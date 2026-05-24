package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// cmdWatch polls a set of files for mtime changes and re-runs a
// chosen command whenever any of them updates. Polling (vs.
// fsnotify) keeps the dependency list small and works the same on
// every OS.
//
// Usage:
//
//	capy watch <library> <script>       # legacy: re-run `capy run lib script`
//	capy watch <library>                # watch just the lib; uses its `run` command on stdin
//	capy watch <library-name> <command> [args]
//
// In all forms, the watched set is: every .capy file in the
// library's directory + the script (if any).
func cmdWatch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: capy watch <library> [<script> | <command> [args...]]")
	}

	// Resolve the library — by name (CAPY_LIBS) or by direct path.
	first := args[0]
	rest := args[1:]
	var libPath string
	var libDir string
	if p, err := resolveLib(first); err == nil {
		libPath = p
		libDir = filepath.Dir(p)
	} else if _, err := os.Stat(first); err == nil {
		libPath = first
		libDir = filepath.Dir(first)
	} else {
		return fmt.Errorf("library %q not found (CAPY_LIBS or path)", first)
	}

	// Determine the watched paths + the command to execute.
	var watched []string
	var argv []string
	if hasCommandViaName(libPath, "run") {
		// Library has commands declared → assume user wants
		// `capy <lib> <cmd> [args]`. If the first remaining arg is
		// a declared command in the library, use it; otherwise
		// fall back to `run` with all `rest` as args.
		cmdName := "run"
		cmdArgs := rest
		if len(rest) > 0 && libraryHasCommand(libPath, rest[0]) {
			cmdName = rest[0]
			cmdArgs = rest[1:]
		}
		argv = append([]string{first, cmdName}, cmdArgs...)
		// Watch every script path mentioned in cmdArgs that exists.
		for _, a := range cmdArgs {
			if st, err := os.Stat(a); err == nil && !st.IsDir() {
				watched = append(watched, a)
			}
		}
	} else {
		// Legacy form: `capy run <library> <script>`.
		if len(rest) < 1 {
			return fmt.Errorf("usage: capy watch <library> <script>")
		}
		argv = append([]string{"run", libPath}, rest...)
		// Watch every existing file arg.
		for _, a := range rest {
			if st, err := os.Stat(a); err == nil && !st.IsDir() {
				watched = append(watched, a)
			}
		}
	}

	// Always watch every .capy file in the library's directory.
	if libDir != "" {
		_ = filepath.WalkDir(libDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".capy") {
				watched = append(watched, path)
			}
			return nil
		})
	}

	if len(watched) == 0 {
		return fmt.Errorf("watch: no files to watch")
	}

	fmt.Fprintf(os.Stderr, "👀 watching %d file(s); re-runs on save (Ctrl-C to exit)\n", len(watched))
	for _, w := range watched {
		fmt.Fprintf(os.Stderr, "    %s\n", w)
	}

	// First run.
	runOnce(argv)
	// Poll loop.
	last := snapshotMtimes(watched)
	for {
		time.Sleep(250 * time.Millisecond)
		cur := snapshotMtimes(watched)
		if !sameMtimes(last, cur) {
			last = cur
			fmt.Fprintln(os.Stderr, "\n--- change detected — re-running ---")
			runOnce(argv)
		}
	}
}

func runOnce(argv []string) {
	self, err := os.Executable()
	if err != nil {
		self = "capy"
	}
	cmd := exec.Command(self, argv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	_ = cmd.Run() // don't bail on non-zero — the next change might fix it
}

func snapshotMtimes(paths []string) map[string]time.Time {
	out := map[string]time.Time{}
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil {
			out[p] = st.ModTime()
		}
	}
	return out
}

func sameMtimes(a, b map[string]time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok || !va.Equal(vb) {
			return false
		}
	}
	return true
}

// libraryHasCommand peeks at the library file's text for a
// `command "NAME"` declaration. Cheap and avoids a full load just
// to check.
func libraryHasCommand(libPath, name string) bool {
	b, err := os.ReadFile(libPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(b), fmt.Sprintf(`command "%s"`, name))
}

func hasCommandViaName(libPath, name string) bool {
	return libraryHasCommand(libPath, name)
}
