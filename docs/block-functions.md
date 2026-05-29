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

## Mode C — multi-token sequence closer (`block_close_seq`)

Modes A and B close on a single keyword or single-character delimiter.
Some grammars — most famously matched-pair HTML — close on a **multi-token
sequence**: `<p>…</p>`, `<div>…</div>`. The closer `</div>` lexes to three
tokens (`</`, `div`, `>`), and a `<div>` body must be terminated by *that
exact sequence* — a stray `</p>` inside it is a mismatched-nesting error.

Declare it with `block_close_seq`, listing the closer's segments. Each
segment is either a **quoted literal** (pre-tokenized the way the lexer
will see it) or a **bare capture name** bound by the opener:

```
function p
    arg literal "<"
    arg literal "p"
    arg literal ">"
    block_close_seq "</p>"
    write `<p>${body}</p>`
end
```

Inside a sequence-closed block, newlines and indentation are
insignificant — the structure comes from the tags, not the layout. The
body is a free-flowing sequence of statements terminated by the closing
sequence.

### Capture-bound closers (one function for every tag)

A ref segment makes the closer **depend on the opener**. An opener that
captures the tag name can close on `</NAME>` generically, so a single
function covers `<div>`, `<p>`, `<span>`, … and each is closed only by its
own matching tag:

```
function element
    arg literal "<"
    arg capture name ident
    arg capture attrs attribute*      # zero or more (see below)
    arg literal ">"
    block_close_seq "</" name ">"     # `name` is a ref to the capture
    write `<${name}${attrs}>${body}</${name}>`
end
```

`<div>` now closes only on `</div>` and `<p>` only on `</p>` — mismatched
nesting (`<div><p></div>`) is a hard parse error. Every ref segment must
name a capture the opener actually binds, or the library fails to load.

## Function-as-type captures (named nonterminals)

A capture's type may name **another library function** instead of a
built-in type. The capture then matches that function's shape and renders
its template:

```
function attribute
    arg capture key ident
    arg literal "="
    arg capture val raw
    write ` ${key}="${val}"`
end
```

`arg capture attrs attribute` (above, in `element`) matches one
`attribute`. Add a quantifier to match several:

| Suffix | Meaning            |
|--------|--------------------|
| (none) | exactly one (mandatory) |
| `*`    | zero or more       |
| `+`    | one or more        |

Two optional separators control repetition — they are **independent**:

| Directive | Role | Applied |
|-----------|------|---------|
| `sep "X"`  | **input** separator | consumed between repetitions while parsing |
| `join "Y"` | **output** separator | inserted between rendered sub-results |

```
arg capture items cell+ sep ","                # comma-separated input
arg capture params param* sep "," join ", "    # comma in, ", " out
```

On interpolation (`${attrs}`, `${items}`) the matched sub-results are
rendered and concatenated — with `join`'s value inserted between them when
set (default: no separator). A `+` capture that matches nothing is a parse
error; a `*` capture that matches nothing renders empty (and `join` never
appears, since there's nothing to separate).

> **Type tip.** A nonterminal's *trailing* capture has no following
> literal to stop on, so prefer single-token types (`raw`, `ident`,
> `word`) over `string` there — a `string` capture runs through the
> expression parser, which can swallow a following delimiter like `>` as
> a comparison operator.

> **`bare` for pure-capture nonterminals.** A function with no `arg
> literal` gets its name auto-prepended as a leading keyword (so `param`
> would require the literal token `param`). If your nonterminal is meant
> to match bare values — e.g. a `param` that matches `a int` — declare it
> `bare` to opt out of the auto-keyword. Nonterminals that already contain
> a literal (like `attribute`'s `=`) need no `bare`.

## Loader validation

You **must** set exactly one of:
- `block.closer:` (Mode A), or
- both `block.open:` and `block.close:` (Mode B), or
- `block_dedent` (indent-closed), or
- `block_verbatim` (raw-byte body), or
- `block_close_seq` (Mode C — multi-token sequence closer).

The loader rejects libraries that set more than one or none.

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
