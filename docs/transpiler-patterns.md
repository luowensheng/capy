# Transpiler Patterns

Capy is, fundamentally, a transpiler: source in, target out. This page
collects the standard ways Capy libraries divide work between
**body templates** (text that flows through statement-by-statement) and
**accumulated context** (state that's assembled at the end).

## Pattern 1: Statement renders directly to body

The simplest pattern. Each statement's `template:` produces text that
appears in the output body in source order.

```yaml
say:
  args: [{ kind: capture, name: msg, type: any }]
  template: "print({{ .msg }})\n"
```

```
say "a"      →     print("a")
say "b"      →     print("b")
```

Used in: most "code generators" where source and output have similar
structure.

## Pattern 2: Statement updates context, body stays empty

Use when the source declares something that contributes to a different
output position (often the top, often deduplicated).

```yaml
import:
  args:
    - { kind: literal, value: "import" }
    - { kind: capture, name: name, type: ident }
  template: ""                        # contributes nothing to body
  run: |
    append context.imports name

file_template: |
  {{- range .context.imports }}import {{ . }}
  {{ end }}
  {{- .body -}}
```

Used in: import collection, decl ordering, "front matter" assembly.

## Pattern 3: Block opener emits header, body emits children, closer emits footer

The canonical control-flow shape.

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

The opener's template references `{{ .body }}` which is the concatenated
output of every statement inside the block. The closer here is silent.

## Pattern 4: Body is irrelevant, context IS the output

Useful for declarative formats (JSON, TOML, YAML configs).

```yaml
set_name:
  args: [{ kind: capture, name: n, type: any }]
  template: ""
  run: |
    set context.name n

file_template: |
  {{ .context | toJSONIndent }}
```

Used in: `samples/transpile-json/`. The body is empty; the entire output is
the marshaled context.

## Pattern 5: Deduplication via context

If the same source can produce duplicate items, use a map-keyed set
instead of a list:

```yaml
context:
  seen_imports: {}                  # map used as a set

functions:
  import:
    args:
      - { kind: literal, value: "import" }
      - { kind: capture, name: name, type: ident }
    template: ""
    run: |
      set context.seen_imports[name] true

file_template: |
  {{- range $name, $_ := .context.seen_imports }}import {{ $name }}
  {{ end }}
  {{ .body }}
```

Each `import json` writes to the same key. `range` over a map iterates
keys.

## Pattern 6: Conditional context updates

The inner DSL's `if` is for library-author logic, not user logic.

```yaml
import:
  args:
    - { kind: literal, value: "import" }
    - { kind: capture, name: name, type: ident }
  template: ""
  run: |
    if (regex_match name "^test_")
        append context.test_imports name
    end
    if not (regex_match name "^test_")
        append context.app_imports name
    end
```

## Pattern 7: Pre/post fragments around a block

When you need to wrap a generated block in target-specific boilerplate,
emit the header in the opener's template, the footer in the closer's
template, and the body via `.body`.

```yaml
begin_class:
  args: [{ kind: capture, name: name, type: ident }]
  block: { closer: end_class }
  template: |
    class {{ .name }}:
    {{ .body | indent 4 }}
end_class:
  template: "    # end class\n"
```

## Pattern 8: One source produces multiple output sections

When the target has distinct sections (e.g. SQL DDL vs DML), accumulate
each into its own context list and join them in the file template.

```yaml
context:
  ddl: []
  dml: []

functions:
  create_table:
    args: [...]
    template: ""
    run: |
      append context.ddl "CREATE TABLE ..."
  insert:
    args: [...]
    template: ""
    run: |
      append context.dml "INSERT INTO ..."

file_template: |
  -- DDL
  {{ range .context.ddl }}{{ . }}
  {{ end }}
  -- DML
  {{ range .context.dml }}{{ . }}
  {{ end }}
```

## Anti-pattern: trying to execute user logic

Capy does not run the source. `if x ... end` in the source emits an `if`
in the target (or whatever the library defines); it doesn't decide at
transpile time whether to skip rendering.

If you want compile-time conditional emission, do it in the **library's
inner `run:` snippet** by reading `context`, not by reading user variables.
