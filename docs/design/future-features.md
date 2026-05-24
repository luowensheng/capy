# Future features — comprehensive design

> Status: **proposed, not implemented.** This doc is the long-form
> roadmap. Each feature gets: what it is, the user pain it
> addresses, who benefits, a concrete design with examples, a
> step-by-step walkthrough, trade-offs, an effort estimate, and a
> shipping recommendation. A summary table at the end ranks
> everything by value × effort.

## Reading guide

The features cluster into nine areas. Skip to whichever you care
about — sections are largely independent, with cross-references
where one depends on another.

| § | Area | Anchor features |
|---|------|----------------|
| 1 | **Foundations** | search path, manifest, directory site |
| 2 | **Multiple implementations of one library** | impl selection, defaults, lockfile |
| 3 | **Versioning** | library version, impl version, pin/resolve |
| 4 | **Authoring & publishing** | scaffolding, local libs, git distribution |
| 5 | **Invocation ergonomics** | file-extension conventions, shebang, short form, compile/run/build subcommands |
| 6 | **Distribution** | WASM packaging, single-binary compiler |
| 7 | **Editor / tooling** | LSP, formatter, watch, local playground, sourcemaps |
| 8 | **SDKs and embedding** | Go, Python, JS, Rust |
| 9 | **Big ideas** | `capy transform`, provenance, capy-on-capy, bundle |

A summary recommendation table at the bottom (§ Decisions)
collects every feature with effort vs. value scores and a
SHIP / DEFER / SKIP verdict.

---

# § 1 Foundations

These three features are infrastructure — nothing else in the doc
works without them. Ship together, in this order.

## 1.1 `CAPY_LIBS` search path

### What it is

An environment variable listing directories where `capy` searches
for libraries by name, mirroring `PATH` / `PYTHONPATH`:

```sh
export CAPY_LIBS="$HOME/.capy/libs:/usr/local/share/capy/libs"
capy run recipe lemon-cake.recipe
# resolves: $HOME/.capy/libs/recipe.capy
# then:    /usr/local/share/capy/libs/recipe.capy
# then:    ./recipe.capy
# then:    error: library "recipe" not found
```

### User pain it addresses

Today every script invocation needs a full path to the library:
`capy run ../shared/recipe.capy lemon-cake.recipe`. As soon as a
team has three libraries reused across ten projects, paths get
out of sync, copies drift, and `cp` becomes the dependency manager.

### Who benefits

- **Script writers:** type `capy run recipe …` instead of pasting
  paths.
- **Library authors:** ship a library by installing it; consumers
  don't reorganise their projects to use it.
- **Teams:** put `/usr/local/share/capy/libs` on every dev box and
  every CI runner; one source of truth.

### Design

| Platform | Default path |
|---|---|
| Linux | `$XDG_CONFIG_HOME/capy/libs/` (or `~/.config/capy/libs/`) |
| macOS | `~/Library/Application Support/Capy/libs/` |
| Windows | `%APPDATA%\Capy\libs\` |
| Fallback | `~/.capy/libs/` |

`CAPY_LIBS` overrides the default; entries are colon-separated
(semicolon on Windows). Resolution is left-to-right, first match
wins.

Subdirectories are part of the name:

```sh
capy run my-team/python script.my-team/python
# resolves: $CAPY_LIBS[*]/my-team/python.capy
```

CLI helpers:

```sh
capy lib list                              # print all libs found in CAPY_LIBS
capy lib which recipe                      # show full path of the resolved lib
capy lib add ./my-recipe                   # copy/symlink into the first writable dir
capy lib add github.com/user/recipes       # clone into the first writable dir
capy lib remove recipe
```

### Walkthrough

```sh
# One-time setup.
mkdir -p ~/.capy/libs

# Pull a community library.
capy lib add github.com/example/recipes
# → cloned into ~/.capy/libs/recipes-capy/

# Or write your own.
mkdir ~/.capy/libs/recipe
cat > ~/.capy/libs/recipe/recipe.capy <<'EOF'
extension html
function recipe
    arg literal "recipe"
    arg capture title string
    block_closer end
    set context.title title
end
# … rest of the library
EOF

# Use it.
echo 'recipe "Lemon cake"\nserves 8\nend' > cake.recipe
capy run recipe cake.recipe > cake.html
```

### Trade-offs

- **Pro:** mirrors decades of POSIX convention; users know how
  search paths work.
- **Pro:** "library by name" is the precondition for every
  short-form invocation in § 5.
- **Con:** introduces a global resolution step — a script's
  behaviour now depends on the environment, which is bad for
  reproducibility unless paired with lockfiles (§ 3).

### Effort

**Small.** ~200 LOC for the resolver, subcommand wiring,
cross-platform default paths.

### Recommendation

**SHIP — first.** Foundation for everything in § 2, § 4, § 5.

---

## 1.2 Library manifest (`capy.toml`)

### What it is

A small TOML file alongside every library file (or inside its
directory). Declares metadata the engine doesn't infer from the
library source:

```toml
# ~/.capy/libs/recipe/capy.toml
[library]
name        = "recipe"
version     = "1.4.0"
description = "Recipe DSL → printable HTML card."
license     = "Source-Available"
authors     = ["Alice <alice@example.com>"]
homepage    = "https://github.com/alice/recipe-capy"

# Output classification (helps the library directory filter).
[output]
extension = "html"
kind      = "html"           # html | json | yaml | toml | code:py | shell | asm | … 

# Implementations of this library's interface (see § 2).
[[impl]]
name        = "html"          # default
file        = "recipe.capy"   # path relative to capy.toml
description = "Printable HTML recipe card."

[[impl]]
name        = "json"
file        = "recipe.json.capy"
description = "Same interface, JSON output for APIs."

[[impl]]
name        = "markdown"
file        = "recipe.md.capy"
description = "Markdown for blogs / READMEs."

# Default impl when none is specified.
default-impl = "html"

