package main

import (
	"path/filepath"
	"testing"

	"github.com/luowensheng/capy/orchestrator"
)

// TestCapyLibRoundTrip verifies that the .capy-form library produces byte-
// identical output to its YAML counterpart for every multi-language-demo
// target. This is the core guarantee of the .capy library format: it's
// the same DTO, parsed differently.
func TestCapyLibRoundTrip(t *testing.T) {
	root := findSamplesRoot(t)
	dir := filepath.Join(root, "multi-language-demo")
	script := filepath.Join(dir, "script.capy")

	for _, lang := range []string{"python", "javascript", "go", "rust", "c"} {
		lang := lang
		t.Run(lang, func(t *testing.T) {
			yamlLib := filepath.Join(dir, "lib_"+lang+".yaml")
			capyLib := filepath.Join(dir, "lib_"+lang+".capy")

			yOut, err := orchestrator.Run(yamlLib, script)
			if err != nil {
				t.Fatalf("yaml run: %v", err)
			}
			cOut, err := orchestrator.Run(capyLib, script)
			if err != nil {
				t.Fatalf("capy run: %v", err)
			}
			if yOut != cOut {
				t.Errorf("outputs differ for %s\n--- yaml ---\n%s\n--- capy ---\n%s", lang, yOut, cOut)
			}
		})
	}
}
