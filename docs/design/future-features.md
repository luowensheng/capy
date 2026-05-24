# Design — what's next for Capy

> Status: **proposed, not implemented.** A grab-bag of features to
> evaluate and prioritise. Each section sketches: what it is, why
> it would help, a concrete shape, open questions, and a rough
> effort estimate. Implementation order is up to discussion.

The features cluster into four groups:

1. **Distribution** — easier install, including WebAssembly.
2. **Ergonomics** — shorter invocation, library auto-discovery,
   shebang-style scripts.
3. **Compilation** — turn a library + script (or just a library)
   into a single runnable artifact.
4. **Ecosystem** — a curated library directory, registry, formatter,
   LSP, watch mode, language SDKs.

---

## 1. Distribution

### 1.1 A curated library directory

**What.** A single page (and machine-readable JSON manifest) listing
every battle-tested Capy library with: name, one-line description,
target output, source repo, license, install command. Hosted on the
project site at `/libraries/` and mirrored in
`docs/assets/library-index.json` for tools to consume.

**Why.** Today a user opens the playground and sees ~60 sample DSLs
— great for browsing, useless for "find me a library that outputs
GitHub Actions YAML." A directory is the entry point.

**Shape.**

```json
[
  {
    "name": "py",
    "target": "Python 3",
    "extension": "py",
    "description": "Tiny imperative DSL → runnable Python.",
    "source": "https://github.com/luowensheng/capy/tree/main/samples/transpile-py",
    "install": "capy lib add capy/py",
    "license": "Source-Available"
  },
  ...
]
```

The page surfaces the JSON via a searchable / filterable table:
filter by target language, output type (HTML, JSON, YAML, code,
shell, asm), domain (web, devops, schemas, games).

**Open questions.**
- Curated by whom? For pre-1.0 it's just the maintainer. For post-1.0
  consider a `community/` namespace with light review (no PRs from
  others today; ask-to-add via issue).
- Versioning. Until libraries themselves carry versions (see § 4.2),
  just link to a commit SHA in the source field.

**Effort.** Small. Mostly content — generate the manifest from the
existing curated playground bundle, write a docs page that loads it.

### 1.2 WebAssembly distribution as an install target

**What.** Make the WASM build first-class: a published `.wasm`
binary plus a thin loader so a browser app can do
```js
const capy = await loadCapy("https://cdn.../capy@0.18.0.wasm");
const out = capy.run(libSrc, scriptSrc);
```

**Why.** Today the WASM lives at
`docs/assets/playground/capy.wasm` and is only used by the
playground. Anyone else wanting to embed Capy in a static site
has to copy the file + `wasm_exec.js` + reverse-engineer the JS
glue. A documented `capy-wasm` package solves that.

**Shape.**
- `npm i @capylang/capy` — installs the wasm bundle + a small
  TypeScript loader.
- `pip install capy-wasm` — same idea via wasmtime-py.
- `capy install --wasm` CLI flag — downloads the right wasm for
  the host's Capy CLI version into `~/.capy/wasm/`.
- The wasm exposes the same `capyRun(libSrc, format, scriptSrc)`
  / `capyDocs` / `capyVersion` surface the playground already uses.

**Open questions.**
- Size: today's wasm is ~6.4 MB. TinyGo could shrink it 5×, but
  TinyGo's reflect support is limited and `text/template` may not
  fly. Worth a benchmark before committing.
- Streaming compile: would let the wasm load incrementally over
  HTTP. Browser support is universal now.

**Effort.** Medium. The hard part is packaging + publishing
flow, not the engineering.

---

## 2. Ergonomics — running Capy like an actual language

The current invocation is `capy run lib.capy script.capy`. That's
explicit, unambiguous, and verbose. Once a user has 10 libraries
and 100 scripts, the verbosity adds up.

### 2.1 Convention-based file extensions

**What.** Allow a script to declare its library via its filename
extension. Library files use `<lib>.capy`; scripts that consume
them use `<filename>.<lib>`. Example:

```
~/proj/
├── recipe.capy           ← library (extension: lib name)
└── lemon-cake.recipe     ← script (extension: chosen lib)
```

Running `capy run lemon-cake.recipe` auto-resolves `recipe.capy`
in the search path (see § 2.2) and uses it.

**Why.** Short, scannable invocation. The OS file-association story
also gets cleaner — register `.recipe` with `capy run` and a
double-click runs it.

**Open questions.**
- Disambiguating "library name" from "output extension" — today the
  library declares `extension html`; if the SCRIPT extension is
  the library name then output extension stays a library-internal
  concern. Slightly different mental model.
- Backwards compat: `capy run lib.capy script.capy` keeps working;
  the new shape is additive.
- What about libraries called `txt` or `md` (collide with common
  content extensions)? Lean: the library author chooses; clashes
  resolve at search-path priority.

