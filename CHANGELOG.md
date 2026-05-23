# Changelog

All notable changes to Capy are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) —
with the important caveat that **while pre-1.0, the library YAML schema
may break between minor versions** (see `CONTRIBUTING.md`).

## [Unreleased]

## [0.8.0] — 2026-05-24

Multi-file generation matures: a `--zip` flag bundles output to a
single archive, three template helpers convert human strings to
language identifiers, and four samples demonstrate the full range —
web app, Android, iOS, libtorch C++ ML trainer.

### Added — CLI / engine

- **`capy run --zip ARCHIVE.zip`** — bundles every declared
  `file:` block into a single zip archive (alternative to
  `--out-dir`). Internal paths use forward slashes for portability.
- **Template helpers** for converting human strings to identifiers:
  - `pascalCase` — `"Habit Tracker"` → `HabitTracker`
  - `camelCase` — `"Habit Tracker"` → `habitTracker`
  - `snakeCase` — `"Habit Tracker"` → `habit_tracker`
  Each strips surrounding quotes so they compose with `string`-typed
  captures.

### Added — samples

- **`samples/webapp-trio/`** — 12-line DSL → `index.html` +
  `app.js` + `styles.css` (habit tracker with localStorage and
  streak counting).
- **`samples/android-app/`** — 15-line declaration → 7 files
  across `app/src/main/{AndroidManifest.xml, java, res/...}` +
  gradle config. Drop into Android Studio.
- **`samples/ios-app/`** — same source shape as android-app →
  SwiftUI App + RootView + Screens + Info.plist + Package.swift.
- **`samples/libtorch-train/`** — 17-line neural-net architecture
  → `model.h` (register_module + forward) + `main.cpp` (training
  loop) + CMakeLists.txt + run.sh. Builds against libtorch with
  CUDA when available.

Each sample's committed `expected/` tree is diffed by CI.

### Added — docs

- **`docs/one-source-many-files.md`** — consistent story behind
  every multi-file sample.
- Homepage card rewritten to lead with project scaffolding.
- Showcase opens with a "Generate a whole project" section.

### Tests

- **`cmd/capy/multitarget_test.go`** diffs every multi-target
  sample against its committed `expected/` tree, plus a
  TestZipOutput that confirms the archive matches the in-memory
  file map.

## [0.7.0] — 2026-05-24

Two complementary features: **actionable errors** with did-you-mean
hints throughout, and **source-file `@import`** for splicing
authored content across files.

### Added — actionable errors

- `domain.CapyError` gains `Hint string` and `File string` fields.
  `FormatWithSource` renders both alongside the caret-pointed
  source line.
- `domain.SuggestClosest(target, candidates, maxDist)` — Levenshtein
  lookup powering "did you mean X?" hints throughout the engine.
- **Upgraded error sites**:
  - **Parser** (no library function matches): suggests the closest
    function name, or lists what's available if no close match.
  - **Type validation** (pattern mismatch): hint shows the offending
    regex so authors can see what's wrong without opening the lib.
  - **Type validation** (options mismatch): hint lists every valid
    option, plus did-you-mean when close.
  - **Library loader** (unknown type): suggests closest type from
    built-ins + library-declared types.
- Error format changed to `file:line:col: message` (was `line N, col
  M: ...`). Editor-clickable. The richer `FormatWithSource` output
  with caret + hint is unchanged.

### Added — source-file `@import`

- `infra.Preprocess(source, dir)` is a line-level preprocessor that
  expands `@import "path"` / `@include "path"` directives at the
  start of a line into the contents of the referenced file.
- **Indentation auto-tracks**: a `    @import "x"` line inlines the
  imported content with each non-blank line prefixed by the same 4
  spaces. Imports nest naturally in surrounding blocks.
- Path resolution is relative to the file containing the directive.
- Cycles detected by absolute path.
- `@import` and `@include` are synonyms.
- `orchestrator.RunMulti` runs the preprocessor over the script
  before the lexer, so this works for every entry point (CLI,
  embedding API, MCP server).

### Added — sample + docs

- **`samples/source-imports/`** — a menu DSL that `@import`s
  shared drinks and desserts sections from `shared/`.
- **`docs/errors-and-debugging.md`** — full reference for the error
  format with worked examples.
- **`docs/multi-file-and-imports.md`** — extended with a
  source-imports section and a comparison table.
