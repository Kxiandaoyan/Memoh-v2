#!/bin/bash
# Local Calendar Tool - Pure Script Implementation
# Supports macOS (Intel & Apple Silicon) and Windows (via Outlook)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect platform
PLATFORM="$(uname -s)"
case "${PLATFORM}" in
    Linux*)     PLATFORM=Linux;;
    Darwin*)    PLATFORM=Mac;;
    CYGWIN*)    PLATFORM=Windows;;
    MINGW*)     PLATFORM=Windows;;
    MSYS*)      PLATFORM=Windows;;
    *)          PLATFORM="UNKNOWN:${PLATFORM}"
esac

# Usage information
usage() {
    cat << EOF
Usage: $(basename "$0") <operation> [options]

Operations:
  list                List calendar events
  create              Create a new event
  update              Update an existing event
  delete              Delete an event
  search              Search for events

Options for 'list':
  --start <datetime>    Start time (ISO 8601, default: now)
  --end <datetime>      End time (ISO 8601, default: 7 days from now)
  --calendar <name>     Calendar name (default: first available calendar)

Options for 'create':
  --title <string>      Event title (required)
  --start <datetime>    Start time (ISO 8601, required)
  --end <datetime>      End time (ISO 8601, required)
  --calendar <name>     Calendar name (default: first available calendar)
  --location <string>   Location
  --notes <string>      Notes/description

Options for 'update':
  --id <string>         Event ID (required)
  --title <string>      New title
  --start <datetime>    New start time
  --end <datetime>      New end time
  --location <string>   New location
  --notes <string>      New notes
  --calendar <name>     Calendar name (default: first available calendar)

Options for 'delete':
  --id <string>         Event ID (required)
  --calendar <name>     Calendar name (default: first available calendar)

Options for 'search':
  --query <string>      Search query (required)
  --calendar <name>     Calendar name (default: search all calendars)

Examples:
  # List events for next 7 days
  $(basename "$0") list

  # Create a new event
  $(basename "$0") create --title "Team Meeting" --start "2026-02-13T14:00:00" --end "2026-02-13T15:00:00"

  # Search for events
  $(basename "$0") search --query "meeting"

Platform: ${PLATFORM}
EOF
    exit 1
}

# Parse arguments
parse_args() {
    OPERATION=""
    TITLE=""
    START_TIME=""
    END_TIME=""
    EVENT_ID=""
    CALENDAR=""
    LOCATION=""
    NOTES=""
    QUERY=""

    OPERATION="$1"
    shift

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --title)
                TITLE="$2"
                shift 2
                ;;
            --start)
                START_TIME="$2"
                shift 2
                ;;
            --end)
                END_TIME="$2"
                shift 2
                ;;
            --id)
                EVENT_ID="$2"
                shift 2
                ;;
            --calendar)
                CALENDAR="$2"
                shift 2
                ;;
            --location)
                LOCATION="$2"
                shift 2
                ;;
            --notes)
                NOTES="$2"
                shift 2
                ;;
            --query)
                QUERY="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
}

# ==================== macOS Implementation ====================

# Escape string for JXA
escape_jxa() {
    echo "$1" | sed 's/\\/\\\\/g; s/"/\\"/g; s/'"'"'/\\'"'"'/g'
}

