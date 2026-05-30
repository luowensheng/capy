package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/olivierdevelops/capy/orchestrator"
)

// cmdNew scaffolds a new project using a library.
//
//	capy new <project-dir> --using <library-name>
//
// If the library declares a `new` command, run it with the project
// directory as the first arg. Otherwise: create the project dir,
// drop a `hello.<lib>` example script into it, plus a small README.
//
// The custom command form lets library authors ship richer
// scaffolding (e.g. a Vite-based React component preview lib
// could create an entire Vite scaffold via its `new` command).
func cmdNew(args []string) error {
	// Go's flag package stops at the first positional. Move flags
	// to the front so `capy new ./my-app --using recipe` works.
	args = reorderFlagsFirst(args)
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	using := fs.String("using", "", "library to scaffold the project with")
	if err := fs.Parse(args); err != nil {
		return err
	}
	pos := fs.Args()
	if len(pos) < 1 {
		return fmt.Errorf("usage: capy new <project-dir> --using <library>")
	}
	projectDir := pos[0]
	if *using == "" {
		return fmt.Errorf("`--using <library>` is required")
	}

	libPath, err := resolveLib(*using)
	if err != nil {
		return err
	}

	// If the library has a `new` command, use it.
	if hasCommand(libPath, "new") {
		// Pass projectDir as the first positional arg.
		return orchestrator.RunCommand(libPath, "new", append([]string{projectDir}, pos[1:]...))
	}

	// Fallback: minimal default scaffolding.
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return err
	}
	example := fmt.Sprintf("# Sample %s script.\n", *using)
	if err := os.WriteFile(
		filepath.Join(projectDir, "hello."+*using),
		[]byte(example), 0644,
	); err != nil {
		return err
	}
	readme := fmt.Sprintf(`# %s

A Capy project using the `+"`%s`"+` library.

## Run

`+"```sh"+`
capy %s run hello.%s
`+"```"+`
`, filepath.Base(projectDir), *using, *using, *using)
	if err := os.WriteFile(filepath.Join(projectDir, "README.md"), []byte(readme), 0644); err != nil {
		return err
	}
	fmt.Printf("✓ created project %q using library %q\n", projectDir, *using)
	fmt.Printf("  cd %s && capy %s run hello.%s\n", projectDir, *using, *using)
	return nil
}

// hasCommand checks whether the library at libPath declares a
// command with the given name.
func hasCommand(libPath, name string) bool {
	// Quick text-level scan to avoid full library load just to
	// check — library load can fail for various reasons we don't
	// care about here.
	b, err := os.ReadFile(libPath)
	if err != nil {
		return false
	}
	needle := fmt.Sprintf("command \"%s\"", name)
	return contains(string(b), needle)
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// reorderFlagsFirst pulls every `--flag VALUE` / `-f VALUE` /
// `--flag=VALUE` pair to the front so Go's flag.Parse picks them
// all up before hitting any positional. Single positional? Two?
// Doesn't matter — flags get moved.
func reorderFlagsFirst(args []string) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) > 1 && a[0] == '-' {
			flags = append(flags, a)
			// If the flag isn't `--name=value` form and has a value
			// arg, take the next one too.
			if !containsByte(a, '=') && i+1 < len(args) && !startsWithDash(args[i+1]) {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		pos = append(pos, a)
	}
	return append(flags, pos...)
}

func containsByte(s string, b byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return true
		}
	}
	return false
}

func startsWithDash(s string) bool { return len(s) > 0 && s[0] == '-' }