# Dependencies on other libraries (see § 3).
[deps]
common-types = { version = "^2.0.0", source = "github:capy/common-types" }
```

### User pain it addresses

Today every piece of library metadata lives in two places:
sometimes as a `description "..."` directive at the top of the
`.capy` file, sometimes implicitly in the file name. To list every
library on a machine, you'd have to parse every file. To check
whether two are compatible, you'd have to read both.

### Who benefits

- **Library directory** (§ 1.3) reads manifests to build the index.
- **`capy lib list`** uses them to print human-readable summaries.
- **Versioning + lockfiles** (§ 3) need a canonical place to read
  `version` from.
- **Multiple-impl support** (§ 2) requires a canonical place to
  enumerate impls.
- **LSP** (§ 7.1) uses the manifest to discover where the library
  file is.

### Design

The manifest is optional for libraries that are just one file with
no fancy features — the engine still loads bare `.capy` files. But
once a library wants versioning, multiple impls, or a published
description, it needs a manifest.

Lookup order when resolving a library by name `X`:

1. `$CAPY_LIBS[i]/X/capy.toml` — directory-style library
   with a manifest. Read the manifest, find the default impl's
   file, load it.
2. `$CAPY_LIBS[i]/X.capy` — bare-file library. No metadata.

### Walkthrough

```sh
capy lib new recipe        # scaffolds a directory with capy.toml
cd ~/.capy/libs/recipe
ls
# capy.toml  recipe.capy  README.md

cat capy.toml
# [library]
# name    = "recipe"
# version = "0.1.0"
# …
```

### Trade-offs

- **Pro:** clean place for metadata that doesn't belong in the
  library source.
- **Pro:** extensible — future features add new keys without
  changing the library source.
- **Con:** another file to learn. Mitigation: `capy lib new`
  scaffolds it; bare-`.capy` libraries continue to work for
  one-off use.

### Effort

**Small.** TOML parser is in the Go standard ecosystem.

### Recommendation

**SHIP — alongside § 1.1.** The manifest IS the unit of
distribution.

---

## 1.3 Library directory site

### What it is

`/libraries/` on the docs site. A searchable, filterable table of
every curated library: name, version, description, output target,
domain, link to source, install command.

Backed by a machine-readable manifest at
`docs/assets/library-index.json` so other tools (LSP, registry,
external indexes) can consume it.

### User pain it addresses

A new user opening the playground sees 60 sample DSLs in a
dropdown — no way to find "the one that outputs Kubernetes
manifests." The samples are organised for the playground, not for
discovery.

### Who benefits

- **New users:** "I need to generate X — what library do I want?"
- **AI agents:** the JSON manifest is the perfect bootstrap
  prompt — "here are the libraries available; pick one."
- **Library authors:** visibility for libraries that are otherwise
  buried in samples/.

### Design

Page layout:

```
┌─────────────────────────────────────────────────────────────┐
│  Libraries                                  [Search ____]    │
│                                                              │
│  Filter:  [Target ▾] [Kind ▾] [Domain ▾] [License ▾]        │
│                                                              │
│  ┌────────────┬───────────────────────────┬─────────────┐   │
│  │ recipe 1.4 │ Recipe DSL → printable    │ html        │   │
│  │            │ HTML card                 │ Source-Av.. │   │
│  ├────────────┼───────────────────────────┼─────────────┤   │
│  │ py 0.18    │ Tiny imperative DSL →     │ code:py     │   │
│  │            │ runnable Python           │ Src-Av..    │   │
│  └────────────┴───────────────────────────┴─────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

Click a row → library detail page with full description, source,
impl list (§ 2), install command, screenshots / generated-output
samples.

### Walkthrough

```sh
# Browser visits luowensheng.github.io/capy/libraries/
# User searches "kubernetes", filters by Kind: yaml
# Picks the `k8s-deploy` library, clicks Install:
capy lib add github.com/example/k8s-deploy

# Now usable.
capy run k8s-deploy app.k8s-deploy
```

### Trade-offs

- **Pro:** turns Capy from "a transpiler engine + some samples"
  into "an ecosystem."
- **Con:** curation is work. Pre-1.0, the maintainer curates; the
  doc page renders the manifest, that's it.

### Effort

**Small.** Static page + JSON manifest generated from
`cmd/playground-bundle` (already enumerates samples).

### Recommendation

**SHIP — alongside § 1.1 / 1.2.** Without it, the search path
has nothing visible to populate from.

---

# § 2 Multiple implementations of one library

### Motivation (please read this first)

A library declares an *interface*: the function shapes the source
file expects to use. An *implementation* defines what each
function emits. Different implementations of the same interface
produce different outputs from the same source.

The simplest example: a chart DSL.

```
chart "Quarterly revenue"
    series "2025" [120 130 145 160]
    series "2024" [100 110 130 150]
end
```

Three impls of the `chart` library:

- **`chart` (mermaid):** emits a Mermaid `xychart-beta` block.
- **`chart` (d3):** emits an HTML page with D3.js.
- **`chart` (ascii):** emits an ASCII bar chart for a terminal.

Same source. The user picks the impl per run.

### Current state

Today: a library is one `.capy` file. To get three outputs you
either write three libraries (`chart-mermaid.capy`,
`chart-d3.capy`, `chart-ascii.capy`) and the source has to commit
to one upfront, or you write three sample subdirectories.

That's not great — the source ends up coupled to a specific impl,
which defeats the "spec-as-source" pitch.

### Who benefits

- **Source authors** stay tool-agnostic. The same `chart` source
  follows them from Slack-as-ASCII to slides-as-Mermaid to
  product-dashboards-as-D3.
- **Library authors** can ship a clean interface library + several
  impls without forcing consumers to commit upfront.
- **Multi-target teams** (Web + Mobile + Backend) get parity:
  declare interface once, ship per-target impls.

## 2.1 Interface + implementation split

### Design

A library directory contains:

