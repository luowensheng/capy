# CLAUDE.md — working notes for this repo

Capy is a **target-agnostic transpiler**: it ships *zero* source-language
grammar. A `.capy` library defines the source language; one source can
render to many text targets. Keep engine changes additive — no existing
library should break.

## Built-in functions list — KEEP IT IN SYNC

The authoritative list of Capy's built-in **template helper functions**
(the ones callable inside `${ … }` and inner-DSL expressions) lives in:

- **Source of truth:** the `funcs` map in `infra/helpers.go`.
- **Cookbook (user-facing):** `docs/function-cookbook.md` — documents and
  shows an example of every helper. Linked in `mkdocs.yml` nav under
  Reference → "Built-in function cookbook", so it publishes to gh-pages.
- **Verified examples:** `samples/builtin-functions/` — a runnable sample
  whose golden output (`script.expected.txt`) is checked in CI.

**RULE (do this every time):** whenever you add, rename, or remove a
built-in helper in `infra/helpers.go`, you MUST in the same change:

1. Update `docs/function-cookbook.md` — the quick-reference table **and** a
   per-function section with a worked example.
2. Update `samples/builtin-functions/` if the helper is demonstrable there,
   and regenerate the golden:
   `go run ./cmd/capy run samples/builtin-functions/lib.capy samples/builtin-functions/script.capy > samples/builtin-functions/script.expected.txt`
3. Mention the change in `docs/whats-new.md`.

The same rule applies to library-authoring directives (`docs/features.md`,
`docs/CAPY_FOR_LLMS.md`) when you add/rename one.

## Repo conventions

- **gh-pages** deploys from `docs/` on push to `main`; verify locally with
  `mkdocs build --strict` (must exit 0).
- **Playground** samples come from the curated list in
  `cmd/playground-bundle/main.go`. It writes JSON to STDOUT — regenerate
  with `go run ./cmd/playground-bundle > docs/assets/playground/samples.json`
  (that file is gitignored; CI rebuilds it).
- **Golden tests** (`cmd/capy/golden_test.go`) discover `samples/*/` and
  pair each `*.capy` with `<base>.expected.txt` (success) or
  `<base>.expected-error.txt` (error, format `LINE:COL: message`).
  `go test ./cmd/capy -run Golden -count=1`. The `-update` flag only
  refreshes EXISTING golden files — pre-create an empty placeholder for a
  brand-new sample before running `-update`.
- **Before committing:** `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`, and `mkdocs build --strict` all green.
- **Git:** commit only when asked; stage files by name (never `git add -A`);
  never commit `.ignore/` files; never force-push `main`; never `--no-verify`.
