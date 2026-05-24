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

	// Positional arguments declared via `arg "name" required "desc"`.
	// Used to generate --help and to validate invocations.
	Args []CommandArg

	// Named flags declared via `flag "--name" "desc" default "v"`.
	Flags []CommandFlag
}

// CommandArg is a positional argument declaration.
type CommandArg struct {
	Name        string
	Required    bool
	Description string
}

// CommandFlag is a flag declaration. Bool flags are presence-only;
// string flags accept `--name VALUE` or `--name=VALUE`.
type CommandFlag struct {
	Name        string // includes the leading dashes, e.g. "--port"
	Description string
	Default     string
	IsBool      bool
}
