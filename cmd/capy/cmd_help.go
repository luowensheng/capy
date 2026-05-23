package main

import (
	"fmt"
	"os"
)

func cmdHelp(name string) error {
	switch name {
	case "run":
		fmt.Println(`capy run <library.yaml> <script.capy>

Transpile a script against a library. Output goes to stdout unless the
library sets `+"`output_file:`"+` or you pass --out.

Flags:
  --out <path>    write output to this file instead of stdout
  --no-color      disable ANSI escape codes (reserved)
  --debug         verbose engine tracing (reserved)`)
	case "check":
		fmt.Println(`capy check <library.yaml>

Parse and validate a library file without running anything. Reports
loaded functions and types if valid, or a structured error otherwise.`)
	case "init":
		fmt.Println(`capy init [<dir>]

Scaffold a starter project (lib.yaml + script.capy + README.md) in the
given directory (default '.'). Refuses to overwrite existing files.`)
	case "version":
		fmt.Println(`capy version

Print the version string baked in at build time.`)
	default:
		fmt.Fprintf(os.Stderr, "no help for %q\n", name)
		printUsage(os.Stdout)
	}
	return nil
}
