# Local Calendar Tool - PowerShell Implementation for Windows
# Supports Microsoft Outlook COM API

param(
    [Parameter(Position=0, Mandatory=$true)]
    [ValidateSet('list', 'create', 'update', 'delete', 'search')]
    [string]$Operation,

    [string]$Title,
    [string]$Start,
    [string]$End,
    [string]$Id,
    [string]$Calendar = 'Calendar',
    [string]$Location,
    [string]$Notes,
    [string]$Query
)

$ErrorActionPreference = 'Stop'

# Output JSON result
function Output-Result($success, $data, $error) {
    $result = @{ success = $success }
    if ($data) { $result.data = $data }
    if ($error) { $result.error = $error }
    $result | ConvertTo-Json -Depth 10 -Compress
}

# Check Outlook availability
function Test-OutlookAvailable {
    try {
        $Outlook = New-Object -ComObject Outlook.Application
        $null = $Outlook.Version
        return $true
    } catch {
        return $false
    }
}

# List events
function Get-CalendarEvents {
    param($StartTime, $EndTime, $CalendarName)

    try {
        $StartTime = if ($StartTime) { [DateTime]::Parse($StartTime) } else { Get-Date }
        $EndTime = if ($EndTime) { [DateTime]::Parse($EndTime) } else { (Get-Date).AddDays(7) }

        $Outlook = New-Object -ComObject Outlook.Application
        $Namespace = $Outlook.GetNamespace('MAPI')
        $CalendarFolder = $Namespace.GetDefaultFolder(9) # olFolderCalendar

        $Items = $CalendarFolder.Items
        $Items.IncludeRecurrences = $true
        $Items.Sort('[Start]')

        # Use standard interval overlap logic instead of strict containment
        # This ensures we catch events that span multiple days or cross midnight
        # Filter: Event overlaps with range if event starts before range ends AND event ends after range starts
        $Filter = "[Start] < '$($EndTime.ToString('g'))' AND [End] > '$($StartTime.ToString('g'))'"
        $FilteredItems = $Items.Restrict($Filter)

        $Events = @()
        foreach ($Item in $FilteredItems) {
            $Events += @{
                eventId = $Item.EntryID
                title = $Item.Subject
                startTime = $Item.Start.ToUniversalTime().ToString('o')
                endTime = $Item.End.ToUniversalTime().ToString('o')
                location = $Item.Location
                notes = $Item.Body
                calendar = $CalendarName
                allDay = $Item.AllDayEvent
            }
        }

        Output-Result -success $true -data @{ events = $Events; count = $Events.Count }
    } catch {
        Output-Result -success $false -error @{ code = 'CALENDAR_ACCESS_ERROR'; message = $_.Exception.Message; recoverable = $true }
    }
}

# Create event
function New-CalendarEvent {
    param($Title, $StartTime, $EndTime, $Location, $Notes, $CalendarName)

    if (-not $Title -or -not $StartTime -or -not $EndTime) {
        Output-Result -success $false -error @{ code = 'INVALID_INPUT'; message = 'title, start, and end are required'; recoverable = $false }
        return
    }

    try {
        $Outlook = New-Object -ComObject Outlook.Application
        $Appointment = $Outlook.CreateItem(1) # olAppointmentItem

        $Appointment.Subject = $Title
        $Appointment.Start = [DateTime]::Parse($StartTime)
        $Appointment.End = [DateTime]::Parse($EndTime)
        if ($Location) { $Appointment.Location = $Location }
        if ($Notes) { $Appointment.Body = $Notes }

        $Appointment.Save()

        Output-Result -success $true -data @{ eventId = $Appointment.EntryID; message = 'Event created successfully' }
    } catch {
        Output-Result -success $false -error @{ code = 'CALENDAR_ACCESS_ERROR'; message = $_.Exception.Message; recoverable = $true }
    }
}

# Update event
function Set-CalendarEvent {
    param($Id, $Title, $StartTime, $EndTime, $Location, $Notes)

    if (-not $Id) {
        Output-Result -success $false -error @{ code = 'INVALID_INPUT'; message = 'id is required'; recoverable = $false }
        return
    }

    try {
        $Outlook = New-Object -ComObject Outlook.Application
        $Namespace = $Outlook.GetNamespace('MAPI')
        $Appointment = $Namespace.GetItemFromID($Id)

        if ($Title) { $Appointment.Subject = $Title }
        if ($StartTime) { $Appointment.Start = [DateTime]::Parse($StartTime) }
        if ($EndTime) { $Appointment.End = [DateTime]::Parse($EndTime) }
        if ($Location) { $Appointment.Location = $Location }
        if ($Notes) { $Appointment.Body = $Notes }

        $Appointment.Save()

        Output-Result -success $true -data @{ eventId = $Appointment.EntryID; message = 'Event updated successfully' }
    } catch {
        Output-Result -success $false -error @{ code = 'CALENDAR_ACCESS_ERROR'; message = $_.Exception.Message; recoverable = $true }
    }
}

