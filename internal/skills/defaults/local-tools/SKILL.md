---
name: local-tools
description: "Create, update, delete, and search calendar events and appointments on macOS (Calendar.app) and Windows (Outlook). Use when the user wants to check their schedule, find free time, create meetings or reminders, reschedule appointments, or manage events directly on their device."
---

# Local Tools Skill

## Environment Variables

No environment variables required. This skill directly accesses system resources.

## Prerequisites

| Platform | Requirements |
|----------|-------------|
| **macOS 10.10+** | Calendar.app (included by default); calendar access permission (prompted on first use); manage in System Settings > Privacy & Security > Calendar |
| **Windows 7+** | Microsoft Outlook installed and configured; PowerShell available (included by default); may require COM access permissions in corporate environments |
| **Linux** | Not supported |

## When to Use This Skill

Use local-tools when the user wants to:
- **View schedule** — list today's meetings, check tomorrow's agenda, see the week ahead
- **Create events** — schedule a meeting, add an appointment, set a reminder
- **Update events** — reschedule, change title/location/notes
- **Delete events** — cancel a meeting, remove an appointment
- **Search events** — find events by keyword, look up birthdays or anniversaries
- **Check availability** — see if a time slot is free, find open windows

**Trigger phrases:**
- "Show me my schedule for tomorrow"
- "Create a meeting at 3 PM"
- "Am I free Thursday afternoon?"
- "Search for calendar events containing 'project'"
- "Delete tomorrow's meeting"
- "Reschedule my 2 PM to 4 PM"

## How It Works

```
┌──────────┐    Bash/PowerShell    ┌─────────────────────────────────────────────────────────────┐
│  Claude  │──────────────────────▶│  calendar.sh / calendar.ps1                                 │
│          │                       │  ├─ macOS: osascript -l JavaScript (JXA) ──▶ Calendar.app   │
│          │                       │  └─ Windows: PowerShell ──▶ Outlook COM API                 │
└──────────┘                       └─────────────────────────────────────────────────────────────┘
```

- **macOS**: `calendar.sh` uses JXA (`osascript -l JavaScript`) to control Calendar.app
- **Windows**: `calendar.ps1` uses PowerShell COM API to control Microsoft Outlook
- Both return structured JSON output

## Calendar Operations

**IMPORTANT: How to Locate the Script**

When you read this SKILL.md file using the Read tool, you receive its absolute path (e.g., `/Users/username/.../SKILLs/local-tools/SKILL.md`).

**To construct the script path:**
1. Take the directory of this SKILL.md file
2. Append `/scripts/calendar.sh` (macOS) or `/scripts/calendar.ps1` (Windows)

**Example:**
```bash
# If SKILL.md is at: /Users/username/path/to/SKILLs/local-tools/SKILL.md
# Then the script is: /Users/username/path/to/SKILLs/local-tools/scripts/calendar.sh

bash "/Users/username/path/to/SKILLs/local-tools/scripts/calendar.sh" <operation> [options]
```

In all examples below, `<skill-dir>/scripts/calendar.sh` is a placeholder. Replace it with the actual absolute path.

### List Events

```bash
# List events for next 7 days (default)
bash "<skill-dir>/scripts/calendar.sh" list

# List events for specific date range
bash "<skill-dir>/scripts/calendar.sh" list \
  --start "2026-02-12T00:00:00" \
  --end "2026-02-19T23:59:59"

# List events from specific calendar (macOS)
bash "<skill-dir>/scripts/calendar.sh" list \
  --calendar "Work"
```

### Create Event

```bash
# Create a simple event
bash "<skill-dir>/scripts/calendar.sh" create \
  --title "Team Meeting" \
  --start "2026-02-13T14:00:00" \
  --end "2026-02-13T15:00:00"

# Create event with location and notes
bash "<skill-dir>/scripts/calendar.sh" create \
  --title "Client Call" \
  --start "2026-02-14T10:00:00" \
  --end "2026-02-14T11:00:00" \
  --calendar "Work" \
  --location "Conference Room A" \
  --notes "Discuss Q1 roadmap"
```

### Update Event

```bash
# Update event title
bash "<skill-dir>/scripts/calendar.sh" update \
  --id "EVENT-ID" \
  --title "Updated Meeting Title"

# Update event time
bash "<skill-dir>/scripts/calendar.sh" update \
  --id "EVENT-ID" \
  --start "2026-02-13T15:00:00" \
  --end "2026-02-13T16:00:00"
```

### Delete Event

```bash
bash "<skill-dir>/scripts/calendar.sh" delete \
  --id "EVENT-ID"
```

### Search Events

```bash
# Search for events containing keyword (searches ALL calendars)
bash "<skill-dir>/scripts/calendar.sh" search \
  --query "meeting"

# Search in specific calendar only
bash "<skill-dir>/scripts/calendar.sh" search \
  --query "project" \
  --calendar "Work"
```

**Note:** When `--calendar` is not specified, the search operation will look through **all available calendars** on both macOS and Windows.

## Output Format

All commands return JSON with the following structure:

### Success Response

```json
{
  "success": true,
  "data": {
    "events": [
      {
        "eventId": "E621F8C4-...",
        "title": "Team Meeting",
        "startTime": "2026-02-13T14:00:00.000Z",
        "endTime": "2026-02-13T15:00:00.000Z",
        "location": "Conference Room",
        "notes": "Weekly sync",
        "calendar": "Work",
        "allDay": false
      }
    ],
    "count": 1
  }
}
```

### Error Response

```json
{
  "success": false,
  "error": {
    "code": "CALENDAR_ACCESS_ERROR",
    "message": "Calendar access permission is required...",
    "recoverable": true,
    "permissionRequired": true
  }
}
```

### Error Codes

