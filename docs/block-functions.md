# Block Functions

A function that opens a body block is declared via the `block:` key on the
function definition. Capy supports two block modes — pick whichever fits the
surface syntax you're aiming for.

## Mode A — named closer + indentation

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

Source:

```
if x
    say "hi"
end
```

- The body is delimited by INDENT / DEDENT tokens (4 spaces or 1 tab per
  level).
- After DEDENT, the engine expects to match the function named in
  `block_closer` (here `end`).
- The closer is itself a library function. Often a silent one (an
  empty `function end ... end`), but you can give it a `write` that
  emits closing text (e.g. `end_route` emitting a `}`).

### Closers with output

```
function begin_route
    arg capture m string
    arg capture p string
    block_closer end_route
    write `route ${m} ${p} {
${body}`
end

function end_route
    write `}
`
end
```

Source:

```
begin_route "GET" "/api/hello"
    say "inside"
end_route
```

Output:

```
route GET /api/hello {
  inside
}
```

## Mode B — explicit delimiters

```
function for
    arg literal "for"
    arg capture var ident
    arg literal "in"
    arg capture iter any
    block_open "{"
    block_close "}"
    write `for ${var} in ${iter} {
${indent 2 body}
}
`
end
```

Source:

```
for x in 40 {
    say x
}
```

- The body begins immediately after the open token (`{`) and ends at the
  close token (`}`). Newlines inside become statement boundaries.
- No closer function involved.
- Useful for `{ ... }` syntax where the braces *are* the delimiters, not
  an indent block.

## Choosing a mode

| If you want…                                                       | Use…  |
|--------------------------------------------------------------------|-------|
| Python/YAML-like indentation                                       | Mode A |
| Curly-brace languages, `do…end`, BEGIN/END pairs                   | Mode A with explicit-text closer |
| `{...}` blocks (Rust/Go/JS-like)                                   | Mode B |
| You want a "block end" function that also emits text (`}`, `END`)  | Mode A with a templated closer |

## Nesting

Both modes nest cleanly. You can have a Mode-A block inside a Mode-B block
and vice versa. The rendered `.body` of the outer template is the
concatenation of the inner statements' rendered output — whatever shape
those statements were.

```yaml
for x in items {
    if x > 0
        say x
    end
}
```

The above produces a `for` template whose `.body` is the rendered `if`
template, whose `.body` is in turn the rendered `say` call.

## Loader validation

You **must** set exactly one of:
- `block.closer:` (Mode A), or
- both `block.open:` and `block.close:` (Mode B).

The loader rejects libraries that set both or neither.

## What if I want both A and B for the same function?

Define two functions (`for_indent` with Mode A + `for_brace` with Mode B)
and let users pick which to use. The matcher picks the longest match.

## Edge cases

- **Empty bodies**: legal in both modes. The rendered `.body` is the empty
  string.
- **Trailing tokens**: a Mode-B block ends at the close token. Any tokens
  before the next NEWLINE after the close are a parse error.
- **Multi-line opener args**: not currently supported. The opener and its
  args must fit on one logical line.
