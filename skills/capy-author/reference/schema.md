# Capy library schema (full, in prose)

```yaml
extension:    <str>      # informational; suggests output file extension
output_file:  <str>      # optional; if set, capy writes here instead of stdout

context:                 # initial schema for accumulated state
  <name>: <list|map|scalar>

types:                   # library-defined argument types
  <TypeName>:
    base:    <built-in kind>      # optional — any|string|int|float|bool
    pattern: <regex>              # optional — applied to value's string form
    options: [<v>, ...]           # optional — enum membership

functions:               # the entire surface grammar
  <key>:
    args:                # ordered list of arg entries
      - { kind: literal, value: "TEXT" }
      - { kind: capture, name: NAME, type: TYPE }
      ...
    template: <string>   # rendered into body per match
    run: |               # inner-DSL snippet, mutates context only
      ...
    block:               # only if this opens a body block
      closer: <function-name>      # Mode A — indent body + named closer
      # OR
      open: "{"
      close: "}"
    priority: <int>      # default 0; higher wins on ambiguous matches

file_template: |         # top-level assembler; receives .body and .context
  <go text/template>
```

## Built-in argument types

| Type     | Captures                                                       |
|----------|----------------------------------------------------------------|
| `any`    | Any single value (number, string, ident, list, object, …).     |
| `ident`  | A single identifier token; bound as a string.                   |
| `raw`    | One identifier OR string token; bound as a string.              |
| `string` | A quoted string — OR a bare identifier.                         |
| `int`    | An integer literal — OR a bare identifier.                      |
| `float`  | A float literal — OR a bare identifier.                         |
| `bool`   | `true`/`false` — OR a bare identifier.                          |

## Auto-name-prepend rule

If `args:` has ZERO `kind: literal` entries, the engine auto-prepends the function's key as a leading literal. As soon as you add any literal, you own the entire shape and the function name does NOT appear in source.

## Capture types in detail

A capture's source text and evaluated value:

| Source | `.Text` (templates) | Evaluated (`run:`) |
|---|---|---|
| `42` | `"42"` | `int64(42)` |
| `"foo"` | `"\"foo\""` | Go string `"foo"` |
| `true` | `"true"` | `bool(true)` |
| `null` | `"null"` | `nil` |
| `x` (bare ident) | `"x"` | `"x"` (string) |
| `a.b.c` | `"a.b.c"` | walks locals → captures → context |
| `[1, 2]` | `"[1, 2]"` | `[]any{1, 2}` |
| `{"k": 1}` | `"{\"k\": 1}"` | `map[string]any{"k":1}` |
| `a > b` | `"a > b"` | comparison evaluated as needed |

## Block modes

### Mode A: named closer + indentation

```yaml
if:
  args:
    - { kind: literal, value: "if" }
    - { kind: capture, name: cond, type: any }
  block: { closer: end }
  template: |
    if {{ .cond }}:
    {{ .body | indent 4 }}
end: {}
```

### Mode B: explicit delimiters

```yaml
for:
  args:
    - { kind: literal, value: "for" }
    - { kind: capture, name: v, type: ident }
    - { kind: literal, value: "in" }
    - { kind: capture, name: i, type: any }
  block: { open: "{", close: "}" }
  template: |
    for {{ .v }} in {{ .i }} {
    {{ .body | indent 2 }}
    }
```

## Validation order for library types

`base` → `pattern` → `options`. Each is optional.
