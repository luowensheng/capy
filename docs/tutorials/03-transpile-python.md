# Tutorial 3: Transpiling to Python

Build a tiny source language that compiles to runnable Python, with `if`
and `loop` block constructs. Estimated time: 15 minutes.

## Goal

Source:

```
import json
say "hello"
x = 42
if x
    say "x is set"
end
loop item in [1, 2, 3]
    say item
end
```

Output (valid Python):

```python
import json
print("hello")
x = 42
if x:
    print("x is set")

for item in [1, 2, 3]:
    print(item)
```

## Step 1 — basic functions

```yaml
extension: py
output_file: ""

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
    args:
      - { kind: capture, name: msg, type: any }
    template: "print({{ .msg }})\n"

  assign:
    args:
      - { kind: capture, name: name, type: ident }
      - { kind: literal, value: "=" }
      - { kind: capture, name: value, type: any }
    template: "{{ .name }} = {{ .value }}\n"
```

Try `capy run` so far — should handle `import`, `say`, and `x = ...`.

## Step 2 — add the `if` block

The `if` block emits Python `if cond:` + indented body. The body is the
already-rendered output of inner statements.

```yaml
functions:
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

Two key bits:

- `block: { closer: end }` — body is indent-delimited; after DEDENT, the
  `end` function must match.
- `{{ .body | indent 4 }}` — `.body` is the concatenated rendered output
  of the inner statements; `indent 4` prefixes every line with 4 spaces
  for Python's syntax.

## Step 3 — add the `loop` block

```yaml
functions:
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
```

Same shape — emits Python `for x in xs:` + indented body.

## Step 4 — file template

Assemble imports at the top, body below:

```yaml
file_template: |
  {{- range .context.imports }}import {{ . }}
  {{ end }}
  {{- .body -}}
```

The `{{-` and `-}}` trim whitespace to keep output tight.

## Step 5 — run

```sh
capy run lib.yaml script.capy
```

You should see valid Python that runs:

```sh
capy run lib.yaml script.capy | python3
# hello
# x is set
# 1
# 2
# 3
```

## What just happened

The transpiler model has three moving parts:

1. **Body** — concatenated per-statement template output. Flows from
   inner statements outward.
2. **Context** — state collected across all statements, regardless of
   nesting. Used for imports here.
3. **File template** — final assembler that gets both.

Block functions reference `{{ .body }}` to get their inner output. The
file template references both `{{ .context }}` and `{{ .body }}`.

## Try it

- Add an `else` companion by defining a second pattern: `else_if` that
  emits `else:` inside the parent if's body. (Hint: harder than it
  looks; you might need to accumulate body parts into context.)
- Make `say` use `{{ .msg | toQuoted }}` so the source can be
  `say hello` (without quotes).
- Add `import_as <name> as <alias>` that emits Python `import name as
  alias`.

## Next

[Tutorial 4: Custom Operators](04-custom-operators.md) goes deeper into
multi-token patterns and the auto-name-prepend rule.
