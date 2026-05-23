package features

import "github.com/luowensheng/capy/domain"

// The outer parser is a pattern matcher driven by the library's functions.
// Each function's args list (literals + typed captures) is the entire shape.
type Parser struct {
	Parse func(toks []domain.Token, lib domain.Library) (domain.Block, error)
}
