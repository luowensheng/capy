// Package capy embeds Capy as a Go library so programs can define their own
// syntax inline without invoking a separate binary or shipping library files.
//
// Quick start:
//
//	lib, err := capy.NewLibrary(`
//	    extension html
//	    function button
//	        arg literal "button"
//	        arg capture label string
//	        template_str "<button>{{ .label }}</button>\n"
//	    end
//	`)
//	if err != nil { log.Fatal(err) }
//
//	out, err := lib.Run(`button "Click me"`)
//	// → <button>"Click me"</button>
//
// The library you pass to NewLibrary can be in either format Capy supports:
//
//   - **Capy-native** (`.capy`) — detected automatically. Recommended for
//     embedded use because it's terser and there's no YAML escaping to fight.
//   - **YAML** (`.yaml`) — pass via NewLibraryYAML when you need YAML
//     features (anchors, complex types) or want to share the library with
//     non-Go tooling.
//
// Why embed Capy in your Go program?
//
//   - You're shipping a CLI that takes a config in a friendlier-than-YAML
//     DSL — write the parser in 50 lines of Capy instead of 500 of Go.
//   - You're building a code generator (Prisma-style schema → migrations)
//     and want users to write `model User { name : string }` instead of
//     calling a Go builder API.
//   - You want hot-swappable grammars: read a library file at startup,
//     let users contribute new ones without recompiling.
package capy

import (
	"github.com/luowensheng/capy/domain"
	"github.com/luowensheng/capy/features"
	"github.com/luowensheng/capy/infra"
	orchfeatures "github.com/luowensheng/capy/orchestrator/features"
)

// Library is a compiled, ready-to-run Capy library. Safe to reuse across
// many Run calls; not safe for concurrent mutation but Run itself is
// re-entrant on a fixed Library.
type Library struct {
	lib    domain.Library
	lex    features.Lexer
	parser features.Parser
	eval   features.Evaluator
}

// NewLibrary compiles a library written in Capy's native syntax (`.capy`
// format) from an in-memory string. The returned Library is ready to Run
// any number of source scripts.
//
// Most embedding callers use this — it has no YAML escaping pain and
// reads natively.
func NewLibrary(librarySrc string) (*Library, error) {
	return newFromBytes("capy", []byte(librarySrc))
}

// NewLibraryYAML compiles a library written in YAML. Use when you have an
// existing YAML library or want yq/JSON-schema tooling.
func NewLibraryYAML(librarySrc string) (*Library, error) {
	return newFromBytes("yaml", []byte(librarySrc))
}

// NewLibraryFromFile reads a library file from disk. The format is chosen
// by the file extension (`.capy` → native; otherwise YAML).
func NewLibraryFromFile(path string) (*Library, error) {
	lex := orchfeatures.MakeLexer()
	loader := orchfeatures.MakeLibraryLoader(infra.YamlParser{}, lex.Tokenize)
	dl, err := loader.Load(path)
	if err != nil {
		return nil, err
	}
	return assemble(dl, lex), nil
}

func newFromBytes(format string, src []byte) (*Library, error) {
	lex := orchfeatures.MakeLexer()
	dl, err := orchfeatures.LoadLibraryFromBytes(format, src, lex.Tokenize)
	if err != nil {
		return nil, err
	}
	return assemble(dl, lex), nil
}

func assemble(dl domain.Library, lex features.Lexer) *Library {
	tpl := orchfeatures.MakeTemplateRenderer(infra.TemplateEngine{})
	return &Library{
		lib:    dl,
		lex:    lex,
		parser: orchfeatures.MakeParser(),
		eval:   orchfeatures.MakeEvaluator(tpl),
	}
}

// Run transpiles a single source script through this library and returns
// the generated output as a string. The library's file_template (if any)
// wraps the output.
//
// Run is safe to call repeatedly on the same Library with different
// sources — each call runs a fresh accumulating context.
func (l *Library) Run(scriptSrc string) (string, error) {
	toks, err := l.lex.Tokenize(scriptSrc)
	if err != nil {
		return "", err
	}
	prog, err := l.parser.Parse(toks, l.lib)
	if err != nil {
		return "", err
	}
	return l.eval.Run(prog, l.lib)
}

// RunMulti is like Run but also returns the rendered multi-file map for
// libraries that declared `file "path":` blocks. The map is empty for
// libraries that don't use multi-file output.
func (l *Library) RunMulti(scriptSrc string) (string, map[string]string, error) {
	toks, err := l.lex.Tokenize(scriptSrc)
	if err != nil {
		return "", nil, err
	}
	prog, err := l.parser.Parse(toks, l.lib)
	if err != nil {
		return "", nil, err
	}
	return l.eval.RunMulti(prog, l.lib)
}

// Extension reports the library's declared `extension:` field — useful
// when you want to write the output to a file with the correct suffix.
func (l *Library) Extension() string { return l.lib.Extension }

// OutputFile reports the library's optional `output_file:` field.
func (l *Library) OutputFile() string { return l.lib.OutputFile }

// FunctionNames returns the sorted list of function names declared by
// the library. Useful for diagnostics and auto-discovery.
func (l *Library) FunctionNames() []string {
	out := make([]string, 0, len(l.lib.Functions))
	for name := range l.lib.Functions {
		out = append(out, name)
	}
	return out
}