```
~/.capy/libs/chart/
├── capy.toml               # declares the interface and lists impls
├── interface.capy          # function shapes only (no `write` calls)
├── impl/
│   ├── mermaid.capy        # impl: emits Mermaid
│   ├── d3.capy             # impl: emits D3 HTML
│   └── ascii.capy          # impl: emits ASCII art
└── README.md
```

`interface.capy` declares the function shapes that scripts will
match against, but every function body is empty (no `write`, no
state mutation). Authors of consuming scripts read THIS file to
know what the library accepts.

```
# interface.capy
extension ""   # impl-dependent; left blank here

function chart
    description "Open a chart with a title."
    arg literal "chart"
    arg capture title string  "Chart title shown above the plot."
    block_closer end
end

function series
    description "One data series on the chart."
    arg literal "series"
    arg capture name string  "Legend label."
    arg capture values any   "List of numbers."
end

function end
end
```

Each impl `extends` the interface, re-declaring each function with
a body:

```
# impl/mermaid.capy
extends "chart"            # inherits arg shapes from interface.capy

extension md
output_file ""

context
    title ""
    series []
end

function chart
    set context.title title
    write `\`\`\`mermaid
xychart-beta
    title "${unquote context.title}"
${body}
\`\`\`
`
end

function series
    append context.series {name: name, values: values}
    write `    line "${unquote name}" ${values}
`
end

function end
end
```

The `extends "chart"` directive tells the loader: "use the arg
shapes from `chart`'s interface file; my function bodies provide
the rendering."

### Walkthrough

```sh
# Install the library (contains all impls).
capy lib add github.com/example/chart

# List impls.
capy lib impl chart
# chart 1.4.0
#   - mermaid (default)
#   - d3
#   - ascii

# Pick one at run time.
capy run --impl mermaid chart revenue.chart > revenue.md
capy run --impl d3      chart revenue.chart > revenue.html
capy run --impl ascii   chart revenue.chart  # prints to terminal

# Or set a default for this shell.
export CAPY_IMPL_CHART=ascii
capy run chart revenue.chart

# Or pin per project.
echo '[impl]\nchart = "d3"' >> capy.toml
```

### Trade-offs

- **Pro:** the same source becomes genuinely portable across
  output targets.
- **Pro:** the interface file IS the documentation. `capy docs
  --interface chart` would describe what the source can express
  without committing to any output shape.
- **Con:** new file (`interface.capy`). Mitigation: a single-impl
  library doesn't need an interface file — just one `.capy` works.

### Effort

**Medium.** Engine learns `extends "X"` directive that loads
function shapes from another file. Loader merges
interface-declared `arg` lines with the impl's bodies. CLI grows
`--impl` flag + impl-selection logic.

### Recommendation

**SHIP — high value.** This is the missing piece that makes
Capy's "spec-as-source" pitch real for multi-target work.

## 2.2 Impl selection: flag, env, lockfile, default

### Design

Selection precedence (highest wins):

1. **CLI flag**: `--impl <name>`
2. **Per-project lockfile**: `[impl] chart = "d3"` in `capy.toml`
   (the project's, not the library's)
3. **Env var**: `CAPY_IMPL_CHART=d3`
4. **Generic env var**: `CAPY_IMPL=d3`
5. **Library's `default-impl`** in its `capy.toml`

If none of the above match, error with the available impl names.

### CLI

```sh
# Discover.
capy lib impl chart

# Run with explicit pick.
capy run --impl mermaid chart script.chart

# See what would be picked.
capy lib resolve chart
# chart 1.4.0
# selected impl: d3 (from project capy.toml)
# search path:   ~/.capy/libs/chart/
```

### Per-project pinning

A project at `~/work/dashboards/` can pin every library AND impl
it uses:

```toml
# ~/work/dashboards/capy.toml
[deps]
chart = { version = "^1.0", source = "github:example/chart" }
table = { version = "^2.1", source = "github:example/table" }

[impl]
chart = "d3"
table = "html"
```

`capy run script.chart` from anywhere inside `~/work/dashboards/`
honours the pin.

### Trade-offs

- **Pro:** consistent across projects but overridable per-script,
  per-shell.
- **Pro:** the env-var form (`CAPY_IMPL_CHART`) is the obvious
  knob for shell experimentation; the lockfile form is the
  obvious knob for CI.

### Effort

**Small** once the impl model exists (§ 2.1).

### Recommendation

**SHIP — with § 2.1.** They're one feature.

---

# § 3 Versioning

### Motivation

Once libraries are real — installed, shared, depended on — they
need versions. Without versions, "library improvement that changed
the indentation" silently regenerates everyone's output and
breaks golden diffs. With versions, upgrades are explicit
decisions.

## 3.1 Library + implementation versions

### Design

Both libraries and individual impls have semantic versions:

```toml
# ~/.capy/libs/chart/capy.toml
[library]
name    = "chart"
version = "1.4.0"           # library version (interface stability)

[[impl]]
name    = "mermaid"
version = "1.2.3"           # impl version (output stability)
file    = "impl/mermaid.capy"

[[impl]]
name    = "d3"
version = "2.0.0-beta.4"
file    = "impl/d3.capy"
```

**Library version** changes when the *interface* changes —
adding/removing function shapes, changing arg types. SemVer:
breaking changes bump MAJOR.

**Impl version** changes when the *output* changes —
indentation tweaks, helper-template fixes, new Mermaid syntax.
Bump MINOR for visual additions, PATCH for fixes.

A given library version can have multiple compatible impl
versions: `chart 1.4.0` ships `mermaid 1.2.3`, `d3 2.0.0-beta.4`,
`ascii 1.1.0`. Upgrading the library bumps the interface; impls
bump independently.

### CLI

```sh
capy lib list
# chart      1.4.0   (mermaid 1.2.3, d3 2.0.0-beta.4, ascii 1.1.0)
# recipe     2.1.5   (html 2.1.5, json 1.0.0)
# k8s        0.7.2   (yaml 0.7.2)

