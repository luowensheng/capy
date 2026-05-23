# scene-dsl

A declarative DSL with no control flow defined — only function calls that emit HTML.

## Files

- `lib.yaml` — two functions: `scene.create_sphere`, `scene.create_ring`. Each takes a `raw` id and an `any` opts object.
- `script.capy` — single-line and multi-line JSON literal arguments.

## Run

```sh
../../capy -lib lib.yaml script.capy
```

## Expected output

```html
<sphere id=moon opts={"size": 16, "color": "#888"}/>
<ring id=ring_a opts={"thickness": 2, "color": "#ff6f00"}/>
```

## What this teaches

- A library can be as small as your DSL needs. No `if`, no `loop`, no `=` — the source language simply has no concept of them.
- Dotted call names (`scene.create_sphere`) are YAML keys with `.`. The pattern compiler splits dotted literals to match how the lexer tokenises them.
- Multi-line JSON arguments work because the lexer suppresses newlines inside `{ }`, `[ ]`, `( )`.
- Captures appear in templates as their **source text** — that's why the opts object in the output looks like the source literal.