- **`infra/preprocessor_test.go`** — 7 tests covering basic import,
  nested imports, indentation preservation, cycle detection, missing
  file, `@include` synonym, malformed directives.
- **`domain/errors_test.go`** — tests for `SuggestClosest`,
  `FormatWithSource` rendering, and the new error format.

### Changed

- Homepage gains an "Errors that tell you how to fix them" card.
- Live showcase opens with two new tabbed sections: error walkthrough
  and source-imports menu sample.
- MkDocs nav exposes `errors-and-debugging.md`.

## [0.6.0] — 2026-05-24

Two structural features: libraries can now declare **multiple
output files** (with subdirectories) in one run, and one library
can **import** others to compose shared types and syntax helpers.

### Added — multi-file output

- **`file "path":` blocks** at the top level of a library. Multiple
  blocks may appear; each declares one output file. Paths may
  contain `/` for subdirectories — the engine creates them as
  needed.

- **`capy run --out-dir DIR`** flag. When the library declares any
  `file:` blocks, --out-dir is required; every block is rendered
  against the same final context+body and written under the dir.

- **Public Go API**: `orchestrator.RunMulti(lib, script)` returns
  both the single-output string and the map of `file:`-block
  outputs. `(*capy.Library).Run` continues to return just the
  string for callers that don't need multi-file.

- **`samples/multi-file-project/`** — 9-line route declaration
  becomes a 6-file FastAPI project tree (README, pyproject.toml,
  .gitignore, src/main.py, src/handlers.py, tests/test_smoke.py).
  Committed under `expected/` as the golden tree; CI test diffs
  every file on every commit.

### Added — library imports

- **`import "path"`** directive at the top of a library. Path is
  relative to the importing file. Imported types, functions,
  context entries, and `file:` blocks are merged in BEFORE the
  importer's declarations. The importer wins on conflict.

- Cycle detection: imports are tracked by absolute path; an
  import cycle is a clean error.

- Format mixing: a `.capy` library can import a `.yaml` library
  and vice versa (the loader sniffs each by file extension).

- **`samples/lib-composition/`** — main `lib.capy` imports
  `common/types.capy` (Email/URL/Semver/Slug) and
  `common/syntax.capy` (tag/note/meta). The merged library
  exposes 6 functions and 4 types.

### Added — docs

- **`docs/multi-file-and-imports.md`** — full reference for both
  features with worked examples and the conflict-resolution rules.
- Homepage gains a "Multi-file projects + library imports" card.
- Live showcase opens with two new tabbed sections:
  "Multi-file projects" and "Library composition."

### Changed

- `domain.Library` gains `Files map[string]string`.
- `infra.RawLibrary` (YAML + Capy parsers) gain `Files` and
  `Imports` fields.
- `features.Evaluator` gains `RunMulti` returning both the single
  output and the per-file map.
- `usecases.RunResult` and `cli.RunOutcome` gain a `Files` field.
- `orchestrator.Run` is preserved for backwards compatibility; new
  callers should prefer `orchestrator.RunMulti`.

## [0.5.0] — 2026-05-24

Repositioning release. Capy is reframed as a tool **anyone** can
use, not just programmers. The homepage opens with an animated
typewriter demo cycling through everyday DSLs (recipe / invite /
meal plan / reading log).

### Added

- **Animated hero** at the top of the homepage
  (`docs/assets/hero/hero.html`). Self-contained HTML/CSS/JS that
  types out source DSL with syntax highlighting, then reveals the
  generated polished output in a side-by-side iframe. Cycles
  through four demos; click dots to jump.

- **Four non-programmer samples**, each a complete library +
  source + golden + README:
  - `samples/recipe-card/` — recipes for home cooks. Six DSL
    keywords (`recipe`, `serves`, `time`, `ingredient`, `step`,
    `tip`). Output: polished HTML recipe card.
  - `samples/event-invite/` — party invitations. Pastel HTML invite
    with RSVP, notes, and "please bring" lines.
  - `samples/weekly-meal-plan/` — weekly dinner planner.
    Green-and-white HTML grid with day-by-day meals + notes.
  - `samples/reading-log/` — children's reading tracker. Bright
    orange certificate with progress bar (pages-read / yearly goal)
    and star ratings (rendered via new `stars` template helper).

- **`docs/for-everyone.md`** — non-programmer guide. Explains the
  vocabulary model with the recipe sample as the worked example,
  walks the three ways to get a library (use a sample, ask an AI,
  write your own), and gives a 5-minute setup.

