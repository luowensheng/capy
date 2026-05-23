---
name: capy-explain
description: Walk through an existing Capy library and explain how it works.
arguments: <path-to-lib.yaml>
---

# /capy-explain

Use when the user points at a `lib.yaml` and asks "what does this do?" or "explain this to me."

## Steps

1. Read the library file.
2. Identify each function: its match shape (from args), its template, its run snippet.
3. Identify each type: its constraints.
4. Identify the context schema and file_template.
5. Produce a concise prose explanation organized as:

### Section 1: What it transpiles

One paragraph: "This library accepts source like X and produces output like Y."

### Section 2: Function-by-function

Bulleted list, one per function. For each:
- Surface form (e.g. `<ident> = <any>`).
- What template emits.
- What run snippet accumulates.
- Whether it opens a block (and how).

### Section 3: Types

If any: one line per library type with its constraints.

### Section 4: Output assembly

What the file template does with context + body.

### Section 5: Example

Run the library against its `script.capy` if one exists and show output.

## When to stop

If the library is large (>20 functions), summarise at a higher level and offer to drill into specific functions on request.
