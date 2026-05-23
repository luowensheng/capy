# Canonical libraries with commentary

## 1. Pure rendering (no state)

A scene-DSL that emits HTML, no `context`, no `run:`.

```yaml
extension: html

functions:
  scene.create_sphere:
    args:
      - { kind: capture, name: id, type: raw }
      - { kind: capture, name: opts, type: any }
    template: "<sphere id={{ .id }} opts={{ .opts }}/>\n"
```

When to use this shape: declarative DSLs where each statement maps 1:1
to a piece of output. No accumulation needed.

## 2. Context-only (pure JSON config target)

Library that doesn't render anything to body — the whole output is the
context as JSON.

```yaml
extension: json

context:
  name: ""
  version: "0.1"
  dependencies: []

functions:
  set_name:
    args: [{ kind: capture, name: n, type: any }]
    template: ""
    run: |
      set context.name n

  depend:
    args:
      - { kind: literal, value: "depend" }
      - { kind: literal, value: "on" }
      - { kind: capture, name: pkg, type: any }
    template: ""
    run: |
      append context.dependencies pkg

file_template: |
  {{ .context | toJSONIndent }}
```

When to use: building config files (`.env`, JSON, TOML, YAML, …).

## 3. Body + context combined (Python transpiler)

```yaml
extension: py

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

  say:
    args: [{ kind: capture, name: msg, type: any }]
    template: "print({{ .msg }})\n"

  if:
    args:
      - { kind: literal, value: "if" }
      - { kind: capture, name: cond, type: any }
    block: { closer: end }
    template: |
      if {{ .cond }}:
      {{ .body | indent 4 }}

  end: {}

file_template: |
  {{- range .context.imports }}import {{ . }}
  {{ end }}
  {{- .body -}}
```

When to use: code generators where some source flows to body, other
source contributes to header/setup/imports.

## 4. Operator-style assignment (no leading function name in source)

```yaml
functions:
  assign:
    args:
      - { kind: capture, name: var, type: ident }
      - { kind: literal, value: "=" }
      - { kind: capture, name: value, type: any }
    template: "{{ .var }} = {{ .value }}\n"
```

Source: `x = 42`. The function key `assign` doesn't appear in source
because the args list has a literal.

## 5. Brace-delimited blocks

```yaml
functions:
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

Source:

```
for x in 40 {
    say x
}
```

When to use: C-style languages where `{` `}` delimit blocks.

## 6. Library-defined typed argument

```yaml
types:
  Identifier:
    pattern: "^[A-Za-z_][A-Za-z0-9_]*$"
  Status:
    options: ["todo", "in-progress", "done"]

functions:
  set:
    args:
      - { kind: capture, name: name, type: Identifier }
      - { kind: capture, name: status, type: Status }
    template: "{{ .name }} = {{ .status }}\n"
```

Source: `set my_task "todo"`. Source: `set 9bad "todo"` → error.
