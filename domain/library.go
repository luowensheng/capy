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
	Functions    map[string]*FuncDef
	Types        map[string]TypeDef
	Context      map[string]any // initial context values (lists, maps, scalars)
	FileTemplate string
}

// FuncDef is a single library-defined source-language construct.
//
//	args:     declarative match shape (literals + typed captures)
//	template: text fragment contributed to the output body when matched
//	run:      context-mutation snippet (does NOT execute user code)
//	block:    when set, this function opens a body block closed by Block.Closer
type FuncDef struct {
	Name     string
	Args     []ArgEntry
	Elements []PatternElement // compiled from Args
	Template string
	Block    *BlockSpec
	Run      string
	RunAST   *InnerBlock
	Priority int
}

// ArgEntry is a single args-list entry with an explicit Kind discriminator.
// Kind = "literal" → only Value is meaningful.
// Kind = "capture" → only Name and Type are meaningful.
type ArgEntry struct {
	Kind  string // "literal" | "capture"
	Value string
	Name  string
	Type  string
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
	Name    string
	Base    string   // any | string | int | float | bool
	Pattern string   // optional regex on the value's string form
	Options []string // optional enum membership
}
