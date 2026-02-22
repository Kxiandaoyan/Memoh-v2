package skills

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// BenchmarkLoadSkillConfig benchmarks loading the skill configuration.
func BenchmarkLoadSkillConfig(b *testing.B) {
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "skills.config.json")

	// Save config to file
	if err := os.WriteFile(configPath, []byte(`{
		"version": 1,
		"description": "Benchmark config",
		"defaults": {
			"skill1": {"order": 1, "enabled": true},
			"skill2": {"order": 2, "enabled": true},
			"skill3": {"order": 3, "enabled": false},
			"skill4": {"order": 4, "enabled": true},
			"skill5": {"order": 5, "enabled": true},
			"skill6": {"order": 6, "enabled": true},
			"skill7": {"order": 7, "enabled": false},
			"skill8": {"order": 8, "enabled": true},
			"skill9": {"order": 9, "enabled": true},
			"skill10": {"order": 10, "enabled": true},
			"skill11": {"order": 11, "enabled": true},
			"skill12": {"order": 12, "enabled": true},
			"skill13": {"order": 13, "enabled": false},
			"skill14": {"order": 14, "enabled": true},
			"skill15": {"order": 15, "enabled": true},
			"skill16": {"order": 16, "enabled": true},
			"skill17": {"order": 17, "enabled": true},
			"skill18": {"order": 18, "enabled": true}
		}
	}`), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = LoadSkillConfig(configPath)
	}
}

// BenchmarkCreateBackup benchmarks creating skill backups.
func BenchmarkCreateBackup(b *testing.B) {
	tmpDir := b.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")
	skillName := "test-skill"

	// Create a skill file
	skillDir := filepath.Join(skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		b.Fatal(err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	content := "---\nname: test-skill\ndescription: Test skill\nversion: 1.0.0\n---\n# Test Skill\n\nThis is a test skill."
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		version := fmt.Sprintf("1.0.%d", i)
		if err := CreateBackup(skillsDir, skillName, version); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListBackups benchmarks listing skill backups.
func BenchmarkListBackups(b *testing.B) {
	tmpDir := b.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")
	skillName := "test-skill"

	// Create a skill file
	skillDir := filepath.Join(skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		b.Fatal(err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	content := "---\nname: test-skill\ndescription: Test skill\nversion: 1.0.0\n---\n# Test Skill\n\nThis is a test skill."
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	// Create 10 backups
	for i := 0; i < 10; i++ {
		version := fmt.Sprintf("1.0.%d", i)
		if err := CreateBackup(skillsDir, skillName, version); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ListBackups(skillsDir, skillName)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHotReloadWatch benchmarks starting and stopping hot reload watchers.
func BenchmarkHotReloadWatch(b *testing.B) {
	tmpDir := b.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		b.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	reloader := NewHotReloader(logger, nil)
	defer reloader.UnwatchAll()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		botID := fmt.Sprintf("bot-%d", i)
		if err := reloader.Watch(ctx, botID, skillsDir); err != nil {
			b.Fatal(err)
		}
		reloader.Unwatch(botID)
	}
}

// BenchmarkConcurrentConfigAccess benchmarks concurrent access to skill configuration.
func BenchmarkConcurrentConfigAccess(b *testing.B) {
	config := &SkillConfig{
		Version:     DefaultConfigVersion,
		Description: "Concurrent access benchmark",
		Defaults: map[string]SkillConfigEntry{
			"skill1":  {Order: 1, Enabled: true},
			"skill2":  {Order: 2, Enabled: true},
			"skill3":  {Order: 3, Enabled: false},
			"skill4":  {Order: 4, Enabled: true},
			"skill5":  {Order: 5, Enabled: true},
			"skill6":  {Order: 6, Enabled: true},
			"skill7":  {Order: 7, Enabled: false},
			"skill8":  {Order: 8, Enabled: true},
			"skill9":  {Order: 9, Enabled: true},
			"skill10": {Order: 10, Enabled: true},
			"skill11": {Order: 11, Enabled: true},
			"skill12": {Order: 12, Enabled: true},
			"skill13": {Order: 13, Enabled: false},
			"skill14": {Order: 14, Enabled: true},
			"skill15": {Order: 15, Enabled: true},
			"skill16": {Order: 16, Enabled: true},
			"skill17": {Order: 17, Enabled: true},
			"skill18": {Order: 18, Enabled: true},
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for i := 1; i <= 18; i++ {
				skillName := fmt.Sprintf("skill%d", i)
				_ = config.GetSkillEntry(skillName, nil)
			}
		}
	})
}

// BenchmarkValidateSkillMetadata benchmarks skill metadata validation.
func BenchmarkValidateSkillMetadata(b *testing.B) {
	metadata := map[string]any{
		"order":       10,
		"enabled":     true,
		"category":    "test",
		"tags":        []string{"tag1", "tag2", "tag3"},
		"author":      "test-author",
		"description": "Test description",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateSkillMetadata(metadata)
	}
}

// BenchmarkConflictDetection benchmarks conflict detection for skills.
func BenchmarkConflictDetection(b *testing.B) {
	tmpDir := b.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")

	// Create some existing skills
	for i := 1; i <= 5; i++ {
		skillName := fmt.Sprintf("skill%d", i)
		skillDir := filepath.Join(skillsDir, skillName)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			b.Fatal(err)
		}
		skillPath := filepath.Join(skillDir, "SKILL.md")
		content := fmt.Sprintf("---\nname: %s\ndescription: Skill %d\nversion: 1.0.0\n---\n# Skill %d", skillName, i, i)
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			b.Fatal(err)
		}
	}

	detector := NewConflictDetector(skillsDir)
	metadata := map[string]any{
		"order":   10,
		"enabled": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectConflicts("new-skill", "1.0.0", metadata)
	}
}

// BenchmarkMultipleSkillsLoad benchmarks loading multiple skills simultaneously.
func BenchmarkMultipleSkillsLoad(b *testing.B) {
	tmpDir := b.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")

	// Create 18 skills
	for i := 1; i <= 18; i++ {
		skillName := fmt.Sprintf("skill%d", i)
		skillDir := filepath.Join(skillsDir, skillName)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			b.Fatal(err)
		}
		skillPath := filepath.Join(skillDir, "SKILL.md")
		content := fmt.Sprintf(`---
name: %s
description: This is skill number %d with some description text
version: 1.0.0
metadata:
  order: %d
  enabled: true
  category: category-%d
  tags: [tag1, tag2, tag3]
---
# Skill %d

This is the main content of skill %d.

## Features
- Feature 1
- Feature 2
- Feature 3

## Usage
Use this skill by invoking it with the proper arguments.

## Examples
Example usage here.
`, skillName, i, i, i%5, i, i)
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			b.Fatal(err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillPath := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
			_, err := os.ReadFile(skillPath)
			if err != nil {
				continue
			}
		}
	}
}

// BenchmarkConcurrentBackupOperations benchmarks concurrent backup creation.
func BenchmarkConcurrentBackupOperations(b *testing.B) {
	tmpDir := b.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")

	// Create multiple skills
	for i := 1; i <= 5; i++ {
		skillName := fmt.Sprintf("skill%d", i)
		skillDir := filepath.Join(skillsDir, skillName)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			b.Fatal(err)
		}
		skillPath := filepath.Join(skillDir, "SKILL.md")
		content := fmt.Sprintf("---\nname: %s\ndescription: Skill %d\nversion: 1.0.0\n---\n# Skill %d", skillName, i, i)
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			skillName := fmt.Sprintf("skill%d", (i%5)+1)
			version := fmt.Sprintf("1.0.%d", i)
			_ = CreateBackup(skillsDir, skillName, version)
			i++
		}
	})
}

