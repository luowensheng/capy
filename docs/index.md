---
title: Capy
hide:
  - navigation
---

# Capy

> **Write something simple. Get something polished.**
>
> Capy turns plain, English-like descriptions into real artifacts —
> printable recipe cards, party invitations, weekly schedules,
> reading logs, full websites, code, configs. Anyone can use it;
> developers can extend it.

<iframe src="assets/hero/hero.html" width="100%" height="540" style="border: 0; border-radius: 12px; box-shadow: 0 12px 40px rgba(0,0,0,0.18); display: block; margin: 8px 0 28px;" title="Capy in action"></iframe>

[Open the playground :material-play-box-outline:](playground.md){ .md-button .md-button--primary }
[Get started in 5 min :material-rocket-launch:](getting-started.md){ .md-button }
[For everyone :material-account-multiple:](for-everyone.md){ .md-button }
[Live demos :material-play-circle:](showcase.md){ .md-button }
[Browse all samples :material-folder-open:](https://github.com/luowensheng/capy/tree/main/samples){ .md-button }

---

## Find what you need

Pick the entry point that matches what you want to do:

<div class="grid cards" markdown>

- :material-play-box-outline: **I want to try Capy right now**

    ➜ [**Open the playground**](playground.md) — runs in your browser, six curated samples, edit + preview + download. No install.

- :material-account-multiple: **I'm not a programmer — what is this?**

    ➜ [**Capy for everyone**](for-everyone.md) — the recipe-card sample as the worked example. Five-minute setup, no code background needed.

- :material-rocket-launch: **I'm a developer evaluating Capy**

    ➜ [**Showcase**](showcase.md) (15+ live demos) → [**Use cases**](use-cases.md) → [**Idea language**](idea-language.md) for the framing.

- :material-school: **I want to learn how it works**

    ➜ [**Getting started**](getting-started.md) (5 min) → [**Tutorial 1**](tutorials/01-hello-world.md) → [**Library authoring**](library-authoring.md) reference.

- :material-puzzle: **I want to write my own library / DSL**

    ➜ [**Library authoring**](library-authoring.md) → [`.capy` library syntax](capy-libraries.md) → [Templates](templates.md) → [Types](types.md) → [Inner DSL](inner-dsl.md).

- :material-file-tree: **I want to scaffold a multi-file project**

    ➜ [**One source → many files**](one-source-many-files.md) — web app, Android, iOS, libtorch C++ scaffolding from a single source.

- :material-palette-swatch: **I want a design system across React/Vue/Svelte**

    ➜ [**Design systems**](design-systems.md) — one component declaration, three frameworks, identical visual semantics.

- :material-test-tube: **I want backend code with auto-wired tests**

    ➜ [**Backend codegen**](backend-codegen.md) — every `handler` declaration emits the Go stub AND a matching smoke test.

- :material-rocket-launch-outline: **I want to supercharge an existing format**

    ➜ [**Extend existing syntax**](extending-existing-syntax.md) — Capy as a preprocessor for SQL, Markdown, Dockerfile, K8s manifests.

- :material-language-go: **I want to embed Capy in my Go program**

    ➜ [**Embedding guide**](embedding.md) — `capy.NewLibrary(src)` + `(*Library).Run(src)`. No subprocess, no library files.

- :material-robot: **I want an AI agent to use Capy**

    ➜ [**MCP server setup**](mcp.md) → [**AI integration cookbook**](cookbook-ai.md) (10 recipes) → drop in [the Claude Code skill](https://github.com/luowensheng/capy/tree/main/skills/capy-mcp).

- :material-lightbulb-on-outline: **I want the philosophy**

    ➜ [**Capy as an idea language**](idea-language.md) — describe ideas, libraries implement them, swap implementations without rewrites.

</div>

---

## Capy in one paragraph

Most people end up writing the same things over and over — recipe
cards, invitations, meal plans, reading logs, configs, API specs,
codebases. Capy lets you describe **what you want** in a few plain
lines, and turns it into **what you'd otherwise have to format by
hand**. The vocabulary is whatever you (or someone before you)
designed for the task — `recipe`, `invite`, `endpoint`, `table`,
`scene`, whatever fits. No syntax to memorize. No code to learn.

The demos above show four ready-made vocabularies. There are 50+
more in the repo — and you can write your own (or ask an AI to)
in about thirty minutes.

---
## What makes Capy different

- **No default grammar.** The library is the entire user-facing
  vocabulary. Two libraries → two different DSLs from the same
  engine.
- **Deterministic output.** Same source + same library = byte-
  identical result, every time. CI-friendly. Reviewable.
- **Multi-target by design.** Swap libraries to retarget; the
  source never changes. The [multi-language demo](showcase.md)
  compiles one source to Python, JavaScript, Go, Rust, AND C.
- **Multi-file output.** A single source can scaffold a whole
  project tree (web app, Android, iOS, libtorch C++ trainer).
  Bundle to a zip with `--zip`.
- **Embeddable as a Go library.** No binary required at runtime;
  your program defines its own DSL inline.
- **AI-friendly.** Ships an [MCP server](mcp.md) and a
  [Claude Code skill](https://github.com/luowensheng/capy/tree/main/skills/capy-mcp)
  so agents can use Capy as a tool.
- **Runs in the browser.** [The playground](playground.md) is the
  compiler itself, compiled to WebAssembly.

---

## Install

```sh
# CLI
go install github.com/luowensheng/capy/cmd/capy@latest

# Embed as a Go library
go get github.com/luowensheng/capy

# MCP server for AI agents
go install github.com/luowensheng/capy/cmd/capy-mcp@latest

# Or grab a release tarball (no Go needed)
curl -fsSL https://raw.githubusercontent.com/luowensheng/capy/main/scripts/install.sh | sh
```

Then run a sample:

```sh
git clone https://github.com/luowensheng/capy
cd capy
capy run samples/recipe-card/lib.capy samples/recipe-card/script.capy > my-recipe.html
open my-recipe.html
```

…or just open the [playground](playground.md) in your browser — no
install at all.

---

## Status

Capy is **v0.10**. The engine is stable; the library schema is
stable; expect occasional additions, not breaking changes. Every
release ships:

- CLI binaries for linux / darwin / windows × amd64 / arm64
- The `capy-mcp` MCP server for AI agents
- A Go library import (`github.com/luowensheng/capy`)
- Updated browser playground

See the [CHANGELOG](https://github.com/luowensheng/capy/blob/main/CHANGELOG.md)
for the full version history.

---

*Open source under MIT. Contributions welcome at
[github.com/luowensheng/capy](https://github.com/luowensheng/capy).*
