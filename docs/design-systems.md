---
title: Design systems & component generation
---

# Design systems with guaranteed look, feel, and style

A company invests months building its visual identity — colors,
spacing, button variants, card layouts. Then six teams pick six
frameworks (React, Vue, Svelte, Angular, SwiftUI, Compose) and the
identity fragments. The "primary button" looks slightly different
in each. Onboarding new components requires hand-translating across
three to ten stacks.

**Capy fixes this by making the visual identity a library.** Authors
declare what a page contains — buttons, cards, fields — in a shared
DSL. The library encodes the house style. Every consumer produces
visually-identical output, in whichever framework they target.

## The pattern

```
            script.capy
        (declared composition)
                │
        ┌───────┼───────┐
        ▼       ▼       ▼
   lib_react  lib_vue  lib_svelte
        │       │       │
        ▼       ▼       ▼
     .tsx     .vue   .svelte
   (Tailwind classes encoded in the library — IDENTICAL in all three)
```

Same source. One library per framework you ship to. The library
encodes:

- **Variants** (primary / ghost / danger button mappings to Tailwind
  classes — defined once, used identically).
- **Layout primitives** (Card: white panel, shadow, padding 6,
  rounded-xl — the same in React's `<section>`, Vue's `<Card>`,
  Svelte's `<Card>`).
- **Sizing scale** (sm / md / lg — same px values across all three).
- **Spacing & typography** — defined in the library, applied uniformly.

## A worked example

[`samples/design-system-components/`](https://github.com/olivierdevelops/capy/tree/main/samples/design-system-components)
ships ONE source and THREE libraries.

The source (8 lines):

```
page "Settings"
    button "Save changes"      variant primary  size lg
    button "Discard"            variant ghost    size lg
    card title "Profile"
        field email     "alice@example.com"
        field display   "Alice Chen"
        field timezone  "America/Los_Angeles"
    end
    card title "Danger zone"
        button "Delete account" variant danger   size md
    end
end
```

Generated **React TSX**:

```tsx
const BUTTON_VARIANT = {
  primary: "bg-indigo-600 hover:bg-indigo-700 text-white",
  ghost:   "bg-transparent hover:bg-slate-100 text-slate-700 border border-slate-200",
  danger:  "bg-red-600 hover:bg-red-700 text-white",
};
const BUTTON_SIZE = { sm: "px-3 py-1 text-sm", md: "px-4 py-2", lg: "px-5 py-3 text-lg" };
// ... Button / Card / Field components ...

export default function SettingsPage() {
  return (
    <main className="max-w-2xl mx-auto p-8 space-y-3">
      <h1 className="text-3xl font-bold text-slate-900">Settings</h1>
      <Button variant="primary" size="lg">Save changes</Button>
      <Button variant="ghost" size="lg">Discard</Button>
      <Card title="Profile">
        <Field label="email" value="alice@example.com" />
        <Field label="display" value="Alice Chen" />
        <Field label="timezone" value="America/Los_Angeles" />
      </Card>
      <Card title="Danger zone">
        <Button variant="danger" size="md">Delete account</Button>
      </Card>
    </main>
  );
}
```

Generated **Vue 3 SFC** — identical Tailwind classes, identical
layout, declared `<script setup>` instead of TSX hooks.

Generated **Svelte** — same again, sibling `.svelte` imports
instead of React's named exports.

Three frameworks, **one source of visual truth**, zero drift. Add
a new component (badge, modal, accordion) by adding one line to
the library; all three consumers pick it up at the next regen.

## What this prevents

- **The "almost-but-not-quite" bug.** Designer says "primary buttons
  are indigo-600." Three teams implement it. Two get it right. One
  uses indigo-500 because that's the closest in their existing
  palette. Three months later QA notices. Now you have to do a
  cross-team audit.
- **The bus factor on tokens.** The senior frontend dev who knew the
  color-token system leaves. The system rots. With Capy, the tokens
  are committed code in the library — anyone can read them.
- **Manual cross-framework translation.** A designer ships a new
  card layout. Three frontend teams reimplement it. Now you have
  three subtly different cards.

## How it composes with the rest of Capy

- **Multi-file output** (`file "...":` blocks) means one source can
  emit a whole React + Vue + Svelte directory tree in one run
  with `capy run --out-dir generated`.
- **Library imports** mean a company can publish a base design-system
  library; product teams import + override one or two functions to
  specialize for their product surface.
- **Type validation** catches `variant unknown` or `size huge` at
  transpile time — a typo in a Tailwind class can't sneak through.
- **The MCP server** means an AI agent can author new components
  with full access to the variant table — no guessing at Tailwind
  classes.

## Where to go from here

- Browse [`samples/design-system-components/`](https://github.com/olivierdevelops/capy/tree/main/samples/design-system-components) — three libraries side by side.
- Write a 4th library targeting your own framework (Lit, Solid,
  Astro). The DSL stays the same.
- Add new component declarations (badge, modal, accordion, tooltip).
  One line in the library; every framework picks it up.