**Effort.** Small. The CLI just needs an "auto-detect when one
positional arg is given" branch.

### 2.2 `CAPY_LIBS` search path

**What.** An environment variable that lists directories where
`capy run` looks for libraries by name. Mirrors `PATH` / `PYTHONPATH`:

```sh
export CAPY_LIBS="$HOME/.capy/libs:/usr/local/share/capy"
capy run lemon-cake.recipe
# resolves: $HOME/.capy/libs/recipe.capy
# then:    /usr/local/share/capy/recipe.capy
# then:    ./recipe.capy
# then:    error: library "recipe" not found
```

**Why.** Once libraries get distribution (see § 1.1), users want
them installed centrally rather than copied into every project.

**Shape.**
- Default: `$XDG_CONFIG_HOME/capy/libs/` on Linux,
  `~/Library/Application Support/Capy/libs/` on macOS,
  `%APPDATA%\Capy\libs\` on Windows; falls back to `~/.capy/libs/`.
- `capy lib add <git-url>` clones into the first writable path.
- `capy lib list` shows every library and where it resolves.

**Open questions.**
- Resolution precedence. Strict left-to-right (Unix tradition) is
  predictable.
- Should subdirectories be allowed (`my-team/python.capy`)?
  Yes — `capy run app.my-team/python` becomes the explicit form
  for collision-prone names.

**Effort.** Small.

### 2.3 Shebang-style scripts

**What.** Let a script declare its own library inline via a
shebang line:

```
#!/usr/bin/env capy --lib recipe
recipe "Lemon olive oil cake"
serves 8
…
end
```

Make the script executable (`chmod +x lemon-cake.recipe`); double-
click it or run it directly. The shebang handles the rest.

**Why.** Same goal as § 2.1 — make Capy feel like a real language —
but works without changing the file extension. Plays well with the
Unix tradition of script-as-executable.

**Shape.**
- `capy --lib <name>` flag accepts a library name (resolved via
  CAPY_LIBS, see § 2.2) and runs the script that follows.
- The shebang line is stripped before lexing.

**Open questions.**
- Windows: shebangs require WSL or a registered `.capy` extension.
  Fine to declare that POSIX-only.

**Effort.** Tiny.

### 2.4 `capy <lib> <script>` short form

**What.** When a libraries-by-name search path exists, allow:

```sh
capy recipe lemon-cake.recipe
capy py app.py        # if app.py is the script and py.capy is the lib
```

The first positional arg is treated as a library name (resolved via
CAPY_LIBS); the second is the script. Even shorter than § 2.1 — no
extension convention needed.

**Open questions.**
- Collision with `capy run` / `capy check` / `capy docs`
  subcommands. Resolve: if the first arg matches a known
  subcommand, treat it as such; otherwise try library-name resolution.

**Effort.** Tiny.

### 2.5 `compile` and `run` as first-class subcommands

**What.** Rename / alias today's `capy run lib.capy script.capy`:

| Today                               | Tomorrow                              |
|-------------------------------------|---------------------------------------|
| `capy run lib.capy script.capy`     | `capy run lib script.lib`             |
| `capy check lib.capy`               | `capy check lib`                      |
| (new)                               | `capy compile lib script.lib -o out`  |
| (new)                               | `capy build lib`                      |

`capy compile X` runs the library on X and writes the output to
the library's declared `output_file` (or to `-o out`). `capy run`
stays for "run and print to stdout."

**Why.** Make the verb match the operation. `run` for ephemeral
results; `compile` for "write the artifact." This is the language
people already use when talking about Capy.

**Effort.** Tiny (it's mostly subcommand aliasing).

---

## 3. Compilation

### 3.1 Library → single-binary compiler

**What.** `capy build lib.capy -o lib` produces a standalone
executable that has the library baked in. Run that executable
against any script:

```sh
$ capy build recipe.capy -o recipe
$ ./recipe lemon-cake.recipe
# generates lemon-cake.html
```

The binary is self-contained — no Capy CLI on the deployment host
required.

**Why.** Two big wins:

1. **Distribution.** A team can ship `recipe` (an executable) to
   end users without telling them what Capy is. They get a tool
   that turns `.recipe` files into HTML.
2. **Speed.** No library-load step at every run; the library is
   pre-compiled into the binary's startup.

**Shape.**

```sh
capy build lib.capy -o my-tool
# embeds lib.capy as a Go-string constant in a tiny main.go,
# uses Go to produce a static binary, runs `go build`.
```

For cross-compilation:

```sh
GOOS=linux GOARCH=amd64 capy build lib.capy -o my-tool-linux
GOOS=windows GOARCH=amd64 capy build lib.capy -o my-tool.exe
```

Or via embedded toolchain:

```sh
capy build lib.capy --target linux/amd64 --target windows/amd64
# produces my-tool-linux-amd64, my-tool-windows-amd64.exe
```

**Open questions.**
- Should compiled binaries support multiple libraries? E.g.
  `capy build --bundle py.capy --bundle js.capy -o capy-multi`
  produces a binary that dispatches on `.py` vs `.js` script
  extensions. Useful for "one Capy install per team" pattern.
- License — section 2(b) of the current LICENSE forbids
  redistribution of the binary. Compiling embeds the library
  AND the engine; treat the compiled artifact as a "redistribution"
  or carve out a compile-time exception. (Probably exception:
  the library AUTHOR is allowed to redistribute their compiled
  tool to their team.)

**Effort.** Medium. The hardest part is the cross-compile UX and
licensing language.

### 3.2 Ahead-of-time library validation as part of compile

**What.** `capy compile` runs every check `capy check` does (type
resolution, block-closer wiring, default validation) PLUS attempts
to render `file_template` against a placeholder context. Surface
template syntax errors at build time, not deploy time.

**Why.** Today a typo in `file_template` like `{{ .context.missing }}`
only surfaces when a script triggers it. Compile-time validation
catches more before users see anything.

**Effort.** Small. The hooks already exist.

### 3.3 Watch mode

**What.** `capy watch lib.capy script.capy` re-runs whenever either
file changes. Print the diff against the previous output so
authors see what their edit changed.

**Why.** The library-author edit loop. Saves a `↑ Enter` cycle and
makes the impact of each library change immediately visible.

**Shape.**
- File watcher on `lib.capy` + `script.capy` + any `import`ed paths
  (transitively).
- Output diff (red/green) via the existing CLI styling.
- `--browser` flag opens the result in a browser tab for HTML
  output, auto-reloading on change.

**Effort.** Small with `fsnotify`.

---

## 4. Ecosystem

### 4.1 LSP server for `.capy` libraries (and downstream scripts)

**What.** `capy-lsp` speaks LSP. Editors get:

- Autocomplete for function names declared in the library.
- Hover docs (showing the `description` of the function under cursor).
- Go-to-definition: jump from a `recipe` keyword in a script to
  the `function recipe` in its library.
- Diagnostics: arg-type mismatches, unknown literals, etc.

**Why.** Once libraries are real and people are writing scripts
against them, editor smarts are a big quality-of-life win — and
they're cheap, because Capy already has all the typed metadata.

**Shape.**
- Two modes: editing a `.capy` library (highlighting + validate)
  and editing a script that references one (autocomplete the
  library's keywords).
- For the script mode, the LSP resolves the library via the same
  CAPY_LIBS chain.

**Open questions.**
- How does the LSP know which library to bind a `.capy` or
  `.<lib>` script to? Either: file extension (§ 2.1), inline
  `# lib: recipe` magic comment, or workspace setting.

