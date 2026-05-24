# Library Authoring Guide

A Capy library is the entire grammar of one source language, plus the
recipe for generating output from it. This doc is the reference
walkthrough.

Libraries are written in **`.capy`** — Capy's native syntax. Terser
than YAML, multi-line templates read natively, same indentation and
string rules as the source files the library will parse. Every example
below is `.capy`. YAML is also accepted as a secondary format for
teams that need it — see [§ Also supported: YAML](#also-supported-yaml)
at the end. Both formats produce byte-identical output.

## File shape

A complete Capy library has these top-level sections (all optional
except at least one `function`):

```
extension py                          # informational; suggests target extension
output_file ""                        # if set, capy writes here instead of stdout

context                               # initial schema for the accumulated context
    imports []
    classes []
end

type Email                            # library-defined argument types
    pattern "^[^@]+@[^@]+\\.[^@]+$"
end

function greet                        # one DSL statement shape
    arg literal "greet"
    arg capture name string
    template_str "..."
    run:
        ...
end

file_template:                        # final-output assembler
    {{- .body -}}
```

## Functions

Each `function NAME … end` block defines one DSL statement shape.
`NAME` is the function's reference name (used for the auto-name-prepend
rule and to name a block's closer); it's not necessarily what appears
in source.

```
function greet
    arg capture name Email             # built-in OR library-declared type
    template_str "Hello, {{ .name }}!\n"
    run:
        append context.greetings name
    priority 0                         # higher wins; default 0
end
```

## `arg` — the match shape

Each `arg` line takes one of two forms:

| Form                                | Meaning                                            |
|-------------------------------------|----------------------------------------------------|
| `arg literal "TEXT"`                | A literal token to match exactly in source.        |
| `arg capture NAME TYPE`             | Bind a captured value of type TYPE to NAME.        |

Both forms accept an optional trailing description string for
[auto-generated docs](library-documentation.md):

```
arg literal "recipe" "Open a new recipe."
arg capture title string "Display name, shown as the H1."
```

### Auto-name-prepend

If a function declares **zero** `arg literal` lines, the engine
prepends a literal of the function's name. So:

```
function greet
    arg capture name any
    template_str "..."
end
```

…matches `greet <any>` — the function name becomes the leading token.

As soon as you add any `arg literal`, you own the entire shape. This
is how you build operator-style functions:

```
function assign
    arg capture var ident
    arg literal "="
    arg capture value any
    template_str "{{ .var }} = {{ .value }}\n"
end
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

Bare identifiers always pass primitive type checks because at the
target language's runtime they could refer to a value of any type.

See [types.md](types.md) for library-defined types with
`pattern`/`options`.

## `template:` — what goes into the body

A Go [`text/template`](https://pkg.go.dev/text/template) with the
captured values + body + context available as data. `.capy` libraries
use a `template:` block (multi-line, indented) or `template_str "…"`
(single-line):

```
function greet
    arg capture name any
    template:
        Hello, {{ .name }}!
end
```

Inside a template you can use:

- `{{ .X }}` — a capture by name.
- `{{ .body }}` — the inner block's rendered output (block functions only).
- `{{ .context.X }}` — the read-only accumulated context.
- Helpers: `indent N`, `toQuoted`, `toPyLit`, `toJSON`, `toJSONIndent`,
  `lower`, `upper`, `join`, `split`, `unescape`, plus more.

See [templates.md](templates.md) for the full helper reference.

## `run:` — what updates the context

A small inner DSL. **Does not execute user source.** It only mutates
the `context` map.

```
function import
    arg literal "import"
    arg capture name ident
    template_str ""
    run:
        append context.imports name
end
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
| `(regex_match value pattern)`         | Boolean expression value.                    |
| `(env "X")`, `(arg N)`, `(read_file "P")` | [Host capabilities](host-capabilities.md). |
| `(os)`, `(arch)`                      | Branch on host platform.                     |
| `error "<message>"`                   | Abort transpilation with a message.          |

See [inner-dsl.md](inner-dsl.md) for full details and examples.

## Block functions

A function that opens a body block declares it with `block_closer`
(named-closer mode) or an explicit delimiter pair:

```
function if
    arg literal "if"
    arg capture cond any
    block_closer end
    template:
        if {{ .cond }}:
        {{ .body | indent 4 }}
end

function end
end
```

Or with explicit delimiters:

```
function for
    arg literal "for"
    arg capture v ident
    arg literal "in"
    arg capture i any
    block_open "{"
    block_close "}"
    template:
        for {{ .v }} in {{ .i }} {
        {{ .body | indent 2 }}
        }
end
```

See [block-functions.md](block-functions.md) for nesting and edge cases.

## `context` — initial schema

Whatever fields your `run:` snippets will manipulate. Lists default to
`[]`, maps to `{}`. The context is rendered into the file template as
`.context`.

```
context
    imports []
    classes []
    vars {}
    total 0
end
```

## `file_template:` — final assembly

Receives `.body` (concatenation of all top-level statements' rendered
templates) and `.context` (final accumulated state). Common patterns:

```
# Python-style: imports at top, then body.
file_template:
    {{- range .context.imports }}import {{ . }}
    {{ end }}
    {{- .body -}}
```

```
# Pure JSON: ignore body entirely, render context.
file_template:
    {{ .context | toJSONIndent }}
```

## `priority`

When two functions could match a prefix, the higher `priority` wins;
ties fall back to "more leading literals wins". You rarely need to set
this explicitly — the default (0) plus the literal-leading bias
handles most cases.

## Validation

`capy check lib.capy` parses + validates a library without running any
source. It reports load-time errors with the offending function and
arg index.

## Suggested authoring loop

```sh
capy init my-lib
cd my-lib
# edit lib.capy + script.capy in your editor
capy run lib.capy script.capy        # see output
capy check lib.capy                  # validate
capy docs lib.capy > REFERENCE.md    # regenerate reference docs
```

When the library stabilises, write a few sample scripts under
`examples/` so behaviour stays pinned as you iterate.

---

## Also supported: YAML

Every library shown above can be expressed in YAML with the same
field names. Use YAML when:

- You want existing YAML tooling (yq, JSON Schema, language servers
  with built-in YAML support) on top of your library.
- You're embedding Capy in a system whose config layer is already
  YAML and want one consistent format.

The translation is mechanical — `function NAME … end` becomes a key
under `functions:`, `arg capture NAME TYPE` becomes
`{ kind: capture, name: NAME, type: TYPE }`, etc.:

```yaml
extension: py
functions:
  greet:
    args:
      - { kind: capture, name: name, type: any }
    template: "Hello, {{ .name }}!\n"
    run: |
      append context.greetings name
file_template: |
  {{- .body -}}
```

Both formats parse into the same in-memory DTO. Output is identical.
The CLI auto-detects the format from the extension (`.capy` or
`.yaml`/`.yml`); the embedded Go API exposes both `capy.NewLibrary`
and `capy.NewLibraryYAML`.
