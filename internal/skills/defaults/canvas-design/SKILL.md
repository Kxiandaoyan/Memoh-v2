---
name: canvas-design
description: "Create original visual designs — posters, banners, flyers, infographics, illustrations, and graphic art — as .png and .pdf files. Generates layouts, applies color palettes, composes typography, and produces print-ready or screen-ready output. Activates when the user asks to design a poster, banner, flyer, infographic, illustration, graphic, piece of art, or any static visual."
license: Complete terms in LICENSE.txt
---

Create original visual designs output as `.pdf` or `.png` files. The workflow has two phases: **Design Philosophy** (output as `.md`) then **Canvas Creation** (output as `.pdf`/`.png`).

## Phase 1: Design Philosophy Creation

Create a short visual philosophy document (`.md` file) that guides the canvas phase.

### Steps

1. **Name the movement** (1-2 words): e.g. "Brutalist Joy", "Chromatic Silence", "Metabolist Dreams"
2. **Write the philosophy** (4-6 paragraphs) covering:
   - Space and form
   - Color and material
   - Scale and rhythm
   - Composition and visual hierarchy
3. **Output** the philosophy as a `.md` file.

### Philosophy Guidelines

- Emphasize visual expression over text — information lives in design, not paragraphs.
- Leave creative room for interpretive choices in the canvas phase.
- Stress expert-level craftsmanship: the final work should appear meticulously crafted by a master.
- Keep the philosophy generic (reusable across contexts) without mentioning the specific subject.

### Example Philosophy

**"Concrete Poetry"**
Communication through monumental form and bold geometry. Massive color blocks, sculptural typography (huge single words, tiny labels), Brutalist spatial divisions. Ideas expressed through visual weight and spatial tension. Text as rare, powerful gesture — only essential words integrated into the visual architecture.

---

## Phase 2: Canvas Creation

Using the philosophy from Phase 1, produce a single-page `.pdf` or `.png` (unless more pages are requested).

### Identifying the Conceptual Thread

Before drawing, identify any subtle reference from the original request. The reference should be woven into form, color, and composition — felt intuitively by those familiar with the subject, invisible to others.

### Canvas Execution Steps

1. **Layout**: Generate the page layout using repeating patterns, geometric shapes, and structured composition. Anchor the design with a limited, intentional color palette.
2. **Typography**: Search the `./canvas-fonts` directory for available fonts. Use different fonts for variety. Integrate type as a visual element — not just typeset text. Keep text minimal and design-forward.
3. **Render**: Generate the final `.pdf` or `.png`.

```
# Example: generate a PDF with Python
from reportlab.lib.pagesizes import A3
from reportlab.pdfgen import canvas

c = canvas.Canvas("output.pdf", pagesize=A3)
width, height = A3

# Background
c.setFillColor("#1a1a2e")
c.rect(0, 0, width, height, fill=1)

# Geometric element
c.setFillColor("#e94560")
c.circle(width / 2, height / 2, 150, fill=1)

# Typography
c.setFillColor("#ffffff")
c.setFont("Helvetica-Bold", 48)
c.drawCentredString(width / 2, height * 0.2, "SILENCE")

c.save()
```

```
# Example: generate a PNG with Pillow
from PIL import Image, ImageDraw, ImageFont

img = Image.new("RGB", (2480, 3508), "#1a1a2e")
draw = ImageDraw.Draw(img)

# Geometric element
draw.ellipse([940, 1454, 1540, 2054], fill="#e94560")

# Typography
font = ImageFont.truetype("./canvas-fonts/sans.ttf", 120)
draw.text((1240, 3000), "SILENCE", fill="#ffffff", font=font, anchor="mm")

img.save("output.png")
```

### Design Rules

- **Boundaries**: All elements must sit within the canvas with proper margins. Nothing overlaps unintentionally; nothing bleeds off the edge.
- **Sophistication**: Output should be museum or magazine quality — never cartoony or amateur.
- **Visual density**: Treat the composition like a scientific diagram — dense, layered patterns that reward sustained viewing.
- **Sparse text**: Text is a contextual accent. Context may call for bold typographic gestures (e.g. a punk poster) or whisper-quiet labels (e.g. a ceramics identity). Either way, keep it minimal.

### Validation Checklist

- [ ] Philosophy `.md` file is saved
- [ ] Canvas `.pdf` or `.png` file is saved
- [ ] No overlapping elements or text
- [ ] All elements within canvas boundaries with margins
- [ ] Color palette is limited and cohesive
- [ ] Typography uses fonts from `./canvas-fonts`
- [ ] Text is minimal — 90% visual, 10% text

---

## Refinement Pass

After initial creation, take a second pass:

- Do NOT add more graphics. Refine what exists.
- Tighten spacing, alignment, and color consistency.
- Ask: "How can I make what's already here more cohesive?" rather than adding new elements.

---

## Multi-Page Option

When additional pages are requested:

1. Create each page as a distinct interpretation of the same philosophy.
2. Vary composition while maintaining the color palette and typographic system.
3. Bundle pages into a single `.pdf` or multiple `.png` files.