**Effort.** Medium. The LSP boilerplate is well-trodden; the
analysis pieces (autocomplete, hover) are straightforward because
the library already exposes them via `capy docs`.

### 4.2 Library versioning + lockfile

**What.** Each library can declare `version "1.2.0"` at the top.
A consumer project pins versions via `capy.lock`:

```
# capy.lock
recipe = "https://github.com/x/recipes-capy@v1.2.0"
py = "https://github.com/y/py-capy@v0.4.1"
```

`capy lib add x/recipes-capy` writes to the lock; `capy lib install`
fetches every locked library.

**Why.** Without versioning, a library improvement that changes
output silently breaks consumers' golden tests. With versioning,
upgrades become explicit decisions.

**Effort.** Medium-large. Need a fetcher (start with git URLs), a
lock format, a cache (`~/.capy/cache/`), and CLI plumbing.

### 4.3 `capy fmt` formatter

**What.** Canonical formatting for `.capy` library files:
- Consistent indent (4 spaces).
- Argument alignment within functions.
- Sorted top-level declarations (extension → context → types →
  functions → file_template).
- Trailing-newline / no-trailing-spaces normalisation.

**Why.** Every language that gets popular ends up needing one. Better
to ship it before opinions calcify.

**Effort.** Medium. The lexer + parser already exist; the formatter
walks the parsed RawLibrary back to text.

### 4.4 Language SDKs

**What.** Capy as a library beyond Go:
- `capy-py` (Python via wasmtime).
- `capy-js` (Node via wasm — already mostly works for the browser
  playground; package it for Node).
- `capy-rs` (Rust via wasmtime).

Each SDK exposes `Library.new(libSrc) → run(scriptSrc)`.

**Why.** Embedding Capy in a Python tool, a JS build pipeline, or
a Rust CLI shouldn't require shelling out to a binary.

**Effort.** Medium per language. Once the WASM is packaged
(§ 1.2), each SDK is a thin wrapper.

