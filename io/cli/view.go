package cli

import (
	"fmt"
	"io"
	"os"
)

// CLIView is the dumb view: it observes a CLIViewModel and renders.
// It maps state enums to user-visible text. No business logic.
type CLIView struct {
	VM     *CLIViewModel
	Stdout io.Writer
	Stderr io.Writer
}

func (v CLIView) Render() int {
	out := v.Stdout
	if out == nil {
		out = os.Stdout
	}
	errw := v.Stderr
	if errw == nil {
		errw = os.Stderr
	}
	switch v.VM.State {
	case StateIdle:
		fmt.Fprintln(errw, "capy: nothing to do")
		return 1
	case StateRunning:
		fmt.Fprintln(errw, "capy: still running")
		return 1
	case StateSuccess:
		if v.VM.File != "" {
			fmt.Fprintf(errw, "wrote %s\n", v.VM.File)
		}
		fmt.Fprint(out, v.VM.Output)
		return 0
	case StateError:
		fmt.Fprintf(errw, "capy: %s\n", v.VM.ErrMsg)
		return 1
	}
	return 1
}
