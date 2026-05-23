# Capy for LLMs — single-page brief

Paste this into a model's context window when you want it to author Capy
library YAML. Covers the schema, the inner DSL, and the common pitfalls.
About 600 lines of prose; designed to be self-contained.

---

## What Capy is

A transpiler engine. You define a source-language grammar + transformation
in a `lib.yaml` file. Capy reads source code, matches each statement
against the library's function shapes, and for each match (a) renders the
function's `template:` into the output body and (b) updates an accumulated
`context` via the function's `run:` snippet. A top-level `file_template:`
assembles `body` + `context` into the final output.

There are NO built-in user-facing keywords. Every shape is library-defined.

---

## The YAML schema

```yaml
extension: <str>             # informational; suggests output file extension
output_file: <str>           # optional; write output here instead of stdout

context:                     # initial schema for accumulated state
  <name>: <list|map|scalar>

types:                       # library-defined argument types
  <TypeName>:
    base: <built-in kind>    # optional: any|string|int|float|bool
    pattern: <regex>         # optional: regex on the value's string form
    options: [<v>, ...]      # optional: enum membership

functions:                   # the entire surface grammar
  <key>:
    args:                    # ordered list of arg entries
      - { kind: literal, value: "TEXT" }
      - { kind: capture, name: NAME, type: TYPE }
      ...
    template: <string>       # rendered into body for each match
    run: |                   # inner-DSL snippet, mutates context only
      <statements>
    block:                   # only if this opens a body block
      closer: <function-name> # Mode A: indent body + named closer
      # OR
      open: "{"               # Mode B: explicit delimiters
      close: "}"
    priority: <int>          # default 0; higher wins on ambiguous matches

file_template: |             # top-level assembler; receives .body and .context
  <go text/template>
```

---

## Args: kind discriminator (CRITICAL)

Every args entry MUST have `kind:` set to either `literal` or `capture`.

- `kind: literal` requires `value: "TEXT"`. NO `name`, NO `type`.
- `kind: capture` requires `name: NAME` and `type: TYPE`. NO `value`.

The loader rejects entries with the wrong fields for the kind. Common
mistakes:
- `{ literal: "if" }` — WRONG (missing `kind:`)
- `{ name: x, type: any }` — WRONG (missing `kind:`)
- `{ kind: literal, value: "if", name: x }` — WRONG (literal can't have name)

## Auto-name-prepend rule

If `args` contains ZERO `kind: literal` entries, the engine auto-prepends
a literal of the function's key. So:

```yaml
greet:
  args: [{ kind: capture, name: name, type: any }]
```

…matches `greet <any>` in source. As soon as you write any literal, you
own the entire shape (function key NOT auto-prepended). That's how you
define operator-style functions like:

```yaml
assign:
  args:
    - { kind: capture, name: var, type: ident }
    - { kind: literal, value: "=" }
    - { kind: capture, name: value, type: any }
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

```yaml
types:
  Email:
    base: string                          # built-in kind check first
    pattern: "^[^@]+@[^@]+\\.[^@]+$"      # regex
  Status:
    options: ["todo", "done"]             # enum
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

Body is delimited by INDENT/DEDENT (4 spaces or 1 tab per level). The
named closer must match after the body.

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

Body delimited by the literal `{` and `}` tokens. No closer function.

---

## Worked example (full Python transpiler library)

```yaml
extension: py
output_file: ""

context:
  imports: []

types:
  Identifier:
    pattern: "^[A-Za-z_][A-Za-z0-9_]*$"

functions:
  import:
    args:
      - { kind: literal, value: "import" }
      - { kind: capture, name: name, type: Identifier }
    template: ""
    run: |
      append context.imports name

  say:
    args:
      - { kind: capture, name: msg, type: any }
    template: "print({{ .msg }})\n"

  assign:
    args:
      - { kind: capture, name: name, type: Identifier }
      - { kind: literal, value: "=" }
      - { kind: capture, name: value, type: any }
    template: "{{ .name }} = {{ .value }}\n"

  if:
    args:
      - { kind: literal, value: "if" }
      - { kind: capture, name: cond, type: any }
    block: { closer: end }
    template: |
      if {{ .cond }}:
      {{ .body | indent 4 }}

  loop:
    args:
      - { kind: literal, value: "loop" }
      - { kind: capture, name: var, type: ident }
      - { kind: literal, value: "in" }
      - { kind: capture, name: iter, type: any }
    block: { closer: end }
    template: |
      for {{ .var }} in {{ .iter }}:
      {{ .body | indent 4 }}

  end: {}

file_template: |
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

1. **Forgetting `kind:`** — every args entry needs it. Validators catch
   this at load time.
2. **Putting business logic in user-script** — Capy transpiles, it doesn't
   execute. `if x ... end` emits an `if`; it doesn't conditionally render.
3. **Quoting confusion** — string captures expose source text *with*
   quotes in templates. Use the value as-is for Python (which uses the
   same quoting) or unquote with `unquote` if needed.
4. **`{}` ambiguity** — `{...}` is an object literal by default. For
   `{...}` blocks, the opener function must declare `block: { open: "{",
   close: "}" }` explicitly.
5. **Indentation must be 4 spaces or 1 tab per level**. 2-space indent
   breaks the lexer.
6. **YAML block scalar (`|`) indentation** in `run:` snippets — inside
   YAML you indent relative to the parent key, but the inner DSL still
   requires its own 4-space block indentation.
7. **No `else`** — use two `if` blocks (one with `not`) for now.
8. **Auto-name-prepend silently disables** the moment you add any literal.
   So `{kind: literal, value: "in"}` somewhere in your args removes the
   automatic `funcname` leading literal.

## CLI quick reference

```sh
capy run <lib.yaml> <script.capy>     # transpile
capy check <lib.yaml>                 # validate library
capy init [<dir>]                     # scaffold
capy version
capy help [<command>]
```

## When in doubt

Run `capy check lib.yaml` after every edit. If it loads cleanly, run
`capy run lib.yaml script.capy` against a minimal script. Errors are
caret-pointed at line:col.
