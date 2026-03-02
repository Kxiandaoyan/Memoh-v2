package containerd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	hostResolvConf    = "/etc/resolv.conf"
	systemdResolvConf = "/run/systemd/resolve/resolv.conf"
	fallbackResolv    = "nameserver 1.1.1.1\nnameserver 8.8.8.8\n"
)

// ResolveConfSource returns a host path to mount as /etc/resolv.conf.
// Priority order:
// 1. systemd-resolved config (if available and contains real DNS servers)
// 2. Host's /etc/resolv.conf filtered to remove Docker internal DNS (127.0.0.11)
// 3. Fallback DNS servers (1.1.1.1 and 8.8.8.8)
func ResolveConfSource(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", ErrInvalidArgument
	}

	// Priority 1: Use host's /etc/resolv.conf, but filter out Docker internal DNS
	// When running inside a Docker container, /etc/resolv.conf points to 127.0.0.11
	// which is Docker's embedded DNS. This doesn't work for bot containers using
	// CNI networking (different network namespace), so we need to extract the real
	// upstream DNS servers.
	if _, err := os.Stat(hostResolvConf); err == nil {
		if filtered, err := filterDockerDNS(hostResolvConf, dataDir); err == nil && filtered != "" {
			return filtered, nil
		}
	}

	// Priority 2: Try systemd-resolved config (bare-metal deployments)
	// This contains the real upstream DNS servers, not Docker's internal DNS
	if runtime.GOOS == "darwin" {
		if ok, err := limaFileExists(systemdResolvConf); err != nil {
			return "", err
		} else if ok {
			return systemdResolvConf, nil
		}
	} else if _, err := os.Stat(systemdResolvConf); err == nil {
		return systemdResolvConf, nil
	}

	// Priority 3: Create fallback resolv.conf
	return createFallbackResolv(dataDir)
}

// filterDockerDNS reads a resolv.conf file and filters out Docker's internal DNS (127.0.0.11).
// If real DNS servers are found in comments (e.g., "# ExtServers: [host(127.0.0.53)]"),
// it extracts and uses them. Otherwise, it copies non-Docker nameservers.
// Returns the path to the filtered resolv.conf file.
func filterDockerDNS(resolvPath, dataDir string) (string, error) {
	file, err := os.Open(resolvPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var nameservers []string
	var otherLines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Extract real DNS from Docker's comment: "# ExtServers: [host(127.0.0.53)]"
		if strings.Contains(line, "# ExtServers:") {
			if servers := extractExtServers(line); len(servers) > 0 {
				nameservers = append(nameservers, servers...)
				continue
			}
		}

		// Skip Docker internal DNS
		if strings.HasPrefix(line, "nameserver 127.0.0.11") {
			continue
		}

		// Collect real nameservers
		if strings.HasPrefix(line, "nameserver ") {
			ns := strings.TrimPrefix(line, "nameserver ")
			ns = strings.TrimSpace(ns)
			// Skip localhost addresses (Docker internal DNS)
			if !strings.HasPrefix(ns, "127.") && ns != "::1" {
				nameservers = append(nameservers, ns)
			}
			continue
		}

		// Keep other lines (search, options, etc.)
		if line != "" && !strings.HasPrefix(line, "#") {
			otherLines = append(otherLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	// If no real nameservers found, return empty to trigger fallback
	if len(nameservers) == 0 {
		return "", fmt.Errorf("no real DNS servers found")
	}

	// Write filtered resolv.conf
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", err
	}

	filteredPath := filepath.Join(dataDir, "resolv.conf")
	var content strings.Builder
	content.WriteString("# Filtered resolv.conf (Docker internal DNS removed)\n")
	for _, ns := range nameservers {
		content.WriteString(fmt.Sprintf("nameserver %s\n", ns))
	}
	for _, line := range otherLines {
		content.WriteString(line + "\n")
	}

	if err := os.WriteFile(filteredPath, []byte(content.String()), 0o644); err != nil {
		return "", err
	}

	return filteredPath, nil
}

// extractExtServers parses Docker's ExtServers comment to extract real DNS servers.
// Example: "# ExtServers: [host(127.0.0.53)]" -> ["127.0.0.53"]
func extractExtServers(line string) []string {
	var servers []string
	// Find content between [ and ]
	start := strings.Index(line, "[")
	end := strings.Index(line, "]")
	if start == -1 || end == -1 || start >= end {
		return servers
	}

	content := line[start+1 : end]
	// Split by comma and extract host(...) entries
	parts := strings.Split(content, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "host(") && strings.HasSuffix(part, ")") {
			server := part[5 : len(part)-1] // Extract content between host( and )
			server = strings.TrimSpace(server)
			if server != "" {
				servers = append(servers, server)
			}
		}
	}

	return servers
}

// createFallbackResolv creates a fallback resolv.conf with public DNS servers.
func createFallbackResolv(dataDir string) (string, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", err
	}
	fallbackPath := filepath.Join(dataDir, "resolv.conf")
	if _, err := os.Stat(fallbackPath); err == nil {
		return fallbackPath, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.WriteFile(fallbackPath, []byte(fallbackResolv), 0o644); err != nil {
		return "", err
	}
	return fallbackPath, nil
}

func limaFileExists(path string) (bool, error) {
	if strings.TrimSpace(path) == "" {
		return false, ErrInvalidArgument
	}
	cmd := exec.Command(
		"limactl",
		"shell",
		"--tty=false",
		"default",
		"--",
		"test",
		"-f",
		path,
	)
	if err := cmd.Run(); err == nil {
		return true, nil
	} else if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("lima test failed for %s: %w", path, err)
	} else {
		return false, fmt.Errorf("lima test failed for %s: %w", path, err)
	}
}
