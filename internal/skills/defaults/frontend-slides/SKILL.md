---
name: frontend-slides
description: "Create stunning, animation-rich HTML presentations from scratch or by converting PowerPoint files. Use when the user wants to build a presentation, convert a PPT/PPTX to web, or create slides for a talk/pitch. Helps non-designers discover their aesthetic through visual exploration rather than abstract choices."
---

# Frontend Slides Skill

Create zero-dependency, animation-rich HTML presentations that run entirely in the browser. This skill helps non-designers discover their preferred aesthetic through visual exploration ("show, don't tell"), then generates production-quality slide decks.

## Core Principles

1. **Zero Dependencies** — Single HTML files with inline CSS/JS. No npm, no build tools.
2. **Show, Don't Tell** — Generate visual previews instead of asking abstract style questions.
3. **Distinctive Design** — Avoid generic "AI slop" aesthetics. Every presentation should feel custom-crafted.
4. **Viewport Fitting (CRITICAL)** — Every slide MUST fit exactly within the viewport. No scrolling within slides, ever.

---

## CRITICAL: Viewport Fitting

Each slide = exactly one viewport height. Content that overflows must be split into multiple slides.

### Content Density Limits

| Slide Type | Maximum Content |
|------------|-----------------|
| Title slide | 1 heading + 1 subtitle + optional tagline |
| Content slide | 1 heading + 4-6 bullet points OR 1 heading + 2 paragraphs |
| Feature grid | 1 heading + 6 cards maximum (2x3 or 3x2 grid) |
| Code slide | 1 heading + 8-10 lines of code maximum |
| Quote slide | 1 quote (max 3 lines) + attribution |
| Image slide | 1 heading + 1 image (max 60vh height) |

### Required Base CSS

Every presentation MUST include this viewport-fitting foundation:

```css
html, body { height: 100%; overflow-x: hidden; }
html { scroll-snap-type: y mandatory; scroll-behavior: smooth; }

.slide {
    width: 100vw;
    height: 100vh;
    height: 100dvh;
    overflow: hidden;
    scroll-snap-align: start;
    display: flex;
    flex-direction: column;
    position: relative;
}

.slide-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;
    max-height: 100%;
    overflow: hidden;
    padding: var(--slide-padding);
}

:root {
    --title-size: clamp(1.5rem, 5vw, 4rem);
    --h2-size: clamp(1.25rem, 3.5vw, 2.5rem);
    --h3-size: clamp(1rem, 2.5vw, 1.75rem);
    --body-size: clamp(0.75rem, 1.5vw, 1.125rem);
    --small-size: clamp(0.65rem, 1vw, 0.875rem);
    --slide-padding: clamp(1rem, 4vw, 4rem);
    --content-gap: clamp(0.5rem, 2vw, 2rem);
}

.card, .container, .content-box {
    max-width: min(90vw, 1000px);
    max-height: min(80vh, 700px);
}

img, .image-container {
    max-width: 100%;
    max-height: min(50vh, 400px);
    object-fit: contain;
}

/* Responsive breakpoints — scale down aggressively */
@media (max-height: 700px) {
    :root { --slide-padding: clamp(0.75rem, 3vw, 2rem); --title-size: clamp(1.25rem, 4.5vw, 2.5rem); }
}
@media (max-height: 600px) {
    :root { --slide-padding: clamp(0.5rem, 2.5vw, 1.5rem); --body-size: clamp(0.7rem, 1.2vw, 0.95rem); }
    .nav-dots, .keyboard-hint, .decorative { display: none; }
}
@media (max-height: 500px) {
    :root { --title-size: clamp(1rem, 3.5vw, 1.5rem); --body-size: clamp(0.65rem, 1vw, 0.85rem); }
}
@media (max-width: 600px) {
    :root { --title-size: clamp(1.25rem, 7vw, 2.5rem); }
    .grid { grid-template-columns: 1fr; }
}

@media (prefers-reduced-motion: reduce) {
    *, *::before, *::after { animation-duration: 0.01ms !important; transition-duration: 0.2s !important; }
    html { scroll-behavior: auto; }
}
```

