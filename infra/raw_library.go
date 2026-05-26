package infra

// RawLibrary is the parser-output DTO. Both the `.capy` parser and
// any future format adapter produce values of this shape; the
// orchestrator's loader maps it into a domain.Library.
//
// The DTO carries STRING-form template fields (Template, Run,
// FileTemplate, Files) for parser tests and debug printing, plus
// new-shape inner-DSL Body strings that the renderer-side AST
// path actually uses. YAML tag annotations remain on the struct
// so the type can still be unmarshalled from a YAML config; the
// engine itself no longer ships a YAML parser.
type RawLibrary struct {
	Extension    string
	OutputFile   string
	Description  string
	Types        map[string]RawType
	Context      map[string]interface{}
	Functions    map[string]RawFunction
	FileTemplate string

	// Multi-file output. Map of relative-path → body source.
	Files map[string]string

	// Imports — relative paths to other library files whose functions,
	// types, and context get merged in before this library's own
	// declarations (which take precedence on conflict).
	Imports []string

	// Preprocess — names of source-level inclusion directives this
	// library opts into (e.g. ["@import", "@include"]). When empty,
	// no preprocessing runs. Keeps "zero predefined grammar" honest.
	Preprocess []string

	// Comments — line-comment markers the library opts into for
	// user scripts. Empty list → user scripts have NO comment
	// syntax. Mirrors Preprocess: the engine ships zero defaults
	// and the library must declare what to recognise.
	Comments []string

	// Manifest fields (set when the library declares them at the
	// top level of its .capy file — `name "X"`, `version "X"`).
	LibName    string
	LibVersion string

	// Commands declared by the library — `command "X" ... end`
	// blocks at the top level. The body is collected as raw text;
	// the loader parses it via the inner-DSL parser.
	Commands map[string]RawCommand

	// Implementations: `impl "NAME" "FILE" ... end` blocks at the
	// top level. Each entry points at a sibling .capy file that
	// provides the actual functions. The CLI picks one per
	// invocation via --impl / CAPY_IMPL / default. When the map
	// is non-empty, the manifest file itself only carries
	// metadata + commands; the real library lives in the chosen
	// impl file.
	Impls map[string]RawImpl

	// DefaultImpl names the impl chosen when neither --impl nor
	// CAPY_IMPL picks one. If empty + Impls is non-empty + only
	// one impl declared, that one is used; otherwise an error.
	DefaultImpl string
}

// RawImpl is one `impl "NAME" "FILE" ... end` declaration.
type RawImpl struct {
	Name        string
	File        string
	Description string
	Version     string
	IsDefault   bool
}

// RawCommand is one `command "X" ... end` declaration.
type RawCommand struct {
	Description string
	Body        string
	Args        []RawCommandArg
	Flags       []RawCommandFlag
}

// RawCommandArg is a positional argument declaration.
type RawCommandArg struct {
	Name        string
	Required    bool
	Description string
}

// RawCommandFlag is a flag declaration.
type RawCommandFlag struct {
	Name        string
	Description string
	Default     string
	IsBool      bool
}

type RawFunction struct {
	Description string
	Args        []RawArg
	Block       *RawBlock
	Priority    int

	// Bare opts the function out of the auto-name-prepend rule. With
	// this flag set, a function declared with only `arg capture` entries
	// matches purely by shape — useful for grammars whose data lines
	// have no leading keyword (e.g. `"1" "2" "3"` as a row of button
	// labels).
	Bare bool

	// Body is the function body — inner-DSL statements including
	// `write` calls. The loader parses it into the FuncDef's
	// TemplateAST + RunAST.
	Body string
}

type RawBlock struct {
	Closer string
	Open   string
	Close  string
	// IsDedent: body ends at the first DEDENT after the opener, with
	// no closer keyword. Used for indent-only blocks (CSS-style rules,
	// YAML-style sections, etc.).
	IsDedent bool
	// IsVerbatim: body is captured as raw source bytes (no nested
	// parsing) until the named `Closer` keyword. Used for code blocks,
	// embedded HTML, or anywhere the body is data not grammar.
	IsVerbatim bool
}

// RawArg is an args-list entry. The Kind discriminator is required.
type RawArg struct {
	Kind        string
	Value       string
	Name        string
	Type        string
	Description string
}

type RawType struct {
	Description string
	Base        string
	Pattern     string
	Options     []string
}
