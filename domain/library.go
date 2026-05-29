package domain

// Library is the result of loading a `.capy` library file. It is
// the entire grammar plus accumulation rules for one
// source-language → target-output transpilation.
//
// Surface-syntax conventions (block start/end, statement
// terminator, arg separator) are fixed for now —
// INDENT/DEDENT for blocks, NEWLINE for statements, whitespace
// for args. A future version will make them configurable
// per-library.
type Library struct {
	Extension  string
	OutputFile string
	// Description is a free-form summary of what the library is for —
	// shown at the top of `capy docs <library>` output.
	Description string
	Functions   map[string]*FuncDef
	Types       map[string]TypeDef
	Context     map[string]any // initial context values (lists, maps, scalars)

	// FileTemplateAST is the parsed write-style body of the library's
	// `file_template ... end` block. The renderer walks this directly.
	// nil when the library has no file_template (renderer uses the
	// top-level body verbatim).
	FileTemplateAST *InnerBlock

	// FilesAST is a multi-file output declaration: each entry is a
	// relative path (may contain write-style `${...}` interpolations
	// for dynamic naming, resolved at render time) → the parsed
	// write-style AST of that file's body. The renderer walks the
	// ASTs directly.
	//
	// When non-empty AND the CLI is invoked with --out-dir, the
	// engine writes every entry to disk and ignores OutputFile. When
	// empty, behaviour falls back to the FileTemplateAST → stdout
	// (or OutputFile) path.
	FilesAST map[string]*InnerBlock

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

	// Impls catalogues every implementation the library declared
	// via `impl "NAME" "FILE" ... end` blocks in its manifest.
	// When non-empty, the manifest file itself carries no
	// functions — the real ones live in the selected impl's
	// file. The CLI picks one per invocation; the loader records
	// the choice in SelectedImpl.
	Impls        map[string]*ImplDef
	DefaultImpl  string // name from the manifest's `default` directive
	SelectedImpl string // populated by the CLI after selection

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

	// Comments is the list of line-comment markers the library opts
	// into for its USER SCRIPTS. The engine ships zero default
	// comment syntax — if Comments is empty, a `#` or `//` at the
	// start of a script line is NOT a comment; it flows into the
	// lexer as ordinary characters (and will usually error).
	//
	// Declare opt-ins in the manifest:
	//
	//   comments
	//       line "#"
	//       line "//"
	//   end
	//
	// Each entry is matched at the start of a line (after any
	// indent) and at any point on a line; everything from the
	// marker to end-of-line is discarded.
	//
	// This declaration ONLY affects user-script lexing. The
	// manifest itself, including inner-DSL `run:` bodies and
	// command bodies, always uses `#` (Capy's own config syntax).
	Comments []string
}

// FuncDef is a single library-defined source-language construct.
//
//	Args:        declarative match shape (literals + typed captures)
//	TemplateAST: write-style body — renders to output text on match
//	RunAST:      state-mutation projection of the body — runs after render
//	Block:       when set, this function opens a body block closed by Block.Closer
type FuncDef struct {
	Name        string
	Description string // free-form, surfaced by `capy docs`
	Args        []ArgEntry
	Elements    []PatternElement // compiled from Args

	// TemplateAST is the parsed write-style body. The renderer walks
	// this directly — state-mutation statements inside it are treated
	// as render-side no-ops (they're handled by RunAST).
	TemplateAST *InnerBlock

	// RunAST is the state-mutation projection of the same body —
	// `set` / `append` / `prepend` / `merge` / `delete` / `call` /
	// `error` statements, with control flow that contains them
	// preserved. Runs AFTER the render pass for each function call.
	RunAST *InnerBlock

	Block    *BlockSpec
	Priority int

	// Lookahead, when non-nil, is a predicate the parser checks after the
	// header matches: the candidate only applies if the token following
	// its statement satisfies the predicate. See Lookahead (missing.md §5).
	Lookahead *Lookahead
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
	// Optional marks a trailing capture that may be omitted at the
	// call site; Default is the source-form value bound when omitted.
	Optional bool
	Default  string
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
	Closer     string
	Open       string
	Close      string
	IsDedent   bool
	IsVerbatim bool
	// Sections, when non-empty, makes this a multi-section block (e.g.
	// `try … rescue … finally … end`). Each entry is an interior section
	// keyword that appears at the opener's indent and introduces its own
	// indented sub-body. The main body and each section body are rendered
	// independently and exposed to the template as `${body}` and a local
	// named after each section keyword (`${rescue}`, `${finally}`). The
	// block is closed by Closer. Enables try/rescue/finally (missing.md §8).
	Sections []string
}

// Lookahead gates a candidate on what follows its header, after the
// trailing newline. It enables context-sensitive keyword reuse
// (missing.md §5): e.g. a flat `os "X"` allowlist entry
// (`when_not_followed_by indent`) coexisting with an `os "X"` that opens
// an indented conditional block (`when_followed_by indent`). At most one
// of the two fields is set.
type Lookahead struct {
	RequireIndent bool // when_followed_by indent
	ForbidIndent  bool // when_not_followed_by indent
}

// PatternElement is one compiled token in the function's match shape.
type PatternElement struct {
	IsCapture bool
	Literal   string
	Name      string
	CapType   string
	// Optional / Default apply to optional trailing capture elements:
	// when the statement ends before this element is reached, the
	// matcher binds Default instead of consuming a token.
	Optional bool
	Default  string
}

// TypeDef is a library-defined argument type. Three optional fields applied
// in order at validation time: Base → Pattern → Options.
type TypeDef struct {
	Name        string
	Description string   // optional, surfaced by `capy docs`
	Base        string   // any | string | int | float | bool
	Pattern     string   // optional regex on the value's string form
	Options     []string // optional enum membership
	// GroupOpen / GroupClose mark a type as a delimited inline group.
	// When set, the capture machinery walks tokens between the open
	// and close delimiters (with depth tracking) and returns the
	// joined source-form text. Constraint fields (Base / Pattern /
	// Options) are mutually exclusive with these.
	GroupOpen  string
	GroupClose string
}
