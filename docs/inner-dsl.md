# Inner DSL Reference

The inner DSL is the language a library author uses inside each function's
`run:` field. It updates the accumulated `context`; it **does not execute
user-script code**. Library authors compose primitives to express "what
metadata should this match contribute to the final output."

## Tokens & expressions

The inner DSL reuses the engine's lexer and value-expression parser. Values
may be:

- Numbers, strings, templates, `true`, `false`, `null`.
- Bare or dotted identifier paths — these resolve, in order, against:
  inner-DSL locals (loop variables), captures from the matched function,
  the root `context` map.
- Lists `[...]` and objects `{...}`.
- Comparison expressions: `a == b`, `a != b`, `a < b`, `a <= b`, `a > b`,
  `a >= b`.
- Unary `not expr`.
- Parenthesized sub-calls: `(regex_match name "^[a-z]+$")`.

## Statements

### `set <path> <value>`

Assigns a value to a field path on the context (or to a local).

```
set context.name "Alice"
set context.config.api.url "https://example.com"
set context.scripts[key] cmd
```

Paths use:
- `.<field>` for map field access.
- `[<expr>]` for dynamic indexing — the expression is evaluated, then its
  string form becomes the key.

### `append <list-path> <value>`

Appends to a list field. Creates the list if it doesn't exist yet.

```
append context.imports name
append context.errors {kind: "warn", msg: msg}
```

### `prepend <list-path> <value>`

Like `append` but inserts at the front.

```
prepend context.docstrings line
```

### `merge <map-path> <map-value>`

Shallow-merges a map into a map field. The value must be a map expression.

```
merge context.headers {"X-Built-With": "capy"}
```

### `delete <path>`

Removes a field or list index.

```
delete context.scripts[old_name]
```

### `if <expr>` … `end`

Library-side conditional update. The expression is evaluated; if truthy,
the body runs.

```
if (regex_match name "^_")
    set context.private true
end
```

### `loop <var> in <expr>` … `end`

Iterates a list, binding the variable in each iteration's local scope.

```
loop tag in tags
    append context.tags tag
end
```

Note: this iterates within a single source statement's `run:` snippet. It
does NOT iterate user-script code — that's what `block:` functions are for.

### Plain calls

A line that starts with an identifier that isn't a statement keyword is
treated as a primitive call:

```
error "expected a positive integer"
```

The only primitive call defined is `error <message>` (abort transpilation).
`regex_match` is also callable but is most useful in expression position
(`(regex_match v "...")`).

## Truthiness

| Value | Truthy? |
|---|---|
| `nil` / `null` | no |
| `false` | no |
| `""` (empty string) | no |
| `0`, `0.0` | no |
| empty list `[]` | no |
| empty map `{}` | no |
| anything else | yes |

## Captures inside `run:`

When you reference a capture by name, you get the **evaluated** value:

- A string literal `"foo"` becomes the Go string `"foo"` (no surrounding quotes).
- A number becomes `int64` or `float64`.
- A list becomes `[]any` of evaluated items.
- An object becomes `map[string]any`.
- A bare identifier (when the source has e.g. `import json`) becomes its
  literal name as a string (`"json"`).

This means `append context.imports name` correctly stores `"json"` without
extra quoting — useful for the file template's `{{ range }}`.

## Context paths

Paths must be rooted at `context` (or at a local introduced by a `loop`).
You cannot directly mutate captures.

```
set context.imports.json true             # ok
set context.imports[name] true            # ok — `name` is a capture
set name "json"                            # ERROR — captures are read-only
```

## Putting it together

```yaml
# transpile-py-style import handling
import:
  args:
    - { kind: literal, value: "import" }
    - { kind: capture, name: name, type: ident }
  template: ""
  run: |
    if (regex_match name "^[a-z][a-z_]*$")
        append context.imports name
    end
    if not (regex_match name "^[a-z][a-z_]*$")
        error "invalid module name"
    end
```

## What's NOT here (and why)

The inner DSL is intentionally small. It does not have:

- User-defined inner functions. Compose with multiple library functions or with `loop`.
- `else` branches. Use two `if` statements or invert with `not`.
- Arithmetic operators (`+`, `-`, …). Compute at template time with helpers, or accumulate into a count.

These omissions keep the runtime tiny and predictable. If you find yourself
wanting them often, open a [feature request](https://github.com/luowensheng/capy/issues).
