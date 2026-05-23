package cli

type CLIState int

const (
	StateIdle CLIState = iota
	StateRunning
	StateSuccess
	StateError
)

type CLIViewModel struct {
	State   CLIState
	Output  string
	File    string
	ErrMsg  string
	useCase RunScriptUseCase
}

func NewCLIViewModel(uc RunScriptUseCase) *CLIViewModel {
	return &CLIViewModel{useCase: uc, State: StateIdle}
}

func (vm *CLIViewModel) Run(scriptPath, libraryPath string) {
	vm.State = StateRunning
	res, err := vm.useCase.Execute(scriptPath, libraryPath)
	if err != nil {
		vm.State = StateError
		vm.ErrMsg = err.Error()
		return
	}
	vm.State = StateSuccess
	vm.Output = res.Output
	vm.File = res.OutputFile
}
