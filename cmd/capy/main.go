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
	// Shebang-style scripts: `#!/usr/bin/env capy --lib recipe` makes
	// the OS invoke `capy --lib recipe <script>`. Honour --lib as
	// "treat the next positional as a script for this library."
	if args[0] == "--lib" {
		if len(args) < 3 {
			return fmt.Errorf("usage: capy --lib <library> <script> [args...]")
		}
		libName := args[1]
		libPath, err := resolveLib(libName)
		if err != nil {
			return err
		}
		// Treat remaining args as: <script> [extra args to command]
		return orchestrator.RunCommand(libPath, "run", args[2:])
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
	case "watch":
		return cmdWatch(args[1:])
	case "fmt":
		return cmdFmt(args[1:])
	case "build":
		return cmdBuild(args[1:])
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
		// Pre-scan args for --impl <name>; the rest flow to the
		// command body unchanged.
		implFlag, rest := extractImplFlag(args[1:])

		implPath, manifestPath, _, resErr := resolveLibWithImpl(args[0], implFlag)
		if resErr == nil {
			// `capy <lib> --help` lists declared commands from the
			// manifest (visible regardless of which impl is picked).
			if len(rest) >= 1 && (rest[0] == "--help" || rest[0] == "-h") {
				return printLibraryHelp(manifestPath)
			}
			if len(rest) == 0 {
				return fmt.Errorf("library %q resolved at %s; pick a command (try: run, build, compile, docs)", args[0], manifestPath)
			}
			return orchestrator.RunCommand(implPath, rest[0], rest[1:])
		}
		// Resolution failed because of the impl selector (multiple
		// impls, no default) — that error is more useful than
		// "library not found."
		if resErr != nil && strings.Contains(resErr.Error(), "impl") {
			return resErr
		}
		// First arg didn't resolve as a library. If it also doesn't
		// exist as a file on disk AND the user supplied a second
		// arg that doesn't look like a script path, they clearly
		// MEANT the short form — report the resolution failure
		// instead of falling through to cmdRun's "no such file."
		if _, statErr := os.Stat(args[0]); statErr != nil {
			if len(args) < 2 || filepath.Ext(args[1]) == "" {
				return resErr
			}
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

// extractImplFlag pulls `--impl <name>` (or `--impl=<name>`) out
// of args. Returns the impl name (empty if not present) and the
// rest of args with the flag removed. Handles both spellings:
//
//	capy chart --impl d3 run …
//	capy chart --impl=d3 run …
func extractImplFlag(args []string) (string, []string) {
	out := make([]string, 0, len(args))
	impl := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--impl" || a == "-i" {
			if i+1 < len(args) {
				impl = args[i+1]
				i++
			}
			continue
		}
		if strings.HasPrefix(a, "--impl=") {
			impl = strings.TrimPrefix(a, "--impl=")
			continue
		}
		out = append(out, a)
	}
	return impl, out
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
