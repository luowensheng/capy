package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/olivierdevelops/capy/domain"
	"github.com/olivierdevelops/capy/infra"
	orchfeatures "github.com/olivierdevelops/capy/orchestrator/features"
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
	// Trust check: warn when the library is NOT on CAPY_LIBS.
	// Library commands can shell out arbitrarily; surface the
	// surprise early. Suppressed via CAPY_TRUST=1 or --trust (CLI).
	if shouldWarnUntrusted(libraryPath) {
		fmt.Fprintf(os.Stderr,
			"warning: library %q is not on CAPY_LIBS — its commands can shell out / write files / read env\n",
			libraryPath)
	}

	lex := orchfeatures.MakeLexer()
	loader := orchfeatures.MakeLibraryLoader(lex.Tokenize)
	lib, err := loader.Load(libraryPath)
	if err != nil {
		return err
	}
	cmd, ok := lib.Commands[cmdName]
	if !ok {
		return fmt.Errorf("library %q has no command %q (declared: %v)",
			libraryPath, cmdName, sortedCommandNames(lib))
	}

	// `--help` / `-h` short-circuit.
	for _, a := range args {
		if a == "--help" || a == "-h" {
			PrintCommandHelp(lib, cmd)
			return nil
		}
	}

	// Parse declared positional args + flags. Anything not consumed
	// overflows into `extra` (still surfaced via context.args).
	posValues, flagValues, extra, err := ParseCommandArgs(cmd, args)
	if err != nil {
		return err
	}

	host := infra.OSHost{
		UserArgs: args,
		BaseDir:  filepath.Dir(libraryPath),
	}

	// Build a fresh execution context.
	ctx := map[string]any{
		"lib_path":    libraryPath,
		"lib_dir":     filepath.Dir(libraryPath),
		"lib_name":    nameOrPath(lib, libraryPath),
		"lib_version": lib.LibVersion,
		"args":        anyList(args),
		"extra":       anyList(extra),
	}
	// Declared positional args appear under their declared names
	// (e.g. `arg "script"` → context.script).
	for name, val := range posValues {
		ctx[name] = val
	}
	// Declared flags appear under context.flags.NAME (trimmed of
	// leading dashes).
	flagsMap := map[string]any{}
	for name, val := range flagValues {
		flagsMap[name] = val
	}
	ctx["flags"] = flagsMap
	// Backwards-compatible numeric aliases for libraries that
	// haven't declared positionals.
	for i, a := range args {
		key := fmt.Sprintf("arg%d", i)
		if _, exists := ctx[key]; !exists {
			ctx[key] = a
		}
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

// shouldWarnUntrusted returns true when libraryPath is NOT a
// descendant of any directory in CAPY_LIBS (or the default
// fallbacks). Suppressed via CAPY_TRUST=1.
func shouldWarnUntrusted(libraryPath string) bool {
	if os.Getenv("CAPY_TRUST") == "1" {
		return false
	}
	abs, err := filepath.Abs(libraryPath)
	if err != nil {
		return true
	}
	for _, dir := range libSearchPathRuntime() {
		da, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		// Trust if libraryPath is under any search-path dir.
		rel, err := filepath.Rel(da, abs)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return false
		}
	}
	return true
}

// libSearchPathRuntime mirrors the CLI's libSearchPath but lives
// in the orchestrator package so we can share the trust check
// between CLI dispatch and `call` (which loops back into this
// package).
func libSearchPathRuntime() []string {
	env := os.Getenv("CAPY_LIBS")
	if env != "" {
		sep := ":"
		if isWindows() {
			sep = ";"
		}
		parts := strings.Split(env, sep)
		var out []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	// CWD is part of the default search path: a project's local
	// library is trusted without requiring CAPY_LIBS to be set.
	out := []string{}
	if cwd, err := os.Getwd(); err == nil {
		out = append(out, cwd)
	}
	home, _ := os.UserHomeDir()
	out = append(out,
		filepath.Join(home, ".config", "capy", "libs"),
		filepath.Join(home, ".capy", "libs"),
		filepath.Join(home, "Library", "Application Support", "Capy", "libs"),
	)
	return out
}

func isWindows() bool {
	return strings.Contains(strings.ToLower(os.Getenv("OS")), "windows")
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
// command-only primitives that need access to the library:
//   - `compile SCRIPT_PATH`  → run the library on a script
//   - `call CMD_NAME arg…`    → invoke another command of THIS library
func (c *commandEnv) dispatch(name string, args []any) (any, bool, error) {
	switch name {
	case "compile":
		// compile SCRIPT_PATH → output string.
		if len(args) != 1 {
			return nil, true, fmt.Errorf("compile expects 1 arg (script path)")
		}
		scriptPath := fmt.Sprintf("%v", args[0])
		if !filepath.IsAbs(scriptPath) {
			cwd, _ := os.Getwd()
			scriptPath = filepath.Join(cwd, scriptPath)
		}
		out, err := Run(c.libraryPath, scriptPath)
		if err != nil {
			return nil, true, err
		}
		return out, true, nil
	case "call":
		// call CMD_NAME arg… — invoke another command of the same
		// library. Returns "" so it can be used either as a
		// statement (`call build context.script`) or as a value
		// expression (`let x = (call build context.script)`).
		if len(args) == 0 {
			return nil, true, fmt.Errorf("call expects at least 1 arg (command name)")
		}
		target := fmt.Sprintf("%v", args[0])
		callArgs := make([]string, len(args)-1)
		for i, a := range args[1:] {
			callArgs[i] = fmt.Sprintf("%v", a)
		}
		if err := RunCommand(c.libraryPath, target, callArgs); err != nil {
			return nil, true, err
		}
		return "", true, nil
	}
	return nil, false, nil
}