# macOS: List events
macos_list_events() {
    local start_time="${START_TIME:-$(date +%Y-%m-%dT%H:%M:%S)}"
    local end_time="${END_TIME:-$(date -v+7d +%Y-%m-%dT%H:%M:%S 2>/dev/null || date -d '+7 days' +%Y-%m-%dT%H:%M:%S)}"
    local calendar="${CALENDAR:-}"

    local script="
        const Calendar = Application('Calendar');

        try {
            // Get calendar - use specified name or first available
            let targetCal;
            const calName = '$(escape_jxa "$calendar")';
            if (calName && calName !== '') {
                targetCal = Calendar.calendars.byName(calName);
            } else {
                const cals = Calendar.calendars();
                if (cals.length === 0) {
                    throw new Error('No calendars found');
                }
                targetCal = cals[0];
            }

            // Parse dates
            const startStr = '$start_time';
            const endStr = '$end_time';

            // Get all events and filter manually (more compatible than whose)
            const allEvents = targetCal.events();
            const result = [];

            for (let i = 0; i < allEvents.length; i++) {
                try {
                    const event = allEvents[i];
                    const eventStart = event.startDate();
                    const eventEnd = event.endDate();

                    // Format dates for comparison (YYYY-MM-DDTHH:mm:ss)
                    const formatDate = (d) => {
                        return d.getFullYear() + '-' +
                               String(d.getMonth() + 1).padStart(2, '0') + '-' +
                               String(d.getDate()).padStart(2, '0') + 'T' +
                               String(d.getHours()).padStart(2, '0') + ':' +
                               String(d.getMinutes()).padStart(2, '0') + ':' +
                               String(d.getSeconds()).padStart(2, '0');
                    };

                    const eventStartStr = formatDate(eventStart);
                    const eventEndStr = formatDate(eventEnd);

                    // Check if event overlaps with query range (standard interval overlap algorithm)
                    if (eventStartStr < endStr && eventEndStr > startStr) {
                        // Infer allDay status from time components (avoid JXA allday() method issues)
                        const isAllDay = (eventStart.getHours() === 0 && eventStart.getMinutes() === 0 &&
                                        eventEnd.getHours() === 0 && eventEnd.getMinutes() === 0);

                        result.push({
                            eventId: event.id(),
                            title: event.summary(),
                            startTime: eventStart.toISOString(),
                            endTime: eventEnd.toISOString(),
                            location: event.location() || '',
                            notes: event.description() || '',
                            calendar: targetCal.name(),
                            allDay: isAllDay
                        });
                    }
                } catch (eventError) {
                    // Skip events that can't be accessed
                }
            }

            JSON.stringify({ success: true, data: { events: result, count: result.length } });
        } catch (e) {
            JSON.stringify({ success: false, error: { code: 'CALENDAR_ACCESS_ERROR', message: e.message, recoverable: true } });
        }
    "

    osascript -l JavaScript -e "$script" 2>&1
}

# macOS: Create event
macos_create_event() {
    if [[ -z "$TITLE" || -z "$START_TIME" || -z "$END_TIME" ]]; then
        echo '{"success":false,"error":{"code":"INVALID_INPUT","message":"title, start, and end are required","recoverable":false}}'
        exit 1
    fi

    local calendar="${CALENDAR:-}"

    local script="
        const Calendar = Application('Calendar');
        
        try {
            // Get calendar - use specified name or first available
            let targetCal;
            const calName = '$(escape_jxa "$calendar")';
            if (calName && calName !== '') {
                targetCal = Calendar.calendars.byName(calName);
            } else {
                const cals = Calendar.calendars();
                if (cals.length === 0) {
                    throw new Error('No calendars found');
                }
                targetCal = cals[0];
            }

            const event = Calendar.Event({
                summary: '$(escape_jxa "$TITLE")',
                startDate: new Date('$START_TIME'),
                endDate: new Date('$END_TIME'),
                location: '$(escape_jxa "$LOCATION")',
                description: '$(escape_jxa "$NOTES")'
            });

            targetCal.events.push(event);

            JSON.stringify({
                success: true,
                data: {
                    eventId: event.id(),
                    message: 'Event created successfully'
                }
            });
        } catch (e) {
            JSON.stringify({ success: false, error: { code: 'CALENDAR_ACCESS_ERROR', message: e.message, recoverable: true } });
        }
    "

    osascript -l JavaScript -e "$script" 2>&1
}

# macOS: Update event
macos_update_event() {
    if [[ -z "$EVENT_ID" ]]; then
        echo '{"success":false,"error":{"code":"INVALID_INPUT","message":"id is required","recoverable":false}}'
        exit 1
    fi

    local calendar="${CALENDAR:-}"
    local updates=""

    [[ -n "$TITLE" ]] && updates="${updates}event.summary = '$(escape_jxa "$TITLE")';"
    [[ -n "$START_TIME" ]] && updates="${updates}event.startDate = new Date('$START_TIME');"
    [[ -n "$END_TIME" ]] && updates="${updates}event.endDate = new Date('$END_TIME');"
    [[ -n "$LOCATION" ]] && updates="${updates}event.location = '$(escape_jxa "$LOCATION")';"
    [[ -n "$NOTES" ]] && updates="${updates}event.description = '$(escape_jxa "$NOTES")';"

    local script="
        const Calendar = Application('Calendar');
        
        try {
            // Get calendar - use specified name or first available
            let targetCal;
            const calName = '$(escape_jxa "$calendar")';
            if (calName && calName !== '') {
                targetCal = Calendar.calendars.byName(calName);
            } else {
                const cals = Calendar.calendars();
                if (cals.length === 0) {
                    throw new Error('No calendars found');
                }
                targetCal = cals[0];
            }

            const event = targetCal.events.byId('$EVENT_ID');
            $updates

            JSON.stringify({
                success: true,
                data: {
                    eventId: event.id(),
                    message: 'Event updated successfully'
                }
            });
        } catch (e) {
            JSON.stringify({ success: false, error: { code: 'CALENDAR_ACCESS_ERROR', message: e.message, recoverable: true } });
        }
    "

    osascript -l JavaScript -e "$script" 2>&1
}

