package features

import "github.com/luowensheng/capy/domain"

// Lexer is a struct of capability functions. The orchestrator builds it.
type Lexer struct {
	Tokenize func(source string) ([]domain.Token, error)
}
