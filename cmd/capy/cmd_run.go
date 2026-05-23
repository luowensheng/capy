package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/luowensheng/capy/domain"
	"github.com/luowensheng/capy/orchestrator"
)

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	out := fs.String("out", "", "override library's output_file (write to this path instead of stdout)")
	outDir := fs.String("out-dir", "", "write multi-file output here. Required when the library declares `file \"...\":` blocks.")
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
	switch {
	case *legacyLib != "" && len(pos) == 1:
		libPath = *legacyLib
		scriptPath = pos[0]
	case len(pos) == 2:
		libPath = pos[0]
		scriptPath = pos[1]
	default:
		return fmt.Errorf("usage: capy run [--out-dir DIR] <library> <script.capy>")
	}

	// Read source for nice error formatting.
	src, _ := os.ReadFile(scriptPath)

	output, files, err := orchestrator.RunMulti(libPath, scriptPath)
	if err != nil {
		return fmt.Errorf("%s", domain.FormatWithSource(err, string(src)))
	}

	// Multi-file output: write every declared file under --out-dir.
	if len(files) > 0 {
		if *outDir == "" {
			return fmt.Errorf("library declared %d `file \"...\":` block(s); pass --out-dir to write them", len(files))
		}
		// Sort paths for deterministic logging.
		paths := make([]string, 0, len(files))
		for p := range files {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		for _, rel := range paths {
			full := filepath.Join(*outDir, rel)
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

	// Single-output: --out takes precedence over the library's output_file:
	if *out != "" {
		return os.WriteFile(*out, []byte(output), 0644)
	}
	fmt.Print(output)
	return nil
}