# macOS: Delete event
macos_delete_event() {
    if [[ -z "$EVENT_ID" ]]; then
        echo '{"success":false,"error":{"code":"INVALID_INPUT","message":"id is required","recoverable":false}}'
        exit 1
    fi

    local calendar="${CALENDAR:-}"

    local script="
        const Calendar = Application('Calendar');
        
        try {
            // Get calendar - use specified name or first available
            let targetCal;
            const calName = '$(escape_jxa "$calendar")';
            if (calName && calName !== '') {
                targetCal = Calendar.calendars.byName(calName);
            } else {
                const cals = Calendar.calendars();
                if (cals.length === 0) {
                    throw new Error('No calendars found');
                }
                targetCal = cals[0];
            }

            const event = targetCal.events.byId('$EVENT_ID');
            event.delete();

            JSON.stringify({
                success: true,
                data: {
                    message: 'Event deleted successfully'
                }
            });
        } catch (e) {
            JSON.stringify({ success: false, error: { code: 'CALENDAR_ACCESS_ERROR', message: e.message, recoverable: true } });
        }
    "

    osascript -l JavaScript -e "$script" 2>&1
}

# macOS: Search events
macos_search_events() {
    if [[ -z "$QUERY" ]]; then
        echo '{"success":false,"error":{"code":"INVALID_INPUT","message":"query is required","recoverable":false}}'
        exit 1
    fi

    local calendar="${CALENDAR:-}"

    local script="
        const Calendar = Application('Calendar');
        const query = '$(escape_jxa "$QUERY")'.toLowerCase();
        
        try {
            const calName = '$(escape_jxa "$calendar")';
            const allCalendars = Calendar.calendars();
            
            if (allCalendars.length === 0) {
                throw new Error('No calendars found');
            }
            
            // Determine which calendars to search
            let calendarsToSearch = [];
            if (calName && calName !== '') {
                // Search specific calendar
                calendarsToSearch = [Calendar.calendars.byName(calName)];
            } else {
                // Search all calendars
                calendarsToSearch = allCalendars;
            }
            
            // Search in all selected calendars
            let allResults = [];
            for (let i = 0; i < calendarsToSearch.length; i++) {
                try {
                    const cal = calendarsToSearch[i];
                    const events = cal.events.whose({
                        _or: [
                            {summary: {_contains: query}},
                            {description: {_contains: query}},
                            {location: {_contains: query}}
                        ]
                    })();
                    
                    const calResults = events.map(event => ({
                        eventId: event.id(),
                        title: event.summary(),
                        startTime: event.startDate().toISOString(),
                        endTime: event.endDate().toISOString(),
                        location: event.location() || '',
                        notes: event.description() || '',
                        calendar: cal.name()
                    }));
                    
                    allResults = allResults.concat(calResults);
                } catch (calError) {
                    // Skip calendars that can't be accessed
                    console.log('Skipping calendar due to error: ' + calError.message);
                }
            }

            JSON.stringify({ success: true, data: { events: allResults, count: allResults.length } });
        } catch (e) {
            JSON.stringify({ success: false, error: { code: 'CALENDAR_ACCESS_ERROR', message: e.message, recoverable: true } });
        }
    "

    osascript -l JavaScript -e "$script" 2>&1
}

# ==================== Windows Implementation ====================

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PS_SCRIPT="$SCRIPT_DIR/calendar.ps1"

