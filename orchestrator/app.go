package orchestrator

import (
	"github.com/luowensheng/capy/infra"
	orchfeatures "github.com/luowensheng/capy/orchestrator/features"
	orchusecases "github.com/luowensheng/capy/orchestrator/usecases"
	orchviews "github.com/luowensheng/capy/orchestrator/views"
)

type AppOrchestrator struct{}

func (AppOrchestrator) RunCLI(scriptPath, libraryPath string) int {
	files := infra.FileReader{}
	yamlP := infra.YamlParser{}
	tplE := infra.TemplateEngine{}

	lex := orchfeatures.MakeLexer()
	parser := orchfeatures.MakeParser()
	tpl := orchfeatures.MakeTemplateRenderer(tplE)
	eval := orchfeatures.MakeEvaluator(tpl)
	libLoader := orchfeatures.MakeLibraryLoader(yamlP, lex.Tokenize)

	rs := orchusecases.MakeRunScript(
		files.Read,
		lex.Tokenize,
		parser.Parse,
		eval.Run,
		libLoader.Load,
		files.Write,
	)
	view := orchviews.MakeCLIView(orchusecases.AsCLIUseCase(rs), scriptPath, libraryPath)
	return view.Render()
}
