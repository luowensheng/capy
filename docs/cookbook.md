# Cookbook

Short, self-contained answers to common needs. Each entry shows just the
delta — copy into your own library and adjust types/names.

## Emit an import block at the top of the output

```yaml
context:
  imports: []

functions:
  import:
    args:
      - { kind: literal, value: "import" }
      - { kind: capture, name: name, type: ident }
    template: ""
    run: |
      append context.imports name

file_template: |
  {{- range .context.imports }}import {{ . }}
  {{ end }}
  {{- .body -}}
```

## Deduplicate context entries

Use a map as a set:

```yaml
context:
  imports: {}                       # map, not list

functions:
  import:
    args:
      - { kind: literal, value: "import" }
      - { kind: capture, name: name, type: ident }
    template: ""
    run: |
      set context.imports[name] true

file_template: |
  {{- range $name, $_ := .context.imports }}import {{ $name }}
  {{ end -}}
```

## Validate that a string matches multiple patterns

Library-defined types apply `base`, `pattern`, and `options` in that order.
For complex multi-rule validation, encode it in the regex (e.g.
`^(?=.*[A-Z])(?=.*[0-9]).{8,}$`):

```yaml
types:
  StrongPassword:
    pattern: "^(?=.*[A-Z])(?=.*[0-9]).{8,}$"
```

For richer logic (AND across distinct patterns, length checks, etc.), use
multiple captures in different functions or open a feature request for
`validate:` snippets.

## Emit indented bodies for a block construct

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

## Render a list of objects from context

```yaml
context:
  routes: []

functions:
  route:
    args:
      - { kind: literal, value: "route" }
      - { kind: capture, name: method, type: any }
      - { kind: capture, name: path, type: any }
    template: ""
    run: |
      append context.routes {method: method, path: path}

file_template: |
  ROUTES = [
  {{- range .context.routes }}
      { "method": {{ .method | toQuoted }}, "path": {{ .path | toQuoted }} },
  {{- end }}
  ]
```

## Support both `do…end` and `{…}` block styles

Define two functions:

```yaml
for_indent:
  args:
    - { kind: literal, value: "for" }
    - { kind: capture, name: v, type: ident }
    - { kind: literal, value: "in" }
    - { kind: capture, name: i, type: any }
    - { kind: literal, value: "do" }
  block: { closer: end }
  template: "for {{ .v }} in {{ .i }}:\n{{ .body | indent 4 }}"
end: {}

for_brace:
  args:
    - { kind: literal, value: "for" }
    - { kind: capture, name: v, type: ident }
    - { kind: literal, value: "in" }
    - { kind: capture, name: i, type: any }
  block: { open: "{", close: "}" }
  template: "for {{ .v }} in {{ .i }}: { {{ .body }} }\n"
```

The matcher picks whichever shape the source uses; the longer literal
prefix wins on ties.

## Emit a function name only if a context flag is set

Inside the file template:

```yaml
file_template: |
  {{ if .context.has_main -}}
  if __name__ == "__main__":
      main()
  {{- end }}
```

Inside a function's template:

```yaml
say:
  args: [{ kind: capture, name: msg, type: any }]
  template: |
    {{ if .context.verbose }}# log: {{ .msg }}{{ end }}
    print({{ .msg }})
```

## Build a deeply nested context

Use dotted paths in `run:`:

```yaml
run: |
  set context.config.api.url "https://example.com"
  set context.config.api.timeout 30
  set context.config.db.host "localhost"
```

You can also use bracket indexing for dynamic keys:

```yaml
run: |
  set context.scripts[name] cmd
```

…where `name` is a capture.

## Stop transpilation with a clear error

```yaml
run: |
  if (regex_match name "^_")
      error "names starting with underscore are reserved"
  end
```

## Have a function that adds nothing to body, just runs side effects

Omit `template:` (defaults to nothing) — only `run:` updates state:

```yaml
register:
  args: [{ kind: capture, name: name, type: ident }]
  run: |
    append context.registered name
```

Body output: empty. Context: gains the new name.

## Capture a multi-line JSON literal as an argument

JSON literals can span lines because the value parser skips newlines inside
`{ }` / `[ ]`:

```
config {
    "host": "localhost",
    "port": 5432
}
```

Just declare the arg as `type: any` and the parser handles the rest.
