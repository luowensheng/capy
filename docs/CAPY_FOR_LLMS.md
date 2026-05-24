# Capy for LLMs — single-page brief

Paste this into a model's context window when you want it to author a
Capy library. Covers the `.capy` schema, the inner DSL, and common
pitfalls. About 500 lines of prose; designed to be self-contained.

---

## What Capy is

A transpiler engine. You define a source-language grammar + transformation
in a **`.capy` library file**. Capy reads source code, matches each
statement against the library's function shapes, and for each match runs
the function's body — a sequence of inner-DSL statements that may
emit output (`write \`...\``) and/or mutate an accumulated `context`
(`set` / `append` / …). A top-level `file_template:` assembles
`body` + `context` into the final output.

There are NO built-in user-facing keywords. Every shape is
library-defined.

**Format note:** Always emit `.capy`. YAML is also accepted as a
secondary format, but `.capy` is the primary surface — terser,
multi-line templates read natively, no YAML escape gotchas. The YAML
form is mentioned only at the bottom of this doc for completeness.

---

## The library schema — Capy-native form (recommended)

```
extension <str>              # informational; suggests output file extension
output_file <str>            # optional; write output here instead of stdout

context                      # initial accumulated state
    <name> []                # empty list
    <name> {}                # empty map
    <name> 0                 # numeric default
    <name> "default"         # string default
end

type <TypeName>              # library-defined argument type
    base <kind>              # optional: any|string|int|float|bool
    pattern "<regex>"        # optional: regex on the value's string form
    options "v1" "v2" "v3"   # optional: enum membership
end

function <NAME>              # one DSL statement shape
    priority <int>           # optional; higher wins ambiguous matches
    arg literal "TEXT"       # match a literal token
    arg capture <NAME> <TYPE> # capture a typed named variable
    block_closer <NAME>      # block opener: body runs until <NAME> appears
    block_open "OPEN" close "CLOSE"   # alternative: explicit delimiters

    # Function body — sequence of inner-DSL statements:
    write `Hello, ${name}!\n`        # emit literal text + interpolations
    append context.greetings name    # mutate state
    # if / for / set / prepend / merge / delete also available
end

file_template:               # whole-file wrapper; captures to EOF
    ...
```

Strings use double quotes (with Go-style escapes `\n` `\t` `\"`
`\\`) or backticks (multi-line, with `${EXPR}` interpolation).
Bare words are accepted for `extension`, type names, and capture
names. Indentation delimits the function body and `file_template:`.

Legacy form: `template:` / `template_str` / `run:` blocks are still
accepted for backwards compatibility — every example in this doc
could be written either way. Prefer the unified body shown above
for new libraries.

(For the YAML form of the same schema, see the end of this doc.
All other examples below are `.capy`.)

---

## Args

Each function's `arg` lines take one of two forms:

- `arg literal "TEXT"` — match this exact token in source.
- `arg capture NAME TYPE` — bind a captured value to NAME.

Both forms accept an optional trailing description string for
auto-generated docs.

## Auto-name-prepend rule

If a function declares ZERO `arg literal` lines, the engine
auto-prepends a literal of the function's name. So:

```
function greet
    arg capture name any
end
```

…matches `greet <any>` in source. As soon as you write any literal,
you own the entire shape (function name NOT auto-prepended). That's
how you define operator-style functions like:

```
function assign
    arg capture var ident
    arg literal "="
    arg capture value any
    template_str "{{ .var }} = {{ .value }}\n"
end
```

…which matches `<ident> = <any>` — no leading `assign` token.

## Built-in capture types

| Type     | Captures                                              |
|----------|-------------------------------------------------------|
| `any`    | Any value expression (number, string, ident, list, object, dotted path, paren-sub-call, comparison). |
| `ident`  | A single identifier token.                            |
| `raw`    | Identifier OR string.                                 |
| `string` | A quoted string — OR a bare identifier.               |
| `int`    | An integer literal — OR a bare identifier.            |
| `float`  | A float literal — OR a bare identifier.               |
| `bool`   | `true`/`false` — OR a bare identifier.                |

Bare identifiers pass primitive type checks because they could refer to
target-language variables.

## Library-defined types

```
type Email
    base string                           # built-in kind check first
    pattern "^[^@]+@[^@]+\\.[^@]+$"       # regex
end

type Status
    options ["todo", "done"]              # enum
end
```

Applied in order: `base` → `pattern` → `options`. All three are optional.

---

## The inner DSL (inside `run:`)

A small fixed language. Updates `context` only — does NOT execute user
code.

### Statements

```
set <path> <value>             # bind a field
append <path> <value>          # push to a list
prepend <path> <value>         # push to front
merge <path> <map-value>       # shallow-merge into a map
delete <path>                  # remove field/key
if <expr>                      # library-side conditional
    ...
end
loop <var> in <expr>           # library-side iteration
    ...
end
error <message>                # abort with a message
```

### Paths

Rooted at `context` (or at a `loop` local). Examples:

```
context.imports
context.config.api.url
context.scripts[name]          # `name` is a capture/local; evaluated to a key
```

### Expressions

