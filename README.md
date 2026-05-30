# Capy

> A transpiler engine with zero default grammar. You define a tiny source
> language in a `.capy` library file, and Capy generates the target output.

[![CI](https://github.com/olivierdevelops/capy/actions/workflows/ci.yml/badge.svg)](https://github.com/olivierdevelops/capy/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/olivierdevelops/capy?include_prereleases)](https://github.com/olivierdevelops/capy/releases)
[![License: Source-Available](https://img.shields.io/badge/License-Source--Available-orange.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/olivierdevelops/capy)](https://goreportcard.com/report/github.com/olivierdevelops/capy)

Capy reads source code, matches each statement against library-defined
function shapes, and for each match (a) renders a template fragment and
(b) updates an accumulated **context**. A top-level `file_template:`
assembles `body` + `context` into the final output file.

There are **no built-in keywords**. `if`, `loop`, `=`, blocks, comments —
all defined by the library, or not at all if your DSL doesn't need them.

---

## 30-second teaser

**`lib.capy`**

```
extension py

context
    imports []
end

function import
    arg literal "import"
    arg capture name ident
    template_str ""
    run:
        append context.imports name
end

function say
    arg capture msg any
    template_str "print({{ .msg }})\n"
end

file_template:
    {{- range .context.imports }}import {{ . }}
    {{ end }}
    {{- .body -}}
```

**`script.capy`**

```
import json
import os
say "hello, world"
```

**Output (Python)**

```python
import json
import os
print("hello, world")
```

The library is the entire grammar. Swap it and the same engine produces
HTML, SQL, JSON, Makefiles — anything you can describe with patterns,
templates, and run blocks.

YAML is supported as a secondary format for teams that need
yq / JSON-schema tooling — same library, same engine. See
[`docs/library-authoring.md`](docs/library-authoring.md#also-supported-yaml).

---

## Why Capy is genuinely useful for AI agents

Two properties most people miss:

1. **Token compression** — agents emit short structured Capy; the engine
   deterministically expands it into long boilerplate-heavy target code.
   12 lines of game-DSL → 67-line runnable canvas game (**5.5×**). 9 lines
   of landing-page DSL → 54 lines of responsive HTML+CSS (**6.0×**). In
   an agent loop the gap compounds — the library is reusable across
   hundreds of invocations.

2. **Sandboxing for free** — the library is the complete grammar. A SQL
   DSL whose `TableName` is an enum **cannot** emit `DROP TABLE`. A
   shell DSL whose `Command` whitelists `ls`/`cat`/`grep` **cannot**
   invoke `rm`. No prompt injection, no post-hoc filtering. The grammar
   is the boundary.

See [docs/ai-agents.md](docs/ai-agents.md) for the token-cost math,
sandboxing patterns, and integrations with Claude Code, Cursor,
Continue, and Aider.

> **50 worked demos** live under [`samples/`](samples/). Compact DSLs
> producing substantial useful targets: a full HTML5 canvas game (12
> lines → 67 lines of working HTML+CSS+JS), responsive landing pages,
> React TSX components with hooks, complete Express/Flask/FastAPI
> servers, production code generators (Python, TypeScript, Go, NASM
> x86-64, Bash, Cobra CLIs), infrastructure (Terraform, Kubernetes,
> Dockerfile, nginx, systemd, GitHub Actions, Prometheus alerts, Chrome
> extensions), schemas (PostgreSQL DDL, Prisma, Zod, XState, GraphQL,
> Protobuf, OpenAPI), and docs (CV, changelog, invoice, blog, Slack
> Block Kit, Mermaid diagrams).

---

## Install

```sh
# Go users — CLI
go install github.com/olivierdevelops/capy/cmd/capy@latest

# Go users — embed Capy as a library
go get github.com/olivierdevelops/capy

# MCP server for AI agents (Claude Desktop, Claude Code, Cursor, Zed)
go install github.com/olivierdevelops/capy/cmd/capy-mcp@latest

# Or: try it in your browser, no install — playground runs Capy compiled
# to WebAssembly with six curated samples (recipe / invite / meal plan /
# reading log / Breakout / Snake). Live editor, preview, download:
# https://olivierdevelops.github.io/capy/playground/

# macOS / Linux (binary, no Go required)
curl -fsSL https://raw.githubusercontent.com/olivierdevelops/capy/main/scripts/install.sh | sh

# Homebrew
brew install olivierdevelops/tap/capy
```

Or download a binary from the [releases page](https://github.com/olivierdevelops/capy/releases).

### Embed in your Go program

No `capy` binary required at runtime. Your program defines its own grammar inline:

```go
import "github.com/olivierdevelops/capy"

lib, _ := capy.NewLibrary(`
    extension html
    function button
        arg literal "button"
        arg capture label string
        template_str "<button>{{ .label }}</button>\n"
    end
`)
out, _ := lib.Run(`button "Click me"`)
// → <button>"Click me"</button>
```

See the [embedding guide](docs/embedding.md) and the runnable
[`examples/embed-html-dsl/`](examples/embed-html-dsl) for full
patterns (loading from disk, multiple grammars per process, hot-swap).

---

## Quick try

```sh
git clone https://github.com/olivierdevelops/capy
cd capy
go build -o capy ./cmd/capy
./capy run samples/recipe-card/lib.capy samples/recipe-card/script.capy
```

---

## Documentation

| Reading order | What it covers |
|---|---|
| [docs/getting-started.md](docs/getting-started.md) | Five-minute tour |
| [docs/embedding.md](docs/embedding.md) | Embedding Capy as a Go library in your own program |
| [docs/mcp.md](docs/mcp.md) | MCP server setup for Claude Desktop / Claude Code / Cursor / Zed |
| [docs/cookbook-ai.md](docs/cookbook-ai.md) | AI integration cookbook (10 recipes) |
| [docs/library-authoring.md](docs/library-authoring.md) | Writing your own `lib.capy` |
| [docs/capy-libraries.md](docs/capy-libraries.md) | `.capy` syntax reference (vs. YAML) |
| [docs/language-reference.md](docs/language-reference.md) | Surface grammar + lexer behavior |
| [docs/inner-dsl.md](docs/inner-dsl.md) | The `run:` operations |
| [docs/types.md](docs/types.md) | `base` / `pattern` / `options` |
| [docs/templates.md](docs/templates.md) | Template helpers |
| [docs/cookbook.md](docs/cookbook.md) | Recipes for common patterns |
| [docs/CAPY_FOR_LLMS.md](docs/CAPY_FOR_LLMS.md) | One-page brief for AI agents |
| [docs/roadmap.md](docs/roadmap.md) | What's planned |

Six worked examples live under [samples/](samples/) — each is a complete
library + script + expected output + README.

---

## Why Capy?

Capy is not a templating engine (it has a parser) and not a parser
generator (it has a runtime). It's something in between: **a
configurable transpiler**, with the configuration written as data.

Compared to alternatives:

| Tool                    | What it does                       | What Capy adds |
|-------------------------|------------------------------------|----------------|
| Jinja, Go templates     | Substitute values into text        | A real parser + accumulated context + types |
| ANTLR, lark, tree-sitter| Parse a language you defined       | Targeted at code generation; ships with a runtime; no Java/Python required |
| Custom Go transpilers   | Full control                       | A `.capy` library replaces hundreds of lines of code per project |
| gomplate, ytt           | Powerful templating with data      | A source language with custom syntax, not just template inputs |

Use Capy when you'd otherwise hand-roll a tiny parser to drive
code-generation: configuration languages, scaffolding tools, DSLs for
domain experts, source-to-source rewrites.

---

## Status

**Pre-1.0.** The library schema may change between minor versions. See
[CHANGELOG.md](CHANGELOG.md) for what's stable; [docs/roadmap.md](docs/roadmap.md)
for what's planned.

---

## Contributing

**Contributions are closed.** No pull requests, no patches. Bug
reports via issues are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md)
for the full statement.

## License

[Capy Source-Available License](LICENSE) — © 2026 Capy authors.

Source-visible for reading, building, and personal evaluation.
Commercial use, redistribution, and derivative works require prior
written permission.
