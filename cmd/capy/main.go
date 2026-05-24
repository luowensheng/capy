// Capy CLI.
//
// Subcommands:
//
//	capy run <library> <script>          transpile a script
//	capy check <library>                 validate a library
//	capy docs <library>                  print auto-generated reference docs
//	capy lib list|which|new|path         manage installed libraries (CAPY_LIBS)
//	capy new <dir> --using <library>     scaffold a new project from a library
//	capy <library-name> <command> [args] dispatch a library command
//	capy init [<dir>]                    legacy: scaffold a starter library
//	capy version
//	capy help [<command>]
//
// Short forms also supported:
//
//	capy <library.capy> <script>         positional run (legacy)
//	capy <script.<libname>>              auto-resolves the library by extension
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luowensheng/capy/orchestrator"
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
	case "lib":
		return cmdLib(args[1:])
	case "new":
		return cmdNew(args[1:])
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
	// Library-name short form: `capy <lib> <command> [args]`. Only
	// if the first arg looks like a library name AND resolves on
	// CAPY_LIBS.
	if libraryNameLooksValid(args[0]) {
		if libPath, err := resolveLib(args[0]); err == nil {
			if len(args) < 2 {
				return fmt.Errorf("library %q resolved at %s; pick a command (try: run, build, compile, docs)", args[0], libPath)
			}
			return orchestrator.RunCommand(libPath, args[1], args[2:])
		}
	}
	// File-extension convention: `capy <script.<libname>>` auto-
	// resolves the library by extension.
	if len(args) == 1 {
		if libPath, ok := resolveLibFromScriptExt(args[0]); ok {
			return orchestrator.RunCommand(libPath, "run", []string{args[0]})
		}
	}
	// Legacy positional form: `capy <library.capy> <script>`.
	return cmdRun(args)
}

// resolveLibFromScriptExt tries to find a library based on a
// script's extension: `cake.recipe` → lookup `recipe` on CAPY_LIBS.
func resolveLibFromScriptExt(scriptPath string) (string, bool) {
	ext := filepath.Ext(scriptPath)
	if ext == "" || ext == ".capy" {
		return "", false
	}
	libname := strings.TrimPrefix(ext, ".")
	libPath, err := resolveLib(libname)
	if err != nil {
		return "", false
	}
	return libPath, true
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, `capy — a transpiler engine driven by a Capy library

Usage:
  capy run <library> <script>            transpile a script
  capy check <library>                   validate a library
  capy docs <library>                    print auto-generated reference docs
  capy lib list                          list installed libraries (CAPY_LIBS)
  capy lib which <name>                  show full path of a library
  capy lib new <name>                    scaffold a new library
  capy lib path                          print the library search path
  capy new <dir> --using <library>       scaffold a new project from a library
  capy <library> <command> [args]        dispatch a library command
  capy version                           print version
  capy help [<command>]                  show detailed help

Examples:
  CAPY_LIBS=~/.capy/libs capy lib list
  capy lib new recipe
  capy recipe run examples/hello.recipe
  capy cake.recipe                       # auto-detects library by extension

See https://luowensheng.github.io/capy/ for documentation.`)
}
