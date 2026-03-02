package containerd

import (
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
// 1. Host's /etc/resolv.conf (works in both bare-metal and container environments)
// 2. systemd-resolved config (if available)
// 3. Fallback DNS servers (1.1.1.1 and 8.8.8.8)
func ResolveConfSource(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", ErrInvalidArgument
	}

	// Priority 1: Use host's /etc/resolv.conf
	// This works reliably in both bare-metal and containerized environments.
	// When running inside a container (e.g., memoh-server), this gives us the
	// container's DNS config which is typically configured by Docker/containerd
	// to use the host's DNS or Docker's embedded DNS (127.0.0.11).
	if _, err := os.Stat(hostResolvConf); err == nil {
		return hostResolvConf, nil
	}

	// Priority 2: Try systemd-resolved config
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
