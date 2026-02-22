package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectConflicts_DuplicateName(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")
	skillName := "test-skill"

	// Create existing skill
	skillDir := filepath.Join(skillsDir, skillName)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	existingContent := "---\nname: test-skill\nversion: 2.0.0\n---\n\nExisting skill"
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(existingContent), 0o644); err != nil {
		t.Fatalf("Failed to write skill file: %v", err)
	}

	// Detect conflicts when trying to install older version
	detector := NewConflictDetector(skillsDir)
	conflicts, err := detector.DetectConflicts(skillName, "1.0.0", nil)
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}

	if len(conflicts) == 0 {
		t.Errorf("Expected version conflict, but got none")
	}

	// Check for version mismatch conflict
	foundVersionConflict := false
	for _, conflict := range conflicts {
		if conflict.Type == ConflictVersionMismatch {
			foundVersionConflict = true
			break
		}
	}

	if !foundVersionConflict {
		t.Errorf("Expected ConflictVersionMismatch, but didn't find it")
	}
}

func TestDetectConflicts_MissingDependency(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")

	detector := NewConflictDetector(skillsDir)

	// Test with array of dependencies
	metadata := map[string]any{
		"dependencies": []any{"missing-skill-1", "missing-skill-2"},
	}

	conflicts, err := detector.DetectConflicts("new-skill", "1.0.0", metadata)
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}

	if len(conflicts) != 2 {
		t.Errorf("Expected 2 dependency conflicts, got %d", len(conflicts))
	}

	for _, conflict := range conflicts {
		if conflict.Type != ConflictDependencyMissing {
			t.Errorf("Expected ConflictDependencyMissing, got %d", conflict.Type)
		}
	}
}

func TestDetectConflicts_NoDependencyConflict(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".skills")

	// Create dependency skill
	depSkillName := "dependency-skill"
	depSkillDir := filepath.Join(skillsDir, depSkillName)
	if err := os.MkdirAll(depSkillDir, 0o755); err != nil {
		t.Fatalf("Failed to create dependency skill directory: %v", err)
	}

	depContent := "---\nname: dependency-skill\nversion: 1.0.0\n---\n\nDependency skill"
	depPath := filepath.Join(depSkillDir, "SKILL.md")
	if err := os.WriteFile(depPath, []byte(depContent), 0o644); err != nil {
		t.Fatalf("Failed to write dependency skill file: %v", err)
	}

	detector := NewConflictDetector(skillsDir)

	// Test with existing dependency
	metadata := map[string]any{
		"dependencies": []any{depSkillName},
	}

	conflicts, err := detector.DetectConflicts("new-skill", "1.0.0", metadata)
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}

	// Should have no dependency conflicts
	for _, conflict := range conflicts {
		if conflict.Type == ConflictDependencyMissing {
			t.Errorf("Unexpected dependency conflict: %s", conflict.Message)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.1", "1.0.0", 1},
		{"v1.0.0", "v2.0.0", -1},
		{"1.0", "1.0.0", -1},
	}

	for _, tt := range tests {
		result := compareVersions(tt.v1, tt.v2)
		if result != tt.expected {
			t.Errorf("compareVersions(%s, %s) = %d, expected %d",
				tt.v1, tt.v2, result, tt.expected)
		}
	}
}

func TestValidateSkillMetadata(t *testing.T) {
	tests := []struct {
		name      string
		metadata  map[string]any
		expectErr bool
	}{
		{
			name:      "nil metadata",
			metadata:  nil,
			expectErr: false,
		},
		{
			name:      "valid metadata",
			metadata:  map[string]any{"order": 1, "enabled": true},
			expectErr: false,
		},
		{
			name:      "reserved key _internal",
			metadata:  map[string]any{"_internal": "value"},
			expectErr: true,
		},
		{
			name:      "reserved key _system",
			metadata:  map[string]any{"_system": "value"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSkillMetadata(tt.metadata)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateSkillMetadata() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}
