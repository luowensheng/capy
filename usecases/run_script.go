package usecases

import "github.com/olivierdevelops/capy/domain"

type RunScript interface {
	Execute(scriptPath, libraryPath string) (RunResult, error)
}

type RunResult struct {
	Output     string
	OutputFile string
	// Files is populated when the library declared `file "path":` blocks.
	Files map[string]string
}

type (
	TokenizeFn      func(source string) ([]domain.Token, error)
	ParseFn         func(toks []domain.Token, src string, lib domain.Library) (domain.Block, error)
	EvaluateFn      func(program domain.Block, lib domain.Library) (string, error)
	EvaluateMultiFn func(program domain.Block, lib domain.Library) (string, map[string]string, error)
	LoadLibFn       func(path string) (domain.Library, error)
	ReadFileFn      func(path string) (string, error)
)