capy lib upgrade chart           # latest matching the lockfile constraint
capy lib upgrade chart --impl d3 # only the d3 impl
```

### Walkthrough

```sh
# Add a library at a specific version.
capy lib add github.com/example/chart@v1.4.0

# Or constrain a range.
capy lib add github.com/example/chart@^1.4

# See what's installed.
capy lib list --versions

# Check for available upgrades.
capy lib outdated
# chart 1.4.0 → 1.5.2 available
# recipe 2.1.5 — up to date
```

### Trade-offs

- **Pro:** the standard SemVer story applies; no Capy-specific
  surprises.
- **Pro:** independent impl versions let an impl maintainer ship a
  fix without re-cutting the entire library.
- **Con:** the cognitive load of two version axes. Mitigation:
  most users won't notice — `capy lib upgrade` just does the
  right thing.

### Effort

**Medium.** Parser for SemVer ranges, resolution algorithm,
lockfile semantics. The Go ecosystem has battle-tested libraries
(`golang.org/x/mod/semver`).

### Recommendation

**SHIP — but after § 1, § 2.** Versioning matters once there are
external libraries to depend on.

## 3.2 Lockfile (`capy.lock`)

### Design

A `capy.lock` next to a project's `capy.toml` records the *exact*
versions resolved at install time. Commit it; CI re-resolves to
the same versions.

```toml
# capy.lock
generated-by = "capy 0.20.0"
generated-at = "2026-05-25T12:00:00Z"

[[lib]]
name    = "chart"
version = "1.4.0"
source  = "github:example/chart"
sha256  = "9c2b5e…a8f7"

[[lib]]
name    = "chart"
impl    = "d3"
version = "2.0.0-beta.4"
sha256  = "f1b9…0142"

[[lib]]
name    = "recipe"
version = "2.1.5"
source  = "github:example/recipe"
sha256  = "5e7a…d31c"
```

### Walkthrough

```sh
# First add.
capy lib add github.com/example/chart@^1.4
# Writes capy.toml AND capy.lock.

# On a fresh checkout.
capy lib install
# Reads capy.lock, fetches each entry, verifies sha256.

# Upgrade.
capy lib upgrade chart
# Bumps lock to highest matching version; commits diff for review.
```

### Trade-offs

- **Pro:** reproducible builds.
- **Pro:** SHAs catch silent upstream tampering.
- **Con:** more files in the repo. Mitigation: just commit them
  alongside `capy.toml`; people are used to it.

### Effort

**Small** on top of § 3.1.

### Recommendation

**SHIP — with § 3.1.**

---

# § 4 Authoring & publishing

### Motivation

A user who finds Capy and likes it should be able to write a
library, install it on their own machine, share it with their team,
and (eventually) publish it. Each step should feel obvious.

## 4.1 `capy lib new` scaffolding

### Design

```sh
capy lib new my-recipe
# Creates ~/.capy/libs/my-recipe/ with:
#   capy.toml          (filled in with name, version 0.1.0, your git author)
#   my-recipe.capy     (a minimal "hello world" library)
#   examples/
#     hello.my-recipe  (a sample script)
#   README.md
#   .gitignore
```

Flags:

```sh
capy lib new my-chart --interface             # scaffold an interface + one impl
capy lib new my-chart --impl mermaid --impl d3
capy lib new my-recipe --target html
capy lib new my-recipe --output-file "recipe.html"
```

### Walkthrough

```sh
$ capy lib new lemon-recipes
✓ created ~/.capy/libs/lemon-recipes/
✓ capy.toml
✓ lemon-recipes.capy (minimal library)
✓ examples/hello.lemon-recipes
✓ README.md

$ cd ~/.capy/libs/lemon-recipes
$ capy run lemon-recipes examples/hello.lemon-recipes
Hello from lemon-recipes!

# Now edit lemon-recipes.capy to add real functions.
# `capy check lemon-recipes` validates as you go.
```

### Trade-offs

- **Pro:** zero-friction onboarding. The output of `capy lib new`
  is already a working library.
- **Pro:** the scaffolded `capy.toml` sets the right defaults
  (license, author from git config).

### Effort

**Small.** Templated file generation.

### Recommendation

**SHIP — early.** Make first-library experience smooth.

## 4.2 Local path libraries

### What it is

A library that lives in your project tree, not in `CAPY_LIBS`.
Useful while developing one — no need to install before testing.

### Design

`capy.toml` accepts path-based deps:

```toml
[deps]
my-recipe = { path = "./libs/my-recipe" }
```

Or invoke directly:

```sh
capy run ./libs/my-recipe/lemon-recipes.capy script.lemon-recipes
```

### Walkthrough

```sh
# Start: write a lib in your project.
mkdir -p my-app/libs/notes
cat > my-app/libs/notes/notes.capy <<'EOF'
extension md
function note
    arg literal "note"
    arg capture text string
    write `> ${unquote text}

`
end
EOF

# Use it via path.
capy run ./libs/notes/notes.capy script.notes

# Or register in project capy.toml.
cat >> my-app/capy.toml <<'EOF'
[deps]
notes = { path = "./libs/notes" }
EOF

capy run notes script.notes     # path-based dep resolved
```

### Trade-offs

- **Pro:** zero-install dev loop.
- **Pro:** library lives in version control alongside the code
  that consumes it.

### Effort

**Tiny.** Already works via direct paths; just add `path = …`
support to `capy.toml`.

### Recommendation

**SHIP — with § 4.1.**

## 4.3 Git-based distribution

### Design

`capy lib add <git-url>` clones into the first writable
`CAPY_LIBS` directory. Standard URL shorthands:

```sh
capy lib add github.com/example/chart            # → https://github.com/example/chart
capy lib add gitlab.com/example/chart            # → https://gitlab.com/example/chart
capy lib add ssh://git@example.com/chart.git
capy lib add github.com/example/chart@v1.4.0     # specific tag
capy lib add github.com/example/chart@main       # branch
```

The clone gets a `version` from the tag (or branch name); the
SHA goes into the lockfile.

### Walkthrough

```sh
$ capy lib add github.com/example/chart
Cloning https://github.com/example/chart → ~/.capy/libs/chart/
Resolved version 1.4.0
Resolved impls: mermaid (1.2.3), d3 (2.0.0-beta.4), ascii (1.1.0)
Updated capy.lock

