package cli

// Protocols the view model needs. Consumer-owned: declared next to the VM.
type RunScriptUseCase interface {
	Execute(scriptPath, libraryPath string) (RunOutcome, error)
}

type RunOutcome struct {
	Output     string
	OutputFile string
	// Files is populated when the library declared `file "path":` blocks.
	Files map[string]string
}