### 4.5 Sourcemaps for generated output

**What.** Optional: every line of generated output carries a
metadata pointer back to the source script line that produced it.
`capy run --sourcemap` emits a `.map.json` alongside the output.

**Why.** When the user runs the generated code (Python / JS /
shell), errors point at the generated file, not the script. A
sourcemap lets a tool re-map them to the original DSL line.

**Open questions.**
- Format: copy V3 source-maps or invent? Copying gets free tooling
  in browsers.

**Effort.** Medium. The evaluator already knows per-statement
positions; surfacing them takes some plumbing.

### 4.6 Library bundle / vendor

**What.** `capy bundle lib.capy` walks every `import` (library
imports + `@import` source imports declared in `preprocess`) and
produces ONE file with everything inlined. Useful for shipping a
library that depends on others without dragging the dependency
tree.

**Why.** A consumer of your library shouldn't have to fetch six
sub-libraries just to use yours.

**Effort.** Small.

### 4.7 Capy playground served from `capy` CLI

**What.** `capy play [<library>]` opens a local playground on
`localhost:8080` with the library you specify (or a blank slate).
Same UI as the hosted playground but using the local CLI's engine
+ all your locally installed libraries.

**Why.** Lets users prototype against private libraries that can't
be uploaded to the public playground. Also: works offline.

**Effort.** Medium. The playground's static assets ship in the
binary via `embed.FS`; the CLI just spins up an HTTP server.

### 4.8 Hot-reload during script development

**What.** Like watch mode (§ 3.3) but bidirectional: edit the
library, the script re-runs; edit the script, ditto. With
`--browser` it goes one step further — the browser preview
hot-reloads.

**Why.** For interactive DSLs (HTML, Mermaid, Markdown) this
collapses the edit-save-refresh loop to edit-see.

**Effort.** Small (extends § 3.3).

---

## 5. Bigger ideas worth considering

### 5.1 Capy as a transformation DSL (`capy transform`)

Today Capy generates text. A natural extension: take an existing
file, parse it via one library, transform the parsed shape, emit
via another. Concretely:

```sh
capy transform --from openapi.capy --to ts-client.capy spec.yaml > client.ts
```

Where both libraries operate on a shared in-memory shape. This
unlocks "modernise this Travis YAML to GitHub Actions" / "convert
this Express server to a Fastify one" use cases.

**Effort.** Large. Probably needs a v0.20+ feature exploration.

### 5.2 Generated-code provenance metadata

Every generated file gets a leading comment with: source script
path + hash, library path + version, Capy version, generation
timestamp. CI can spot stale generated code by comparing the
hash to the current source. Already done ad-hoc by most libraries;
formalise it via a `capy generate` flag.

**Effort.** Tiny.

### 5.3 Capy-on-Capy (libraries that GENERATE other Capy libraries)

A meta-DSL whose target is `.capy` source. Use case: bootstrap a
new DSL by describing it at a higher level — "I want CRUD
endpoints for these models" → a Capy library that generates the
route DSL + handler templates.

**Effort.** Trivial as a sample; conceptually fascinating.

---

## Priority order (my opinion)

If we ship these one at a time, the order that compounds best:

1. **§ 1.1 Library directory** + **§ 2.2 CAPY_LIBS** — without these,
   none of the ergonomic shortcuts have anything to resolve to.
2. **§ 2.1 / 2.3 / 2.4** — short-form invocation. Makes the tool
   feel like a language.
3. **§ 3.1 Compilation** — the "ship a tool to your team" story.
   Probably the highest end-user payoff of anything here.
4. **§ 1.2 WASM distribution** — opens the door to embedding in
   non-Go ecosystems without the SDK work.
5. **§ 4.1 LSP** + **§ 4.3 `capy fmt`** — quality-of-life for
   library authors and downstream script writers.
6. **§ 4.2 Versioning** — once enough libraries exist that
   pinning matters.
7. **§ 3.3 Watch** + **§ 4.7 `capy play`** + **§ 4.8 Hot-reload** —
   inner-loop polish, after the outer-loop primitives exist.
8. **§ 4.4 Language SDKs** — depends on § 1.2.
9. **§ 4.5 Sourcemaps**, **§ 4.6 Bundle**, **§ 5.1 transform**,
   **§ 5.2 provenance**, **§ 5.3 capy-on-capy** — refinements and
   research bets.

## Out of scope (intentionally)

- A "real" general-purpose programming language. Capy is a
  transpiler engine and stays one. No new control structures
  inside templates beyond `for` / `if`. No expression DSL beyond
  helper calls.
- A built-in package manager with central registry. Distribute
  libraries via git URLs; if someone wants a registry, they can
  build one on top.
- Cloud hosting / SaaS. Capy ships as binaries and source; running
  it is the user's job.
