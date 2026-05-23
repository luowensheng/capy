package features

import "github.com/luowensheng/capy/domain"

// The evaluator is a TRANSPILER driver. It walks the parsed program,
// rendering each function's template into the body and mutating the live
// context via each function's run snippet. Then it renders the file template
// with the accumulated context+body to produce the final output.
type Evaluator struct {
	Run func(program domain.Block, lib domain.Library) (string, error)
	// RunMulti returns BOTH the file_template-rendered single output AND
	// a map of every `file "path":` template rendered against the same
	// final context+body. Used for multi-file project generation.
	RunMulti func(program domain.Block, lib domain.Library) (string, map[string]string, error)
}
