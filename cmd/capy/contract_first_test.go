package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/olivierdevelops/capy/orchestrator"
)

// TestContractFirstAPI guards the grammar-as-contract demo. One source
// (script.capy) feeds three libraries (openapi/typescript/markdown);
// every output is golden-tested. If any library drifts, CI fails.
func TestContractFirstAPI(t *testing.T) {
	root := findSamplesRoot(t)
	dir := filepath.Join(root, "contract-first-api")
	script := filepath.Join(dir, "script.capy")

	for _, target := range []string{"openapi", "typescript", "markdown"} {
		target := target
		t.Run(target, func(t *testing.T) {
			lib := filepath.Join(dir, "lib_"+target+".capy")
			gold := filepath.Join(dir, "script_"+target+".expected.txt")

			got, err := orchestrator.Run(lib, script)
			if err != nil {
				t.Fatalf("run %s: %v", target, err)
			}
			want, err := os.ReadFile(gold)
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			if string(want) != got {
				t.Errorf("output drift for %s — regenerate with:\n  capy run %s %s > %s",
					target, filepath.Base(lib), filepath.Base(script), filepath.Base(gold))
			}
		})
	}
}
