package orchestrator

import (
	"github.com/olivierdevelops/capy/infra"
	orchfeatures "github.com/olivierdevelops/capy/orchestrator/features"
	orchusecases "github.com/olivierdevelops/capy/orchestrator/usecases"
	orchviews "github.com/olivierdevelops/capy/orchestrator/views"
)

type AppOrchestrator struct{}

func (AppOrchestrator) RunCLI(scriptPath, libraryPath string) int {
	files := infra.FileReader{}

	lex := orchfeatures.MakeLexer()
	parser := orchfeatures.MakeParser()
	eval := orchfeatures.MakeEvaluator()
	libLoader := orchfeatures.MakeLibraryLoader(lex.Tokenize)

	rs := orchusecases.MakeRunScriptMulti(
		files.Read,
		lex.Tokenize,
		parser.Parse,
		eval.Run,
		eval.RunMulti,
		libLoader.Load,
		files.Write,
	)
	view := orchviews.MakeCLIView(orchusecases.AsCLIUseCase(rs), scriptPath, libraryPath)
	return view.Render()
}
