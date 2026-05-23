# Contributing to Capy

Thanks for taking the time to contribute. This document covers what you need to know to do so productively.

## Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating you are expected to uphold it. Report unacceptable behavior by following the instructions there.

## Project status

Capy is **pre-1.0**. The library YAML schema and engine internals may change between minor versions. Each breaking change is called out in `CHANGELOG.md`.

## How to file a good issue

- **Bug**: minimal reproduction (a tiny `lib.yaml` + `script.capy` that exhibits the problem), expected output, actual output, your OS + Go version + Capy version.
- **Feature**: what you're trying to do, why the current model doesn't fit, and the smallest YAML-level change that would unblock you.
- **Library request**: a description of the source language you want to transpile and what the target looks like.

There are issue templates for each — please use them.

## Development setup

```sh
git clone https://github.com/luowensheng/capy
cd capy
go test ./...               # run all tests
go build -o capy ./cmd/capy # build the CLI
./capy run samples/transpile-py/lib.yaml samples/transpile-py/script.capy
```

Go 1.22 or later is required.

## Running tests

```sh
go test ./...                        # unit + golden tests
go test -run TestGolden ./...        # just the sample integration tests
go test -cover ./...                 # with coverage
```

The golden tests compare each `samples/*/script.capy` output to `samples/*/expected.txt`. If you intentionally change behavior, regenerate the expected files:

```sh
go test ./... -update                # rewrite expected.txt with current output
```

## Code style

- Run `gofmt` (or `goimports`) on save.
- `golangci-lint run` should be clean before submitting.
- The codebase follows the **VHCO** layout — see [docs/architecture.md](docs/architecture.md). The six top-level Go folders (`domain`, `features`, `usecases`, `io`, `infra`, `orchestrator`) are not negotiable.
- New `Impl` types live in `orchestrator/`. Other modules declare shapes, not wiring.
- Tests live next to the code (`foo.go` ↔ `foo_test.go`).

## Pull-request flow

1. Open an issue first if the change is non-trivial; we'll discuss the approach.
2. Fork → branch → commit.
3. Ensure `go test ./...` and `golangci-lint run` are clean.
4. Open a PR using the template. Reference the issue.
5. Expect review feedback; small follow-up commits are encouraged.

We squash-merge on the green path. Keep the PR title imperative ("add foo", "fix bar"); CHANGELOG line is added in the merge.

## Adding a new sample

When adding a sample under `samples/<name>/`:

- `lib.yaml` — the library
- `script.capy` — the input
- `expected.txt` — the verified output (generated via `go test ./... -update`)
- `README.md` — what it teaches, why it's interesting

A new sample should demonstrate a feature the existing samples don't cover. Don't add a "kitchen sink" sample; small focused libs read better.

## Adding a new inner-DSL primitive

The inner DSL (used in `run:` snippets) is the engine's vocabulary. Adding a primitive is a meaningful change.

1. Add the statement form (if applicable) to `domain/ast.go`.
2. Parse it in `orchestrator/features/inner_parser.go`.
3. Execute it in `orchestrator/features/inner_evaluator.go`.
4. Document it in `docs/inner-dsl.md`.
5. Add unit tests and at least one sample that uses it.

## Adding a new template helper

In `infra/template_engine.go`, add an entry to the `funcs` map. Document in `docs/templates.md`. Add a unit test.

## Adding a new built-in type kind

Built-in kinds (`any`, `string`, `int`, `float`, `bool`, `ident`, `raw`) are deliberately small. If you think you need a new one, open an issue first — usually a library-defined type covers the case.

## License

By contributing you agree your contributions will be licensed under the [MIT License](LICENSE) that covers the project.
