package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/luowensheng/capy/domain"
	"github.com/luowensheng/capy/orchestrator"
	orchfeatures "github.com/luowensheng/capy/orchestrator/features"
)

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	out := fs.String("out", "", "override library's output_file (write to this path instead of stdout)")
	outDir := fs.String("out-dir", "", "write multi-file output here. Required for libraries with `file \"...\":` blocks (unless --zip used).")
	zipPath := fs.String("zip", "", "bundle multi-file output as a zip archive at this path (alternative to --out-dir)")
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
	var userArgs []string
	switch {
	case *legacyLib != "" && len(pos) >= 1:
		libPath = *legacyLib
		scriptPath = pos[0]
		userArgs = pos[1:]
	case len(pos) >= 2:
		libPath = pos[0]
		scriptPath = pos[1]
		// Anything beyond <library> <script> is a positional arg for
		// the library to consume via the inner `arg N` primitive.
		userArgs = pos[2:]
	case len(pos) == 1:
		// One-arg form: `capy run <script.ext>` — auto-resolve the
		// library by looking for a `.capy` file matching the script's
		// extension (e.g. `main.interface` → `interface.capy`), or
		// `lib.capy` in the script's directory.
		scriptPath = pos[0]
		lp, err := autoResolveLib(scriptPath)
		if err != nil {
			return err
		}
		libPath = lp
		// If the resolved library declares a `command "run"`, dispatch
		// through it (same path as `capy <script.ext>` without the
		// explicit `run` subcommand) so users get the library's full
		// `run` behaviour — including any post-render side-effects like
		// opening a browser. Falling back to the lower-level transpile
		// path only when no such command exists keeps the legacy
		// behaviour for libraries that haven't declared one.
		if libraryHasCommand(libPath, "run") {
			return orchestrator.RunCommand(libPath, "run", []string{scriptPath})
		}
		userArgs = nil
	default:
		return fmt.Errorf("usage: capy run [--out-dir DIR | --zip ARCHIVE.zip] [<library>] <script> [args...]")
	}

	// Read source for nice error formatting.
	src, _ := os.ReadFile(scriptPath)

	output, files, err := orchestrator.RunMultiWithArgs(libPath, scriptPath, userArgs)
	if err != nil {
		return fmt.Errorf("%s", domain.FormatWithSource(err, string(src)))
	}

	// Multi-file output path. Each file is either written under --out-dir
	// or bundled into --zip. Exactly one of those should be set.
	if len(files) > 0 {
		switch {
		case *zipPath != "" && *outDir != "":
			return fmt.Errorf("use --zip OR --out-dir, not both")
		case *zipPath != "":
			return writeZip(*zipPath, files)
		case *outDir != "":
			return writeTree(*outDir, files)
		default:
			return fmt.Errorf("library declared %d `file \"...\":` block(s); pass --out-dir DIR or --zip ARCHIVE.zip to write them", len(files))
		}
	}

	// Single-output. Precedence: --out flag → library's `output_file`
	// directive → stdout.
	if *out != "" {
		return os.WriteFile(*out, []byte(output), 0644)
	}
	if libOut := peekOutputFile(libPath); libOut != "" {
		full := libOut
		if !filepath.IsAbs(full) {
			full = filepath.Join(filepath.Dir(scriptPath), libOut)
		}
		return os.WriteFile(full, []byte(output), 0644)
	}
	fmt.Print(output)
	return nil
}

// peekOutputFile reloads the library just to read its `output_file:`
// directive. Cheap (the library is small), and avoids threading the
// value through orchestrator.RunMultiWithArgs.
func peekOutputFile(libPath string) string {
	lex := orchfeatures.MakeLexer()
	loader := orchfeatures.MakeLibraryLoader(lex.Tokenize)
	lib, err := loader.Load(libPath)
	if err != nil {
		return ""
	}
	return lib.OutputFile
}


// writeTree writes every (path, content) pair under root, creating
// subdirectories as needed. Paths are sorted for deterministic logging.
func writeTree(root string, files map[string]string) error {
	paths := sortedKeys(files)
	for _, rel := range paths {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(files[rel]), 0644); err != nil {
			return fmt.Errorf("write %s: %v", full, err)
		}
		fmt.Fprintf(os.Stderr, "wrote %s (%d bytes)\n", full, len(files[rel]))
	}
	return nil
}

// writeZip bundles every (path, content) into a single zip archive. Paths
// inside the zip are preserved verbatim (subdirectories supported by
// archive/zip natively).
func writeZip(zipFile string, files map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(zipFile), 0755); err != nil {
		return err
	}
	f, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	paths := sortedKeys(files)
	for _, rel := range paths {
		// Use forward slashes inside zip archives — POSIX convention,
		// works on Windows tools too.
		zipRel := filepath.ToSlash(rel)
		w, err := zw.Create(zipRel)
		if err != nil {
			return fmt.Errorf("zip entry %s: %v", zipRel, err)
		}
		if _, err := w.Write([]byte(files[rel])); err != nil {
			return fmt.Errorf("zip write %s: %v", zipRel, err)
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	stat, _ := f.Stat()
	size := int64(0)
	if stat != nil {
		size = stat.Size()
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d entries, %d bytes)\n", zipFile, len(files), size)
	return nil
}

// autoResolveLib finds the library file to use when the user invokes
// `capy run` with only a script path. Resolution order, all relative
// to the script's directory:
//
//  1. <ext>.capy  — extension-name match (e.g. main.interface → interface.capy)
//  2. lib.capy    — conventional default name
//
// If neither exists, returns an error pointing at both candidates.
func autoResolveLib(scriptPath string) (string, error) {
	dir := filepath.Dir(scriptPath)
	ext := strings.TrimPrefix(filepath.Ext(scriptPath), ".")
	candidates := []string{}
	if ext != "" {
		candidates = append(candidates, filepath.Join(dir, ext+".capy"))
	}
	candidates = append(candidates, filepath.Join(dir, "lib.capy"))
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("no library found for %q (looked for: %s)",
		scriptPath, strings.Join(candidates, ", "))
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