$ capy run chart my-data.chart > out.md
```

### Trade-offs

- **Pro:** uses tooling everyone already has (`git`).
- **Pro:** zero hosting infrastructure on Capy's side.
- **Con:** less convenient than `npm install x`. Mitigation: most
  developers are used to git URLs for tooling these days.

### Effort

**Small.** Standard subprocess wrapping; existing libraries
(`go-git`) for in-process git ops.

### Recommendation

**SHIP — with § 1.1.**

## 4.4 Future: registry

### What it is

`registry.capy.dev` (or similar) — central index of published
libraries, served as static JSON. `capy lib publish` uploads a
manifest there.

### When to build

After § 1 through § 4.3 are stable and there's enough community
demand that "find me a library that does X" needs more than
`/libraries/`.

### Effort

**Large.** Hosting, moderation, abuse mitigation, semver storage,
search. Don't build prematurely.

### Recommendation

**DEFER** — until ecosystem demand justifies it.

---

# § 5 Invocation ergonomics

### Motivation

The current invocation `capy run lib.capy script.capy` is
explicit and unambiguous. Once a user has 10 libraries and 100
scripts, that explicitness gets tedious. These features layer
shortcuts on top of the foundation work in § 1.

## 5.1 File-extension convention

### Design

A script whose extension is the name of a library is auto-
resolved:

```
~/proj/
├── recipe.capy             # library (file = lib-name + .capy)
└── lemon-cake.recipe       # script (extension = lib-name)
```

```sh
capy run lemon-cake.recipe
# auto-resolves: $CAPY_LIBS[*]/recipe/recipe.capy or ./recipe.capy
```

### Walkthrough

```sh
$ capy lib new recipe
$ cat > cake.recipe <<'EOF'
recipe "Lemon olive oil cake"
ingredient "Flour" "2 cups"
end
EOF

$ capy run cake.recipe          # auto-detects lib from .recipe extension
# … HTML output …
```

### Edge cases

- Extension collides with a common format (`.md`, `.json`):
  resolution still walks `CAPY_LIBS`; if a library named `md`
  exists, it's used; if not, the CLI errors with a hint.
- Multiple positional args: first one ending in `.capy` is the
  library, everything else is script + script args.

### Trade-offs

- **Pro:** lets users register `.recipe` with their OS to
  double-click → run.
- **Pro:** scannable: `ls *.recipe` shows everything that uses
  the recipe DSL.

### Effort

**Small.** Add detection in the CLI's `run` subcommand.

### Recommendation

**SHIP — high value, low cost.**

## 5.2 Shebang scripts

### Design

A `.capy` script declares its library inline via a shebang:

```
#!/usr/bin/env capy --lib recipe
recipe "Lemon olive oil cake"
ingredient "Flour" "2 cups"
end
```

Make executable; run directly:

```sh
chmod +x cake.recipe
./cake.recipe
```

### Walkthrough

```sh
$ chmod +x cake.recipe
$ ./cake.recipe
# … HTML output to stdout …

$ ./cake.recipe --out cake.html
# … written to cake.html …
```

### Trade-offs

- **Pro:** the Unix shebang model — works the way scripts work.
- **Pro:** no extension restrictions — the file can be named
  anything.
- **Con:** Windows requires WSL or explicit `.capy` file
  associations.

### Effort

**Tiny.** Strip shebang in the source-reader; add `--lib` flag.

### Recommendation

**SHIP — alongside § 5.1.**

## 5.3 Library-name short form

### Design

Once libraries can be resolved by name, the CLI's first positional
arg can be one:

```sh
capy recipe cake.recipe
capy chart --impl d3 revenue.chart
capy py app.py
```

If the first arg matches a known subcommand (`run`, `check`,
`docs`, …), treat it as the subcommand. Otherwise treat it as a
library name and route to `run`.

### Trade-offs

- **Pro:** shortest possible invocation.
- **Con:** small ambiguity risk if a library name collides with
  a future subcommand. Mitigation: subcommand names live in a
  fixed (small) set.

### Effort

**Tiny.**

### Recommendation

**SHIP — alongside § 5.1.**

## 5.4 `compile` / `run` / `build` subcommand split

### Design

Rename verbs to match operations:

| Verb | Operation |
|------|-----------|
| `capy run LIB SCRIPT` | Run lib on script, print to stdout |
| `capy compile LIB SCRIPT -o OUT` | Run, write to OUT (or library's `output_file`) |
| `capy build LIB -o EXE` | Produce a standalone binary that runs LIB on any script (§ 6.2) |
| `capy check LIB` | Validate library without running |
| `capy docs LIB` | Render the auto-generated reference doc |

Old `capy run lib.capy script.capy` keeps working as an alias.

### Walkthrough

```sh
$ capy run py app.py
# … Python output to stdout …

$ capy compile py app.py -o app.py.out
# … wrote app.py.out (412 bytes) …

$ capy build py -o py-tool
# … built py-tool (5.8 MB) …

