# Changelog

All notable changes to Capy are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) —
with the important caveat that **while pre-1.0, the library YAML schema
may break between minor versions** (see `CONTRIBUTING.md`).

## [Unreleased]

## [0.3.0] — 2026-05-24

AI-integration release. Capy now ships an MCP server, a dedicated
Claude Code skill, and a cookbook of integration patterns.

### Added

- **`capy-mcp`** — a Model Context Protocol server (`cmd/capy-mcp`)
  exposing three tools over stdio JSON-RPC 2.0:
  - `capy_check` — validate a library; list its functions/types.
  - `capy_run` — transpile a script through an inline library string.
  - `capy_run_file` — same, with paths to existing files on disk.

  Format (`yaml` / `capy` / `auto`) is sniffed from the first
  non-comment line. Shipped as a separate binary in every release
  archive (`capy-mcp`, alongside `capy`). Works with Claude Desktop,
  Claude Code, Cursor, Zed, and any MCP-aware client.

- **`skills/capy-mcp/SKILL.md`** — Claude Code skill describing
  *when* to reach for Capy via MCP and how to operate the three
  tools. Pairs with the pre-existing `skills/capy-author/SKILL.md`.

- **`docs/mcp.md`** — full MCP setup guide with config snippets for
  Claude Desktop, Claude Code, Cursor, and direct JSON-RPC.

- **`docs/cookbook-ai.md`** — ten copy-pasteable recipes covering
  drop-in MCP install, sandboxed agent loops, token compression
  math, AI-builds-library-human-uses-it, typed safe surfaces,
  one-DSL-many-targets, embedded Go agents, skill+MCP wiring,
  self-correcting agents, and prompt-side guidance.

### Changed

- `.goreleaser.yaml` builds and ships the `capy-mcp` binary
  alongside `capy` in every release archive across all five
  platforms (linux/darwin/windows × amd64/arm64).
- Homepage feature grid: new "MCP server + Claude Code skill" card.
- MkDocs nav: "For AI agents" group expanded with MCP setup and the
  integration cookbook.

## [0.2.0] — 2026-05-24

A substantial feature release. Two new entry points (Capy-native
library files and embedding-as-Go-library), seven new showcase
samples including playable HTML5 games, and several engine fixes.

### Added

- **`.capy` library format**: libraries can now be written in Capy's
  own native syntax in addition to YAML. Loader dispatches on file
  extension; same DTO, same engine, byte-identical output. `.capy`
  libraries support `extension`, `output_file`, `function`, `type`,
  and `file_template` top-level blocks. See [`docs/capy-libraries.md`](docs/capy-libraries.md).

- **Embedding API** — top-level `github.com/luowensheng/capy` Go
  package lets programs define DSLs inline and transpile in-memory
  with no separate binary. Public API:

  ```go
  capy.NewLibrary(src) / NewLibraryYAML(src) / NewLibraryFromFile(path)
  (*Library).Run(scriptSrc) (string, error)
  (*Library).Extension() / OutputFile() / FunctionNames()
  ```

  Runnable example at [`examples/embed-html-dsl/`](examples/embed-html-dsl).
  Guide: [`docs/embedding.md`](docs/embedding.md).

- **`type NAME ... end` blocks in `.capy` libraries** with `base`,
  `pattern`, and `options` directives — full parity with the YAML
  `types:` section.

- **Seven new showcase samples**:
  - `samples/multi-language-demo/` — one source → Python, JavaScript,
    Go, Rust, and C (every library ships in BOTH .yaml and .capy form).
  - `samples/3d-tools-demo/` — one scene → Blender, SketchUp, Rhino,
    Unity, Unreal scripts.
  - `samples/typed-config-dsl/` — named typed captures + custom types
    with pattern/options/base; valid + invalid input both committed.
  - `samples/interactive-breakout/` — event-driven Breakout DSL with
    `on_key` / `on_event` primitives; 18-line DSL → 226-line working
    game with playable iframe in the docs.
  - `samples/interactive-snake/` — event-driven Snake with dual key
    bindings (arrows + WASD), event handlers, localStorage scoring.

### Changed

- Outer parser now recursively dispatches library types with `base:`
  to the base type's token rules. Without this, `type Port { base: int }`
  + `port 8443` failed because the lib-type path only accepted
  ident/string tokens.

- `file_template:` in `.capy` libraries now captures to EOF and uses
  the first non-blank line's indent as the strip width — so authors
  can place template actions (e.g. `{{ .body | indent 8 }}`) at
  column 0 for clean nested indentation.

- Golden test harness now picks up samples whose library is either
  `lib.yaml` or `lib.capy`, and skips `lib.capy` from the script glob.

- Infra parsers gain `ParseBytes` methods for in-memory parsing.
  New public `orchfeatures.LoadLibraryFromBytes(format, src, tokenize)`
  used by the embedding API.

### Docs

- New showcase sections: playable games (event-driven DSL), named
  variables & type checking, one scene → five 3D tools, one source
  → five programming languages, `.capy` library format.
- New pages: `docs/embedding.md`, `docs/capy-libraries.md`.
- Homepage feature grid expanded.

## [0.1.0] — 2026-05-23

Initial public release.

### Added

- **Zero default grammar** engine: every user-facing token shape is defined
  by the loaded library.
- **`functions:`** with kind-discriminated `args:` (`{kind: literal, value}`
  or `{kind: capture, name, type}`). Auto-name-prepend rule when args has
  no literals.
- **`types:`** with three optional validators applied in order: `base`,
  `pattern` (regex), `options` (enum).
- **`context:`** for the accumulated transpilation state; **`run:`** for
  context-mutation snippets in a small inner DSL (`set`, `append`,
  `prepend`, `merge`, `delete`, `if`, `loop`, `regex_match`, `error`).
- **`template:`** per function for body output; **`file_template:`** for
  final-file assembly using `.body` + `.context`.
- **Two block modes**: indent/dedent + named closer (`block: { closer }`)
  or explicit delimiter pair (`block: { open, close }`).
- **Template helpers**: `indent`, `lower`, `upper`, `join`, `toQuoted`,
  `toPyLit`, `toJSON`, `toJSONIndent`.
- Object-literal keys accept quoted strings OR bare identifiers
  (`{name: "x"}` is valid).
- Six samples: `empty-engine`, `assembly`, `types`, `scene-dsl`,
  `transpile-py`, `transpile-json`.
- VHCO project layout.

### Known limitations

- No `else` arm on inner `if` (single-arm only).
- No argument `default:` values.
- No multi-output: each library produces exactly one file.
- No configurable surface syntax (statement terminator, arg separator)
  — deferred to a future version.
- No `import` between library files.

[Unreleased]: https://github.com/luowensheng/capy/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/luowensheng/capy/releases/tag/v0.1.0
