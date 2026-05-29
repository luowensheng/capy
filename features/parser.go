package features

import "github.com/luowensheng/capy/domain"

// The outer parser is a pattern matcher driven by the library's functions.
// Each function's args list (literals + typed captures) is the entire shape.
type Parser struct {
	// Parse turns a token stream into a program. `src` is the original
	// source the tokens came from — needed so `block_verbatim` bodies
	// can capture the raw byte range (preserving blank lines and
	// comment-marker lines that produce no tokens). Pass the exact
	// string handed to the lexer.
	Parse func(toks []domain.Token, src string, lib domain.Library) (domain.Block, error)
}
