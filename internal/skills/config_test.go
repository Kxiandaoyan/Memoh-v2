package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSkillConfig(t *testing.T) {
	// Test 1: Loading a valid config file
	t.Run("ValidConfig", func(t *testing.T) {
		// Create a temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "skills.config.json")
		configData := `{
			"version": 1,
			"description": "Test config",
			"defaults": {
				"test-skill": {
					"order": 42,
					"enabled": true
				}
			}
		}`
		if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		config := LoadSkillConfig(configPath)
		if config == nil {
			t.Fatal("Expected config, got nil")
		}
		if config.Version != 1 {
			t.Errorf("Expected version 1, got %d", config.Version)
		}
		if config.Description != "Test config" {
			t.Errorf("Expected 'Test config', got '%s'", config.Description)
		}
		if len(config.Defaults) != 1 {
			t.Errorf("Expected 1 default entry, got %d", len(config.Defaults))
		}
		entry, ok := config.Defaults["test-skill"]
		if !ok {
			t.Fatal("Expected 'test-skill' in defaults")
		}
		if entry.Order != 42 {
			t.Errorf("Expected order 42, got %d", entry.Order)
		}
		if !entry.Enabled {
			t.Error("Expected enabled true")
		}
	})

	// Test 2: Non-existent file returns default config
	t.Run("NonExistentFile", func(t *testing.T) {
		config := LoadSkillConfig("/nonexistent/path/skills.config.json")
		if config == nil {
			t.Fatal("Expected default config, got nil")
		}
		if config.Version != DefaultConfigVersion {
			t.Errorf("Expected default version %d, got %d", DefaultConfigVersion, config.Version)
		}
		if len(config.Defaults) != 0 {
			t.Errorf("Expected empty defaults, got %d entries", len(config.Defaults))
		}
	})

	// Test 3: Invalid JSON returns default config
	t.Run("InvalidJSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid.json")
		if err := os.WriteFile(configPath, []byte("not valid json"), 0644); err != nil {
			t.Fatalf("Failed to create invalid config: %v", err)
		}

		config := LoadSkillConfig(configPath)
		if config == nil {
			t.Fatal("Expected default config, got nil")
		}
		if config.Version != DefaultConfigVersion {
			t.Errorf("Expected default version, got %d", config.Version)
		}
	})

	// Test 4: Incompatible version returns default config
	t.Run("IncompatibleVersion", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "old-version.json")
		configData := `{
			"version": 999,
			"description": "Future version",
			"defaults": {}
		}`
		if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		config := LoadSkillConfig(configPath)
		if config == nil {
			t.Fatal("Expected default config, got nil")
		}
		if config.Version != DefaultConfigVersion {
			t.Errorf("Expected default version %d, got %d", DefaultConfigVersion, config.Version)
		}
	})

	// Test 5: Empty path returns default config
	t.Run("EmptyPath", func(t *testing.T) {
		config := LoadSkillConfig("")
		if config == nil {
			t.Fatal("Expected default config, got nil")
		}
		if config.Version != DefaultConfigVersion {
			t.Errorf("Expected default version, got %d", config.Version)
		}
	})
}

func TestGetSkillEntry(t *testing.T) {
	config := &SkillConfig{
		Version:     1,
		Description: "Test",
		Defaults: map[string]SkillConfigEntry{
			"skill-a": {Order: 10, Enabled: true},
			"skill-b": {Order: 20, Enabled: false},
		},
	}

	// Test 1: Skill in config, no metadata override
	t.Run("ConfigOnly", func(t *testing.T) {
		entry := config.GetSkillEntry("skill-a", nil)
		if entry.Order != 10 {
			t.Errorf("Expected order 10, got %d", entry.Order)
		}
		if !entry.Enabled {
			t.Error("Expected enabled true")
		}
	})

	// Test 2: Skill not in config, use defaults
	t.Run("DefaultValues", func(t *testing.T) {
		entry := config.GetSkillEntry("unknown-skill", nil)
		if entry.Order != DefaultSkillOrder {
			t.Errorf("Expected default order %d, got %d", DefaultSkillOrder, entry.Order)
		}
		if entry.Enabled != DefaultSkillEnabled {
			t.Errorf("Expected default enabled %v, got %v", DefaultSkillEnabled, entry.Enabled)
		}
	})

	// Test 3: Metadata overrides config
	t.Run("MetadataOverride", func(t *testing.T) {
		metadata := map[string]any{
			"order":   5,
			"enabled": false,
		}
		entry := config.GetSkillEntry("skill-a", metadata)
		if entry.Order != 5 {
			t.Errorf("Expected order 5 (from metadata), got %d", entry.Order)
		}
		if entry.Enabled {
			t.Error("Expected enabled false (from metadata)")
		}
	})

	// Test 4: Partial metadata override
	t.Run("PartialMetadataOverride", func(t *testing.T) {
		metadata := map[string]any{
			"order": 99,
		}
		entry := config.GetSkillEntry("skill-b", metadata)
		if entry.Order != 99 {
			t.Errorf("Expected order 99 (from metadata), got %d", entry.Order)
		}
		if entry.Enabled {
			t.Error("Expected enabled false (from config)")
		}
	})

	// Test 5: Metadata with float64 (common in JSON parsing)
	t.Run("MetadataFloat64", func(t *testing.T) {
		metadata := map[string]any{
			"order":   float64(33),
			"enabled": true,
		}
		entry := config.GetSkillEntry("unknown-skill", metadata)
		if entry.Order != 33 {
			t.Errorf("Expected order 33, got %d", entry.Order)
		}
		if !entry.Enabled {
			t.Error("Expected enabled true")
		}
	})
}

func TestValidateConfig(t *testing.T) {
	// Test 1: Valid config
	t.Run("ValidConfig", func(t *testing.T) {
		config := &SkillConfig{
			Version:     1,
			Description: "Valid",
			Defaults: map[string]SkillConfigEntry{
				"skill-a": {Order: 10, Enabled: true},
			},
		}
		if err := config.ValidateConfig(); err != nil {
			t.Errorf("Expected valid config, got error: %v", err)
		}
	})

	// Test 2: Invalid version
	t.Run("InvalidVersion", func(t *testing.T) {
		config := &SkillConfig{
			Version:     999,
			Description: "Invalid version",
			Defaults:    map[string]SkillConfigEntry{},
		}
		if err := config.ValidateConfig(); err == nil {
			t.Error("Expected error for invalid version")
		}
	})

	// Test 3: Negative order
	t.Run("NegativeOrder", func(t *testing.T) {
		config := &SkillConfig{
			Version:     1,
			Description: "Negative order",
			Defaults: map[string]SkillConfigEntry{
				"skill-a": {Order: -1, Enabled: true},
			},
		}
		if err := config.ValidateConfig(); err == nil {
			t.Error("Expected error for negative order")
		}
	})

	// Test 4: Empty skill name
	t.Run("EmptySkillName", func(t *testing.T) {
		config := &SkillConfig{
			Version:     1,
			Description: "Empty name",
			Defaults: map[string]SkillConfigEntry{
				"": {Order: 10, Enabled: true},
			},
		}
		if err := config.ValidateConfig(); err == nil {
			t.Error("Expected error for empty skill name")
		}
	})
}
