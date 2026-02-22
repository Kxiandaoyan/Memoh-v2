# Skills Configuration System

## Overview

The skills configuration system allows you to control which skills are enabled and their display order through a centralized configuration file.

## Configuration File

The configuration file is located at `internal/skills/defaults/skills.config.json` and follows this format:

```json
{
  "version": 1,
  "description": "Description of this configuration",
  "defaults": {
    "skill-name": {
      "order": 10,
      "enabled": true
    }
  }
}
```

### Fields

- **version** (int): Configuration schema version. Current version is 1.
- **description** (string): Human-readable description of this configuration.
- **defaults** (object): Map of skill names to their configuration.

### Skill Configuration Entry

Each skill can have the following properties:

- **order** (int): Sort order for the skill. Lower numbers appear first. Default: 100
- **enabled** (bool): Whether the skill is enabled. Default: true

## Configuration Priority

When determining a skill's configuration, the system uses the following priority (highest to lowest):

1. **Metadata** in the skill's SKILL.md frontmatter
2. **Configuration file** (skills.config.json)
3. **Default values** (order: 100, enabled: true)

### Example: Metadata Override

In your SKILL.md file:

```markdown
---
name: my-skill
description: My custom skill
metadata:
  order: 5
  enabled: false
---
# My Skill Content
```

This will override any configuration in skills.config.json.

## Error Handling

The configuration system is designed to be resilient:

- **Missing configuration file**: Uses default values
- **Invalid JSON**: Falls back to default configuration
- **Incompatible version**: Falls back to default configuration
- **Missing skill in config**: Uses default values (order: 100, enabled: true)

This ensures that the system always starts successfully, even with configuration issues.

## API Changes

The skill listing API now includes two additional fields:

```json
{
  "skills": [
    {
      "name": "skill-name",
      "description": "Skill description",
      "content": "...",
      "metadata": {},
      "order": 10,
      "enabled": true
    }
  ]
}
```

Skills are automatically:
- **Filtered**: Disabled skills (enabled: false) are excluded from the list
- **Sorted**: Skills are sorted by their `order` field (ascending)

## Implementation Details

### Files Modified/Created

1. **internal/skills/config.go** - Configuration loading and merging logic
2. **internal/skills/defaults/skills.config.json** - Default configuration
3. **internal/handlers/skills.go** - Integration with skill loading
4. **internal/skills/config_test.go** - Comprehensive test suite

### Key Functions

- `LoadSkillConfig(path string)` - Loads configuration from a file
- `LoadDefaultSkillConfig()` - Loads configuration from default location
- `GetSkillEntry(skillName, metadata)` - Gets merged configuration for a skill
- `ValidateConfig()` - Validates configuration structure

## Usage Example

To disable a skill, edit `internal/skills/defaults/skills.config.json`:

```json
{
  "version": 1,
  "description": "My custom configuration",
  "defaults": {
    "old-skill": {
      "order": 100,
      "enabled": false
    },
    "important-skill": {
      "order": 1,
      "enabled": true
    }
  }
}
```

The next time skills are loaded:
- `old-skill` will not appear in the list
- `important-skill` will appear first (order: 1)
- Other skills will use default order (100)

## Testing

Run the test suite:

```bash
go test ./internal/skills/...
```

The test suite covers:
- Valid configuration loading
- Missing/invalid file handling
- Version compatibility
- Metadata override priority
- Edge cases (empty names, negative orders, etc.)
