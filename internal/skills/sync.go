package skills

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	// SkillsConfigFileName is the name of the skills configuration file.
	SkillsConfigFileName = "skills.config.json"

	// SkillFileName is the name of the main skill file.
	SkillFileName = "SKILL.md"
)

// SyncDefaultSkills synchronizes default skills to a bot's skills directory.
// It reads all skills from defaultsDir and copies them to botSkillsDir.
// If force=true, it will overwrite existing skills.
// If force=false, it will skip existing skills to preserve bot-specific customizations.
// Returns the count of skills synced and any error encountered.
func SyncDefaultSkills(botSkillsDir, defaultsDir string, force bool) (int, error) {
	if botSkillsDir == "" {
		return 0, fmt.Errorf("botSkillsDir cannot be empty")
	}
	if defaultsDir == "" {
		return 0, fmt.Errorf("defaultsDir cannot be empty")
	}

	// Ensure bot skills directory exists
	if err := os.MkdirAll(botSkillsDir, 0o755); err != nil {
		return 0, fmt.Errorf("create bot skills directory: %w", err)
	}

	// Read all entries in defaults directory
	entries, err := os.ReadDir(defaultsDir)
	if err != nil {
		return 0, fmt.Errorf("read defaults directory: %w", err)
	}

	syncCount := 0
	for _, entry := range entries {
		// Skip files (we only care about skill directories)
		if !entry.IsDir() {
			continue
		}

		skillName := entry.Name()

		// Skip special directories (like .backups or .git)
		if len(skillName) > 0 && skillName[0] == '.' {
			continue
		}

		// Source and destination paths
		srcSkillDir := filepath.Join(defaultsDir, skillName)
		destSkillDir := filepath.Join(botSkillsDir, skillName)

		// Check if skill already exists
		if _, err := os.Stat(destSkillDir); err == nil {
			// Skill exists
			if !force {
				// Skip to preserve customizations
				continue
			}
			// force=true: we'll overwrite
		} else if !os.IsNotExist(err) {
			// Some other error occurred
			return syncCount, fmt.Errorf("stat skill directory %s: %w", destSkillDir, err)
		}

		// Copy the entire skill directory
		if err := copyDir(srcSkillDir, destSkillDir); err != nil {
			return syncCount, fmt.Errorf("copy skill %s: %w", skillName, err)
		}

		syncCount++
	}

	return syncCount, nil
}

// InitializeBotSkills initializes a bot's skills directory with all default skills.
// This function is called when creating a new bot.
// It copies all default skills and creates a default skills.config.json.
func InitializeBotSkills(botID, dataRoot string) error {
	if botID == "" {
		return fmt.Errorf("botID cannot be empty")
	}
	if dataRoot == "" {
		return fmt.Errorf("dataRoot cannot be empty")
	}

	// Construct paths
	botSkillsDir := filepath.Join(dataRoot, "bots", botID, ".skills")

	// Determine defaults directory
	// Try to find it relative to the current executable or working directory
	defaultsDir, err := findDefaultsDir()
	if err != nil {
		return fmt.Errorf("find defaults directory: %w", err)
	}

	// Sync all default skills with force=true (for new bot initialization)
	count, err := SyncDefaultSkills(botSkillsDir, defaultsDir, true)
	if err != nil {
		return fmt.Errorf("sync default skills: %w", err)
	}

	// Create a default skills.config.json for the bot
	if err := createDefaultSkillConfig(botSkillsDir); err != nil {
		return fmt.Errorf("create default skill config: %w", err)
	}

	// Log success (in production, this would go to a proper logger)
	fmt.Printf("Initialized bot %s with %d default skills\n", botID, count)

	return nil
}

// copyDir recursively copies a directory tree.
// It preserves file permissions and copies all files and subdirectories.
func copyDir(src, dest string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	// Create destination directory with same permissions
	if err := os.MkdirAll(dest, srcInfo.Mode()); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}

	// Read source directory entries
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read source directory: %w", err)
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, destPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dest, preserving permissions.
func copyFile(src, dest string) error {
	// Get source file info for permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source file: %w", err)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file with same permissions
	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy content
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("copy file content: %w", err)
	}

	// Ensure data is written to disk
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("sync destination file: %w", err)
	}

	return nil
}

// findDefaultsDir attempts to locate the defaults directory.
// It tries multiple locations relative to the executable and working directory.
func findDefaultsDir() (string, error) {
	// Try relative to current working directory first
	cwd, err := os.Getwd()
	if err == nil {
		possiblePaths := []string{
			filepath.Join(cwd, "internal", "skills", "defaults"),
			filepath.Join(cwd, "skills", "defaults"),
		}
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	// Try relative to executable
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		possiblePaths := []string{
			filepath.Join(execDir, "internal", "skills", "defaults"),
			filepath.Join(execDir, "skills", "defaults"),
		}
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("defaults directory not found")
}

// createDefaultSkillConfig creates a default skills.config.json for a new bot.
// This file should be minimal and let the bot customize as needed.
func createDefaultSkillConfig(botSkillsDir string) error {
	configPath := filepath.Join(botSkillsDir, SkillsConfigFileName)

	// Don't overwrite existing config
	if _, err := os.Stat(configPath); err == nil {
		return nil // Config already exists, skip
	}

	// Create a minimal default configuration
	config := SkillConfig{
		Version:     DefaultConfigVersion,
		Description: "Bot-specific skill configuration",
		Defaults:    make(map[string]SkillConfigEntry),
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