// BenchmarkSkillConfigUpdate benchmarks updating skill configuration.
func BenchmarkSkillConfigUpdate(b *testing.B) {
	config := &SkillConfig{
		Version:     DefaultConfigVersion,
		Description: "Update benchmark",
		Defaults:    make(map[string]SkillConfigEntry),
	}

	// Pre-populate with some skills
	for i := 1; i <= 18; i++ {
		skillName := fmt.Sprintf("skill%d", i)
		config.Defaults[skillName] = SkillConfigEntry{Order: i, Enabled: i%2 == 0}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		skillName := fmt.Sprintf("skill%d", (i%18)+1)
		enabled := i%2 == 0
		_ = config.UpdateSkillEnabled(skillName, enabled)
	}
}

// BenchmarkHotReloadTrigger simulates hot reload events.
func BenchmarkHotReloadTrigger(b *testing.B) {
	tmpDir := b.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		b.Fatal(err)
	}

	var triggerCount int
	var mu sync.Mutex
	onChange := func(botID string) {
		mu.Lock()
		triggerCount++
		mu.Unlock()
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	reloader := NewHotReloader(logger, onChange)
	defer reloader.UnwatchAll()

	ctx := context.Background()
	botID := "test-bot"
	if err := reloader.Watch(ctx, botID, skillsDir); err != nil {
		b.Fatal(err)
	}

	// Create a test skill
	skillDir := filepath.Join(skillsDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		skillPath := filepath.Join(skillDir, "SKILL.md")
		content := fmt.Sprintf("---\nname: test-skill\ndescription: Test %d\nversion: 1.0.%d\n---\n# Test", i, i)
		_ = os.WriteFile(skillPath, []byte(content), 0644)
		time.Sleep(1 * time.Millisecond) // Small delay to allow file system event processing
	}

	// Wait a bit for debounced events to settle
	time.Sleep(1 * time.Second)
}
