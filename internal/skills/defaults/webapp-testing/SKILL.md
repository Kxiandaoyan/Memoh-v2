---
name: webapp-testing
description: Automated web application testing using browser automation. Use when asked to test a web app, verify UI behavior, check for regressions, or automate browser interactions such as clicking, form filling, navigation, and screenshot capture.
---

# Web App Testing

Test web applications through browser automation.

## Quick Start

```bash
# Launch browser and navigate
exec: agent-browser open https://example.com

# Take screenshot
exec: agent-browser screenshot --output /data/screenshot.png

# Click element
exec: agent-browser click --selector "#submit-btn"

# Fill form
exec: agent-browser fill --selector "input[name=email]" --value "test@example.com"

# Assert text present
exec: agent-browser assert-text --text "Success"
```

## Testing Workflow

1. **Setup**: Navigate to the target URL
2. **Interact**: Perform user actions (click, fill, navigate)
3. **Assert**: Verify expected outcomes (text, elements, URL)
4. **Screenshot**: Capture evidence of results
5. **Report**: Summarize pass/fail with screenshots

## Common Patterns

### Login flow
```
open URL → fill username → fill password → click submit → assert dashboard visible
```

### Form submission
```
fill fields → click submit → assert success message or error handling
```

### Navigation
```
click nav link → assert URL changed → assert page content
```

## Validation & Error Handling

- After every form submission, assert that either a success message or a specific error message appears — never assume the action succeeded silently
- Validate that error states render correctly (e.g., inline field errors, toast notifications, disabled buttons)
- When an element is not found, capture a screenshot before reporting failure to aid debugging

### Retry Pattern for Flaky Elements

Elements that load asynchronously (e.g., after an API call or animation) may not be immediately available. Use a wait-and-retry approach:

```
# Wait for element before interacting
exec: agent-browser wait --selector "#dynamic-content" --timeout 5000
exec: agent-browser click --selector "#dynamic-content"

# If assert fails, wait briefly and retry once
exec: agent-browser wait --selector ".result-panel" --timeout 3000
exec: agent-browser assert-text --text "Results loaded"
```

## Guidelines

- Always take a screenshot after key interactions as evidence
- Test both happy paths and error cases
- Use specific CSS selectors over generic ones
- Report failures with the exact selector and expected vs actual state