# Delete event
function Remove-CalendarEvent {
    param($Id)

    if (-not $Id) {
        Output-Result -success $false -error @{ code = 'INVALID_INPUT'; message = 'id is required'; recoverable = $false }
        return
    }

    try {
        $Outlook = New-Object -ComObject Outlook.Application
        $Namespace = $Outlook.GetNamespace('MAPI')
        $Appointment = $Namespace.GetItemFromID($Id)

        $Appointment.Delete()

        Output-Result -success $true -data @{ message = 'Event deleted successfully' }
    } catch {
        Output-Result -success $false -error @{ code = 'CALENDAR_ACCESS_ERROR'; message = $_.Exception.Message; recoverable = $true }
    }
}

# Search events in a single folder
function Search-CalendarFolder {
    param($Folder, $Query, $FolderName)
    
    $Items = $Folder.Items
    $Items.Sort('[Start]')
    
    $SearchQuery = $Query.ToLower()
    $Results = @()
    
    foreach ($Item in $Items) {
        try {
            $MatchSubject = $Item.Subject -and $Item.Subject.ToLower().Contains($SearchQuery)
            $MatchBody = $Item.Body -and $Item.Body.ToLower().Contains($SearchQuery)
            $MatchLocation = $Item.Location -and $Item.Location.ToLower().Contains($SearchQuery)
            
            if ($MatchSubject -or $MatchBody -or $MatchLocation) {
                $Results += @{
                    eventId = $Item.EntryID
                    title = $Item.Subject
                    startTime = $Item.Start.ToUniversalTime().ToString('o')
                    endTime = $Item.End.ToUniversalTime().ToString('o')
                    location = $Item.Location
                    notes = $Item.Body
                    calendar = $FolderName
                }
            }
        } catch {
            # Skip items that can't be accessed
        }
    }
    
    return $Results
}

# Search events across all calendars
function Find-CalendarEvents {
    param($Query, $CalendarName)

    if (-not $Query) {
        Output-Result -success $false -error @{ code = 'INVALID_INPUT'; message = 'query is required'; recoverable = $false }
        return
    }

    try {
        $Outlook = New-Object -ComObject Outlook.Application
        $Namespace = $Outlook.GetNamespace('MAPI')
        
        $AllEvents = @()
        
        if ($CalendarName) {
            # Search specific calendar
            $CalendarFolder = $Namespace.GetDefaultFolder(9) # olFolderCalendar
            $Results = Search-CalendarFolder -Folder $CalendarFolder -Query $Query -FolderName $CalendarFolder.Name
            $AllEvents += $Results
        } else {
            # Search all calendar folders
            # Start with default calendar
            try {
                $DefaultCalendar = $Namespace.GetDefaultFolder(9)
                $Results = Search-CalendarFolder -Folder $DefaultCalendar -Query $Query -FolderName $DefaultCalendar.Name
                $AllEvents += $Results
            } catch {
                Write-Host "Warning: Could not access default calendar: $($_.Exception.Message)"
            }
            
            # Search additional calendar folders in the mailbox
            try {
                $RootFolder = $Namespace.Folders
                foreach ($Folder in $RootFolder) {
                    try {
                        # Try to get calendar folders from each root folder
                        $CalendarFolders = @()
                        
                        # Check if this folder has a Calendar subfolder
                        foreach ($SubFolder in $Folder.Folders) {
                            if ($SubFolder.DefaultItemType -eq 1 -or $SubFolder.Name -like '*Calendar*') {
                                # 1 = olAppointmentItem
                                $CalendarFolders += $SubFolder
                            }
                        }
                        
                        foreach ($CalFolder in $CalendarFolders) {
                            try {
                                $Results = Search-CalendarFolder -Folder $CalFolder -Query $Query -FolderName $CalFolder.Name
                                $AllEvents += $Results
                            } catch {
                                # Skip folders that can't be accessed
                            }
                        }
                    } catch {
                        # Skip folders that can't be accessed
                    }
                }
            } catch {
                Write-Host "Warning: Could not enumerate all calendars: $($_.Exception.Message)"
            }
        }

        Output-Result -success $true -data @{ events = $AllEvents; count = $AllEvents.Count }
    } catch {
        Output-Result -success $false -error @{ code = 'CALENDAR_ACCESS_ERROR'; message = $_.Exception.Message; recoverable = $true }
    }
}

# Main
if (-not (Test-OutlookAvailable)) {
    Output-Result -success $false -error @{ code = 'OUTLOOK_NOT_AVAILABLE'; message = 'Microsoft Outlook is not installed or not accessible'; recoverable = $true }
    exit 1
}

switch ($Operation) {
    'list' { Get-CalendarEvents -StartTime $Start -EndTime $End -CalendarName $Calendar }
    'create' { New-CalendarEvent -Title $Title -StartTime $Start -EndTime $End -Location $Location -Notes $Notes -CalendarName $Calendar }
    'update' { Set-CalendarEvent -Id $Id -Title $Title -StartTime $Start -EndTime $End -Location $Location -Notes $Notes }
    'delete' { Remove-CalendarEvent -Id $Id }
    'search' { Find-CalendarEvents -Query $Query -CalendarName $Calendar }
    default {
        Output-Result -success $false -error @{ code = 'INVALID_OPERATION'; message = "Unknown operation: $Operation"; recoverable = $false }
    }
}
