package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultConfigVersion is the current version of the skills configuration format.
	DefaultConfigVersion = 1

	// DefaultSkillOrder is the default order value for skills not specified in config.
	DefaultSkillOrder = 100

	// DefaultSkillEnabled is the default enabled state for skills not specified in config.
	DefaultSkillEnabled = true
)

// SkillConfig represents the configuration for all skills.
type SkillConfig struct {
	Version     int                          `json:"version"`
	Description string                       `json:"description"`
	Defaults    map[string]SkillConfigEntry  `json:"defaults"`
}

// SkillConfigEntry represents the configuration for a single skill.
type SkillConfigEntry struct {
	Order   int  `json:"order"`
	Enabled bool `json:"enabled"`
}

// LoadSkillConfig loads the skill configuration from the specified file path.
// If the file does not exist or cannot be parsed, it returns a default configuration
// without returning an error, ensuring system stability.
func LoadSkillConfig(path string) *SkillConfig {
	// Default configuration if file doesn't exist or fails to load
	defaultConfig := &SkillConfig{
		Version:     DefaultConfigVersion,
		Description: "Default configuration (config file not loaded)",
		Defaults:    make(map[string]SkillConfigEntry),
	}

	// If path is empty, return default
	if path == "" {
		return defaultConfig
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return defaultConfig
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		// Return default config on read error
		return defaultConfig
	}

	// Parse JSON
	var config SkillConfig
	if err := json.Unmarshal(data, &config); err != nil {
		// Return default config on parse error
		return defaultConfig
	}

	// Validate version
	if config.Version != DefaultConfigVersion {
		// Incompatible version, return default config
		return defaultConfig
	}

	// Ensure defaults map exists
	if config.Defaults == nil {
		config.Defaults = make(map[string]SkillConfigEntry)
	}

	return &config
}

// GetSkillEntry returns the configuration entry for a specific skill.
// Priority: metadata > config.json > default values
func (sc *SkillConfig) GetSkillEntry(skillName string, metadata map[string]any) SkillConfigEntry {
	entry := SkillConfigEntry{
		Order:   DefaultSkillOrder,
		Enabled: DefaultSkillEnabled,
	}

	// First, check config.json
	if configEntry, ok := sc.Defaults[skillName]; ok {
		entry.Order = configEntry.Order
		entry.Enabled = configEntry.Enabled
	}

	// Then, override with metadata if present
	if metadata != nil {
		if orderVal, ok := metadata["order"]; ok {
			switch v := orderVal.(type) {
			case int:
				entry.Order = v
			case float64:
				entry.Order = int(v)
			case int64:
				entry.Order = int(v)
			}
		}
		if enabledVal, ok := metadata["enabled"]; ok {
			if boolVal, ok := enabledVal.(bool); ok {
				entry.Enabled = boolVal
			}
		}
	}

	return entry
}

// LoadDefaultSkillConfig loads the skill configuration from the default location.
// This is a convenience function that locates the config file relative to the
// skills package.
func LoadDefaultSkillConfig() *SkillConfig {
	// Try to find the config file relative to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return LoadSkillConfig("")
	}

	execDir := filepath.Dir(execPath)

	// Try multiple potential locations
	possiblePaths := []string{
		filepath.Join(execDir, "internal", "skills", "defaults", "skills.config.json"),
		filepath.Join(execDir, "skills", "defaults", "skills.config.json"),
		"internal/skills/defaults/skills.config.json",
		"skills/defaults/skills.config.json",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return LoadSkillConfig(path)
		}
	}

	// If none found, return default config
	return LoadSkillConfig("")
}

// ValidateConfig validates the skill configuration.
func (sc *SkillConfig) ValidateConfig() error {
	if sc.Version != DefaultConfigVersion {
		return fmt.Errorf("unsupported config version: %d (expected %d)", sc.Version, DefaultConfigVersion)
	}

	// Validate each entry
	for name, entry := range sc.Defaults {
		if name == "" {
			return fmt.Errorf("skill name cannot be empty")
		}
		if entry.Order < 0 {
			return fmt.Errorf("skill '%s': order must be non-negative", name)
		}
	}

	return nil
}

// UpdateSkillEnabled updates the enabled state of a skill in the configuration.
func (sc *SkillConfig) UpdateSkillEnabled(skillName string, enabled bool) error {
	if skillName == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	if sc.Defaults == nil {
		sc.Defaults = make(map[string]SkillConfigEntry)
	}

	entry := sc.Defaults[skillName]
	entry.Enabled = enabled
	sc.Defaults[skillName] = entry

	return nil
}

// UpdateSkillOrder updates the order of a skill in the configuration.
func (sc *SkillConfig) UpdateSkillOrder(skillName string, order int) error {
	if skillName == "" {
		return fmt.Errorf("skill name cannot be empty")
	}
	if order < 0 {
		return fmt.Errorf("order must be non-negative")
	}

	if sc.Defaults == nil {
		sc.Defaults = make(map[string]SkillConfigEntry)
	}

	entry := sc.Defaults[skillName]
	entry.Order = order
	sc.Defaults[skillName] = entry

	return nil
}

// SaveSkillConfigTo saves the skill configuration to the specified file path (atomic write).
func SaveSkillConfigTo(config *SkillConfig, configPath string) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if err := config.ValidateConfig(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmpPath := configPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}

	if err := os.Rename(tmpPath, configPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

// SaveSkillConfig saves the skill configuration to the specified bot's data directory.
// Deprecated: Use SaveSkillConfigTo with an explicit path instead.
func SaveSkillConfig(config *SkillConfig, botID string) error {
	if botID == "" {
		return fmt.Errorf("botID cannot be empty")
	}
	dataRoot := os.Getenv("MCP_DATA_ROOT")
	if dataRoot == "" {
		dataRoot = "./data"
	}
	configPath := filepath.Join(dataRoot, "bots", botID, ".skills", "skills.config.json")
	return SaveSkillConfigTo(config, configPath)
}
