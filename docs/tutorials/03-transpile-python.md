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

```
extension py

context
    imports []
end

function import
    arg literal "import"
    arg capture name ident
    append context.imports name
end

function say
    arg capture msg any
    write `print(${msg})
`
end

function assign
    arg capture name ident
    arg literal "="
    arg capture value any
    write `${name} = ${value}
`
end
```

Try `capy run` so far — should handle `import`, `say`, and `x = ...`.

## Step 2 — add the `if` block

The `if` block emits Python `if cond:` + indented body. The body is the
already-rendered output of inner statements.

```
function if
    arg literal "if"
    arg capture cond any
    block_closer end
    write `if ${cond}:
${indent 4 body}
`
end

function end
end
```

Two key bits:

- `block_closer end` — body is indent-delimited; after DEDENT, the
  `end` function must match.
- `${indent 4 body}` — `body` is the concatenated rendered output of
  the inner statements; `indent 4 body` prefixes every line with 4
  spaces for Python's syntax.

## Step 3 — add the `loop` block

```
function loop
    arg literal "loop"
    arg capture var ident
    arg literal "in"
    arg capture iter any
    block_closer end
    write `for ${var} in ${iter}:
${indent 4 body}
`
end
```

Same shape — emits Python `for x in xs:` + indented body.

## Step 4 — file template

Assemble imports at the top, body below:

```
file_template
    for imp in context.imports
        write `import ${imp}
`
    end
    write body
end
```

Each `write` emits exactly the bytes inside the backticks — no
whitespace-trimming sigils needed; you control the output directly.

## Step 5 — run

```sh
capy run lib.capy script.capy
```

You should see valid Python that runs:

```sh
capy run lib.capy script.capy | python3
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

Block functions reference `${body}` (or `${indent N body}` for
indented output) to get their inner output. The file template
references both `${context...}` and `${body}` / `write body`.

## Try it

- Add an `else` companion by defining a second pattern: `else_if` that
  emits `else:` inside the parent if's body. (Hint: harder than it
  looks; you might need to accumulate body parts into context.)
- Make `say` use `${msg | toQuoted}` so the source can be
  `say hello` (without quotes).
- Add `import_as <name> as <alias>` that emits Python `import name as
  alias`.

## Next

[Tutorial 4: Custom Operators](04-custom-operators.md) goes deeper into
multi-token patterns and the auto-name-prepend rule.
