package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luowensheng/capy/domain"
	"github.com/luowensheng/capy/infra"
	orchfeatures "github.com/luowensheng/capy/orchestrator/features"
)

// RunCommand looks up command NAME on the library at libraryPath
// and runs its body. Positional args after the command name are
// exposed to the body via `args.0`, `args.1`, … and `args` (list).
//
// The command body has access to:
//   - `compile <script>` primitive that runs the library on a script
//   - shell-like host primitives (write_file, mktemp, exec, …)
//   - standard inner-DSL state mutation (set / append / let / if /
//     for)
func RunCommand(libraryPath, cmdName string, args []string) error {
	lex := orchfeatures.MakeLexer()
	yp := infra.YamlParser{}
	loader := orchfeatures.MakeLibraryLoader(yp, lex.Tokenize)
	lib, err := loader.Load(libraryPath)
	if err != nil {
		return err
	}
	cmd, ok := lib.Commands[cmdName]
	if !ok {
		return fmt.Errorf("library %q has no command %q (declared: %v)",
			libraryPath, cmdName, sortedCommandNames(lib))
	}

	host := infra.OSHost{
		UserArgs: args,
		BaseDir:  filepath.Dir(libraryPath),
	}

	// Build a fresh execution context: command bodies see `args` and
	// `lib_dir` and `args.N` paths.
	ctx := map[string]any{
		"lib_path":    libraryPath,
		"lib_dir":     filepath.Dir(libraryPath),
		"lib_name":    nameOrPath(lib, libraryPath),
		"lib_version": lib.LibVersion,
		"args":        anyList(args),
	}
	// Expose args.0, args.1, … as named keys too for ergonomic access.
	for i, a := range args {
		ctx[fmt.Sprintf("arg%d", i)] = a
	}

	// Eval the body.
	ev := &orchfeatures.InnerEvaluator{Context: ctx, Host: host}
	// Provide a captures map that carries the library handle so
	// `compile script` can reach back into the library to render.
	cmdEnv := newCommandEnv(libraryPath, lib, host)
	ev.OnUnknownCall = cmdEnv.dispatch
	if err := ev.Exec(cmd.Body, map[string]domain.CaptureValue{}); err != nil {
		return err
	}
	return nil
}

func sortedCommandNames(lib domain.Library) []string {
	out := make([]string, 0, len(lib.Commands))
	for n := range lib.Commands {
		out = append(out, n)
	}
	// Simple sort — small slice, doesn't matter.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func nameOrPath(lib domain.Library, libraryPath string) string {
	if lib.LibName != "" {
		return lib.LibName
	}
	return strings.TrimSuffix(filepath.Base(libraryPath), filepath.Ext(libraryPath))
}

func anyList(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// commandEnv captures the library handle + host so command-body
// primitives that need them (like `compile script`) can call back
// into the engine to render the library on a given script.
type commandEnv struct {
	libraryPath string
	lib         domain.Library
	host        domain.Host
}

func newCommandEnv(libraryPath string, lib domain.Library, host domain.Host) *commandEnv {
	return &commandEnv{libraryPath: libraryPath, lib: lib, host: host}
}

// dispatch is the fallback for unknown inner calls. It implements
// command-only primitives that need access to the library (today:
// `compile`).
func (c *commandEnv) dispatch(name string, args []any) (any, bool, error) {
	switch name {
	case "compile":
		// compile SCRIPT_PATH → output string.
		if len(args) != 1 {
			return nil, true, fmt.Errorf("compile expects 1 arg (script path)")
		}
		scriptPath := fmt.Sprintf("%v", args[0])
		// Resolve relative to CWD; OS user typed it from the shell.
		if !filepath.IsAbs(scriptPath) {
			cwd, _ := os.Getwd()
			scriptPath = filepath.Join(cwd, scriptPath)
		}
		out, err := Run(c.libraryPath, scriptPath)
		if err != nil {
			return nil, true, err
		}
		return out, true, nil
	}
	return nil, false, nil
}
