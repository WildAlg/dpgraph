# dpgraph website

Public site for [dpgraph](https://github.com/WildAlg/dpgraph) and the paper *Practical and Accurate Local Edge Differentially Private Graph Algorithms*.

Built with [Astro](https://astro.build) + Tailwind + [Motion](https://motion.dev). Static, deploys to GitHub Pages.

## Develop

```bash
cd website
npm install
npm run dev      # http://localhost:4321/dpgraph/
```

## Build

```bash
npm run build    # → ./dist
npm run preview  # serves ./dist
```

## Structure

```
src/
  layouts/     # Base.astro (chrome) and Doc.astro (sidebar + TOC)
  pages/       # routes — index.astro, paper.astro, docs/*.mdx
  components/
    animations/   # React islands: LedpAnimation, KCoreAnimation, EdgeOrientAnimation
  styles/      # globals.css — fonts and Tailwind base
public/
  figures/     # static SVG fallbacks of the animations (reduced-motion)
```

## Deploy

CI in `.github/workflows/site.yml` builds and deploys to GitHub Pages on every
push to `main` that touches `website/**`.

## Aesthetic

Research-paper editorial: Instrument Serif display, Inter body, JetBrains Mono
code. Off-white paper / ink-black palette with a single ink-blue accent.
Signal-orange is reserved exclusively for "noise" / privacy callouts in the
animations.