- Numbers, strings (including ${interp}), `true`, `false`, `null`.
- Identifier paths resolve in order: locals (loop vars), captures, context.
- Lists `[1, 2, 3]`, objects `{"k": "v", name: "Alice"}` (keys can be unquoted idents).
- Comparison: `==`, `!=`, `<`, `<=`, `>`, `>=`. Unary `not expr`.
- `(regex_match value pattern)` returns a boolean — useful in `if` conditions.

### Captures inside `run:`

When you reference a capture, you get the **evaluated** value:

- String literal `"foo"` → Go string `"foo"` (no quotes).
- Number `42` → `int64(42)`.
- List `[1, 2]` → `[]any{int64(1), int64(2)}`.
- Object `{a: 1}` → `map[string]any{"a": int64(1)}`.
- Bare identifier `x` → string `"x"`.

So `append context.imports name` for source `import json` correctly stores
`"json"` (without quotes).

---

## Templates

Per-function `template:` data:

- `.<capture>` — the captured source text (with quotes for strings,
  bracket syntax for lists).
- `.body` — the rendered inner block (block functions only).
- `.context` — read-only snapshot.

File-template data:

- `.body` — concatenation of all top-level statements' rendered output.
- `.context` — final accumulated context.

### Helpers

- `indent N` — pad every line with N spaces. Use for block bodies.
- `lower`, `upper` — case.
- `join SEP <list>` — joiner.
- `toQuoted` — wrap a string in `"…"`.
- `toPyLit` — Python literal formatting (True/False/None, lists, dicts).
- `toJSON`, `toJSONIndent` — JSON marshal.

---

## Two block modes

### Mode A: named closer + indentation

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

Body is delimited by INDENT/DEDENT (4 spaces or 1 tab per level). The
named closer must match after the body.

### Mode B: explicit delimiters

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

Body delimited by the literal `{` and `}` tokens. No closer function.

---

## Worked example (full Python transpiler library)

```
extension py

context
    imports []
end

type Identifier
    pattern "^[A-Za-z_][A-Za-z0-9_]*$"
end

function import
    arg literal "import"
    arg capture name Identifier
    append context.imports name
end

function say
    arg capture msg any
    write `print(${msg})
`
end

function assign
    arg capture name Identifier
    arg literal "="
    arg capture value any
    write `${name} = ${value}
`
end

function if
    arg literal "if"
    arg capture cond any
    block_closer end
    write `if ${cond}:
${indent 4 body}
`
end

function loop
    arg literal "loop"
    arg capture var ident
    arg literal "in"
    arg capture iter any
    block_closer end
    write `for ${var} in ${iter}:
${indent 4 body}
`
end

function end
end

file_template:
    {{- range .context.imports }}import {{ . }}
    {{ end }}
    {{- .body -}}
```

Source:

```
import json
say "hello"
x = 42
if x
    say x
end
```

Output:

```python
import json
print("hello")
x = 42
if x:
    print(x)
```

---

## Common pitfalls

1. **Putting business logic in user-script** — Capy transpiles, it
   doesn't execute. `if x ... end` emits an `if`; it doesn't
   conditionally render.
2. **Quoting confusion** — string captures expose source text *with*
   quotes in templates. Use the value as-is for Python (which uses
   the same quoting) or strip with the `unquote` helper if needed.
3. **`{}` ambiguity** — `{...}` is an object literal by default. For
   `{...}` blocks, the opener function must declare `block_open "{"`
   and `block_close "}"` explicitly.
4. **Indentation must be 4 spaces or 1 tab per level.** 2-space
   indent breaks the lexer.
5. **No `else`** — use two `if` blocks (one with `not`) for now.
6. **Auto-name-prepend silently disables** the moment you add any
   `arg literal`. So an `arg literal "in"` line somewhere removes
   the automatic function-name leading literal.

## CLI quick reference

```sh
capy run <lib.capy> <script.capy>     # transpile
capy check <lib.capy>                 # validate library
capy docs <lib.capy>                  # auto-generate reference docs
capy init [<dir>]                     # scaffold
capy version
capy help [<command>]
```

## When in doubt

Run `capy check lib.capy` after every edit. If it loads cleanly, run
`capy run lib.capy script.capy` against a minimal script. Errors are
caret-pointed at line:col.

---

## Also supported: YAML (secondary format)

Every library above can be expressed in YAML — same field names,
same engine, byte-identical output. Use YAML only when you need
existing YAML tooling (yq, JSON Schema). The mapping is mechanical:

| `.capy`                                | YAML                                                |
|----------------------------------------|-----------------------------------------------------|
| `function NAME … end`                  | key `NAME:` under `functions:`                      |
| `arg literal "X"`                      | `{ kind: literal, value: "X" }`                     |
| `arg capture N T`                      | `{ kind: capture, name: N, type: T }`               |
| `block_closer end`                     | `block: { closer: end }`                            |
| `block_open "{"` + `block_close "}"`   | `block: { open: "{", close: "}" }`                  |
| `template:` block                      | `template: \|` block scalar                          |
| `run:` block                           | `run: \|` block scalar                              |
| `type NAME … end`                      | key `NAME:` under `types:`                          |

The CLI auto-detects format from the file extension (`.capy` vs
`.yaml`/`.yml`). The embedded Go API has `capy.NewLibrary` for
`.capy` and `capy.NewLibraryYAML` for YAML. If you're authoring a
new library, stick with `.capy`.
