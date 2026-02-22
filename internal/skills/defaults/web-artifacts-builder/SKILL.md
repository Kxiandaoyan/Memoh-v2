---
name: web-artifacts-builder
description: Build interactive web artifacts — single-file HTML/CSS/JS apps, data visualizations, dashboards, games, and interactive demos. Use when asked to create something visual and interactive that runs in a browser.
---

# Web Artifacts Builder

Create self-contained, interactive web artifacts delivered as single HTML files.

## Output Format

Always produce a **single `index.html` file** saved to `/data/` that includes all CSS and JS inline. The file must work without any server or build step.

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>App Title</title>
  <style>
    /* All CSS inline */
  </style>
</head>
<body>
  <!-- Content -->
  <script>
    // All JS inline
  </script>
</body>
</html>
```

## Common Artifact Types

### Data visualization
Use Chart.js (CDN) for charts. Load data inline as JS arrays.

```html
<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
```

### Interactive dashboard
Tailwind CSS (CDN) for rapid layout:

```html
<script src="https://cdn.tailwindcss.com"></script>
```

### Simple game / simulation
Vanilla JS + Canvas API. No dependencies needed.

### Form / calculator
HTML form elements + JS event listeners. Validate client-side.

## Guidelines

- **Mobile-first**: Use responsive design (flexbox/grid)
- **Dark/light**: Default to a clean light theme unless asked otherwise
- **Performance**: Inline small images as base64; link CDN assets for large libraries
- **Accessibility**: Semantic HTML, ARIA labels on interactive elements
- **Save path**: `/data/artifact-name.html` or `/data/index.html`

## Workflow

1. Understand the desired output with an example or mockup description
2. Choose the minimal set of libraries needed
3. Build the full HTML in one file
4. Write to `/data/` using the `write` tool
5. Confirm the file path so the user can open it
