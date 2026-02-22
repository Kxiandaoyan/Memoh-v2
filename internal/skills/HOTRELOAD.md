# Skill Hot-Reloading and Version Management

This document describes the skill hot-reloading and version management system implemented in the Memoh-v2-saas backend.

## Features

### 1. Version Control

Every skill now supports versioning through the YAML frontmatter:

```yaml
---
name: my-skill
description: My awesome skill
version: 1.0.0
metadata:
  order: 10
  enabled: true
---

# Skill content here
```

- **Version field**: Follows semantic versioning (e.g., `1.0.0`, `2.1.3`)
- **Automatic versioning**: If no version is specified, defaults to `1.0.0`
- **Version comparison**: System compares versions when updating skills to detect conflicts

### 2. Hot-Reloading

Skills can be reloaded without restarting the service using file system monitoring.

**Implementation**:
- Uses `fsnotify` to monitor the `.skills/` directory for changes
- Watches for file creates, writes, removes, and renames
- Debounces rapid changes (500ms) to avoid excessive reloads
- Automatically watches new subdirectories when created

**API Endpoint**:
```
POST /bots/{bot_id}/container/skills/reload
```

**Request**:
```json
{
  "force": false  // Optional: force reload even if no changes detected
}
```

**Response**:
```json
{
  "ok": true
}
```

### 3. Conflict Detection

The system detects conflicts before upserting skills:

**Conflict Types**:
1. **Duplicate Name**: Skill with same name already exists
2. **Version Mismatch**: Attempting to install older version over newer one
3. **Dependency Missing**: Required dependency skill not installed

**API Endpoint**:
```
POST /bots/{bot_id}/container/skills/conflicts
```

**Request**:
```json
{
  "name": "my-skill",
  "version": "1.0.0",
  "metadata": {
    "dependencies": ["other-skill"]
  }
}
```

**Response**:
```json
{
  "has_conflicts": true,
  "conflicts": [
    {
      "type": 2,
      "message": "Existing version 2.0.0 is newer than 1.0.0",
      "details": "Skill 'my-skill' has version 2.0.0, attempting to install 1.0.0"
    }
  ]
}
```

### 4. Backup and Rollback

**Automatic Backups**:
- Created automatically before any skill modification
- Stored in `.skills/.backups/{skill-name}/`
- Filename format: `SKILL-{version}-{timestamp}.md`
- Keeps last 10 backups per skill (configurable)

**List Backups Endpoint**:
```
GET /bots/{bot_id}/container/skills/{name}/backups
```

**Response**:
```json
{
  "backups": [
    {
      "skill_name": "my-skill",
      "version": "1.0.0",
      "timestamp": "2026-02-22T10:30:00Z",
      "file_path": "/data/bots/bot123/.skills/.backups/my-skill/SKILL-1.0.0-20260222-103000.md"
    }
  ]
}
```

**Rollback Endpoint**:
```
POST /bots/{bot_id}/container/skills/{name}/rollback
```

**Request**:
```json
{
  "version": "1.0.0"
}
```

**Response**:
```json
{
  "ok": true
}
```

## Architecture

### Modules

1. **hotreload.go**: File system monitoring and hot-reload logic
   - `HotReloader`: Main hot-reload manager
   - `Watch()`: Start watching a bot's skills directory
   - `Unwatch()`: Stop watching a directory

2. **backup.go**: Backup and rollback functionality
   - `CreateBackup()`: Create skill backup
   - `ListBackups()`: List all backups for a skill
   - `RollbackSkill()`: Restore skill from backup
   - `DeleteBackups()`: Clean up backups

3. **conflict.go**: Conflict detection
   - `ConflictDetector`: Detects skill conflicts
   - `DetectConflicts()`: Check for conflicts before upsert
   - `ValidateSkillMetadata()`: Validate metadata

### Handler Integration

The `skills.go` handler integrates all features:
- Updated `UpsertSkills()` to create backups and detect conflicts
- Added new endpoints for reload, rollback, and conflict detection
- Enhanced `SkillItem` struct with `Version` field
- Updated `parseSkillFile()` to parse version from frontmatter

## Usage Examples

### 1. Enable Hot-Reload for a Bot

```go
import (
    "github.com/Kxiandaoyan/Memoh-v2-saas/internal/skills"
)

// Create hot-reloader
hotReloader := skills.NewHotReloader(logger, func(botID string) {
    log.Printf("Skills changed for bot: %s", botID)
    // Trigger skill reload logic here
})

// Start watching
err := hotReloader.Watch(ctx, "bot123", "/data/bots/bot123/.skills")
if err != nil {
    log.Fatal(err)
}

// Stop watching when done
defer hotReloader.Unwatch("bot123")
```

### 2. Create Skill with Version

```bash
curl -X POST http://localhost:8080/bots/bot123/container/skills \
  -H "Content-Type: application/json" \
  -d '{
    "skills": [
      {
        "name": "my-skill",
        "description": "My skill",
        "version": "1.0.0",
        "content": "---\nname: my-skill\nversion: 1.0.0\n---\n\n# My Skill"
      }
    ]
  }'
```

### 3. Check for Conflicts Before Installing

```bash
curl -X POST http://localhost:8080/bots/bot123/container/skills/conflicts \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-skill",
    "version": "1.0.0",
    "metadata": {
      "dependencies": ["other-skill"]
    }
  }'
```

### 4. Rollback to Previous Version

```bash
# List backups
curl http://localhost:8080/bots/bot123/container/skills/my-skill/backups

# Rollback
curl -X POST http://localhost:8080/bots/bot123/container/skills/my-skill/rollback \
  -H "Content-Type: application/json" \
  -d '{"version": "1.0.0"}'
```

## Configuration

### Environment Variables

- `MCP_DATA_ROOT`: Root directory for bot data (default: `./data`)

### Constants (in code)

- `BackupDirName`: Directory name for backups (default: `.backups`)
- `MaxBackupsPerSkill`: Maximum backups per skill (default: `10`)
- Debounce interval for hot-reload: `500ms`

## Security Considerations

1. **Path Validation**: All skill names are validated to prevent path traversal attacks
2. **Metadata Validation**: Reserved metadata keys (`_internal`, `_system`) are blocked
3. **Backup Cleanup**: Old backups are automatically cleaned up to prevent disk space exhaustion
4. **Version Comparison**: Uses simple string comparison for version numbers

## Performance

- **File Watching**: Minimal overhead, uses OS-native file system notifications
- **Debouncing**: Prevents excessive reloads during rapid file changes
- **Backup Rotation**: Automatic cleanup keeps disk usage bounded

## Testing

Run tests:
```bash
go test ./internal/skills/...
```

Tests cover:
- Backup creation and restoration
- Conflict detection
- Version comparison
- Metadata validation
- Hot-reload functionality

## Future Enhancements

1. **Dependency Resolution**: Automatic installation of missing dependencies
2. **Version Constraints**: Support for version ranges (e.g., `>=1.0.0, <2.0.0`)
3. **Migration Scripts**: Auto-migration when skill schema changes
4. **Skill Registry**: Central registry for published skills
5. **Delta Updates**: Only reload changed skills instead of all skills
