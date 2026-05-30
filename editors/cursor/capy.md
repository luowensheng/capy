# Cursor rule for Capy

Drop this into `.cursor/rules/capy.md` (or `Cursor → Settings → Rules`).

```
Apply when the project contains a `lib.yaml` or `*.capy` file.

You are an assistant that designs and edits Capy libraries (see
https://github.com/olivierdevelops/capy).

Capy is a transpiler engine driven by a YAML library. It does NOT execute
user-script code — it matches source against library function shapes and
renders templates while accumulating a `context` value.

Library shape (the entire schema):

  extension, output_file, context, types, functions, file_template

functions[].args entries MUST have an explicit `kind:` discriminator:
  { kind: literal, value: "TEXT" }
  { kind: capture, name: NAME, type: TYPE }

Auto-name-prepend: if args has zero literals, the function key is the
leading literal. With any literal, you own the entire shape (function
key NOT auto-prepended).

Inner DSL (`run:`) operations:
  set <path> <value>
  append/prepend <list-path> <value>
  merge <map-path> <map>
  delete <path>
  if <expr> ... end
  loop <var> in <expr> ... end
  error <message>
  (regex_match value pattern) — for if conditions

Template helpers:
  indent N, lower, upper, join, toQuoted, toPyLit, toJSON, toJSONIndent

Two block modes (use exactly one):
  block: { closer: <name> }            # Mode A — indent + closer
  block: { open: "{", close: "}" }     # Mode B — delimiters

Indentation is 4 spaces or 1 tab per level. No `else` arm. Captures
appear as source text in templates, as evaluated values in run snippets.

After every edit: `capy check lib.yaml`.
```

For full reference, paste `docs/CAPY_FOR_LLMS.md` from the repo.
