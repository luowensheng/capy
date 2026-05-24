# Roadmap

What's planned for Capy after v0.1.0. Not commitments — this is direction.
Open an issue if any of these matter to you so we know to prioritise.

## v0.2 — quality of life

- **`else` arm on inner `if`**. Currently single-arm; the workaround is
  two `if` blocks with `not`.
- **Argument `default:` values** on captures. Mirror the original README
  spec.
- **`capy fmt`** — canonical formatter for source files. Probably an
  opinionated formatter (no config knobs).
- **`capy watch`** — re-run on file change for fast iteration.
- **Coverage badge** on the README.
- **`capy lint <lib.capy>`** — non-load-time checks (unused functions, dead
  captures, etc.).

## v0.3 — schema power

- **`validate` types written in inner Capy**. The most expressive form
  from the original README. Requires a small bootstrap loop.
- **Library composition / `import`**. Splitting a big library across
  files; merging types/functions/context defaults.
- **Custom inner-DSL primitives** registered from the orchestrator
  (`MakeEvaluator(WithPrimitive(...))`) — for embedders who need a
  domain-specific op.

## v0.4 — surface flexibility

- **Configurable syntax** (deferred from earlier): per-library statement
  terminator, argument separator, block delimiters. Will be opt-in;
  defaults stay as today.
- **Customizable comment syntax** (currently only `#`).
- **Trailing-comma tolerance everywhere**.

## v0.5 — multi-output

- **`outputs:`** — a library produces multiple files (e.g. one per class,
  one per route). Each accumulated context slot can target a different
  file. Requires a path-template selector.
- **`output_file_template`** — interpolate the output filename from
  context.

## v0.x — ecosystem

- **LSP server** for the source language defined by a library (using the
  library's schema for completion + diagnostics).
- **Tree-sitter grammar** generation from a library.
- **WASM build** to run Capy in the browser.
- **`awesome-capy`** — curated list of libraries (sql, sql-builder,
  graphql, dockerfile, terraform, etc.).

## Self-hosting milestone

- **Capy's inner DSL written in Capy.** The current parser/evaluator pair
  for function bodies is in Go; rewriting them as a Capy library would be a
  satisfying self-host moment and force the language to be expressive
  enough.

---

If you want to drive any of these, open an issue with a sketch of the
schema change and a small example library. Contributions welcome.
