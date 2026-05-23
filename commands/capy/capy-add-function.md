---
name: capy-add-function
description: Add a function definition to an existing Capy library.
---

# /capy-add-function

Use when the user wants to extend a `lib.yaml` with a new function.

## Steps

1. Confirm the path to `lib.yaml` (or detect from the current dir).
2. Ask, if not already provided:
   - **Function purpose** in one sentence.
   - **Source shape** — show one example line.
   - **Output shape** — show what to emit.
   - Whether it opens a block (Mode A or Mode B).
   - Required arg types.
3. Compose the YAML entry:
   - Choose `kind: literal` and `kind: capture` args.
   - If first source token is the function name, args may be capture-only (auto-name-prepend).
   - If first source token is a symbol or different word, prefix with `{kind: literal, value: "..."}`.
4. Edit `lib.yaml` to add the new function.
5. Add a line to `script.capy` exercising the new function.
6. Run `capy run` and show the output. If it fails, iterate.
7. Once verified, optionally regenerate the golden.

## Checks

- `capy check lib.yaml` should remain clean.
- Every capture name referenced in `template:` and `run:` must appear in `args:`.
- Block-opener functions must reference an existing closer function (Mode A) or use `open:`/`close:` (Mode B).

## Skip the work if

The user already has a similar function. Suggest editing the existing one and explain why.
