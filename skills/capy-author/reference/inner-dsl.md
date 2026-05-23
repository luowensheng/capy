# Inner DSL reference (the `run:` field)

A small fixed language. Updates context only. NEVER executes user-script
code.

## Statements

```
set <path> <value>             # bind a field to value
append <list-path> <value>     # push to a list
prepend <list-path> <value>    # push to front
merge <map-path> <map-value>   # shallow-merge maps
delete <path>                  # remove

if <expr>                      # library-side conditional
    ...
end

loop <var> in <expr>           # library-side iteration
    ...
end

error <message>                # abort
```

## Paths

Rooted at `context` or at a local introduced by `loop`. Access via:

- `context.field` — map field.
- `context.list[i]` or `context.map[key]` — dynamic index (the expression
  is evaluated, then its string form becomes the key).
- `context.a.b.c` — deeply nested.

## Expressions

- Numbers, strings (with `${interp}`), `true`, `false`, `null`.
- Identifier paths: locals → captures → context (in order).
- Lists `[...]`, objects `{...}` (keys may be unquoted idents).
- Comparison: `==`, `!=`, `<`, `<=`, `>`, `>=`.
- Unary `not expr`.
- `(regex_match value pattern)` — boolean, often used in `if`.

## Truthiness

| Value | Truthy? |
|---|---|
| `nil` / `null` | no |
| `false` | no |
| `""` (empty string) | no |
| `0` / `0.0` | no |
| `[]` (empty list) | no |
| `{}` (empty map) | no |
| anything else | yes |

## Idioms

### Accumulate to a list

```
run: |
    append context.imports name
```

### Dedupe via map-as-set

```
run: |
    set context.imports[name] true
```

Then in the file template:

```
{{- range $name, $_ := .context.imports }}import {{ $name }}
{{ end }}
```

### Conditional accumulation

```
run: |
    if (regex_match name "^_")
        set context.private true
    end
    if not (regex_match name "^_")
        append context.public name
    end
```

### Bail out with a clear error

```
run: |
    if (regex_match name "^[0-9]")
        error "names cannot start with a digit"
    end
```

### Fan out one source line to multiple context entries

```
run: |
    loop tag in tags
        append context.tags tag
    end
```

Note this iterates a **captured list value** — not user-script code.
