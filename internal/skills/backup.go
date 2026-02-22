package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// BackupDirName is the directory name for skill backups.
	BackupDirName = ".backups"

	// MaxBackupsPerSkill is the maximum number of backups to keep per skill.
	MaxBackupsPerSkill = 10
)

// BackupInfo represents metadata about a skill backup.
type BackupInfo struct {
	SkillName string    `json:"skill_name"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	FilePath  string    `json:"file_path"`
}

// CreateBackup creates a backup of a skill before modification.
func CreateBackup(skillsDir, skillName, version string) error {
	if skillName == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	// Read current skill file
	skillPath := filepath.Join(skillsDir, skillName, "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Skill doesn't exist yet, no backup needed
			return nil
		}
		return fmt.Errorf("read skill file: %w", err)
	}

	// Ensure backup directory exists
	backupDir := filepath.Join(skillsDir, BackupDirName, skillName)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	// Generate backup filename with timestamp and version
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("SKILL-%s-%s.md", version, timestamp)
	backupPath := filepath.Join(backupDir, backupName)

	// Write backup file
	if err := os.WriteFile(backupPath, content, 0o644); err != nil {
		return fmt.Errorf("write backup file: %w", err)
	}

	// Clean up old backups
	if err := cleanupOldBackups(backupDir, MaxBackupsPerSkill); err != nil {
		// Log but don't fail on cleanup error
		return nil
	}

	return nil
}

// ListBackups returns all backups for a skill, sorted by timestamp (newest first).
func ListBackups(skillsDir, skillName string) ([]BackupInfo, error) {
	backupDir := filepath.Join(skillsDir, BackupDirName, skillName)

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, fmt.Errorf("read backup directory: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Parse filename: SKILL-{version}-{timestamp}.md
		parts := strings.SplitN(entry.Name(), "-", 3)
		if len(parts) < 3 {
			continue
		}

		version := parts[1]
		timestampStr := strings.TrimSuffix(parts[2], ".md")

		timestamp, err := time.Parse("20060102-150405", timestampStr)
		if err != nil {
			continue
		}

		backups = append(backups, BackupInfo{
			SkillName: skillName,
			Version:   version,
			Timestamp: timestamp,
			FilePath:  filepath.Join(backupDir, entry.Name()),
		})
	}

	// Sort by timestamp, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// RollbackSkill restores a skill from a backup.
func RollbackSkill(skillsDir, skillName, toVersion string) error {
	if skillName == "" {
		return fmt.Errorf("skill name cannot be empty")
	}
	if toVersion == "" {
		return fmt.Errorf("version cannot be empty")
	}

	// Find the backup
	backups, err := ListBackups(skillsDir, skillName)
	if err != nil {
		return fmt.Errorf("list backups: %w", err)
	}

	var targetBackup *BackupInfo
	for i := range backups {
		if backups[i].Version == toVersion {
			targetBackup = &backups[i]
			break
		}
	}

	if targetBackup == nil {
		return fmt.Errorf("backup for version %s not found", toVersion)
	}

	// Read backup content
	content, err := os.ReadFile(targetBackup.FilePath)
	if err != nil {
		return fmt.Errorf("read backup file: %w", err)
	}

	// Create backup of current version before rollback
	currentSkillPath := filepath.Join(skillsDir, skillName, "SKILL.md")
	if _, err := os.Stat(currentSkillPath); err == nil {
		_ = CreateBackup(skillsDir, skillName, "rollback-"+time.Now().Format("20060102-150405"))
	}

	// Write rolled-back content
	skillDir := filepath.Join(skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return fmt.Errorf("create skill directory: %w", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, content, 0o644); err != nil {
		return fmt.Errorf("write skill file: %w", err)
	}

	return nil
}

// cleanupOldBackups removes old backups, keeping only the most recent N.
func cleanupOldBackups(backupDir string, keepCount int) error {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return err
	}

	// Filter only .md files
	var files []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry)
		}
	}

	// If we have fewer than keepCount, nothing to clean up
	if len(files) <= keepCount {
		return nil
	}

	// Sort by modification time
	type fileWithTime struct {
		name    string
		modTime time.Time
	}
	var filesWithTime []fileWithTime
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			continue
		}
		filesWithTime = append(filesWithTime, fileWithTime{
			name:    f.Name(),
			modTime: info.ModTime(),
		})
	}

	sort.Slice(filesWithTime, func(i, j int) bool {
		return filesWithTime[i].modTime.After(filesWithTime[j].modTime)
	})

	// Remove old files
	for i := keepCount; i < len(filesWithTime); i++ {
		filePath := filepath.Join(backupDir, filesWithTime[i].name)
		_ = os.Remove(filePath)
	}

	return nil
}

// DeleteBackups removes all backups for a skill.
func DeleteBackups(skillsDir, skillName string) error {
	backupDir := filepath.Join(skillsDir, BackupDirName, skillName)
	if err := os.RemoveAll(backupDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete backup directory: %w", err)
	}
	return nil
}