$ ./py-tool app.py > app.out      # py-tool is now self-contained
```

### Trade-offs

- **Pro:** verbs match what users say out loud.
- **Pro:** `run` ≠ `compile` makes the "writes a file" intent
  explicit.

### Effort

**Tiny** for the rename + aliases. `build` is § 6.2.

### Recommendation

**SHIP — early.** Verbs cost nothing; clarity pays forever.

---

# § 6 Distribution

## 6.1 WASM packaging

### Design

The wasm bundle (today, `docs/assets/playground/capy.wasm`) gets
published as proper packages:

```sh
# JavaScript / TypeScript.
npm install @capylang/capy
```

```js
import { loadCapy } from "@capylang/capy";
const capy = await loadCapy();
const out = await capy.run(libSrc, scriptSrc);
```

```sh
# Python via wasmtime.
pip install capy-wasm
```

```python
from capy import Capy
capy = Capy()
out = capy.run(lib_src, script_src)
```

```sh
# Browser CDN.
<script src="https://cdn.jsdelivr.net/npm/@capylang/capy@0.20.0/dist/capy.js"></script>
```

### User pain

Today: embed Capy in a static React app and you have to copy three
files (`capy.wasm`, `wasm_exec.js`, your own glue) and figure out
how to load them. The npm package solves all three.

### Walkthrough

```sh
$ npm create vite@latest my-app -- --template vanilla
$ cd my-app
$ npm install @capylang/capy

# index.js:
import { loadCapy } from "@capylang/capy";
const capy = await loadCapy();
document.body.textContent = await capy.run(libSrc, scriptSrc);

$ npm run dev    # works
```

### Trade-offs

- **Pro:** instant access from JS / Python / browser without any
  Capy CLI install.
- **Con:** WASM is ~6 MB today; with TinyGo could be ~1 MB but
  text/template support is fragile.

### Effort

**Medium.** Mostly packaging + CI to publish on each release tag.

### Recommendation

**SHIP — after § 1 / 2 stabilise.** Wasm is how Capy reaches
non-Go users.

## 6.2 `capy build` — single-binary compiler

### What it is

`capy build LIB -o EXE` produces a standalone executable
that has LIB baked in. Run it against any script:

```sh
$ capy build recipe -o recipe-tool
$ ./recipe-tool cake.recipe
# … HTML to stdout …
$ ./recipe-tool cake.recipe -o cake.html
```

The binary embeds:
- The Capy engine.
- The chosen library (and its default impl, or one passed via
  `--impl`).
- The selected `output_file` default.

### User pain

Today: share a Capy-based tool with non-developers and you must
explain "first install Go, then `go install capy`, then download
my library, then run …". `capy build` collapses to "here's a
binary, double-click it."

### Walkthrough

```sh
# Team-internal: build a tool that takes .recipe files.
$ capy build recipe --impl html -o recipe-renderer

# Ship recipe-renderer to non-developers.
# They drop a .recipe file on it (or run from terminal) and get HTML.

# Cross-compile for distribution.
$ capy build recipe --target linux/amd64 --target darwin/arm64 --target windows/amd64
✓ recipe-linux-amd64    (5.8 MB)
✓ recipe-darwin-arm64   (6.0 MB)
✓ recipe.exe            (6.1 MB)
```

### Multi-library compilation

```sh
$ capy build --lib recipe --lib chart -o my-tools
$ ./my-tools recipe cake.recipe > out.html
$ ./my-tools chart revenue.chart > out.md
```

The compiled binary dispatches on the first arg.

### Trade-offs

- **Pro:** "ship a tool to your team" becomes one step.
- **Pro:** removes Capy as a runtime dependency on the deploy
  host. Useful for CI, scripts, customer hand-offs.
- **Con:** binary size (~6 MB per target). Acceptable for CLI
  tooling.
- **Licensing question:** the current LICENSE forbids
  redistribution; needs a carve-out for compiled binaries that
  the library author intends to ship.

### Effort

**Medium.** Code generator emits a tiny Go `main.go` that embeds
the lib via `//go:embed`, then runs `go build`. Cross-compile via
GOOS/GOARCH.

### Recommendation

**SHIP — high value.** This is the "make Capy distributable"
feature.

---

# § 7 Editor / tooling

## 7.1 LSP server

### What it is

`capy-lsp` speaks Language Server Protocol. Editors get:

- **Autocomplete** for declared library functions when editing a
  script.
- **Hover docs** showing function descriptions and arg types.
- **Go-to-definition** from a function call in a script to the
  function declaration in the library.
- **Diagnostics**: unknown function, arg-type mismatch, unclosed
  block, etc. Inline, as you type.
- **Rename** for library functions (and their usages across
  every script in the workspace).

### User pain

Editing a `.recipe` script without an LSP means: type `serv` →
wait for run → see "unknown function" → re-read library → fix.
With LSP: type `serv` → see "serves" in the dropdown → tab.

### Walkthrough

```sh
# Install the LSP.
go install github.com/luowensheng/capy/cmd/capy-lsp@latest

# VS Code: install the Capy extension; it auto-launches capy-lsp.
# Vim: add to config.
# Emacs: add to LSP config.
```

```
# In a .recipe script:
recipe "Lemon cake"
    ing       <─ LSP suggests `ingredient`
            (capture name: string, capture qty: string)
```

### Library autodetection

The LSP needs to know which library a script uses. Resolution
order:

1. File extension (`.recipe` → library `recipe`).
2. Inline magic comment: `# capy: lib=recipe`.
3. Workspace setting in `.capy/lsp.toml`.
4. Project `capy.toml` declares a default library.

### Trade-offs

- **Pro:** library-authors get safe rename across thousands of
  consumer scripts.
- **Pro:** "spec-as-source" gets editor smarts that catch bugs
  before they reach `run`.

### Effort

**Medium.** Standard LSP boilerplate; analysis pieces are easy
because the library already exposes the data (`capy docs`
generates everything an LSP needs from a library file).

### Recommendation

**SHIP — high value once ecosystem grows.** Worth it once
people have ~5+ libraries.

## 7.2 `capy fmt`

### Design

Canonical formatter for `.capy` library files:

- 4-space indents.
- Argument alignment within functions.
- Sorted top-level declarations (extension → context → types →
  functions → file_template).
- Trailing-newline / no-trailing-space normalisation.

```sh
capy fmt lib.capy            # rewrite in place
capy fmt --check lib.capy    # exit 1 if not formatted
capy fmt --diff lib.capy     # print diff
```

### Walkthrough

