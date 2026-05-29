package orchusecases

import (
	"github.com/luowensheng/capy/domain"
	"github.com/luowensheng/capy/io/cli"
	"github.com/luowensheng/capy/usecases"
)

type RunScriptImpl struct {
	read       usecases.ReadFileFn
	tokenize   usecases.TokenizeFn
	parse      usecases.ParseFn
	evaluate   usecases.EvaluateFn
	evalMulti  usecases.EvaluateMultiFn
	loadLib    usecases.LoadLibFn
	writeOut   func(path, content string) error
}

func MakeRunScript(
	read usecases.ReadFileFn,
	tokenize usecases.TokenizeFn,
	parse usecases.ParseFn,
	evaluate usecases.EvaluateFn,
	loadLib usecases.LoadLibFn,
	writeOut func(path, content string) error,
) *RunScriptImpl {
	return &RunScriptImpl{read: read, tokenize: tokenize, parse: parse, evaluate: evaluate, loadLib: loadLib, writeOut: writeOut}
}

// MakeRunScriptMulti wires both the single-file evaluator (for backwards
// compatibility) and the multi-file evaluator. The latter is used when
// the library declared any `file "path":` blocks.
func MakeRunScriptMulti(
	read usecases.ReadFileFn,
	tokenize usecases.TokenizeFn,
	parse usecases.ParseFn,
	evaluate usecases.EvaluateFn,
	evalMulti usecases.EvaluateMultiFn,
	loadLib usecases.LoadLibFn,
	writeOut func(path, content string) error,
) *RunScriptImpl {
	return &RunScriptImpl{
		read:      read,
		tokenize:  tokenize,
		parse:     parse,
		evaluate:  evaluate,
		evalMulti: evalMulti,
		loadLib:   loadLib,
		writeOut:  writeOut,
	}
}

func (r *RunScriptImpl) Execute(scriptPath, libraryPath string) (usecases.RunResult, error) {
	lib := domain.Library{Functions: map[string]*domain.FuncDef{}, Types: map[string]domain.TypeDef{}, Context: map[string]any{}, FilesAST: map[string]*domain.InnerBlock{}}
	if libraryPath != "" {
		l, err := r.loadLib(libraryPath)
		if err != nil {
			return usecases.RunResult{}, err
		}
		lib = l
	}
	src, err := r.read(scriptPath)
	if err != nil {
		return usecases.RunResult{}, err
	}
	toks, err := r.tokenize(src)
	if err != nil {
		return usecases.RunResult{}, err
	}
	prog, err := r.parse(toks, src, lib)
	if err != nil {
		return usecases.RunResult{}, err
	}

	var out string
	var files map[string]string
	if r.evalMulti != nil {
		out, files, err = r.evalMulti(prog, lib)
	} else {
		out, err = r.evaluate(prog, lib)
	}
	if err != nil {
		return usecases.RunResult{}, err
	}
	if lib.OutputFile != "" {
		if err := r.writeOut(lib.OutputFile, out); err != nil {
			return usecases.RunResult{}, err
		}
	}
	return usecases.RunResult{Output: out, OutputFile: lib.OutputFile, Files: files}, nil
}

type cliAdapter struct{ inner *RunScriptImpl }

func (a cliAdapter) Execute(s, l string) (cli.RunOutcome, error) {
	res, err := a.inner.Execute(s, l)
	if err != nil {
		return cli.RunOutcome{}, err
	}
	return cli.RunOutcome{Output: res.Output, OutputFile: res.OutputFile, Files: res.Files}, nil
}

func AsCLIUseCase(r *RunScriptImpl) cli.RunScriptUseCase { return cliAdapter{inner: r} }
