package features

import "github.com/luowensheng/capy/domain"

// The evaluator is a TRANSPILER driver. It walks the parsed program,
// rendering each function's template into the body and mutating the live
// context via each function's run snippet. Then it renders the file template
// with the accumulated context+body to produce the final output.
type Evaluator struct {
	Run func(program domain.Block, lib domain.Library) (string, error)
}