### Overflow Prevention Checklist

Before generating any presentation, verify:
1. Every `.slide` has `height: 100vh; height: 100dvh; overflow: hidden;`
2. All font sizes and spacing use `clamp()` or viewport units
3. Content containers have `max-height` constraints
4. Grids use `auto-fit` with `minmax()` for responsive columns
5. Breakpoints exist for heights: 700px, 600px, 500px
6. Content per slide respects density limits above

When content does not fit: split into multiple slides. Never reduce font size below readable limits, remove spacing, or allow scrolling.

---

## Phase 0: Detect Mode

Determine the user's intent:

- **Mode A: New Presentation** — Create slides from scratch. Proceed to Phase 1.
- **Mode B: PPT Conversion** — Convert a .ppt/.pptx file. Proceed to Phase 4.
- **Mode C: Enhancement** — Improve an existing HTML presentation. Read the file, then enhance.

---

## Phase 1: Content Discovery (New Presentations)

Ask via AskUserQuestion:

1. **Purpose**: Pitch deck / Teaching / Conference talk / Internal presentation
2. **Length**: Short (5-10) / Medium (10-20) / Long (20+)
3. **Content readiness**: All content ready / Rough notes / Topic only

If user has content, ask them to share it.

---

## Phase 2: Style Discovery

### Style Presets

| Preset | Vibe | Best For |
|--------|------|----------|
| Bold Signal | Confident, high-impact | Pitch decks, keynotes |
| Dark Botanical | Elegant, sophisticated | Premium brands |
| Creative Voltage | Energetic, retro-modern | Creative pitches |
| Neon Cyber | Futuristic, techy | Tech startups |
| Notebook Tabs | Editorial, organized | Reports, reviews |

Additional presets: Electric Studio, Pastel Geometry, Split Pastel, Vintage Editorial, Terminal Green, Swiss Modern, Paper & Ink.

### Style Selection Flow

Ask the user how they want to choose:

**Option A: Guided Discovery (Default)**
1. Ask about desired audience feeling: Impressed/Confident, Excited/Energized, Calm/Focused, Inspired/Moved (multi-select up to 2)
2. Based on mood, generate **3 distinct style previews** as mini HTML files in `.claude-design/slide-previews/` (style-a.html, style-b.html, style-c.html)
3. Each preview: a single animated title slide (~50-100 lines), self-contained, showing typography, color palette, and animation style
4. Present previews and ask user to pick or mix elements

**Option B: Direct Selection** — User picks a preset by name, skip to Phase 3.

**Font/color rules**: Never use Inter, Roboto, Arial, system fonts, or purple-on-white gradients. Use distinctive pairings (Clash Display, Satoshi, Cormorant Garamond, DM Sans, etc.).

---

## Phase 3: Generate Presentation

### File Structure

```
presentation.html    # Self-contained presentation
assets/              # Images, if any
```

### HTML Architecture

Follow this structure:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Presentation Title</title>
    <link rel="stylesheet" href="https://api.fontshare.com/v2/css?f[]=...">
    <style>
        /* CSS custom properties for theme — see Required Base CSS above */
        /* Base styles, slide container, responsive breakpoints */
        /* Animations: .reveal pattern with staggered delays */
    </style>
</head>
<body>
    <div class="progress-bar"></div>
    <nav class="nav-dots"><!-- Generated by JS --></nav>

    <section class="slide title-slide">
        <h1 class="reveal">Presentation Title</h1>
        <p class="reveal">Subtitle</p>
    </section>

    <section class="slide">
        <h2 class="reveal">Slide Title</h2>
        <p class="reveal">Content...</p>
    </section>

    <script>
        /* SlidePresentation class — handles keyboard/touch/scroll navigation,
           progress bar, nav dots, and Intersection Observer for .visible class */
        class SlidePresentation {
            constructor() { /* ... */ }
        }
        new SlidePresentation();
    </script>