```sh
$ cat lib.capy
extension html
function   greet
   arg literal "greet"
  arg capture name string
      write `Hello, ${name}!`
end

$ capy fmt lib.capy
$ cat lib.capy
extension html

function greet
    arg literal "greet"
    arg capture name string
    write `Hello, ${name}!`
end
```

### Trade-offs

- **Pro:** removes a class of code-review nitpicks.
- **Pro:** plays nice with the LSP — format-on-save just works.

### Effort

**Medium.** Parser already exists; the formatter walks the parsed
RawLibrary back to text using consistent rules.

### Recommendation

**SHIP — after LSP** (or together; they share the parser
infrastructure).

## 7.3 Watch mode

### Design

`capy watch LIB SCRIPT` re-runs whenever any input changes. Prints
the diff between the previous and current output so authors see
the impact of each edit.

```sh
$ capy watch recipe cake.recipe
👀 watching recipe.capy, cake.recipe
=== 12:01:03 ===
Hello, world!

# edit recipe.capy …

=== 12:01:14 === diff:
-Hello, world!
+Welcome!
```

Flags:
- `--browser`: open the result in a browser tab and live-reload
  on each save (great for HTML / Markdown DSLs).
- `--out FILE`: write to file in addition to printing.

### Walkthrough

```sh
$ capy watch chart revenue.chart --browser
# browser opens with current chart.
# edit revenue.chart, save → chart updates in the browser
# edit chart.capy, save → chart updates with new styles
```

### Trade-offs

- **Pro:** the tight inner loop most DSL authors want.
- **Pro:** discoverable — `capy watch` is the obvious command for
  "iterate fast."

### Effort

**Small.** `fsnotify` + the existing CLI.

### Recommendation

**SHIP — early polish.** Big quality-of-life win.

## 7.4 Local playground (`capy play`)

### Design

`capy play [LIB]` spins up an HTTP server on `localhost:8080`
serving the same playground UI used at
`luowensheng.github.io/capy/playground/`. Difference:

- All libraries from `CAPY_LIBS` are available (not just curated
  samples).
- Private / unpublished libraries work.
- Runs offline.

```sh
$ capy play recipe
🚀 serving on http://localhost:8080
   library: recipe (~/.capy/libs/recipe/)
   editor:  http://localhost:8080
```

### Walkthrough

```sh
$ capy play
🚀 serving on http://localhost:8080
   all libraries from CAPY_LIBS

# Browser opens; dropdown lists every installed library.
# Edit scripts in the browser; instant feedback.
```

### Trade-offs

- **Pro:** authoring experience that doesn't need network.
- **Pro:** private libraries can be played without uploading.

### Effort

**Medium.** Static assets ship via `//go:embed`; the CLI just
spins an HTTP server.

### Recommendation

**SHIP — after the hosted playground stabilises.**

## 7.5 Sourcemaps

### What it is

Optional metadata mapping generated output lines back to source
script lines.

```sh
$ capy compile py app.py -o app.py.out --sourcemap app.map.json
```

`app.map.json` records that line 12 of `app.py.out` came from
line 3 of `app.py`. Tools (or future Capy CLI flags) can then
translate Python runtime errors back to the original DSL.

### Trade-offs

- **Pro:** when target runtime errors point at line numbers
  nobody recognises, the sourcemap rescues them.
- **Con:** non-trivial to maintain through helper calls and
  indentation. Mitigation: best-effort, document the limits.

### Effort

**Medium-Large.** Templating engine needs to thread position
metadata through every `write` call.

### Recommendation

**DEFER — until users hit the pain.** Worth doing then, not
now.

---

# § 8 SDKs

## 8.1 Go SDK (current — already shipped)

`import "github.com/luowensheng/capy"` works today.

## 8.2 Python SDK

### Design

```python
from capy import Library
lib = Library.from_path("~/.capy/libs/recipe")
out = lib.run(open("cake.recipe").read())
```

Backed by the WASM bundle (§ 6.1) via `wasmtime-py`.

### Walkthrough

```sh
$ pip install capy
```

```python
from capy import Library

# Embed a library inline.
lib = Library.from_string("""
extension html
function greet
    arg capture name any
    write `Hello, ${name}!
`
end
""")

print(lib.run('greet "world"'))
# Hello, "world"!
```

### Effort

**Medium per language.** Once § 6.1 lands, each SDK is a thin
wrapper.

### Recommendation

**SHIP Python first** (largest non-Go developer base).
JavaScript and Rust follow on demand.

## 8.3 JavaScript SDK

