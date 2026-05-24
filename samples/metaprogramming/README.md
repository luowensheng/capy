# metaprogramming

**A Capy source that defines its own DSL primitives, inline.**

The library here is intentionally minimal — only one function
(`print`). Everything else — `heading`, `quote`, `todo` — lives in
the SOURCE file as `define ... end` blocks.

This is Capy's metaprogramming feature: a source file can extend
the grammar with new patterns, without requiring the library
author to add them.

## What you write

```
# Define new patterns INLINE — they immediately become callable.
define heading
    arg literal "heading"
    arg capture text string
    template:
        # {{ .text | unquote }}

end

define quote
    arg literal "quote"
    arg capture text string
    arg capture who string
    template:
        > {{ .text | unquote }}
        >
        > — *{{ .who | unquote }}*

end

define checklist_item
    arg literal "todo"
    arg capture done ident
    arg capture text string
    template:
        - [{{ if eq .done "yes" }}x{{ else }} {{ end }}] {{ .text | unquote }}
end

# Use them.
heading "Today's todos"
todo yes "Ship metaprogramming feature"
todo no  "Document on the docs site"
quote "Description over implementation." "an anonymous Capy enthusiast"
```

## What you get

```markdown
# Today's todos
- [x] Ship metaprogramming feature
- [ ] Document on the docs site
> Description over implementation.
>
> — *an anonymous Capy enthusiast*
```

## When metaprogramming is the right tool

- **Source has repetitive boilerplate.** Five `<callout type="note">…</callout>`-shaped statements that all want the same surrounding HTML? Define a `note` pattern once at the top of the file.
- **You don't have edit access to the library.** A consumer can extend a vendor library without forking it.
- **Quick experiments.** Try a new DSL shape inline before promoting it to the library.

## When it isn't

- **Multiple files would use the same pattern.** Move it to the library or a `@import`-ed shared `.capy` file.
- **The pattern needs to interact with `run:` blocks across files.** Library-level features compose more cleanly.

## How it works

When Capy compiles your source, before lexing it runs a tiny pre-pass
that scans for top-level `define NAME ... end` blocks. Each block is
parsed using the same syntax as a library function declaration, then
the resulting functions are merged into the working library (source
defines override library entries with the same name).

The remaining source — with the `define` blocks stripped — is parsed
normally against the augmented library.

[Metaprogramming docs →](../../docs/metaprogramming.md)
