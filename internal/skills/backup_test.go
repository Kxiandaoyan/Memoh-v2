package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateBackup(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")
	skillName := "test-skill"

	// Create a skill file
	skillDir := filepath.Join(skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	skillContent := `---
name: test-skill
description: Test skill
version: 1.0.0
---

# Test Skill

This is a test skill.
`
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("Failed to write skill file: %v", err)
	}

	// Create backup
	if err := CreateBackup(skillsDir, skillName, "1.0.0"); err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify backup exists
	backupDir := filepath.Join(skillsDir, BackupDirName, skillName)
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 backup file, got %d", len(entries))
	}
}

func TestListBackups(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")
	skillName := "test-skill"

	// Create multiple backups
	backupDir := filepath.Join(skillsDir, BackupDirName, skillName)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create backup files
	versions := []string{"1.0.0", "1.1.0", "2.0.0"}
	for _, version := range versions {
		timestamp := time.Now().Add(-time.Hour).Format("20060102-150405")
		filename := "SKILL-" + version + "-" + timestamp + ".md"
		content := "---\nname: test-skill\nversion: " + version + "\n---\n\nTest content"
		if err := os.WriteFile(filepath.Join(backupDir, filename), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write backup file: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// List backups
	backups, err := ListBackups(skillsDir, skillName)
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}

	if len(backups) != len(versions) {
		t.Fatalf("Expected %d backups, got %d", len(versions), len(backups))
	}

	// Verify backups are sorted by timestamp (newest first)
	for i := 0; i < len(backups)-1; i++ {
		if backups[i].Timestamp.Before(backups[i+1].Timestamp) {
			t.Errorf("Backups not sorted correctly: %v should be after %v",
				backups[i].Timestamp, backups[i+1].Timestamp)
		}
	}
}

func TestRollbackSkill(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")
	skillName := "test-skill"

	// Create backup
	backupDir := filepath.Join(skillsDir, BackupDirName, skillName)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	oldVersion := "1.0.0"
	oldContent := "---\nname: test-skill\nversion: 1.0.0\n---\n\nOld version"
	timestamp := time.Now().Format("20060102-150405")
	backupFile := "SKILL-" + oldVersion + "-" + timestamp + ".md"
	if err := os.WriteFile(filepath.Join(backupDir, backupFile), []byte(oldContent), 0o644); err != nil {
		t.Fatalf("Failed to write backup file: %v", err)
	}

	// Create current skill
	skillDir := filepath.Join(skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	currentContent := "---\nname: test-skill\nversion: 2.0.0\n---\n\nNew version"
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(currentContent), 0o644); err != nil {
		t.Fatalf("Failed to write skill file: %v", err)
	}

	// Rollback
	if err := RollbackSkill(skillsDir, skillName, oldVersion); err != nil {
		t.Fatalf("RollbackSkill failed: %v", err)
	}

	// Verify rollback
	restoredContent, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("Failed to read restored skill file: %v", err)
	}

	if string(restoredContent) != oldContent {
		t.Errorf("Rollback content mismatch.\nExpected: %s\nGot: %s", oldContent, string(restoredContent))
	}
}

func TestCleanupOldBackups(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create 15 backup files
	for i := 0; i < 15; i++ {
		filename := filepath.Join(backupDir, fmt.Sprintf("backup-%02d.md", i))
		if err := os.WriteFile(filename, []byte("test"), 0o644); err != nil {
			t.Fatalf("Failed to write backup file: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Cleanup, keep only 10
	if err := cleanupOldBackups(backupDir, 10); err != nil {
		t.Fatalf("cleanupOldBackups failed: %v", err)
	}

	// Verify count
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}

	if len(entries) != 10 {
		t.Errorf("Expected 10 backups after cleanup, got %d", len(entries))
	}
}
