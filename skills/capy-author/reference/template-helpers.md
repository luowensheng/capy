# Template helpers

Capy uses Go [`text/template`](https://pkg.go.dev/text/template) with these
extra helpers.

| Helper        | Usage                                          | Effect                                            |
|---------------|------------------------------------------------|---------------------------------------------------|
| `indent`      | `{{ .body \| indent 4 }}`                      | Prefix every line by N spaces. Use for block bodies. |
| `lower`       | `{{ .s \| lower }}`                            | Lowercase.                                        |
| `upper`       | `{{ .s \| upper }}`                            | Uppercase.                                        |
| `join`        | `{{ join ", " .list }}`                        | Join a list as strings.                           |
| `toQuoted`    | `{{ .s \| toQuoted }}`                         | Wrap in `"…"` with JSON escaping.                 |
| `toPyLit`     | `{{ .v \| toPyLit }}`                          | Python literal (True/False/None/quoted strings).  |
| `toJSON`      | `{{ .v \| toJSON }}`                           | Compact JSON.                                     |
| `toJSONIndent`| `{{ .context \| toJSONIndent }}`               | Pretty JSON.                                      |

## Available variables

Per-function template:

- `.<capture>` — captured source text.
- `.body` — rendered inner block (only on block-opener functions).
- `.context` — read-only context snapshot.

File template:

- `.body` — full top-level body.
- `.context` — final context.

## Whitespace control

Go templates use `{{-` and `-}}` to trim surrounding whitespace.

```yaml
file_template: |
  {{- range .context.imports }}import {{ . }}
  {{ end }}
  {{- .body -}}
```

The leading `-` on `{{- range ... }}` trims the preceding newline; the
trailing `-` on `{{- .body -}}` trims following whitespace.

## Common idioms

### Imports at top

```yaml
file_template: |
  {{- range .context.imports }}import {{ . }}
  {{ end }}
  {{- .body -}}
```

### JSON file

```yaml
file_template: |
  {{ .context | toJSONIndent }}
```

### Indented block

```yaml
template: |
  if {{ .cond }}:
  {{ .body | indent 4 }}
```

### Optional emission based on context

```yaml
template: |
  {{ if .context.verbose }}# log emitted from {{ .name }}{{ end }}
  print({{ .msg }})
```

### Map iteration with key + value

```yaml
file_template: |
  {{ range $k, $v := .context.tasks }}{{ $k }}: {{ $v }}
  {{ end }}
```