# Windows: Execute PowerShell script
windows_execute() {
    local op="$1"
    local args=""

    # Build arguments
    [[ -n "$TITLE" ]] && args="$args -Title '$TITLE'"
    [[ -n "$START_TIME" ]] && args="$args -Start '$START_TIME'"
    [[ -n "$END_TIME" ]] && args="$args -End '$END_TIME'"
    [[ -n "$EVENT_ID" ]] && args="$args -Id '$EVENT_ID'"
    [[ -n "$CALENDAR" ]] && args="$args -Calendar '$CALENDAR'"
    [[ -n "$LOCATION" ]] && args="$args -Location '$LOCATION'"
    [[ -n "$NOTES" ]] && args="$args -Notes '$NOTES'"
    [[ -n "$QUERY" ]] && args="$args -Query '$QUERY'"

    # Escape single quotes for PowerShell by doubling them
    args=$(echo "$args" | sed "s/'/''/g")

    # shellcheck disable=SC2086
    powershell -ExecutionPolicy Bypass -File "$PS_SCRIPT" -Operation "$op" $args 2>&1
}

# Windows: List events via Outlook
windows_list_events() {
    windows_execute "list"
}

# Windows: Create event via Outlook
windows_create_event() {
    if [[ -z "$TITLE" || -z "$START_TIME" || -z "$END_TIME" ]]; then
        echo '{"success":false,"error":{"code":"INVALID_INPUT","message":"title, start, and end are required","recoverable":false}}'
        exit 1
    fi
    windows_execute "create"
}

# Windows: Update event via Outlook
windows_update_event() {
    if [[ -z "$EVENT_ID" ]]; then
        echo '{"success":false,"error":{"code":"INVALID_INPUT","message":"id is required","recoverable":false}}'
        exit 1
    fi
    windows_execute "update"
}

# Windows: Delete event via Outlook
windows_delete_event() {
    if [[ -z "$EVENT_ID" ]]; then
        echo '{"success":false,"error":{"code":"INVALID_INPUT","message":"id is required","recoverable":false}}'
        exit 1
    fi
    windows_execute "delete"
}

# Windows: Search events via Outlook
windows_search_events() {
    if [[ -z "$QUERY" ]]; then
        echo '{"success":false,"error":{"code":"INVALID_INPUT","message":"query is required","recoverable":false}}'
        exit 1
    fi
    windows_execute "search"
}

# ==================== Permission Helper ====================

# Check if error is permission-related
is_permission_error() {
    local error_msg="$1"
    [[ "$error_msg" == *"不能获取对象"* ]] || \
    [[ "$error_msg" == *"not authorized"* ]] || \
    [[ "$error_msg" == *"Permission denied"* ]] || \
    [[ "$error_msg" == *"Access denied"* ]] || \
    [[ "$error_msg" == *"CALENDAR_ACCESS_ERROR"* ]]
}

# Try to trigger permission dialog on macOS (development helper)
try_trigger_permission() {
    if [[ "$PLATFORM" == "Mac" ]]; then
        # Try to access Calendar to trigger system permission dialog
        osascript -l JavaScript -e 'Application("Calendar").name()' 2>/dev/null || true
    fi
}

# ==================== Main ====================

main() {
    if [[ $# -lt 1 ]]; then
        usage
    fi

    parse_args "$@"

    local result=""
    local exit_code=0

    case "$PLATFORM" in
        Mac)
            case "$OPERATION" in
                list)   result=$(macos_list_events) ;;
                create) result=$(macos_create_event) ;;
                update) result=$(macos_update_event) ;;
                delete) result=$(macos_delete_event) ;;
                search) result=$(macos_search_events) ;;
                *)      usage ;;
            esac
            ;;
        Windows)
            case "$OPERATION" in
                list)   result=$(windows_list_events) ;;
                create) result=$(windows_create_event) ;;
                update) result=$(windows_update_event) ;;
                delete) result=$(windows_delete_event) ;;
                search) result=$(windows_search_events) ;;
                *)      usage ;;
            esac
            ;;
        *)
            echo "{\"success\":false,\"error\":{\"code\":\"PLATFORM_NOT_SUPPORTED\",\"message\":\"Platform $PLATFORM is not supported\",\"recoverable\":false}}"
            exit 1
            ;;
    esac

    # Check if result indicates permission error
    if is_permission_error "$result"; then
        # Try to trigger permission dialog (non-blocking)
        try_trigger_permission
        
        # Return enhanced error message
        echo "{\"success\":false,\"error\":{\"code\":\"CALENDAR_ACCESS_ERROR\",\"message\":\"Calendar permission required. Please grant access in System Settings > Privacy & Security > Calendars, then try again.\",\"recoverable\":true,\"permissionRequired\":true}}"
        exit 1
    fi

    echo "$result"
}

main "$@"
