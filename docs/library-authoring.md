# Library Authoring Guide

A Capy library is the entire grammar of one source language, plus the
recipe for generating output from it. This doc is the reference
walkthrough.

A library can be written in either of two interchangeable formats:

- **`.capy`** — Capy's native syntax. **Recommended for new libraries.**
  Terser, multi-line templates read natively, same indentation and string
  rules as the source files the library will parse.
- **`.yaml`** — same library expressed in YAML. Useful when you want
  downstream tooling (yq, JSON schema, language servers).

Both formats parse into the same in-memory DTO and run through the same
engine. Output is byte-identical. See [.capy libraries](capy-libraries.md)
for the format comparison.

The reference below shows both forms side by side. Skip whichever you
don't need.

## File shape

A complete Capy library has these top-level sections (all optional
except `functions`):

### Capy-native form

```
extension     py                  # informational; suggests target extension
output_file   ""                  # if set, capy writes here instead of stdout

context                           # initial schema for the accumulated context
    imports []
    classes []
end

type Email                        # library-defined argument types
    pattern "^[^@]+@[^@]+\\.[^@]+$"
end

function greet                    # one DSL statement shape
    arg literal "greet"
    arg capture name string
    template_str "..."
    run:
        ...
end

file_template:                    # final-output assembler
    {{- .body -}}
```

### YAML form (same library, same engine)

```yaml
extension:    py                # informational; suggests the target file extension
output_file:  ""                # if set, capy writes here instead of stdout

context:                        # initial schema for the accumulated context
  imports: []
  classes: []

types:                          # library-defined argument types
  Email:
    pattern: "^[^@]+@[^@]+\\.[^@]+$"

functions:                      # the entire surface grammar
  greet:
    args: [...]
    template: "..."
    run:      |
      ...
    block:    { ... }           # only when this function opens a body block

file_template: |                # the final-output assembler
  {{- .body -}}
```

## `functions:`

Each entry is a function the library author defines. The map **key** is the
function's reference name (used for the auto-name-prepend rule and to name a
block's closer); it's not necessarily what appears in source.

```yaml
functions:
  greet:                                    # function reference name
    args:                                   # ordered list of args
      - { kind: capture, name: name, type: Email }
    template: "Hello, {{ .name }}!\n"
    run: |
      append context.greetings name
    priority: 0                             # higher wins; default 0
```

## `args:` — the match shape

Every entry has an explicit `kind:` discriminator:

| `kind:`     | Required fields                | Meaning                                         |
|-------------|--------------------------------|-------------------------------------------------|
| `literal`   | `value: "TEXT"`                | A literal token to match exactly in source.     |
| `capture`   | `name: NAME`, `type: TYPE`     | Bind a value of type TYPE to NAME.              |

The loader validates that the right fields appear for each kind.

### Auto-name-prepend

If `args` contains **zero** `kind: literal` entries, the engine prepends a
literal of the function's key. So:

```yaml
greet:
  args:
    - { kind: capture, name: name, type: any }
```

…matches `greet <any>` — i.e. the function key becomes the leading token.

As soon as you write any `kind: literal`, you own the entire shape. This is
how you build operator-style functions:

```yaml
assign:
  args:
    - { kind: capture, name: var, type: ident }
    - { kind: literal, value: "=" }
    - { kind: capture, name: value, type: any }
```

…matches `<ident> = <any>`. No leading `assign` token in source.

## Built-in types

`any`, `ident`, `raw`, `string`, `int`, `float`, `bool`.

| Type   | What it captures                                                                                                   |
|--------|--------------------------------------------------------------------------------------------------------------------|
| `any`  | Any single value expression (number, string, ident, list, object, bool, null, dotted ident, parenthesized sub-call). |
| `ident`| A single identifier token; bound as a string.                                                                       |
| `raw`  | One identifier OR string token; bound as a string.                                                                   |
| `string` | A quoted string literal — OR a bare identifier (transpile-mode permissive).                                       |
| `int`/`float`/`bool` | The respective literal — OR a bare identifier.                                                          |

