package main

import (
	"fmt"
	"os"

	orchfeatures "github.com/luowensheng/capy/orchestrator/features"
)

// `capy check <library.yaml>` parses + validates a library file without
// running any source. Useful for libraries-as-data CI.
func cmdCheck(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: capy check <library.yaml>")
	}
	path := args[0]
	if _, err := os.Stat(path); err != nil {
		return err
	}
	loader := orchfeatures.MakeLibraryLoader(orchfeatures.MakeLexer().Tokenize)
	lib, err := loader.Load(path)
	if err != nil {
		return err
	}
	fmt.Printf("ok — %d function(s), %d type(s)\n", len(lib.Functions), len(lib.Types))
	for name := range lib.Functions {
		fmt.Printf("  function %s\n", name)
	}
	for name := range lib.Types {
		fmt.Printf("  type     %s\n", name)
	}
	return nil
}
