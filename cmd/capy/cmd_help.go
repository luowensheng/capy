package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	orchfeatures "github.com/olivierdevelops/capy/orchestrator/features"
)

// printLibraryHelp loads a library and prints its declared commands.
func printLibraryHelp(libPath string) error {
	lex := orchfeatures.MakeLexer()
	loader := orchfeatures.MakeLibraryLoader(lex.Tokenize)
	lib, err := loader.Load(libPath)
	if err != nil {
		return err
	}
	libName := lib.LibName
	if libName == "" {
		libName = strings.TrimSuffix(filepath.Base(libPath), filepath.Ext(libPath))
	}
	if lib.LibVersion != "" {
		fmt.Printf("%s %s\n", libName, lib.LibVersion)
	} else {
		fmt.Printf("%s\n", libName)
	}
	if lib.Description != "" {
		fmt.Printf("%s\n", lib.Description)
	}
	fmt.Println()
	fmt.Println("COMMANDS")
	if len(lib.Commands) == 0 {
		fmt.Println("    (none declared — the default `run` renders to stdout)")
	}
	names := make([]string, 0, len(lib.Commands))
	for n := range lib.Commands {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		c := lib.Commands[n]
		desc := c.Description
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Printf("    %-12s  %s\n", n, desc)
	}
	fmt.Printf("\nRun `capy %s <command> --help` for command-specific help.\n", libName)
	return nil
}

func cmdHelp(name string) error {
	switch name {
	case "run":
		fmt.Println(`capy run <library.yaml> <script.capy>

Transpile a script against a library. Output goes to stdout unless the
library sets ` + "`output_file:`" + ` or you pass --out.

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