</body>
</html>
```

### Required JavaScript Features

1. **SlidePresentation Class** — Keyboard navigation (arrows, space), touch/swipe, mouse wheel, progress bar, nav dots
2. **Intersection Observer** — Add `.visible` class to trigger CSS entrance animations

Optional enhancements (based on style): custom cursor, particle backgrounds, parallax, 3D tilt, magnetic buttons, counter animations.

### Code Quality

- Comment each CSS/JS section with what it does and how to modify it
- Semantic HTML (`<section>`, `<nav>`, `<main>`), ARIA labels, keyboard navigation
- Reduced motion support via `prefers-reduced-motion`
- All viewport-fitting CSS from the Required Base CSS section above

---

## Phase 4: PPT Conversion

1. **Extract** content using Python with `python-pptx` — pull text, images, notes from each slide
2. **Confirm** extracted structure with the user
3. **Style** — proceed to Phase 2 for style selection
4. **Generate** HTML preserving all text, images (from assets folder), slide order, and speaker notes

---

## Phase 5: Delivery

1. Clean up `.claude-design/slide-previews/` if it exists
2. Open the presentation with `open [filename].html`
3. Provide summary with file path, style name, slide count, navigation instructions, and customization tips (`:root` variables, font link, `.reveal` timings)

---

## Animation Patterns

### Entrance Animations

```css
/* Fade + Slide Up */
.reveal { opacity: 0; transform: translateY(30px); transition: opacity 0.6s var(--ease-out-expo), transform 0.6s var(--ease-out-expo); }
.visible .reveal { opacity: 1; transform: translateY(0); }

/* Scale In */
.reveal-scale { opacity: 0; transform: scale(0.9); transition: opacity 0.6s, transform 0.6s var(--ease-out-expo); }

/* Blur In */
.reveal-blur { opacity: 0; filter: blur(10px); transition: opacity 0.8s, filter 0.8s var(--ease-out-expo); }
```

### Background Effects

```css
/* Gradient Mesh */
.gradient-bg {
    background: radial-gradient(ellipse at 20% 80%, rgba(120, 0, 255, 0.3) 0%, transparent 50%),
                radial-gradient(ellipse at 80% 20%, rgba(0, 255, 200, 0.2) 0%, transparent 50%),
                var(--bg-primary);
}

/* Grid Pattern */
.grid-bg {
    background-image: linear-gradient(rgba(255,255,255,0.03) 1px, transparent 1px),
                      linear-gradient(90deg, rgba(255,255,255,0.03) 1px, transparent 1px);
    background-size: 50px 50px;
}
```

---

## Style-to-Effect Mapping

| Feeling | Key Techniques |
|---------|----------------|
| Dramatic / Cinematic | Slow fade-ins (1-1.5s), dark backgrounds, spotlight effects, parallax |
| Techy / Futuristic | Neon glow, particle systems, grid patterns, monospace accents, glitch effects |
| Playful / Friendly | Bouncy easing, large border-radius, pastels, floating animations |
| Professional / Corporate | Subtle fast animations (200-300ms), clean sans-serif, precise spacing |
| Calm / Minimal | Very slow motion, high whitespace, muted palette, serif typography |
| Editorial / Magazine | Strong type hierarchy, pull quotes, grid-breaking layouts, one accent color |

---

## Troubleshooting

| Issue | Fix |
|-------|-----|
| Fonts not loading | Check Fontshare/Google Fonts URL; verify font names match in CSS |
| Animations not triggering | Verify Intersection Observer is running; check `.visible` class |
| Scroll snap not working | Ensure `scroll-snap-type` on html and `scroll-snap-align` on slides |
| Mobile issues | Disable heavy effects at 768px; test touch events; reduce particle count |
| Performance | Use `will-change` sparingly; prefer `transform`/`opacity` animations; throttle scroll handlers |

---

## Related Skills

- **learn** — Generate FORZARA.md documentation for the presentation
- **frontend-design** — For more complex interactive pages beyond slides
- **design-and-refine:design-lab** — For iterating on component designs
