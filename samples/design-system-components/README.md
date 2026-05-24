# design-system-components

**One component declaration → React + Vue + Svelte components.**

The same `script.capy` runs through three libraries. Each emits
idiomatic code for its framework — but the visual identity (button
variants, card layout, field row styling) is **identical**, because
it lives in the libraries.

## The source (8 lines)

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

## Three targets, one source

```sh
# React TSX
capy run --out-dir react-out  lib_react.capy  script.capy

# Vue 3 SFC
capy run --out-dir vue-out    lib_vue.capy    script.capy

# Svelte component
capy run --out-dir svelte-out lib_svelte.capy script.capy
```

Each produces a single component file named `SettingsPage.<ext>`
where the page title becomes a PascalCase identifier via the
`pascalCase` template helper.

## What the libraries enforce

Each library has the same variant + size table:

```
BUTTON_VARIANT = {
  primary: "bg-indigo-600 hover:bg-indigo-700 text-white",
  ghost:   "bg-transparent hover:bg-slate-100 text-slate-700 border border-slate-200",
  danger:  "bg-red-600 hover:bg-red-700 text-white",
}
BUTTON_SIZE = { sm: "px-3 py-1 text-sm", md: "px-4 py-2", lg: "px-5 py-3 text-lg" }
```

These tokens are the **house style**. Define them once per
framework; every component picks them up. Need to change the
primary color? Edit one line in each library; every page across
the codebase regenerates.

## Why this matters

Without Capy, every framework's team would re-implement the design
system. Three teams. Three implementations. They drift. With Capy,
the source is one composition; the libraries are three faithful
translators. Drift is impossible because there's only one source.

Add Lit / Solid / Astro / SwiftUI as needed. The DSL stays the
same.

[Pattern docs → `docs/design-systems.md`](../../docs/design-systems.md)