Bare identifiers always pass primitive type checks because at the target
language's runtime they could refer to a value of any type.

See [types.md](types.md) for library-defined types with `pattern`/`options`.

## `template:` — what goes into the body

A Go [`text/template`](https://pkg.go.dev/text/template) with the captured
values + body + context available as data:

```yaml
greet:
  args: [{ kind: capture, name: name, type: any }]
  template: "Hello, {{ .name }}!\n"
```

Inside a template you can use:

- `{{ .X }}` — a capture by name.
- `{{ .body }}` — the inner block's rendered output (block functions only).
- `{{ .context.X }}` — the read-only accumulated context.
- Helpers: `indent N`, `toQuoted`, `toPyLit`, `toJSON`, `toJSONIndent`, `lower`, `upper`, `join`.

See [templates.md](templates.md) for the full helper reference.

## `run:` — what updates the context

A small inner DSL. **Does not execute user source.** It only mutates the
`context` map.

```yaml
import:
  args:
    - { kind: literal, value: "import" }
    - { kind: capture, name: name, type: ident }
  template: ""
  run: |
    append context.imports name
```

Operations available:

| Form                                  | Effect                                       |
|---------------------------------------|----------------------------------------------|
| `set <path> <value>`                  | Set a field at a dotted/indexed path.        |
| `append <list-path> <value>`          | Append to a list.                            |
| `prepend <list-path> <value>`         | Prepend to a list.                           |
| `merge <map-path> <map-value>`        | Shallow-merge into a map.                    |
| `delete <path>`                       | Remove a field/key.                          |
| `if <expr>` ... `end`                 | Library-side conditional update.             |
| `loop <var> in <expr>` ... `end`      | Library-side iteration.                      |
| `regex_match value pattern`           | Boolean expression value.                    |
| `error <message>`                     | Abort transpilation with a message.          |

See [inner-dsl.md](inner-dsl.md) for full details and examples.

## `block:` — opening a body block

A function that opens a block declares it with one of two modes:

### Mode A: named-closer + INDENT/DEDENT body

```yaml
if:
  args:
    - { kind: literal, value: "if" }
    - { kind: capture, name: cond, type: any }
  block: { closer: end }
  template: |
    if {{ .cond }}:
    {{ .body | indent 4 }}
end: {}                           # closer is itself a function; can be silent
```

### Mode B: explicit delimiter pair

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

See [block-functions.md](block-functions.md) for nesting and edge cases.

## `context:` — initial schema

Whatever fields your `run:` snippets will manipulate. Lists default to `[]`,
maps to `{}`. The context is rendered into the file template as `.context`.

```yaml
context:
  imports: []
  classes: []
  vars: {}
  total: 0
```

## `file_template:` — final assembly

Receives `.body` (concatenation of all top-level statements' rendered
templates) and `.context` (final accumulated state). Common patterns:

```yaml
# Python-style: imports at top, then body.
file_template: |
  {{- range .context.imports }}import {{ . }}
  {{ end }}
  {{- .body -}}
```

```yaml
# Pure JSON: ignore body entirely, render context.
file_template: |
  {{ .context | toJSONIndent }}
```

## `priority:`

When two functions could match a prefix, the higher `priority:` wins; ties
fall back to "more leading literals wins". You rarely need to set this
explicitly — the default (0) plus literal-leading bias handles most cases.

## Validation

`capy check lib.yaml` parses + validates a library without running any
source. It reports load-time errors with the offending function and arg
index.

## Suggested authoring loop

```sh
capy init my-lib
cd my-lib
# edit lib.yaml + script.capy in your editor
capy run lib.yaml script.capy        # see output
capy check lib.yaml                  # validate
```

When the lib stabilises, write a few sample scripts under `examples/` so the
behavior stays pinned as you iterate.
