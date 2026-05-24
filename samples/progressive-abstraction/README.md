# progressive-abstraction

**Same library, three abstraction levels — pick exactly how much
control you want.**

Most generators force one abstraction: high-level (you're stuck
with whatever it produces) or low-level (you write everything).
Capy doesn't. One library can expose primitives at multiple
granularities — start with a one-liner, peel back layers as you
need more control.

This sample ships ONE landing-page library and THREE source files
showing the same library used at three different levels of detail.

## Level 1 — minimal (~4 lines)

`script_minimal.capy`:

```
landing "Capy" tagline "Describe what you want. Capy produces what you need." cta_text "Open the playground" cta_link "https://..."
```

You declare WHAT; the library decides everything else (layout,
colors, fonts, footer text, default copy). 48-line HTML output.

## Level 2 — block style (~12 lines)

`script_medium.capy`:

```
landing "Capy"
    hero "Capy" "Describe what you want. Capy produces what you need."

    feature "Zero default grammar"  "Every keyword is defined by the library."
    feature "55+ samples"           "Recipes, invoices, Android apps."
    feature "Browser playground"    "The compiler runs as WebAssembly."

    cta "Open the playground" "https://..."
end
```

You take control of **which sections appear and in what order**.
The library still owns visual identity. 66-line HTML output.

## Level 3 — escape hatches (~30 lines)

`script_full.capy`:

```
landing "Capy — Pro"
    raw_head "<meta name=\"theme-color\" content=\"#4f46e5\">"
    style_override "body { background: linear-gradient(...); } ..."

    hero "Capy — Pro" "Same engine. Same grammar. Take exactly the control you need."

    feature "Zero default grammar"  "..."
    feature "Metaprogramming"       "..."

    raw_section "<section style='...'>...custom HTML...</section>"
    cta "Open the playground" "..."
    raw_footer "<a href='https://github.com/luowensheng/capy'>...</a>"
end
```

Now you have **escape hatches**: drop raw HTML into `<head>`,
override the stylesheet, inject custom sections, replace the
footer. The library never blocks you. 75-line HTML output.

## Why this matters

Most no-code / low-code tools are excellent until you need
something they didn't anticipate — then you're stuck rewriting in
the underlying technology. Capy lets you:

- **Start small.** Level 1 for prototyping, sketches, "I just need
  it working."
- **Grow into more control.** Level 2 when you need a specific
  feature list, ordering, or layout choice.
- **Drop to raw when needed.** Level 3 for the 10% that needs to
  look unique.

You're never forced into a higher level than the task requires;
you're also never forced lower. The library is the SUBSTRATE; the
source picks the granularity.

## Run

```sh
capy run lib.capy script_minimal.capy > out1.html
capy run lib.capy script_medium.capy  > out2.html
capy run lib.capy script_full.capy    > out3.html
```

All three open in any browser. Compare the source verbosity to the
output complexity — Level 1 is 4 lines, Level 3 is ~30, and Level 3's
output has features Level 1's doesn't (gradient text, hover states,
metrics section, custom footer).

[Pattern docs →](../../docs/progressive-abstraction.md)
