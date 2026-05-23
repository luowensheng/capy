# Tutorial 4: Custom Operators and Block Contexts

How to define patterns that DON'T start with a function-name literal —
operator-style syntax like `x = 1`, `a + b`, `x := y`. Estimated time: 10
minutes.

## The auto-name-prepend rule recap

If your `args:` has **zero** `kind: literal` entries, Capy auto-prepends
the function's key as a leading literal. That's how `greet <any>` works
without you writing `{kind: literal, value: "greet"}`.

The moment you add any literal, the rule turns OFF. Now the args list IS
the entire shape — and the function key is just a reference name.

## Pattern A: assignment

```yaml
assign:
  args:
    - { kind: capture, name: var, type: ident }
    - { kind: literal, value: "=" }
    - { kind: capture, name: value, type: any }
  template: "{{ .var }} = {{ .value }}\n"
```

Matches `x = 1`. The function key `assign` does not appear in source.

## Pattern B: walrus-operator style

```yaml
walrus:
  args:
    - { kind: capture, name: var, type: ident }
    - { kind: literal, value: ":=" }
    - { kind: capture, name: value, type: any }
  template: "{{ .var }} := {{ .value }}\n"
```

The lexer reads `:=` as one `PUNCT` token because both characters are in
the punctuation set — your literal must match the full text.

## Pattern C: arithmetic-flavored assignment

```yaml
assign_add:
  args:
    - { kind: capture, name: var, type: ident }
    - { kind: literal, value: "=" }
    - { kind: capture, name: a, type: any }
    - { kind: literal, value: "+" }
    - { kind: capture, name: b, type: any }
  template: "{{ .var }} = {{ .a }} + {{ .b }}\n"
```

Matches `x = 4 + 5`. Five tokens, three captures, two literals.

## Pattern overlap and priority

`x = 4 + 5` could match BOTH `assign` (with `value` = `4`, then leftover
`+ 5` — no, actually that wouldn't match because the trailing tokens
don't fit the pattern → match fails) and `assign_add` (full match).

What about `x = 1`? Only `assign` matches because there's no `+` to anchor
the `assign_add` pattern.

When two patterns DO genuinely overlap, Capy picks the one that:

1. Has higher `priority:` (default 0).
2. Has more leading literal tokens (so longer, more specific patterns
   win).

You can usually rely on (2). Set `priority:` explicitly only when (2)
doesn't pick the right one.

## Block constructs with custom delimiters

A `for x in xs { ... }` C-style block:

```yaml
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

This is Mode B (delimiter blocks). The opener consumes everything up to
`{`, then the body is parsed until `}`. Newlines inside the body are
statement boundaries; the `}` ends the block.

Contrast with Mode A (indent + named closer) from Tutorial 3.

## Combining both block modes

You can declare two functions with the same surface keyword, one Mode A
and one Mode B:

```yaml
for_indent:
  args: [{kind: literal, value: "for"}, ..., {kind: literal, value: "do"}]
  block: { closer: end }
  template: ...

for_brace:
  args: [{kind: literal, value: "for"}, ...]
  block: { open: "{", close: "}" }
  template: ...
```

The matcher picks based on which delimiter shows up in source.

## Object literals with unquoted keys

For configuration DSLs, unquoted keys read better:

```
set config {host: "localhost", port: 5432, ssl: true}
```

Object literals accept either quoted strings OR bare identifiers as keys
— no library setting needed.

## Try it

- Add a `pipe` operator: `x |> f` that emits `f(x)` in Python.
- Build a tiny expression language with `+`, `-`, `*`, `/` as
  multi-token patterns. (Capy doesn't have operator precedence — patterns
  are flat — so this requires careful library design.)
- Add a `where` clause to a `find` block: `find x in xs where x > 0
  { ... }`.

## What you've learned across the tutorials

| Tutorial | Concept |
|---|---|
| 1 | Single function + template |
| 2 | Types, context, file template |
| 3 | Block functions (Mode A), body composition |
| 4 | Operator patterns, delimiter blocks (Mode B), priority |

You now have the full toolkit. Look at the [cookbook](../cookbook.md)
for more recipes, or build something real and share it via
[Discussions](https://github.com/luowensheng/capy/discussions).