| Code | Meaning | Recoverable |
|------|---------|-------------|
| `CALENDAR_ACCESS_ERROR` | Permission denied or calendar not accessible | Yes |
| `INVALID_INPUT` | Missing required parameters | No |
| `EVENT_NOT_FOUND` | Event ID not found | No |
| `OUTLOOK_NOT_AVAILABLE` | Microsoft Outlook not installed (Windows) | Yes |

## Date Format Guidelines

When using the `list` command with time ranges:

1. **Always use ISO 8601 format**: `YYYY-MM-DDTHH:mm:ss`
2. **Use local timezone**: Do NOT use UTC or timezone suffixes (like +08:00 or Z)
3. **Calculate dates yourself**: Do NOT use shell command substitution like `$(date ...)`
4. **Claude should compute dates**: Based on current date, calculate target dates directly
5. **Examples**:
   - Today at midnight: `2026-02-13T00:00:00`
   - Today at end of day: `2026-02-13T23:59:59`
   - Tomorrow morning: `2026-02-14T09:00:00`
   - Next week Monday: `2026-02-16T00:00:00`

**Why**: The script expects local time strings that match your system timezone. Shell substitutions may not execute correctly in all environments.

## Common Patterns

### Pattern 1: Schedule Management

```bash
# User asks: "What meetings do I have today?"
# Claude's approach: Calculate today's date and query full day from 00:00 to 23:59
# IMPORTANT: Claude should replace 2026-02-13 with the actual current date
bash "<skill-dir>/scripts/calendar.sh" list \
  --start "2026-02-13T00:00:00" \
  --end "2026-02-13T23:59:59"

# User asks: "What's on my schedule tomorrow?"
# Claude should calculate tomorrow's date (e.g., if today is 2026-02-13, tomorrow is 2026-02-14)
bash "<skill-dir>/scripts/calendar.sh" list \
  --start "2026-02-14T00:00:00" \
  --end "2026-02-14T23:59:59"
```

### Pattern 2: Meeting Scheduling

```bash
# User asks: "Schedule a meeting for tomorrow at 3 PM"
# Claude's approach:
bash "<skill-dir>/scripts/calendar.sh" create \
  --title "Meeting" \
  --start "2026-02-13T15:00:00" \
  --end "2026-02-13T16:00:00" \
  --calendar "Work"
```

### Pattern 3: Event Search

```bash
# User asks: "Find all meetings about the project"
# Claude's approach:
bash "<skill-dir>/scripts/calendar.sh" search \
  --query "project" \
  --calendar "Work"
```

### Pattern 4: Availability Check

```bash
# User asks: "Am I free tomorrow afternoon?"
# Claude's approach:
# 1. List tomorrow's events
# 2. Analyze time slots
# 3. Report availability
bash "<skill-dir>/scripts/calendar.sh" list \
  --start "2026-02-14T00:00:00" \
  --end "2026-02-14T23:59:59"
```

## Known Behaviors

### Time Range Matching

The `list` command uses **interval overlap detection**:
- Returns events that have **any overlap** with the query time range
- Does NOT require events to be fully contained within the range

**Examples:**
- Query: 2026-02-13 00:00:00 to 23:59:59
- Returns:
  - ✅ Events fully on Feb 13 (e.g., 10:00-11:00)
  - ✅ Multi-day events spanning Feb 13 (e.g., Feb 12 10:00 - Feb 14 10:00)
  - ✅ Events crossing midnight (e.g., Feb 13 23:30 - Feb 14 00:30)
  - ❌ Events entirely before Feb 13 (e.g., Feb 12 10:00-11:00)
  - ❌ Events entirely after Feb 13 (e.g., Feb 14 10:00-11:00)

### All-Day Events

- Treated as spanning from 00:00:00 to 23:59:59 on their date(s)
- Multi-day all-day events (e.g., Feb 12-14) will appear when querying any date within that range

### Time Precision

- Comparisons use second-level precision
- Milliseconds are ignored in date comparisons

### Recurring Events

- Each occurrence is treated as a separate event instance
- The script returns individual occurrences within the queried time range

## Best Practices

- **Check before creating**: List existing events to avoid conflicts before creating new ones
- **Search before updating/deleting**: Always search first to get the correct event ID
- **Use specific calendars** (macOS): Specify `--calendar "Work"` to keep events organized
- **Execute directly**: Do not trial-and-error; run the command once and report results
- **Handle errors gracefully**: Parse JSON response; if permission error, tell user to open System Settings > Privacy & Security > Calendar
- **Use default calendar**: If no calendar name specified, the script automatically uses the first available calendar
- **Do not expose internals**: Do not show error stacks, technical details, or script source code to users

## Limitations

- Does not support advanced recurring event queries or modifying recurring event rules
- No support for attendees or meeting invitations
- Windows: Only works with Microsoft Outlook (not Windows Calendar, Google Calendar, etc.)
- All dates must be in ISO 8601 format; return values are converted to UTC

## Troubleshooting

### macOS

**Permission Denied:**
```
Error: Calendar access permission is required
```
**Solution:** Open System Settings > Privacy & Security > Calendar, authorize Terminal or LobsterAI

**Script Not Found:**
```
bash: calendar.sh: No such file or directory
```
**Solution:** Ensure you're using the absolute path from SKILL.md's directory + `/scripts/calendar.sh`

### Windows

**Outlook Not Found:**
```
Error: Microsoft Outlook is not installed or not accessible
```
**Solution:** Install Microsoft Outlook and ensure it's properly configured

**PowerShell Execution Policy:**
```
Error: Execution of scripts is disabled on this system
```
**Solution:** Run PowerShell as Administrator and execute:
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

## Related Skills

- **imap-smtp-email** - For email-based meeting invitations
- **scheduled-task** - For recurring calendar synchronization
