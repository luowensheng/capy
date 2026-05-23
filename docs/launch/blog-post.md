# Capy — a transpiler engine in 1000 lines of Go

*Draft. Tighten before publishing.*

---

I keep writing tiny parsers. A YAML-ish config language for one project,
a route DSL for another, a `Makefile`-generator for a third. Each time
it's the same shape: lex a few tokens, recognise some patterns, fill out
a template, emit some output text.

**Capy** is the engine I wished I had: define your DSL in a YAML file,
write your input, get your output. No code generation, no parser
generator, no template engine alone — a small runtime that does all
three.

```sh
go install github.com/luowensheng/capy/cmd/capy@latest
```

## The shape

A Capy library declares **functions**, **types**, and a **file template**:

```yaml
extension: py
context: { imports: [] }

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

file_template: |
  {{- range .context.imports }}import {{ . }}
  {{ end }}
  {{- .body -}}
```

A source file written against this library:

```
import json
import os
say "hello"
```

…transpiles to:

```python
import json
import os
print("hello")
```

Capy doesn't know what "import" or "say" means. The library does.

## Why not just templating?

Go's `text/template` is great for substituting values. But Capy
**parses** input. The library declares what shapes are valid, types
them, and matches statements before rendering. You don't pre-process
your source by hand; the engine does it.

Compared to a parser generator like ANTLR or tree-sitter: Capy is
deliberately smaller in grammar power (no operator precedence, no
recursive descent over user-defined recursion), but it ships with a
runtime that emits text. You don't have to write the generator
yourself.

## What the engine actually does

For each line of source:

1. Try to match against each library function's pattern (literals +
   typed captures, ordered).
2. On a match: validate types, render the function's template into the
   output body, run the function's `run:` snippet to mutate the
   accumulated `context`.
3. After all statements: render the top-level `file_template:` with
   `body` + `context`.

That's it. No execution of user source. Capy is a transpiler — it turns
input text into output text.

## Two grammars, one engine

There's a quiet thing happening in the design: Capy has **two grammars**.

- **The outer grammar** is whatever the library defines. There are no
  reserved words; `if`, `loop`, `=`, `+`, `:=` are just text the library
  may or may not give meaning. A library with no functions rejects every
  input.

- **The inner grammar** is the small fixed DSL inside each library's
  `run:` field. It has hardcoded `if`/`loop`/`set`/`append`/`merge` —
  enough to update the accumulated context, never user code.

This split is the whole trick. The outer grammar gives library authors
total surface freedom; the inner grammar gives them just enough
imperative power to thread state through the transpilation.

## What I've built with it so far

Real things, not just demos:

- A `.env` generator with typed key names (SCREAMING_SNAKE pattern).
- A test-case DSL that emits `t.Run("name", func(t *testing.T) { … })`
  blocks.
- A tiny SQL builder.

The first one took 20 minutes including writing the script that
exercises it.

## What's missing

Pre-1.0. The library YAML schema may change. Specifically:

- No `else` arm on inner `if`. Use two `if`s.
- No argument defaults (yet).
- No multi-output (each library produces one file).
- No configurable surface syntax beyond what's there (block delimiters,
  arg separators — fixed for v0.1).

See [docs/roadmap.md](https://github.com/luowensheng/capy/blob/main/docs/roadmap.md).

## Try it

```sh
go install github.com/luowensheng/capy/cmd/capy@latest
git clone https://github.com/luowensheng/capy
cd capy
capy run samples/transpile-py/lib.yaml samples/transpile-py/script.capy
```

Then `capy init my-dsl` to scaffold your own.

[GitHub](https://github.com/luowensheng/capy) · [Discussions](https://github.com/luowensheng/capy/discussions) · MIT licensed.
