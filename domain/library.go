package domain

// Library is the result of loading a YAML library file. It is the entire
// grammar plus accumulation rules for one source-language → target-output
// transpilation.
//
// Surface-syntax conventions (block start/end, statement terminator, arg
// separator) are fixed for now — INDENT/DEDENT for blocks, NEWLINE for
// statements, whitespace for args. A future version will make them
// configurable per-library.
type Library struct {
	Extension    string
	OutputFile   string
	// Description is a free-form summary of what the library is for —
	// shown at the top of `capy docs <library>` output.
	Description  string
	Functions    map[string]*FuncDef
	Types        map[string]TypeDef
	Context      map[string]any // initial context values (lists, maps, scalars)
	FileTemplate string

	// Files is a multi-output declaration: each entry is a relative path
	// (may include slashes for subdirs) → a Go template rendered with
	// `.context` and `.body` exactly like FileTemplate.
	//
	// When non-empty AND the CLI is invoked with an --out-dir, the engine
	// writes every entry to disk and ignores FileTemplate/OutputFile.
	// When empty, behavior is unchanged: FileTemplate goes to stdout (or
	// OutputFile if set).
	//
	// Each path key is itself a Go template rendered against the same
	// .context + .body data, so libraries can name outputs dynamically:
	//   file "{{ .context.name | pascalCase }}.tsx":
	//       …
	Files map[string]string

	// Commands declared in the library's manifest. Each maps a verb
	// name (e.g. "build", "serve", "new") to a CommandDef. The CLI
	// dispatches `capy <lib> <name>` to the matching command. The
	// inner-DSL body of a command can shell out, write files, etc.
	// — see CommandDef.
	Commands map[string]*CommandDef

	// Manifest metadata for tooling (`capy lib list`, library
	// directory site). All fields are optional; absence means the
	// library was loaded as a bare .capy file with no manifest.
	LibName    string // canonical name declared in the manifest
	LibVersion string // semver string

	// Preprocess is the list of source-level inclusion directives the
	// library OPTS INTO. The engine ships zero default preprocessing —
	// if Preprocess is empty, lines like `@import "x.capy"` are NOT
	// recognised and flow into the lexer as ordinary tokens (where
	// they'll likely fail to match a function). A library that wants
	// text-level file inclusion declares the directives explicitly:
	//
	//   preprocess
	//       include "@import"
	//       include "@include"
	//   end
	//
	// Each declared directive is processed identically (read the quoted
	// path, splice the file's bytes into the source, recurse). The
	// declaration is just an opt-in switch on a per-name basis. This
	// keeps Capy's "zero predefined grammar" promise intact: even
	// directives that look universal come from the library, not the
	// engine.
	Preprocess []string
}

// FuncDef is a single library-defined source-language construct.
//
//	args:     declarative match shape (literals + typed captures)
//	template: text fragment contributed to the output body when matched
//	run:      context-mutation snippet (does NOT execute user code)
//	block:    when set, this function opens a body block closed by Block.Closer
type FuncDef struct {
	Name        string
	Description string // free-form, surfaced by `capy docs`
	Args        []ArgEntry
	Elements    []PatternElement // compiled from Args
	Template    string
	Block       *BlockSpec
	Run         string
	RunAST      *InnerBlock
	Priority    int
}

// ArgEntry is a single args-list entry with an explicit Kind discriminator.
// Kind = "literal" → only Value is meaningful.
// Kind = "capture" → only Name and Type are meaningful.
type ArgEntry struct {
	Kind        string // "literal" | "capture"
	Value       string
	Name        string
	Type        string
	Description string // optional, only meaningful when Kind=capture
}

// BlockSpec marks a function as a block opener. There are two modes:
//
//  1. Named-closer mode  (default):
//     block: { closer: <function-name> }
//     The body is delimited by INDENT/DEDENT; the closer function runs after.
//
//  2. Delimiter mode:
//     block: { open: "{", close: "}" }
//     The body is delimited by exact tokens. No closer function involved.
//     Useful for `for x in 40 { ... }` style.
type BlockSpec struct {
	Closer string
	Open   string
	Close  string
}

// PatternElement is one compiled token in the function's match shape.
type PatternElement struct {
	IsCapture bool
	Literal   string
	Name      string
	CapType   string
}

// TypeDef is a library-defined argument type. Three optional fields applied
// in order at validation time: Base → Pattern → Options.
type TypeDef struct {
	Name        string
	Description string   // optional, surfaced by `capy docs`
	Base        string   // any | string | int | float | bool
	Pattern     string   // optional regex on the value's string form
	Options     []string // optional enum membership
}
