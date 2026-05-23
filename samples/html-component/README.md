# html-component

JSX-ish component DSL → HTML. Demonstrates brace-delimited block functions.

## Run

```sh
../../capy run lib.yaml script.capy
```

## What this teaches

- `block: { open: "{", close: "}" }` for explicit-delimiter blocks (Mode B).
- The block opener's template references `{{ .body }}` to emit the
  rendered child statements between its own HTML wrapper.
- Multi-line `{ ... }` block bodies work because the lexer emits newlines
  even inside brackets.
