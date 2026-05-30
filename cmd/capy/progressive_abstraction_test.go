package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/olivierdevelops/capy/orchestrator"
)

// TestProgressiveAbstraction runs the SAME library against three scripts
// (minimal / medium / full) and verifies each produces its committed
// expected output. This is the core demonstration that one library can
// expose multiple abstraction levels.
func TestProgressiveAbstraction(t *testing.T) {
	root := findSamplesRoot(t)
	dir := filepath.Join(root, "progressive-abstraction")
	lib := filepath.Join(dir, "lib.capy")
	for _, level := range []string{"minimal", "medium", "full"} {
		level := level
		t.Run(level, func(t *testing.T) {
			script := filepath.Join(dir, "script_"+level+".capy")
			out, err := orchestrator.Run(lib, script)
			if err != nil {
				t.Fatalf("run %s: %v", level, err)
			}
			want, err := os.ReadFile(filepath.Join(dir, "script_"+level+".expected.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if string(want) != out {
				t.Errorf("output drift for %s — regenerate via:\n  capy run lib.capy script_%s.capy > script_%s.expected.txt", level, level, level)
			}
		})
	}
}
