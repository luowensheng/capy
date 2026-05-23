package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/luowensheng/capy/domain"
	"github.com/luowensheng/capy/orchestrator"
)

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	out := fs.String("out", "", "override library's output_file (write to this path instead of stdout)")
	debug := fs.Bool("debug", false, "enable verbose engine tracing (currently a no-op)")
	noColor := fs.Bool("no-color", false, "disable colored output (reserved)")
	// Legacy compatibility:
	legacyLib := fs.String("lib", "", "(legacy) library path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	_ = debug
	_ = noColor

	pos := fs.Args()
	var libPath, scriptPath string
	switch {
	case *legacyLib != "" && len(pos) == 1:
		libPath = *legacyLib
		scriptPath = pos[0]
	case len(pos) == 2:
		libPath = pos[0]
		scriptPath = pos[1]
	default:
		return fmt.Errorf("usage: capy run <library.yaml> <script.capy>")
	}

	// Read source for nice error formatting.
	src, _ := os.ReadFile(scriptPath)

	output, err := orchestrator.Run(libPath, scriptPath)
	if err != nil {
		return fmt.Errorf("%s", domain.FormatWithSource(err, string(src)))
	}
	// `output` is already the rendered file. If --out provided, write there.
	if *out != "" {
		return os.WriteFile(*out, []byte(output), 0644)
	}
	// Otherwise the library may have set output_file: but we already wrote via the orchestrator? No — the
	// orchestrator's Run uses the orchestrator package's MakeRunScript only for the CLI flow; Run() here
	// bypasses that and returns the string. Print to stdout if no --out given.
	fmt.Print(output)
	return nil
}
