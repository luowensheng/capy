package domain

// CommandDef is one library-declared command — a custom verb the
// CLI exposes as `capy <lib> <name> [args]`. The command body is
// an extended inner-DSL program (sequence of InnerStmt) with
// shell-like primitives available: exec, write_file, mktemp,
// mktemp_dir, cd, print, let, etc.
//
// The body runs against a per-invocation execution context that
// holds:
//   - `args`  — positional CLI args after the command name
//   - `flags` — flag values declared by the command (future)
//   - locals introduced via `let X = …`
//
// A library can override built-in commands (run, compile, check,
// docs) by declaring its own command with the same name. If a
// library declares no commands, the built-in default for `run`
// is used.
type CommandDef struct {
	Name        string
	Description string
	Body        InnerBlock
	// Body raw text — kept around so we can re-tokenise / re-parse
	// when needed by tooling. Empty after the first compile.
	BodyRaw string
}
