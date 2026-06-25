---
name: impeccable
description: Design guidance for production-grade frontend interfaces. 23 commands (craft/polish/audit/bolder/distill/animate/typeset/layout...), anti-slop detector rules. Use /impeccable.
---
# Impeccable v3.8.0

Designs and iterates production-grade frontend interfaces. Real working code, committed design choices, exceptional craft.

## Absolute bans (match-and-refuse)

- **Side-stripe borders.** `border-left/right` > 1px as colored accent. Rewrite with full borders or nothing.
- **Gradient text.** `background-clip: text` + gradient. Use solid color instead.
- **Glassmorphism as default.** Blurs/glass cards decoratively. Rare and purposeful only.
- **Hero-metric template.** Big number + small label + supporting stats + gradient accent.
- **Identical card grids.** Same-sized cards with icon+heading+text repeated endlessly.
- **Tiny uppercase tracked eyebrow above every section.** Max 1 per 3 sections.
- **Numbered section markers (01/02/03) as default scaffolding.**
- **Text overflow.** Test headings at every breakpoint; reduce clamp max if it overflows.

## General rules

### Color
- Body text ≥4.5:1 contrast; large text ≥3:1
- Gray text on colored background is washed out — use darker shade of bg's own hue
- Use OKLCH for new projects
- Cream/sand/beige body bg is the saturated AI default of 2026 — avoid it

### Typography
- Cap body line length at 65-75ch
- Don't pair similar fonts; pair on contrast axis (serif+sans, geometric+humanist)
- Display heading max 6rem; letter-spacing ≥ -0.04em
- `text-wrap: balance` on h1-h3; `text-wrap: pretty` on prose

### Layout
- Cards are the lazy answer. Nested cards = always wrong.
- Flexbox for 1D, Grid for 2D
- Semantic z-index scale (dropdown→sticky→modal-backdrop→modal→toast→tooltip). Never 999/9999.

### Motion
- Intentional, not afterthought. Ease-out with exponential curves. No bounce/elastic.
- `@media (prefers-reduced-motion: reduce)` is not optional
- Stagger list items; suppress the uniform-reflex, not all motion
- Reveal animations: content must be visible before transition triggers

### Interaction
- Dropdowns in `overflow:hidden` containers clip. Use popover API or portal.

## The AI slop test
If someone could look at this interface and say "AI made that", it's failed.

## Codex-specific defects (refuse-and-rewrite)
- `border + box-shadow` ghost-card pattern on the same element
- `border-radius: 32px+` on cards/sections/inputs
- Hand-drawn sketchy SVG illustrations (`feTurbulence`, `feDisplacementMap`)
- `repeating-linear-gradient` stripe backgrounds
- Meta-criticism copy (layering ironic modifiers)

## When invoked
- First run initial setup: gather PRODUCT.md, DESIGN.md, read existing tokens/components
- Match the project register: brand (marketing/landing/portfolio) vs product (app/dashboard/tool)
- New project without tokens: generate brand seed color via palette, compose with OKLCH
- Produce ready-to-ship code, not prototypes. Battle-test with screenshots.
