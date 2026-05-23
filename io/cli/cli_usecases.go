package cli

// Protocols the view model needs. Consumer-owned: declared next to the VM.
type RunScriptUseCase interface {
	Execute(scriptPath, libraryPath string) (RunOutcome, error)
}

type RunOutcome struct {
	Output     string
	OutputFile string
}
