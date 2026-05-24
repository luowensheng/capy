package domain

// ImplDef is one implementation of a library's interface. The
// library's manifest catalogues every available impl; the CLI
// selects one per invocation. The selected impl's File is what
// the loader actually reads to get the FuncDefs / file_template
// / etc.
//
// Multiple impls let the same source language target multiple
// outputs — a `chart` DSL can have impls that emit Mermaid, D3,
// or ASCII; the same source survives the swap.
type ImplDef struct {
	Name        string
	File        string // path relative to the manifest file's directory
	Description string
	Version     string
	IsDefault   bool
}
