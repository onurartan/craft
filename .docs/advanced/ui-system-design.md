# Internal UI/UX System Design Document
**Target Audience**: Frontend Engineers / Internal Team
**Scope**: `docs/index.html` (Landing Page) & `docs/docs.html` (Documentation SPA)

This document serves as the absolute source of truth for the styling, architecture, and interaction patterns of the Craft web assets.

---

## 1. Core Aesthetic Philosophy

The design is engineered to evoke trust, performance, and modernity—drawing direct inspiration from standard-setters like **Vercel**, **Linear**, and **Stripe Docs**. 

**The three pillars of this UI are:**
1. **Absolute Darkness**: Using `#000000` (pure black) rather than dark grays for the base background. This maximizes contrast and makes neon glow effects pop.
2. **Glassmorphism (Subtle)**: Minimalist frosted glass effects rather than heavy shadows.
3. **Typography over Graphics**: Using font weights and negative letter spacing (`tracking-tight`) to establish hierarchy instead of relying on borders or colored boxes.

---

## 2. Color Palette Architecture

We bypassed default Tailwind grays in favor of a custom-tailored monochrome scale mixed with low-opacity pure whites.

### Base Colors
- `bg-[#000000]`: The absolute root background (`body`).
- `bg-[#0A0A0A]`: The primary "Surface" color (Cards, Panels, Navbar). It provides just enough lift off the pure black background.
- `bg-[#050505]`: The "Sub-surface" color (used for the IDE left-sidebar) to create inner depth.

### Accents & Borders
30
<truncated 3389 bytes>
plication (SPA) Architecture (`docs.html`)

The Documentation Portal is a client-side SPA built without React/Vue, relying solely on Alpine.js and browser APIs.

### The State Machine (`docsApp()`)
```javascript
function docsApp() {
  return {
    docs: [],          // Flat list of markdown files
    groups: [],        // Categorized docs for the sidebar
    activeDoc: null,   // The currently selected markdown document
    mdContent: '',     // The rendered HTML from Marked.js
    isLoading: true,   // Controls loading skeletons
  }
}
```

### Routing & Deep Linking
We leverage the browser's Hash API to allow users to bookmark specific pages.
1. User clicks a link -> `loadDoc(doc)` runs.
2. We update the URL without refreshing: `window.history.pushState(null, '', '#' + doc.name.replace('.md', ''));`
3. We listen for manual URL changes (Back/Forward buttons): `window.addEventListener('hashchange', ...)`

### Markdown Parsing pipeline
1. **Fetch**: Download raw text from GitHub API or local path.
2. **Parse**: Pass raw text to `marked.parse()`.
3. **Inject**: Set `this.mdContent = html`. Alpine.js injects this via `x-html="mdContent"`.
4. **Highlight**: Wait for DOM update (`$nextTick`), then execute `Prism.highlightAllUnder(document.getElementById('markdown-content'))`.

### The Content Container
```html
<article class="prose prose-invert prose-sm md:prose-base max-w-none">
```
We use Tailwind's Typography plugin (`@tailwindcss/typography`). 
- `prose-invert` automatically styles the markdown HTML (H1, p, ul, strong) for dark mode.
- We aggressively override the `prose` defaults in the `<style>` tag to match our custom typography (removing underlines from links, tightening margins, and forcing Prism.js to handle code blocks with transparent backgrounds).

