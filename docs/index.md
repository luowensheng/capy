---
title: Capy
hide:
  - navigation
---

# Capy

> A transpiler engine with **zero default grammar**. You define a tiny
> source language in YAML; Capy generates the target output.

Capy reads source code, matches each statement against library-defined
function shapes, and for each match (a) renders a template fragment and
(b) updates an accumulated **context**. A top-level `file_template:`
assembles `body` + `context` into the final output file.

There are **no built-in keywords**. `if`, `loop`, `=`, blocks, comments —
all defined by the library, or not at all if your DSL doesn't need them.

---

## 30-second teaser

`lib.yaml`:

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

`script.capy`:

```
import json
import os
say "hello, world"
```

Output (Python):

```python
import json
import os
print("hello, world")
```

The library is the entire grammar. Swap it and the same engine produces
HTML, SQL, JSON, Makefiles — anything you can describe with `args:` +
`template:` + `run:`.

---

## Install

```sh
# Go users
go install github.com/luowensheng/capy/cmd/capy@latest

# macOS / Linux (binary, no Go required)
curl -fsSL https://raw.githubusercontent.com/luowensheng/capy/main/scripts/install.sh | sh
```

Or download a binary from the [releases page](https://github.com/luowensheng/capy/releases).

---

## Where to go next

<div class="grid cards" markdown>

- :material-rocket-launch: **[Getting started](getting-started.md)**

    Install, run a sample, and understand the four things every library
    controls.

- :material-pencil: **[Library authoring](library-authoring.md)**

    The reference walkthrough for writing your own `lib.yaml`.

- :material-book: **[Tutorials](tutorials/01-hello-world.md)**

    Four progressive lessons: Hello → config DSL → Python transpiler →
    custom operators.

- :material-toolbox: **[Cookbook](cookbook.md)**

    Recipes for common patterns.

- :material-language-python: **[CAPY_FOR_LLMS](CAPY_FOR_LLMS.md)**

    Single-page brief tuned for AI agents.

- :material-folder-open: **50 demos**

    Full library + script + golden output for every kind of target —
    [see the catalog](https://github.com/luowensheng/capy/tree/main/samples).

</div>

---

## What 50 demos look like

Compact source DSLs producing substantial, useful target code:

| Demo | Target | Source → output |
|---|---|---|
| canvas-game | HTML5 canvas game | 12 lines → 67-line runnable page with sprites, key handlers, RAF game loop |
| react-component | React TSX | 11 lines → typed component with `useState`/`useEffect` |
| landing-page | Responsive HTML | 10 lines → 72-line page with embedded CSS |
| transpile-py | Python | full transpiler: imports, blocks, control flow, indented bodies |
| assembly | x86-64 NASM | high-level source → real assembly; `.data` from accumulated symbols |
| postgres-schema | PostgreSQL DDL | tables + columns + indexes + foreign keys |
| kubernetes | k8s manifest | pure context accumulation → multi-section YAML |
| terraform | HCL | resource blocks with arbitrary inner statements |
| slack-blocks | Block Kit JSON | message DSL → webhook-ready payload |
| openapi | OpenAPI 3 YAML | endpoints + schemas → Swagger-ready spec |

[Browse all 50 demos →](https://github.com/luowensheng/capy/tree/main/samples)

---

## Why Capy?

Not a templating engine (it has a parser). Not a parser generator (it
has a runtime). Something in between: **a configurable transpiler**,
with the configuration written as data.

| Tool                    | What it does                       | What Capy adds |
|-------------------------|------------------------------------|----------------|
| Jinja, Go templates     | Substitute values into text        | A real parser + accumulated context + types |
| ANTLR, lark, tree-sitter| Parse a language you defined       | Targeted at code generation; ships with a runtime; no Java/Python required |
| Custom Go transpilers   | Full control                       | A YAML schema replaces hundreds of lines of code per project |
| gomplate, ytt           | Powerful templating with data      | A source language with custom syntax, not just template inputs |

Use Capy when you'd otherwise hand-roll a tiny parser to drive
code-generation: configuration languages, scaffolding tools, DSLs for
domain experts, source-to-source rewrites.

---

## Status

**Pre-1.0.** The library YAML schema may change between minor versions.
See [CHANGELOG](https://github.com/luowensheng/capy/blob/main/CHANGELOG.md)
for what's stable; [roadmap](roadmap.md) for what's planned.

[MIT licensed](https://github.com/luowensheng/capy/blob/main/LICENSE).
