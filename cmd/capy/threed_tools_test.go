package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luowensheng/capy/orchestrator"
)

// TestThreeDToolsDemo runs every lib_*.capy in samples/3d-tools-demo against
// the shared script and verifies the output matches the committed golden.
// This guards the demo against accidental engine regressions or template
// changes that would silently break the docs.
func TestThreeDToolsDemo(t *testing.T) {
	root := findSamplesRoot(t)
	dir := filepath.Join(root, "3d-tools-demo")
	script := filepath.Join(dir, "script.capy")

	for _, tool := range []string{"blender", "sketchup", "rhino", "unity", "unreal"} {
		tool := tool
		t.Run(tool, func(t *testing.T) {
			lib := filepath.Join(dir, "lib_"+tool+".capy")
			gold := filepath.Join(dir, "lib_"+tool+".expected.txt")

			got, err := orchestrator.Run(lib, script)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			want, err := os.ReadFile(gold)
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			if string(want) != got {
				t.Errorf("output mismatch — regenerate with: capy run %s %s > %s",
					filepath.Base(lib), filepath.Base(script), filepath.Base(gold))
			}
		})
	}
}
