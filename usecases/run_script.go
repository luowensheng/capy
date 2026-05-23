package usecases

import "github.com/luowensheng/capy/domain"

type RunScript interface {
	Execute(scriptPath, libraryPath string) (RunResult, error)
}

type RunResult struct {
	Output     string
	OutputFile string
}

type (
	TokenizeFn func(source string) ([]domain.Token, error)
	ParseFn    func(toks []domain.Token, lib domain.Library) (domain.Block, error)
	EvaluateFn func(program domain.Block, lib domain.Library) (string, error)
	LoadLibFn  func(path string) (domain.Library, error)
	ReadFileFn func(path string) (string, error)
)
