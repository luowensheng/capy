# Architecture

This doc describes how the engine is laid out internally. For library
authors, this isn't required reading — it's here for anyone curious
about the internals.

## Top-level layout

```
domain/         entities (Token, Library, FuncDef, FuncCall, CapyError, …)
features/       capability struct declarations (Lexer, Parser, Evaluator, …)
usecases/       contracts the orchestrator wires up (RunScript)
io/cli/         CLI view + view model + the use-case protocol it needs
infra/          adapters to external systems (file IO, YAML, text/template)
orchestrator/   the only place where things get assembled
```

The six folders are not negotiable. If you feel the urge to add `utils/` or
`shared/`, re-read the responsibilities below.

## Data flow

```
script.capy ──► Lexer ──► tokens ──► Outer Parser ──► AST (FuncCall) ──┐
                                          ▲                              │
                                          │                              ▼
                                          │                       Outer Evaluator
                                          │                              │
                                  Library (FuncDef) ◄── Library Loader   │
                                          │                              │
                                          ▼                              ▼
                                    type defs                Inner Evaluator
                                                          (mutates context via run:)
                                                                         │
                                                                         ▼
                                                               File template render
                                                                         │
                                                                         ▼
                                                                       output
```

## Module responsibilities

### `domain/`

Pure data types with no behavior beyond simple constructors. Token, AST,
Library shape, errors. Imports nothing internal.

### `features/`

Each external capability is declared as a struct of function fields:

```go
type Lexer struct { Tokenize func(source string) ([]domain.Token, error) }
type Parser struct { Parse func([]domain.Token, domain.Library) (domain.Block, error) }
type Evaluator struct { Run func(domain.Block, domain.Library) (string, error) }
// ...
```

This is a deliberate VHCO move: features declare **shapes**, not
implementations. The orchestrator builds the function values.

### `usecases/`

Higher-level user-visible operations. Currently just `RunScript`. Each
declares the function-type aliases for the capabilities it needs from
features. No implementations; only contracts.

### `io/cli/`

A dumb view (renders state enums) + a view-model (handles flow control) +
the use-case protocol the view-model needs. No business logic in the view.

### `infra/`

External-system adapters. `FileReader`, `YamlParser`, `TemplateEngine`. No
knowledge of the domain types — the orchestrator maps between them.

### `orchestrator/`

The **only** module that imports concrete types from other modules. Every
`make_*` factory lives here:

```
orchestrator/features/
  make_lexer.go
  make_parser.go
  make_evaluator.go
  make_library_loader.go
  inner_parser.go
  inner_evaluator.go
  value_parser.go
  expr_to_text.go
orchestrator/usecases/make_run_script.go
orchestrator/views/make_cli_view.go
orchestrator/app.go
orchestrator/run.go             # programmatic entry point
```

## The two grammars

Capy has two grammars in the engine:

1. **Outer (zero default)** — user-facing source. Matched against
   library-defined function shapes. No hard-coded keywords.

2. **Inner (small fixed grammar)** — the language inside each library's
   `run:` field. Has a fixed parser/evaluator pair (`inner_parser.go` /
   `inner_evaluator.go`).

Both grammars share the same lexer (it's purely lexical and library-
agnostic) and the same value-expression parser (`value_parser.go`).

## Captures: dual face

Each capture is parsed once but exposed two ways:

- To templates → as the source text (so `if x > 0` emits literal `if x > 0:`
  in Python).
- To the inner DSL → as the evaluated value (so `append context.x value`
  stores the Go value).

This is implemented in `make_evaluator.go` (`renderTemplate` uses `.Text`)
and `inner_evaluator.go` (`resolvePath` evaluates `.Expr`).

## Error positions

`domain.CapyError { Line, Col, Msg }` is the structured error type. The
outer parser populates it from token positions. The CLI calls
`domain.FormatWithSource` to render the caret block.

## Testing

- Golden tests (`cmd/capy/golden_test.go`) walk `samples/*/` and compare
  each script's actual output to a stored `*.expected.txt` /
  `*.expected-error.txt`. Regenerate with `go test ./... -update`.
- Unit tests live next to source files (`foo.go` ↔ `foo_test.go`).

## Adding a feature

A typical change touches:

1. `domain/` — the data shape (a new field, a new struct).
2. `infra/yaml_parser.go` — the YAML DTO (if user-visible).
3. `orchestrator/features/make_library_loader.go` — the mapping.
4. The relevant feature implementation (`make_parser.go`, `make_evaluator.go`,
   or `inner_evaluator.go`).
5. A new sample under `samples/` + golden.
6. Docs under `docs/` describing the new field.

`features/` and `usecases/` only change when you're adding a whole new
capability (rare).
