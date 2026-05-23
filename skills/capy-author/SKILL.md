---
name: capy-author
description: Use this skill when the user wants to design or modify a Capy library — a YAML file that defines a tiny source language and how to transpile it to a target language (Python, JSON, SQL, HTML, etc.). Triggers on phrases like "build a Capy library for…", "transpile this to X", "write a Capy DSL", "edit lib.yaml", or when there's a `.capy` source file in context.
tools: [Read, Write, Edit, Bash]
---

# Capy Library Author

When the user wants a custom DSL or code generator and Capy is appropriate, design a `lib.yaml` + `script.capy` pair that delivers the target output.

## How to operate

1. **Clarify the target.** Ask what target language (Python, JSON, SQL, Makefile, …) and what user-facing source-language shape they want — show them 2–3 source snippets and the matching outputs you'd produce.

2. **Author `lib.yaml`** using the patterns in [reference/](reference/). Validate as you go with `capy check lib.yaml`.

3. **Write `script.capy`** that exercises every pattern.

4. **Verify with `capy run lib.yaml script.capy`**. Iterate on mismatch.

5. **Capture the golden** (`<base>.expected.txt`) so future changes regress safely.

## Hard rules (do NOT violate)

- Every `args:` entry MUST have a `kind:` field (`literal` or `capture`). The loader rejects entries without it.
- `kind: literal` requires `value:`; NOT `name` or `type`.
- `kind: capture` requires `name:` and `type:`; NOT `value:`.
- Inside `run:` snippets the indentation is **4 spaces or 1 tab per level** — same as the outer grammar.
- Captures resolve to **source text** in templates and **evaluated values** in `run:`. Don't confuse them.
- `block:` is either `{ closer: <name> }` OR `{ open: "{", close: "}" }`. Never both.

## What's in [reference/](reference/)

| File | What's in it |
|---|---|
| `schema.md` | The whole YAML schema in prose. |
| `inner-dsl.md` | Every operation `run:` supports. |
| `template-helpers.md` | Every helper available in templates. |
| `examples.md` | Six canonical libraries with line-by-line commentary. |
| `pitfalls.md` | Mistakes the model should not make. Read first. |

## Standard delivery shape

When you finish, present the user with:

- The path to `lib.yaml`.
- The path to `script.capy`.
- The path to the verified golden (`<base>.expected.txt`).
- A 5-line summary of what's defined and why.
- The exact `capy run` command to reproduce.

If anything in the user's request can't be expressed in v0.1 (e.g.
`validate:` Capy snippets, `else` arm, multi-output), say so explicitly
and propose a workaround using what's available.
