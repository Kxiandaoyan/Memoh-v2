package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncDefaultSkills(t *testing.T) {
	// Create temporary directories
	tmpDir := t.TempDir()
	defaultsDir := filepath.Join(tmpDir, "defaults")
	botSkillsDir := filepath.Join(tmpDir, "bot-skills")

	// Create mock default skills
	skill1Dir := filepath.Join(defaultsDir, "skill1")
	skill2Dir := filepath.Join(defaultsDir, "skill2")
	if err := os.MkdirAll(skill1Dir, 0o755); err != nil {
		t.Fatalf("create skill1 dir: %v", err)
	}
	if err := os.MkdirAll(skill2Dir, 0o755); err != nil {
		t.Fatalf("create skill2 dir: %v", err)
	}

	// Create SKILL.md files
	if err := os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte("skill1 content"), 0o644); err != nil {
		t.Fatalf("write skill1 file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte("skill2 content"), 0o644); err != nil {
		t.Fatalf("write skill2 file: %v", err)
	}

	// Create a subdirectory with assets in skill1
	assetsDir := filepath.Join(skill1Dir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("create assets dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "image.png"), []byte("fake image"), 0o644); err != nil {
		t.Fatalf("write asset file: %v", err)
	}

	// Test initial sync
	t.Run("InitialSync", func(t *testing.T) {
		count, err := SyncDefaultSkills(botSkillsDir, defaultsDir, false)
		if err != nil {
			t.Fatalf("sync failed: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 skills synced, got %d", count)
		}

		// Verify skills were copied
		skill1Content, err := os.ReadFile(filepath.Join(botSkillsDir, "skill1", "SKILL.md"))
		if err != nil {
			t.Fatalf("read skill1: %v", err)
		}
		if string(skill1Content) != "skill1 content" {
			t.Errorf("skill1 content mismatch")
		}

		// Verify assets were copied
		assetContent, err := os.ReadFile(filepath.Join(botSkillsDir, "skill1", "assets", "image.png"))
		if err != nil {
			t.Fatalf("read asset: %v", err)
		}
		if string(assetContent) != "fake image" {
			t.Errorf("asset content mismatch")
		}
	})

	// Test sync without force (should skip existing)
	t.Run("SyncWithoutForce", func(t *testing.T) {
		// Modify skill1 in bot directory
		modifiedContent := []byte("modified skill1")
		if err := os.WriteFile(filepath.Join(botSkillsDir, "skill1", "SKILL.md"), modifiedContent, 0o644); err != nil {
			t.Fatalf("write modified skill1: %v", err)
		}

		// Sync again without force
		count, err := SyncDefaultSkills(botSkillsDir, defaultsDir, false)
		if err != nil {
			t.Fatalf("sync failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 skills synced (all exist), got %d", count)
		}

		// Verify skill1 was not overwritten
		content, err := os.ReadFile(filepath.Join(botSkillsDir, "skill1", "SKILL.md"))
		if err != nil {
			t.Fatalf("read skill1: %v", err)
		}
		if string(content) != "modified skill1" {
			t.Errorf("skill1 was overwritten when it shouldn't be")
		}
	})

	// Test sync with force (should overwrite)
	t.Run("SyncWithForce", func(t *testing.T) {
		// Sync with force
		count, err := SyncDefaultSkills(botSkillsDir, defaultsDir, true)
		if err != nil {
			t.Fatalf("sync failed: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 skills synced, got %d", count)
		}

		// Verify skill1 was overwritten
		content, err := os.ReadFile(filepath.Join(botSkillsDir, "skill1", "SKILL.md"))
		if err != nil {
			t.Fatalf("read skill1: %v", err)
		}
		if string(content) != "skill1 content" {
			t.Errorf("skill1 was not overwritten when it should be")
		}
	})

	// Test adding new skill
	t.Run("AddNewSkill", func(t *testing.T) {
		// Create new skill in defaults
		skill3Dir := filepath.Join(defaultsDir, "skill3")
		if err := os.MkdirAll(skill3Dir, 0o755); err != nil {
			t.Fatalf("create skill3 dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(skill3Dir, "SKILL.md"), []byte("skill3 content"), 0o644); err != nil {
			t.Fatalf("write skill3 file: %v", err)
		}

		// Sync without force
		count, err := SyncDefaultSkills(botSkillsDir, defaultsDir, false)
		if err != nil {
			t.Fatalf("sync failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 new skill synced, got %d", count)
		}

		// Verify new skill was copied
		content, err := os.ReadFile(filepath.Join(botSkillsDir, "skill3", "SKILL.md"))
		if err != nil {
			t.Fatalf("read skill3: %v", err)
		}
		if string(content) != "skill3 content" {
			t.Errorf("skill3 content mismatch")
		}
	})
}

func TestSyncDefaultSkills_SkipHiddenDirs(t *testing.T) {
	tmpDir := t.TempDir()
	defaultsDir := filepath.Join(tmpDir, "defaults")
	botSkillsDir := filepath.Join(tmpDir, "bot-skills")

	// Create a hidden directory (starts with .)
	hiddenDir := filepath.Join(defaultsDir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatalf("create hidden dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "SKILL.md"), []byte("hidden"), 0o644); err != nil {
		t.Fatalf("write hidden file: %v", err)
	}

	// Create a normal skill
	normalDir := filepath.Join(defaultsDir, "normal")
	if err := os.MkdirAll(normalDir, 0o755); err != nil {
		t.Fatalf("create normal dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(normalDir, "SKILL.md"), []byte("normal"), 0o644); err != nil {
		t.Fatalf("write normal file: %v", err)
	}

	// Sync
	count, err := SyncDefaultSkills(botSkillsDir, defaultsDir, false)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 skill synced (hidden should be skipped), got %d", count)
	}

	// Verify hidden was not copied
	if _, err := os.Stat(filepath.Join(botSkillsDir, ".hidden")); !os.IsNotExist(err) {
		t.Errorf("hidden directory should not be copied")
	}

	// Verify normal was copied
	if _, err := os.Stat(filepath.Join(botSkillsDir, "normal")); err != nil {
		t.Errorf("normal directory should be copied")
	}
}

func TestInitializeBotSkills(t *testing.T) {
	tmpDir := t.TempDir()
	dataRoot := filepath.Join(tmpDir, "data")

	// Create defaults directory
	defaultsDir := filepath.Join(tmpDir, "internal", "skills", "defaults")
	if err := os.MkdirAll(defaultsDir, 0o755); err != nil {
		t.Fatalf("create defaults dir: %v", err)
	}

	// Create mock default skills
	skillNames := []string{"skill1", "skill2", "skill3"}
	for i, skillName := range skillNames {
		skillDir := filepath.Join(defaultsDir, skillName)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("create skill dir: %v", err)
		}
		content := []byte("skill content " + string(rune('0'+i+1)))
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), content, 0o644); err != nil {
			t.Fatalf("write skill file: %v", err)
		}
	}

	// Change working directory to tmpDir so findDefaultsDir can find it
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(oldWd)

	// Initialize bot skills
	botID := "test-bot-123"
	if err := InitializeBotSkills(botID, dataRoot); err != nil {
		t.Fatalf("initialize bot skills failed: %v", err)
	}

	// Verify bot skills directory exists
	botSkillsDir := filepath.Join(dataRoot, "bots", botID, ".skills")
	if _, err := os.Stat(botSkillsDir); err != nil {
		t.Fatalf("bot skills directory not created: %v", err)
	}

	// Verify skills were copied
	verifySkillNames := []string{"skill1", "skill2", "skill3"}
	for _, skillName := range verifySkillNames {
		skillPath := filepath.Join(botSkillsDir, skillName, "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			t.Errorf("%s not copied: %v", skillName, err)
		}
	}

	// Verify skills.config.json was created
	configPath := filepath.Join(botSkillsDir, "skills.config.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}

	var config SkillConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if config.Version != DefaultConfigVersion {
		t.Errorf("expected config version %d, got %d", DefaultConfigVersion, config.Version)
	}
	if config.Defaults == nil {
		t.Errorf("config.Defaults should not be nil")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.txt")
	destPath := filepath.Join(tmpDir, "dest.txt")

	// Create source file with specific permissions
	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	// Copy file
	if err := copyFile(srcPath, destPath); err != nil {
		t.Fatalf("copy file failed: %v", err)
	}

	// Verify content
	destContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read dest file: %v", err)
	}
	if string(destContent) != string(content) {
		t.Errorf("content mismatch: expected %q, got %q", content, destContent)
	}

	// Verify permissions
	srcInfo, _ := os.Stat(srcPath)
	destInfo, _ := os.Stat(destPath)
	if srcInfo.Mode() != destInfo.Mode() {
		t.Errorf("permissions mismatch: expected %v, got %v", srcInfo.Mode(), destInfo.Mode())
	}
}

func TestCopyDir(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	// Create source directory structure
	// source/
	//   file1.txt
	//   subdir/
	//     file2.txt
	//     deepdir/
	//       file3.txt
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir", "deepdir"), 0o755); err != nil {
		t.Fatalf("create source structure: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o644); err != nil {
		t.Fatalf("write file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0o644); err != nil {
		t.Fatalf("write file2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "subdir", "deepdir", "file3.txt"), []byte("content3"), 0o644); err != nil {
		t.Fatalf("write file3: %v", err)
	}

	// Copy directory
	if err := copyDir(srcDir, destDir); err != nil {
		t.Fatalf("copy dir failed: %v", err)
	}

	// Verify structure
	testCases := []struct {
		path    string
		content string
	}{
		{"file1.txt", "content1"},
		{"subdir/file2.txt", "content2"},
		{"subdir/deepdir/file3.txt", "content3"},
	}

	for _, tc := range testCases {
		destPath := filepath.Join(destDir, tc.path)
		content, err := os.ReadFile(destPath)
		if err != nil {
			t.Errorf("read %s: %v", tc.path, err)
			continue
		}
		if string(content) != tc.content {
			t.Errorf("%s: expected %q, got %q", tc.path, tc.content, content)
		}
	}
}

func TestSyncDefaultSkills_EmptyParams(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		botSkillsDir  string
		defaultsDir   string
		expectedError string
	}{
		{
			name:          "EmptyBotSkillsDir",
			botSkillsDir:  "",
			defaultsDir:   tmpDir,
			expectedError: "botSkillsDir cannot be empty",
		},
		{
			name:          "EmptyDefaultsDir",
			botSkillsDir:  tmpDir,
			defaultsDir:   "",
			expectedError: "defaultsDir cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SyncDefaultSkills(tt.botSkillsDir, tt.defaultsDir, false)
			if err == nil {
				t.Error("expected error, got nil")
			}
			if err.Error() != tt.expectedError {
				t.Errorf("expected error %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestInitializeBotSkills_EmptyParams(t *testing.T) {
	tests := []struct {
		name          string
		botID         string
		dataRoot      string
		expectedError string
	}{
		{
			name:          "EmptyBotID",
			botID:         "",
			dataRoot:      "/tmp",
			expectedError: "botID cannot be empty",
		},
		{
			name:          "EmptyDataRoot",
			botID:         "bot123",
			dataRoot:      "",
			expectedError: "dataRoot cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitializeBotSkills(tt.botID, tt.dataRoot)
			if err == nil {
				t.Error("expected error, got nil")
			}
			if err.Error() != tt.expectedError {
				t.Errorf("expected error %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestCreateDefaultSkillConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config
	if err := createDefaultSkillConfig(tmpDir); err != nil {
		t.Fatalf("create config failed: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, "skills.config.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Verify content
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var config SkillConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if config.Version != DefaultConfigVersion {
		t.Errorf("expected version %d, got %d", DefaultConfigVersion, config.Version)
	}
	if config.Defaults == nil {
		t.Errorf("Defaults should not be nil")
	}

	// Test that it doesn't overwrite existing config
	modifiedConfig := SkillConfig{
		Version:     DefaultConfigVersion,
		Description: "Modified config",
		Defaults:    map[string]SkillConfigEntry{"test": {Order: 1, Enabled: true}},
	}
	modifiedData, _ := json.MarshalIndent(modifiedConfig, "", "  ")
	if err := os.WriteFile(configPath, modifiedData, 0o644); err != nil {
		t.Fatalf("write modified config: %v", err)
	}

	// Call createDefaultSkillConfig again
	if err := createDefaultSkillConfig(tmpDir); err != nil {
		t.Fatalf("create config failed: %v", err)
	}

	// Verify config was not overwritten
	data, _ = os.ReadFile(configPath)
	var finalConfig SkillConfig
	json.Unmarshal(data, &finalConfig)
	if finalConfig.Description != "Modified config" {
		t.Errorf("config was overwritten when it shouldn't be")
	}
}
