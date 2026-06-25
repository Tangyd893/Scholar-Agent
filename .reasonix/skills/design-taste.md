---
name: design-taste
description: Anti-slop frontend skill. Infers design direction from brief, tunes 3 dials (variance/motion/density), enforces layout/typography/color rules. Use /taste.
---
# Taste-Skill v2 (design-taste-frontend)

Anti-slop frontend skill. The agent reads the brief, infers the right design direction, and ships interfaces that do not look templated.

## 0. Brief Inference

Before any code, read the room: page kind, vibe words, reference signals, audience, brand assets, quiet constraints. Output a one-line "Design Read": "Reading this as: <page kind> for <audience>, with a <vibe> language."

Anti-Default: Do NOT default to AI-purple gradients, centered hero over dark mesh, three equal feature cards, Inter + slate-900.

## 1. Three Dials

| Dial | Range | Default |
|------|-------|---------|
| DESIGN_VARIANCE | 1 (symmetry) - 10 (chaos) | 8 |
| MOTION_INTENSITY | 1 (static) - 10 (cinematic) | 6 |
| VISUAL_DENSITY | 1 (airy) - 10 (packed) | 4 |

Infer from design read. Minimalist → 5-6/3-4/2-3; Agency → 9-10/8-10/3-4.

## 2. Design System Map

Match brief to real design system (Fluent UI, Material, Carbon, Polaris, Primer, GOV.UK, shadcn/ui). Don't invent CSS when official packages exist. One system per project.

For aesthetics without official systems (glassmorphism, bento, brutalism, editorial, dark tech), build with native CSS + Tailwind. Label approximations honestly.

## 3. Stack Defaults

- React/Next.js with RSC. Wrap providers in `"use client"`.
- Tailwind v4. Motion (`motion/react`) for animation.
- Fonts: `next/font` or self-hosted. Never Google Fonts `<link>`.
- Icons priority: Phosphor > Hugeicons > Radix Icons > Tabler. Discouraged: lucide-react. One family per project.
- Grid over flex-math. `h-dvh` not `h-screen`. `max-w-[1400px]`.

## 4. Design Engineering Directives

### Typography
- **Discouraged default**: Inter. Prefer Geist, Outfit, Cabinet Grotesk, Satoshi.
- **Serif VERY discouraged as default.** Only when brand explicitly names it or genuinely editorial/luxury.
- **BANNED defaults**: Fraunces, Instrument_Serif (LLM-favorite display serifs).
- Italic descender clearance: `leading-[1.1]` minimum for italic display words with y/g/j/p/q.

### Color
- Max 1 accent. Saturation < 80%. One palette per project.
- **LILA RULE**: AI Purple/Blue glow is discouraged as default.
- **PREMIUM-CONSUMER PALETTE BAN**: beige/cream/brass/clay/oxblood as default for premium brands = banned. Rotate: Cold Luxury, Forest, Black+Tan, Cobalt+Cream, Terracotta+Slate, Olive+Brick, monochrome+pop.

### Layout
- **Anti-Center Bias**: Centered hero avoided when VARIANCE > 4.
- **Hero discipline**: max 4 text elements, top padding max `pt-24`, headline max 2 lines.
- **Eyebrow restraint**: max 1 eyebrow per 3 sections.
- **Split-header ban**: left headline + right explainer as section header = banned.
- **Navigation**: single line desktop, max 80px height.
- **Bento**: varied cells, no empty cells, 2-3 cells need real visual variation.

### Cards & Data
- Cards only when elevation communicates hierarchy. Shadow tinted to bg hue.
- Shape consistency: one corner-radius scale per page.
- Full interactive states: loading (skeletons), empty, error, tactile feedback.
- Button contrast WCAG AA min. CTA text on one line. No duplicate CTA intent per page.

### Image Strategy
- Image-gen tool first. Real web images second. Never text-only pages with fake screenshots.

## 5. Pre-Flight Checklist

Before shipping: audit font scale + hero height, button contrast, CTA labels (no wrap, no duplicates), eyebrow count, layout repetition, bento cell count, palette consistency.
