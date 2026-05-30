package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/olivierdevelops/capy/domain"
	orchfeatures "github.com/olivierdevelops/capy/orchestrator/features"
)

// `capy docs <library>` parses the library and emits Markdown reference
// documentation listing every function, type, and the arg descriptions
// authors attached via `description "..."` directives.
func cmdDocs(args []string) error {
	fs := flag.NewFlagSet("docs", flag.ContinueOnError)
	out := fs.String("out", "", "write to this path instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	pos := fs.Args()
	if len(pos) != 1 {
		return fmt.Errorf("usage: capy docs [--out path] <library>")
	}
	libPath := pos[0]
	if _, err := os.Stat(libPath); err != nil {
		return err
	}

	loader := orchfeatures.MakeLibraryLoader(orchfeatures.MakeLexer().Tokenize)
	lib, err := loader.Load(libPath)
	if err != nil {
		return err
	}
	md := domain.RenderLibraryDocs(lib)

	if *out != "" {
		return os.WriteFile(*out, []byte(md), 0644)
	}
	fmt.Print(md)
	return nil
}
