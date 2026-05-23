# Templates

Capy uses Go's [`text/template`](https://pkg.go.dev/text/template) for both
the per-function `template:` and the top-level `file_template:`. This page
documents the data model and the Capy-specific helpers.

## Per-function template data

When a function's template is rendered, these variables are available:

| Variable        | Source                                                                     |
|-----------------|----------------------------------------------------------------------------|
| `.<capture>`    | One entry per capture, the captured source text as a string.               |
| `.body`         | The inner block's rendered output (block functions only).                  |
| `.context`      | Read-only snapshot of the current accumulated context.                     |

```yaml
greet:
  args: [{ kind: capture, name: name, type: any }]
  template: "Hello, {{ .name }}!\n"
```

`.name` is the source text of the captured value (`"Alice"` with quotes, or
`Alice` if the user passed a bare identifier).

## File template data

When the top-level `file_template:` is rendered, these variables are
available:

| Variable     | Source                                                                |
|--------------|-----------------------------------------------------------------------|
| `.body`      | Concatenation of all top-level statements' rendered templates.        |
| `.context`   | The final accumulated context.                                        |

```yaml
file_template: |
  {{- range .context.imports }}import {{ . }}
  {{ end }}
  {{- .body -}}
```

## Helpers

Beyond Go's stdlib helpers, Capy provides:

### `indent N`

Indents every line of a string by N spaces. Most useful for block bodies.

```yaml
template: |
  if {{ .cond }}:
  {{ .body | indent 4 }}
```

### `toQuoted`

Wraps a string in JSON-style double quotes (with proper escaping). Useful
for emitting string literals in target languages.

```yaml
say:
  args: [{ kind: capture, name: msg, type: any }]
  template: "print({{ .msg | toQuoted }})\n"
```

### `toPyLit`

Formats a Go value as a Python literal (`True`/`False`, `None`, quoted
strings, list/dict syntax). Useful when accumulated `context` carries real
Go values you want to splat into Python.

```yaml
file_template: |
  CONFIG = {{ .context.config | toPyLit }}
```

### `toJSON` / `toJSONIndent`

Marshal any value to JSON. Compact and pretty respectively. Excellent for
config-file targets.

```yaml
file_template: |
  {{ .context | toJSONIndent }}
```

### `lower` / `upper`

Case helpers.

### `join SEP <list>`

Join a list of strings (or any-types coerced to strings).

```yaml
file_template: |
  scripts: {{ join ", " .context.script_names }}
```

## Common patterns

### Imports at top, body below

```yaml
file_template: |
  {{- range .context.imports }}import {{ . }}
  {{ end }}
  {{- .body -}}
```

### A block function emitting an indented body

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

### Pure-context output (no body emission)

```yaml
functions:
  set_name:
    args: [{ kind: capture, name: n, type: any }]
    template: ""
    run: |
      set context.name n

file_template: |
  {{ .context | toJSONIndent }}
```

### Computing a value with template logic

Inside a single template you can use Go's `if`/`with`/`range`:

```yaml
template: |
  {{ if .verbose }}# Generated for {{ .target }}{{ end }}
  print({{ .msg }})
```

## Whitespace control

Go templates use `{{-` and `-}}` to trim surrounding whitespace. Helpful for
keeping output clean when ranges produce stray newlines:

```yaml
file_template: |
  {{- range .context.imports }}import {{ . }}
  {{ end -}}
  {{ .body }}
```
