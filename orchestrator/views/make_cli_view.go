package orchviews

import (
	"github.com/olivierdevelops/capy/io/cli"
)

func MakeCLIView(uc cli.RunScriptUseCase, scriptPath, libraryPath string) cli.CLIView {
	vm := cli.NewCLIViewModel(uc)
	vm.Run(scriptPath, libraryPath)
	return cli.CLIView{VM: vm}
}