Same as 8.2, packaged for Node + browser. Already half-built
(the playground's loader).

## 8.4 Rust SDK

`capy = "0.1"` via crates.io. Uses `wasmtime` crate. Useful for
Rust-based CLIs that want Capy as a config layer.

---

# § 9 Big ideas

## 9.1 `capy transform`

### What it is

Take a file, parse it via one library's interpreter, transform via
inner-DSL operations, emit via another library:

```sh
capy transform old.travis.yml --from travis --to gh-actions > .github/workflows/ci.yml
```

The two libraries share an in-memory representation; the transform
is a function from one context to another.

### User pain

Migrating from one tool to another (Travis → GH Actions, Express
→ Fastify, OpenAPI 2 → OpenAPI 3) means hand-editing files. With
`capy transform`, write one library per side and the transform is
declarative.

### Effort

**Large.** Probably v0.25+.

### Recommendation

**RESEARCH** — explore as a v0.x sample; commit to engine
support only once a real use case proves it.

## 9.2 Generated-code provenance

### Design

Every generated file gets a leading comment:

```python
# Generated by Capy 0.20.0 on 2026-05-25T12:00:00Z
# library: recipe@1.4.0 (sha256:9c2b…)
# source:  /Users/x/cake.recipe (sha256:5e7a…)
# Do not edit; regenerate with: capy compile recipe cake.recipe
```

CI checks the header against current state and fails if drifted.

### Effort

**Tiny.** A `--provenance` flag adds the header.

### Recommendation

**SHIP — small, high value for teams with generated code in
git.**

## 9.3 Capy-on-Capy

### What it is

A Capy library whose output is *another* `.capy` library.
Bootstrap new DSLs by describing them at a higher level:

```
schema User
    field email string required
    field name string
end

generate-crud User
```

Output: a `.capy` library with `create_user`, `update_user`,
`delete_user`, `list_users` functions targeting your preferred
backend.

### Effort

**Tiny as a sample, conceptually fascinating.** No engine changes
required — it's just a Capy library.

### Recommendation

**SHIP as a sample** to demonstrate the meta-pattern.

## 9.4 Bundle / vendor

### Design

`capy bundle LIB` walks every dependency (library imports and
source-side `@import`s) and inlines them into one file. The
output is hermetic: ship one `.capy`, no other files needed.

### Effort

**Small.**

### Recommendation

**SHIP — useful for distribution.**

---

# § Decisions — summary table

Rough effort × value ranking. **Value** is "users who notice on
day one." **Effort** is "developer-weeks at one full-time."

| Feature | Effort | Value | Verdict | Priority |
|---|---|---|---|---|
| § 1.1 `CAPY_LIBS` search path | S | ⭐⭐⭐⭐⭐ | **SHIP** | 1 |
| § 1.2 Library manifest | S | ⭐⭐⭐⭐⭐ | **SHIP** | 1 |
| § 1.3 Library directory site | S | ⭐⭐⭐⭐ | **SHIP** | 1 |
| § 2.1 Multiple impls | M | ⭐⭐⭐⭐⭐ | **SHIP** | 2 |
| § 2.2 Impl selection | S | ⭐⭐⭐⭐ | **SHIP** (with 2.1) | 2 |
| § 3.1 Versioning | M | ⭐⭐⭐⭐ | **SHIP** | 3 |
| § 3.2 Lockfile | S | ⭐⭐⭐⭐ | **SHIP** (with 3.1) | 3 |
| § 4.1 `lib new` scaffolding | S | ⭐⭐⭐⭐ | **SHIP** | 2 |
| § 4.2 Local path libs | XS | ⭐⭐⭐ | **SHIP** | 2 |
| § 4.3 Git distribution | S | ⭐⭐⭐⭐ | **SHIP** | 3 |
| § 4.4 Registry | L | ⭐⭐ | **DEFER** | — |
| § 5.1 File-ext convention | S | ⭐⭐⭐⭐ | **SHIP** | 4 |
| § 5.2 Shebang scripts | XS | ⭐⭐⭐ | **SHIP** | 4 |
| § 5.3 Short form | XS | ⭐⭐⭐ | **SHIP** | 4 |
| § 5.4 compile/run/build verbs | XS | ⭐⭐⭐ | **SHIP** | 4 |
| § 6.1 WASM packaging | M | ⭐⭐⭐⭐ | **SHIP** | 5 |
| § 6.2 Single-binary compiler | M | ⭐⭐⭐⭐⭐ | **SHIP** | 5 |
| § 7.1 LSP | M | ⭐⭐⭐⭐ | **SHIP** | 6 |
| § 7.2 `capy fmt` | M | ⭐⭐⭐ | **SHIP** | 6 |
| § 7.3 Watch mode | S | ⭐⭐⭐⭐ | **SHIP** | 5 |
| § 7.4 `capy play` | M | ⭐⭐⭐ | **SHIP** | 7 |
| § 7.5 Sourcemaps | M | ⭐⭐ | **DEFER** | — |
| § 8.2 Python SDK | M | ⭐⭐⭐⭐ | **SHIP** (after 6.1) | 6 |
| § 8.3 JS SDK | S | ⭐⭐⭐⭐ | **SHIP** (after 6.1) | 6 |
| § 8.4 Rust SDK | M | ⭐⭐⭐ | **SHIP** (after 6.1) | 8 |
| § 9.1 `capy transform` | L | ⭐⭐ | **RESEARCH** | — |
| § 9.2 Provenance metadata | XS | ⭐⭐⭐⭐ | **SHIP** | 5 |
| § 9.3 Capy-on-Capy | XS | ⭐⭐ | **SHIP as sample** | 4 |
| § 9.4 Bundle/vendor | S | ⭐⭐⭐ | **SHIP** | 7 |

## Recommended release plan

**v0.19** — Foundations (Priority 1–2)
- `CAPY_LIBS` + manifest + library directory site (§ 1).
- `capy lib new` + local path libs (§ 4.1, 4.2).
- Multiple impls + selection (§ 2).
- Capy-on-Capy as a sample (§ 9.3).

**v0.20** — Versioning + distribution (Priority 3)
- Library/impl versions + lockfile (§ 3).
- Git-based `capy lib add` (§ 4.3).
- Bundle/vendor (§ 9.4).

**v0.21** — Ergonomics (Priority 4)
- File-extension convention (§ 5.1).
- Shebang + short form (§ 5.2, 5.3).
- compile/run/build subcommand split (§ 5.4).

**v0.22** — Distribution & polish (Priority 5)
- Single-binary compiler (§ 6.2).
- WASM packaging (§ 6.1).
- Watch mode (§ 7.3).
- Provenance metadata (§ 9.2).

**v0.23** — Tooling (Priority 6)
- LSP (§ 7.1).
- `capy fmt` (§ 7.2).
- Python + JS SDKs (§ 8.2, 8.3).

**v0.24** — Inner-loop polish (Priority 7)
- Local `capy play` (§ 7.4).

**v0.25+** — Out of the way bets (Priority 8 / research)
- Rust SDK (§ 8.4).
- Sourcemaps (§ 7.5).
- `capy transform` research (§ 9.1).

## Out of scope (intentionally)

- A "real" general-purpose programming language. Capy is a
  transpiler engine and stays one. No new control structures
  inside templates beyond `for` / `if`. No expression DSL beyond
  helper calls.
- Centralised cloud hosting / SaaS. Distribute via git URLs.
- A registry until § 1–4 are stable and demand is real.
