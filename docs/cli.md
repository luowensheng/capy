# CLI Reference

`capy` is the command-line front-end. All subcommands also accept
`-h`/`--help` for inline help.

## `capy run <library.yaml> <script.capy>`

Transpile a script against a library. Output goes to stdout unless the
library sets `output_file:` or you pass `--out`.

```sh
capy run samples/transpile-py/lib.yaml samples/transpile-py/script.capy
capy run lib.yaml app.capy --out=app.py
```

Flags:

| Flag           | Effect                                                 |
|----------------|--------------------------------------------------------|
| `--out <path>` | Write output to this file (overrides library setting). |
| `--no-color`   | Disable ANSI escape codes (reserved).                  |
| `--debug`      | Verbose engine tracing (reserved).                     |
| `-lib <path>`  | (legacy) library path; equivalent to positional arg 1. |

Errors are printed with line + column + caret:

```
error: no library function matches token "x"
  1 │ x = 1
    │ ^
```

## `capy check <library.yaml>`

Parse and validate a library without running any source. Reports loaded
functions and types if valid, or a structured error otherwise. Exit code
0 = valid, 1 = invalid.

```sh
capy check lib.yaml
# ok — 5 function(s), 2 type(s)
#   function greet
#   function assign
#   ...
#   type     Email
#   type     Status
```

Use this in CI alongside `go test ./...` to catch broken libraries early.

## `capy init [<dir>]`

Scaffold a starter project (`lib.yaml`, `script.capy`, `README.md`) in the
target directory (default `.`). Refuses to overwrite existing files.

```sh
capy init my-dsl
# created my-dsl/lib.yaml
# created my-dsl/script.capy
# created my-dsl/README.md
```

## `capy version`

Print the version baked in at build time. `dev` if you built from source
without `-ldflags "-X main.version=..."`.

## `capy help [<command>]`

Top-level help when called without arguments; per-command detailed help
when called with one.

## Exit codes

| Code | Meaning                                                    |
|------|------------------------------------------------------------|
| 0    | Success.                                                   |
| 1    | An error occurred (syntax, validation, I/O). See stderr.   |
| 2    | Reserved for usage errors (not currently distinguished).   |

## Embedding Capy

If you want to call Capy from Go code, import
`capylang/orchestrator`:

```go
import "capylang/orchestrator"

out, err := orchestrator.Run("lib.yaml", "script.capy")
// or, when contents are already in memory:
out, err := orchestrator.RunStrings(libYAML, "lib.yaml", scriptSource)
```

Errors are `*domain.CapyError` values; use `domain.FormatWithSource(err, src)`
to render them with a caret.