- **Template helpers** (`infra/template_engine.go`):
  `add` / `sub` / `mul` for numeric math, `percent` for progress-bar
  calculations clamped to 0-100, and `stars` for rendering 1-5
  ratings as ★★★★☆.

### Changed

- Homepage `docs/index.md` opens with the animated hero and a
  reframed lead ("Write something simple. Get something polished.")
  before the feature grid. Adds a "For everyone" button to the
  hero CTAs.

- Live showcase opens with a new "Capy for everyday things — no
  coding needed" section with playable iframes for all four new
  samples.

- MkDocs nav exposes `for-everyone.md` directly under Home.

## [0.4.0] — 2026-05-24

Two new patterns get first-class documentation and worked samples,
plus a new template helper they both rely on.

### Added

- **`samples/contract-first-api/`** — REST API DSL where the
  grammar IS the contract. One source (`script.capy`) feeds three
  libraries (OpenAPI YAML / TypeScript client / Markdown docs);
  each has a committed golden snapshot. CI test
  (`cmd/capy/contract_first_test.go`) re-runs all three on every
  commit so library drift fails fast. Adding a 4th target is a
  ~30-line `.capy` file.

- **`samples/supercharge-markdown/`** — Capy as a Markdown
  preprocessor. 26-line DSL with `post` / `tag` / `para` / `h2` /
  `bullet` / `callout` / `card` / `code` becomes 48 lines of real
  Markdown with YAML frontmatter, blockquote callouts, and inline
  HTML metric cards. Drop into Hugo / Jekyll / MkDocs / Astro
  unchanged.

- **`samples/supercharge-sql/`** — Capy as a Postgres DDL
  preprocessor. Macros (`pk`, `fk`, `timestamps`, `soft_delete`,
  `index`) expand into idiomatic SQL. The database doesn't know
  Capy ran.

- **`docs/grammar-as-contract.md`** — the workflow doc:
  user describes intent → agent drafts library → grammar acts as
  contract → consumers build against it before targets land → CI
  goldens guard against drift. Includes the "build before tested"
  argument.

- **`docs/extending-existing-syntax.md`** — the supercharge-an-
  existing-format playbook with the canonical recipe and a table
  of host formats this pattern fits (SQL, Markdown, HTML,
  Dockerfile, GitHub Actions, Terraform, Kubernetes, OpenAPI,
  Mermaid, …).

- **Template helpers**: new `trimSuffix` and `trimPrefix` in the
  template engine. Useful for generators that emit trailing
  commas (`{{ .body | trimSuffix ",\n" }}`) — needed by
  `supercharge-sql` and similar patterns.

### Changed

- Homepage adds two cards: "Grammar as contract" and "Supercharge
  an existing syntax."
- Live showcase opens with two new tabbed sections demonstrating
  both patterns end-to-end.
- MkDocs nav exposes both new top-level pages.

## [0.3.1] — 2026-05-24

Reframing release. Capy is now consistently presented as a library
written in **Capy's own `.capy` syntax**, with YAML as one supported
alternative format rather than the canonical form. Plus one missing
parser feature needed to make the docs accurate.

### Added

- **`context ... end` blocks in `.capy` libraries.** Same DTO as the
  YAML `context:` section — declares initial context fields with
  defaults (`[]`, `{}`, scalars). Brings `.capy` to feature parity
  with YAML for the four most common library sections (`functions`,
  `types`, `context`, `file_template`).

### Changed

- README, `docs/index.md`, `docs/getting-started.md`,
  `docs/library-authoring.md`, `docs/CAPY_FOR_LLMS.md`, and both
  `skills/*/SKILL.md` files reframed: Capy is a transpiler engine
  driven by a `.capy` library; YAML is supported for the same
  library when downstream tooling needs it.
- README 30-second teaser rewritten in `.capy` syntax (verified to
  run end-to-end).
- Homepage feature cards no longer say "define in YAML" — they say
  "define in a `.capy` library (or YAML)".
- CAPY_FOR_LLMS now shows the schema in `.capy` form first, with
  YAML alongside.

### Why

For users new to Capy, leading with YAML was misleading: it
implied Capy *is* a YAML configuration framework, when it's actually
a transpiler engine that happens to accept libraries in two
equivalent textual formats. The `.capy` form is more idiomatic and
shares lexical rules with the source files the library will parse —
so authors only need one mental model.

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
