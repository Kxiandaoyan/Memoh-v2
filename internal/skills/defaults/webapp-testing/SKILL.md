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

## Guidelines

- Always take a screenshot after key interactions as evidence
- Test both happy paths and error cases
- Use specific CSS selectors over generic ones
- Report failures with the exact selector and expected vs actual state
