# Templates

Capy emits text via `write` calls inside function bodies and
`file_template`. The string body of a `write` call is a backtick
literal with `${EXPR}` interpolation — that's the primary surface.

Under the hood Capy translates `write \`...\`` and the control
flow that wraps it into Go [`text/template`](https://pkg.go.dev/text/template)
syntax, so the helpers below are Go-template helpers but you call
them Capy-style via `${func arg arg}`.

## Per-function values in scope

Inside a function body, these are visible to `${EXPR}` interpolation:

| Reference          | Source                                                          |
|--------------------|-----------------------------------------------------------------|
| `${<capture>}`     | One entry per capture; the captured source text as a string.    |
| `${body}`          | The inner block's rendered output (block functions only).       |
| `${context.X}`     | Read-only snapshot of the current accumulated context.          |
| `${func arg arg}`  | Call a helper inline (see Helpers below).                       |

```
function greet
    arg capture name any
    write `Hello, ${name}!
`
end
```

`${name}` is the source text of the captured value (`"Alice"` with
quotes, or `Alice` if the user passed a bare identifier).

## file_template values in scope

In `file_template`:

| Reference        | Source                                                              |
|------------------|---------------------------------------------------------------------|
| `${body}`        | Concatenation of all top-level statements' written output.          |
| `${context.X}`   | The final accumulated context.                                      |

```
file_template
    for imp in context.imports
        write `import ${imp}
`
    end
    write body
end
```

## Helpers

Beyond Go's stdlib helpers, Capy provides:

### `indent N`

Indents every line of a string by N spaces. Most useful for block bodies.

```
function if
    arg literal "if"
    arg capture cond any
    block_closer end
    write `if ${cond}:
${indent 4 body}
`
end
```

### `toQuoted`

Wraps a string in JSON-style double quotes (with proper escaping).
Useful for emitting string literals in target languages.

```
function say
    arg capture msg any
    write `print(${toQuoted msg})
`
end
```

### `toPyLit`

Formats a Go value as a Python literal (`True`/`False`, `None`,
quoted strings, list/dict syntax). Useful when accumulated `context`
carries real Go values you want to splat into Python.

```
file_template
    write `CONFIG = ${toPyLit context.config}
`
end
```

### `toJSON` / `toJSONIndent`

Marshal any value to JSON. Compact and pretty respectively.
Excellent for config-file targets.

```
file_template
    write (toJSONIndent context)
end
```

### `lower` / `upper`

Case helpers.

### `join SEP <list>`

Join a list of strings (or any-types coerced to strings).

```
file_template
    write `scripts: ${join ", " context.script_names}
`
end
```

### `escapeHtml <string>`

HTML-entity-escape the five characters every HTML emitter has to
neutralise: `&`, `<`, `>`, `"`, `'`. Wrap any free-form capture that
flows into HTML so the output stays safe even if the capture
contains markup. The verbose name is deliberate: `${html x}` would
read as "make this HTML"; `${escapeHtml x}` makes the intent
("escape this FOR HTML") obvious at the call site.

```
function p
    arg literal "p"
    arg capture text string
    write `<p>${escapeHtml text}</p>
`
end
```

Source `p "Look at <script>"` then emits `<p>Look at &lt;script&gt;</p>`
— no markup injection. Composes with `unquote` / `decoded` for
captures that need their quotes stripped first:
`${escapeHtml (decoded text)}`.

### `decoded <string>`

Strip outer quotes (if present) AND fully resolve Go-style escape
sequences (`\"` → `"`, `\n` → newline, `\t` → tab, `\\` → `\`,
plus `\xNN` / `\uNNNN`). Use this when a `string`-typed capture
contains escaped quotes or whitespace that the user wrote
literally — `p "He said \"hi\""` becomes `He said "hi"`, not
`He said \"hi\"` like `${unquote x}` would produce.

```
function p
    arg literal "p"
    arg capture text string
    write `<p>${escapeHtml (decoded text)}</p>
`
end
```

Existing libraries that want the source-text quoted form (YAML
output, TypeScript string literals, markdown frontmatter) keep
using `${x}` directly — `decoded` is opt-in.

## Common patterns

### Imports at top, body below

```
file_template
    for imp in context.imports
        write `import ${imp}
`
    end
    write body
end
```

### A block function emitting an indented body

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

### Pure-context output (no body emission)

```
function set_name
    arg capture n any
    set context.name n
end

file_template
    write (toJSONIndent context)
end
```

### Mixing output and state in one function

```
function section
    arg literal "section"
    arg capture title string
    block_closer end
    # Record this section in the TOC AND emit its heading.
    append context.toc title
    write `## ${title}

${body}
`
end
```

## Whitespace

There is no `{{- -}}` trimming sigil in the unified shape. You
control whitespace by where `write` is called and what bytes are
inside the backtick literal. To avoid blank lines between iterations,
make sure each `write` ends with the exact newline you want and no
more:

```
for imp in context.imports
    write `import ${imp}
`
end
```
