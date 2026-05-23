---
name: capy-author
description: Use this skill when the user wants to design or modify a Capy library — a file in `.capy` native syntax (or YAML) that defines a tiny source language and how to transpile it to a target (Python, JSON, SQL, HTML, etc.). Triggers on phrases like "build a Capy library for…", "transpile this to X", "write a Capy DSL", "edit lib.capy", "edit lib.yaml", or when there's a `.capy` source file in context.
tools: [Read, Write, Edit, Bash]
---

# Capy Library Author

When the user wants a custom DSL or code generator and Capy is
appropriate, design a `lib.capy` + `script.capy` pair that delivers
the target output.

**Default to `.capy` for new libraries.** It's terser, multi-line
templates read natively, and there's no YAML escape pain. Use YAML
only when the user explicitly wants it or has an existing `.yaml`
library to edit.

## How to operate

1. **Clarify the target.** Ask what target language (Python, JSON,
   SQL, Makefile, …) and what user-facing source-language shape they
   want — show them 2–3 source snippets and the matching outputs
   you'd produce.

2. **Author `lib.capy`** using the patterns in [reference/](reference/).
   Validate as you go with `capy check lib.capy`.

3. **Write `script.capy`** that exercises every pattern.

4. **Verify with `capy run lib.capy script.capy`**. Iterate on
   mismatch.

5. **Capture the golden** (`<base>.expected.txt`) so future changes
   regress safely.

## Hard rules (do NOT violate)

For `.capy` libraries:

- `arg literal "TEXT"` matches a literal token.
- `arg capture NAME TYPE` captures a typed named variable.
- `template_str "..."` for single-line templates; `template:` for
  multi-line indented blocks.
- A function definition is `function NAME ... end`. Inside, `run:`
  introduces an indented inner-DSL block.
- `file_template:` is always last and captures to EOF.

For YAML libraries (same library, same engine):

- Every `args:` entry MUST have a `kind:` field (`literal` or
  `capture`). The loader rejects entries without it.
- `kind: literal` requires `value:`; NOT `name` or `type`.
- `kind: capture` requires `name:` and `type:`; NOT `value:`.

Universal:

- Inside `run:` snippets the indentation is **4 spaces or 1 tab per
  level** — same as the outer grammar.
- Captures resolve to **source text** in templates and **evaluated
  values** in `run:`. Don't confuse them.
- `block:` (YAML) or `block_closer` / `block_open close` (Capy) is
  either a named closer OR an open/close delimiter pair. Never both.

## What's in [reference/](reference/)

| File | What's in it |
|---|---|
| `schema.md` | The whole library schema in both `.capy` and YAML forms. |
| `inner-dsl.md` | Every operation `run:` supports. |
| `template-helpers.md` | Every helper available in templates. |
| `examples.md` | Six canonical libraries with line-by-line commentary. |
| `pitfalls.md` | Mistakes the model should not make. Read first. |

## Standard delivery shape

When you finish, present the user with:

- The path to `lib.capy` (or `lib.yaml`).
- The path to `script.capy`.
- The path to the verified golden (`<base>.expected.txt`).
- A 5-line summary of what's defined and why.
- The exact `capy run` command to reproduce.

If anything in the user's request can't be expressed in v0.3 (e.g.
`validate:` Capy snippets, `else` arm, multi-output), say so
explicitly and propose a workaround using what's available.
