package orchestrator

import (
	"os"
	"path/filepath"

	"github.com/luowensheng/capy/infra"
	orchfeatures "github.com/luowensheng/capy/orchestrator/features"
)

// Run loads a library from disk, reads a script, and produces the transpiled
// output as a string. Intended for embedding Capy programmatically (and for
// tests).
func Run(libraryPath, scriptPath string) (string, error) {
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}
	libSrc, err := os.ReadFile(libraryPath)
	if err != nil {
		return "", err
	}
	return RunStrings(string(libSrc), libraryPath, string(src))
}

// RunStrings is like Run but takes the library and script contents directly.
// `libraryPath` is used only to resolve relative paths inside the YAML (e.g.
// future `import:` directives) — pass an empty string if you don't care.
func RunStrings(libraryYAML, libraryPath, scriptSrc string) (string, error) {
	yp := infra.YamlParser{}
	tplE := infra.TemplateEngine{}
	lex := orchfeatures.MakeLexer()
	parser := orchfeatures.MakeParser()
	tpl := orchfeatures.MakeTemplateRenderer(tplE)
	eval := orchfeatures.MakeEvaluator(tpl)

	// Parse YAML in-memory via a temp file to keep the YAML parser path stable.
	// Most production callers should pass a real libraryPath via Run() above.
	if libraryPath == "" {
		// write to a temp file so the parser's file-based API still works
		tmp, err := os.CreateTemp("", "capy-lib-*.yaml")
		if err != nil {
			return "", err
		}
		defer os.Remove(tmp.Name())
		if _, err := tmp.WriteString(libraryYAML); err != nil {
			return "", err
		}
		tmp.Close()
		libraryPath = tmp.Name()
	}
	_ = filepath.Base // reserved for future use

	libLoader := orchfeatures.MakeLibraryLoader(yp, lex.Tokenize)
	lib, err := libLoader.Load(libraryPath)
	if err != nil {
		return "", err
	}
	toks, err := lex.Tokenize(scriptSrc)
	if err != nil {
		return "", err
	}
	prog, err := parser.Parse(toks, lib)
	if err != nil {
		return "", err
	}
	return eval.Run(prog, lib)
}
