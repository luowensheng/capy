# Types

Capy types validate captured argument values at evaluation time. The library
author declares types in the `types:` section; functions reference them by
name in `args[].type`.

## Built-in kinds

These are always available and don't need a `types:` entry:

| Kind     | Captures                                                                |
|----------|-------------------------------------------------------------------------|
| `any`    | Any single value expression. No validation.                             |
| `ident`  | A single identifier token; bound as a string.                           |
| `raw`    | An identifier OR a string token; bound as a string (no quotes).         |
| `string` | A quoted string literal — or a bare ident (transpile-mode permissive).  |
| `int`    | An integer literal — or a bare ident.                                   |
| `float`  | A float literal — or a bare ident.                                      |
| `bool`   | `true`/`false` — or a bare ident.                                       |

The "or a bare ident" rule exists because in transpile mode the captured
text flows into the target language; a bare ident might be a variable that
holds a value of the expected type at the target's runtime.

## Library-defined types

Declared in `types:` with three optional fields applied in this order:

```yaml
types:
  Email:
    base:    string                          # 1. type-kind check
    pattern: "^[^@]+@[^@]+\\.[^@]+$"          # 2. regex on the value's string form
  Status:
    options: ["todo", "in-progress", "done"] # 3. enum membership
  PositiveInt:
    base:    int
    pattern: "^[1-9][0-9]*$"
```

Each field is optional. The most common patterns:

- `pattern:` only — for free-form strings that must match a regex.
- `options:` only — for enums.
- `base:` + `pattern:` — to constrain a primitive kind further.

## How validation runs

For each captured argument with a declared type:

1. If the type is a built-in kind, the engine checks the captured **source
   text** (e.g. `int` requires the text to parse as an integer literal).
   Bare identifiers always pass.
2. If the type is library-defined:
   - If `base:` is set, run the built-in check for that base kind first.
   - If `pattern:` is set, compile + match against the value's string form
     (with surrounding quotes stripped from string literals).
   - If `options:` is set, check membership against the value's string form.

If any check fails, the engine returns a structured error like:

```
function "set_email" arg "e": value "not-an-email" does not match pattern for type "Email"
```

## Using types in `args:`

```yaml
functions:
  set_email:
    args:
      - { kind: capture, name: e, type: Email }
    template: "email = {{ .e }}\n"
```

The capture's `type:` may be either a built-in kind or any name from your
`types:` section. The loader validates that every referenced type resolves —
typos are caught at `capy check` time.

## Patterns: practical tips

- Use `^...$` anchors. Without anchors, a regex like `[a-z]+` accepts
  anything that contains lowercase letters.
- Use `\\` to escape inside YAML strings (`"\\d"` in YAML is the regex `\d`).
- Test patterns with a quick `capy run` against a sample script before
  shipping them.

## Options: practical tips

- Options match the captured string text after quote-stripping. So:
  - `set_status "todo"` → option string `"todo"` ✓
  - `set_status todo` (bare ident) → also `"todo"` ✓
- Put the most common options first; the engine scans linearly (it's a tiny
  list).

## What's NOT here yet

- **`validate:` snippets** written in inner Capy. The README originally
  envisioned this. We've kept `pattern:` + `options:` + `base:` as the v0.1
  surface; full snippets land in a future version.
- **Type composition / inheritance**. Each type stands alone.
- **Custom built-in kinds**. Only the six above; ask via an issue if you
  need more.
