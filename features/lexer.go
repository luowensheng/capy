package features

import "github.com/olivierdevelops/capy/domain"

// Lexer is a struct of capability functions. The orchestrator builds it.
type Lexer struct {
	// Tokenize lexes a source string using the engine-default
	// comment markers (`#`). Used for the manifest format and
	// inner-DSL run/command bodies, which are Capy's own config
	// syntax.
	Tokenize func(source string) ([]domain.Token, error)

	// TokenizeWith lexes a source string with the supplied list
	// of line-comment markers. Used for USER SCRIPTS, where the
	// library's `comments` declaration controls what (if anything)
	// counts as a comment. An empty markers list means no comment
	// syntax — `#`, `//`, etc. flow through as ordinary tokens.
	TokenizeWith func(source string, commentMarkers []string) ([]domain.Token, error)
}
