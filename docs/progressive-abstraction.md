---
title: Progressive abstraction
hide:
  - toc
---

<div class="capy-hero" markdown>

<span class="capy-eyebrow">PROGRESSIVE ABSTRACTION</span>

# Pick exactly how much control you want

Most generators force **one** level of abstraction. High-level means
you're stuck with whatever it decides. Low-level means you write
everything. Capy lets the SAME library expose primitives at
**multiple granularities** — start with a one-liner, peel back
layers as you need more control, drop to raw HTML/CSS when the
abstraction isn't enough.

You're never forced into a higher level than the task requires; and
you're never forced lower than your interest.

</div>

---

## Three scripts, one library, same artifact type

All three sources below run through the **same** `lib.capy`. The
library exposes:

- A one-shot `landing "Title" tagline "..." cta_text "..." cta_link "..."` line
- Block-style `landing ... end` containing `hero`, `feature`, `cta`
- Escape hatches: `raw_head`, `style_override`, `raw_section`, `raw_footer`

The script picks how much of that surface to use.

---

### Level 1 — minimal (~4 lines)

<div class="split" markdown>

<div markdown>

```
landing "Capy"
        tagline "Describe what you want. Capy produces what you need."
        cta_text "Open the playground"
        cta_link "https://luowensheng.github.io/capy/playground/"
```

You declare **what**; the library decides **everything else** —
layout, colors, fonts, default footer text. The output is opinionated,
on-brand, and complete.

</div>

<div class="visual" markdown>

<div class="browser-frame">
  <div class="chrome">
    <span class="lights"><span class="r"></span><span class="y"></span><span class="g"></span></span>
    <span class="url">level-1-minimal.html</span>
  </div>
  <iframe src="../assets/demos/abstraction-level-1.html" sandbox="allow-scripts allow-same-origin" title="Level 1"></iframe>
</div>

</div>
</div>

---

### Level 2 — block style (~12 lines)

<div class="split" markdown>

<div markdown>

```
landing "Capy"
    hero "Capy" "Describe what you want. Capy produces what you need."

    feature "Zero default grammar"  "Every keyword is defined by the library."
    feature "55+ samples"           "Recipes, invoices, Android apps."
    feature "Browser playground"    "Compiler runs as WebAssembly."
    feature "MCP server included"   "Plug into Claude / Cursor / Zed."

    cta "Open the playground" "https://..."
end
```

You take control of **which sections appear and in what order**.
The library still owns visual identity — fonts, colors, spacing —
but you decided to add a feature grid and what's in it.

</div>

<div class="visual" markdown>

<div class="browser-frame">
  <div class="chrome">
    <span class="lights"><span class="r"></span><span class="y"></span><span class="g"></span></span>
    <span class="url">level-2-medium.html</span>
  </div>
  <iframe src="../assets/demos/abstraction-level-2.html" sandbox="allow-scripts allow-same-origin" title="Level 2"></iframe>
</div>

</div>
</div>

---

### Level 3 — escape hatches (~30 lines)

<div class="split" markdown>

<div markdown>

```
landing "Capy — Pro"
    raw_head "<meta name='theme-color' content='#4f46e5'>"

    style_override "body { background: linear-gradient(...); }
                    .hero h1 { ... }"

    hero "Capy — Pro" "Same engine. Same grammar. Take exactly the control you need."

    feature "..." "..."
    feature "..." "..."
    feature "Metaprogramming" "Source declares its own DSL primitives."

    raw_section "<section style='...'>...custom HTML...</section>"

    cta "Open the playground" "..."

    raw_footer "<a href='https://github.com/...'>github.com/luowensheng/capy</a>"
end
```

Now you have **escape hatches**. Drop literal HTML into `<head>`;
extend the stylesheet; inject arbitrary sections; replace the
footer. The library never gets in your way.

</div>

<div class="visual" markdown>

<div class="browser-frame">
  <div class="chrome">
    <span class="lights"><span class="r"></span><span class="y"></span><span class="g"></span></span>
    <span class="url">level-3-full.html</span>
  </div>
  <iframe src="../assets/demos/abstraction-level-3.html" sandbox="allow-scripts allow-same-origin" title="Level 3"></iframe>
</div>

</div>
</div>

---

## Why this matters

Most no-code / low-code tools are excellent until you need something
they didn't anticipate. Then you have three options:

1. **Live with the limitation** — your output is "good enough."
2. **Switch tools entirely** — abandon the high-level abstraction.
3. **Hack around it** — fragile workarounds that break on upgrades.

Capy gives you a fourth: **drop one level lower in the same
library**. The abstraction ladder is part of the design:

```
Level 1: one-line `landing "..." tagline "..." cta_text "..."`
Level 2: `landing ... end` block with explicit sections
Level 3: above + raw_head / style_override / raw_section / raw_footer
Level 4: write your own library
```

You climb the ladder as your needs grow. You never have to start
over — yesterday's Level-1 source still runs through today's
extended library.

## How libraries are designed for this

A library that supports progressive abstraction provides:

1. **High-level one-shot functions** — most users start here.
2. **Block-style primitives** for explicit composition.
3. **Escape hatches** — `raw_*` primitives that accept literal target
   syntax and inject it at well-defined points in the output.
4. **Override hooks** — `style_override`, `theme`, `option` — that
   change library defaults without writing the whole file.

The escape hatches are the key: they make the library **non-trapping**.
A user can always reach the underlying medium (HTML, SQL, Kotlin,
Swift, whatever) when the abstraction isn't enough.

## When to design a library this way

- **Different users want different levels of control.** Marketing
  team uses Level 1, product team uses Level 2, brand designer needs
  Level 3 for the launch page.
- **The output medium is rich** — HTML, CSS, configuration files
  where users routinely need a niche feature.
- **You want to encourage starting small and graduating.** Lower
  the barrier to entry; let power users grow without leaving.

When to NOT design this way:

- **The output is rigidly constrained** (Protobuf, K8s API types).
  Escape hatches would break downstream validation.
- **You explicitly want to enforce constraints** (typed configs that
  must be valid; design-system tokens that can't be overridden).

## See it in the playground

The progressive-abstraction sample is in the playground dropdown
under **Patterns** → "Progressive abstraction" (level 1 / 2 / 3
each available individually). Edit any of them, hit Run, see the
output change live.

[Full sample → `samples/progressive-abstraction/`](https://github.com/luowensheng/capy/tree/main/samples/progressive-abstraction)
