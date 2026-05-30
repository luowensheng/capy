package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/olivierdevelops/capy/orchestrator"
)

// TestMultiFileProject guards the multi-file generation feature: one source
// + one library produces a tree of files under expected/.
func TestMultiFileProject(t *testing.T) {
	root := findSamplesRoot(t)
	dir := filepath.Join(root, "multi-file-project")
	lib := filepath.Join(dir, "lib.capy")
	script := filepath.Join(dir, "script.capy")

	_, files, err := orchestrator.RunMulti(lib, script)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected files to be populated")
	}

	for rel, got := range files {
		expectedPath := filepath.Join(dir, "expected", rel)
		want, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Errorf("missing expected file %s: %v", rel, err)
			continue
		}
		if string(want) != got {
			t.Errorf("%s drift — regenerate with:\n  capy run --out-dir expected lib.capy script.capy\n--- want ---\n%s\n--- got ---\n%s",
				rel, string(want), got)
		}
	}
}

// TestLibImports verifies the import directive merges types and functions
// from the imported library into the main one.
func TestLibImports(t *testing.T) {
	root := findSamplesRoot(t)
	dir := filepath.Join(root, "lib-composition")
	lib := filepath.Join(dir, "lib.capy")
	script := filepath.Join(dir, "script.capy")

	out, err := orchestrator.Run(lib, script)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(dir, "script.expected.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != out {
		t.Errorf("output drift:\n--- want ---\n%s\n--- got ---\n%s", string(want), out)
	}
	if !strings.Contains(out, "**Note:**") {
		t.Error("expected imported `note` function to render its callout")
	}
}
