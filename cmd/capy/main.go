// Capy CLI.
//
// Subcommands:
//
//	capy run <library.yaml> <script.capy>
//	capy check <library.yaml>
//	capy init [<dir>]
//	capy version
//	capy help [<command>]
//
// Top-level invocation styles also supported for ergonomics:
//
//	capy <library.yaml> <script.capy>     (run shorthand)
//	capy -lib <library.yaml> <script.capy> (legacy flag form)
package main

import (
	"fmt"
	"os"
)

// version is set at build time via -ldflags "-X main.version=v0.1.0".
var version = "dev"

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func dispatch(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return nil
	}
	switch args[0] {
	case "run":
		return cmdRun(args[1:])
	case "check":
		return cmdCheck(args[1:])
	case "docs":
		return cmdDocs(args[1:])
	case "init":
		return cmdInit(args[1:])
	case "version", "--version", "-v":
		fmt.Println("capy", version)
		return nil
	case "help", "--help", "-h":
		if len(args) > 1 {
			return cmdHelp(args[1])
		}
		printUsage(os.Stdout)
		return nil
	}
	// Legacy/positional form: try to treat as `run`.
	return cmdRun(args)
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, `capy — a transpiler engine driven by a YAML library

Usage:
  capy run <library.yaml> <script.capy>   transpile a script
  capy check <library.yaml>               validate a library without running anything
  capy init [<dir>]                       scaffold a new library project
  capy version                            print version
  capy help [<command>]                   show detailed help

See https://github.com/luowensheng/capy for documentation.`)
}
