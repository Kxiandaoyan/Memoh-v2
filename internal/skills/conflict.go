package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConflictType represents the type of skill conflict detected.
type ConflictType int

const (
	// ConflictNone indicates no conflict.
	ConflictNone ConflictType = iota
	// ConflictDuplicateName indicates a skill with the same name already exists.
	ConflictDuplicateName
	// ConflictVersionMismatch indicates a version conflict.
	ConflictVersionMismatch
	// ConflictDependencyMissing indicates a required dependency is missing.
	ConflictDependencyMissing
)

// Conflict represents a skill conflict.
type Conflict struct {
	Type    ConflictType `json:"type"`
	Message string       `json:"message"`
	Details string       `json:"details,omitempty"`
}

// ConflictDetector detects conflicts in skills.
type ConflictDetector struct {
	skillsDir string
}

// NewConflictDetector creates a new ConflictDetector.
func NewConflictDetector(skillsDir string) *ConflictDetector {
	return &ConflictDetector{
		skillsDir: skillsDir,
	}
}

// DetectConflicts checks for conflicts when upserting a skill.
func (cd *ConflictDetector) DetectConflicts(skillName string, newVersion string, newMetadata map[string]any) ([]Conflict, error) {
	var conflicts []Conflict

	// Check for duplicate name
	skillPath := filepath.Join(cd.skillsDir, skillName, "SKILL.md")
	if _, err := os.Stat(skillPath); err == nil {
		// Skill exists, check version
		existingContent, err := os.ReadFile(skillPath)
		if err == nil {
			parsed := parseSkillFileForConflict(string(existingContent), skillName)

			// Compare versions
			if parsed.Version != "" && newVersion != "" {
				cmp := compareVersions(parsed.Version, newVersion)
				if cmp > 0 {
					conflicts = append(conflicts, Conflict{
						Type:    ConflictVersionMismatch,
						Message: fmt.Sprintf("Existing version %s is newer than %s", parsed.Version, newVersion),
						Details: fmt.Sprintf("Skill '%s' has version %s, attempting to install %s", skillName, parsed.Version, newVersion),
					})
				}
			}
		}
	}

	// Check dependencies if specified in metadata
	if newMetadata != nil {
		if deps, ok := newMetadata["dependencies"].([]any); ok {
			for _, dep := range deps {
				if depName, ok := dep.(string); ok {
					if !cd.skillExists(depName) {
						conflicts = append(conflicts, Conflict{
							Type:    ConflictDependencyMissing,
							Message: fmt.Sprintf("Required dependency '%s' is missing", depName),
							Details: fmt.Sprintf("Skill '%s' requires '%s' to be installed", skillName, depName),
						})
					}
				}
			}
		} else if deps, ok := newMetadata["dependencies"].(string); ok {
			// Handle comma-separated string
			depNames := strings.Split(deps, ",")
			for _, depName := range depNames {
				depName = strings.TrimSpace(depName)
				if depName != "" && !cd.skillExists(depName) {
					conflicts = append(conflicts, Conflict{
						Type:    ConflictDependencyMissing,
						Message: fmt.Sprintf("Required dependency '%s' is missing", depName),
						Details: fmt.Sprintf("Skill '%s' requires '%s' to be installed", skillName, depName),
					})
				}
			}
		}
	}

	return conflicts, nil
}

// skillExists checks if a skill exists in the skills directory.
func (cd *ConflictDetector) skillExists(skillName string) bool {
	skillPath := filepath.Join(cd.skillsDir, skillName, "SKILL.md")
	_, err := os.Stat(skillPath)
	return err == nil
}

// parseSkillFileForConflict is a simplified parser for conflict detection.
func parseSkillFileForConflict(raw string, fallbackName string) struct {
	Name    string
	Version string
} {
	result := struct {
		Name    string
		Version string
	}{
		Name:    fallbackName,
		Version: "1.0.0",
	}

	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "---") {
		return result
	}

	// Find closing "---"
	rest := trimmed[3:]
	closingIdx := strings.Index(rest, "\n---")
	if closingIdx < 0 {
		return result
	}

	frontmatterRaw := rest[:closingIdx]

	// Simple key-value parsing (avoid YAML dependency here)
	lines := strings.Split(frontmatterRaw, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "name" && value != "" {
			result.Name = value
		} else if key == "version" && value != "" {
			result.Version = value
		}
	}

	return result
}

// compareVersions compares two semantic version strings.
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
// This is a simplified comparison that handles basic semantic versioning.
func compareVersions(v1, v2 string) int {
	// Remove 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Split by dots
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		// Simple string comparison for numeric parts
		if parts1[i] < parts2[i] {
			return -1
		} else if parts1[i] > parts2[i] {
			return 1
		}
	}

	// If all compared parts are equal, longer version is considered newer
	if len(parts1) < len(parts2) {
		return -1
	} else if len(parts1) > len(parts2) {
		return 1
	}

	return 0
}

// ValidateSkillMetadata validates skill metadata for common issues.
func ValidateSkillMetadata(metadata map[string]any) error {
	if metadata == nil {
		return nil
	}

	// Check for reserved keys
	reservedKeys := []string{"_internal", "_system"}
	for _, key := range reservedKeys {
		if _, ok := metadata[key]; ok {
			return fmt.Errorf("metadata key '%s' is reserved", key)
		}
	}

	return nil
}
