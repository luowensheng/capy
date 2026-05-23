package domain

// Tokens are LEXICAL only. There are no grammar keywords (no if/loop/end/=).
// Identifier-like words are TokIdent. Operators (=, ==, <, > , : . , !=, <=, >=, +, -, ...)
// are TokPunct with their literal text. The library decides what any of those mean.
type TokenKind int

const (
	TokIdent TokenKind = iota
	TokNumber
	TokString   // "..." or '...' — content stored raw, supports ${} at eval time
	TokTemplate // `...` — same: content stored raw, supports ${} at eval time
	TokPunct    // = == != < > <= >= , : . + - * / etc.
	TokLParen
	TokRParen
	TokLBrace
	TokRBrace
	TokLBrack
	TokRBrack
	TokNewline
	TokIndent
	TokDedent
	TokEOF
)

type Token struct {
	Kind TokenKind
	Text string
	Line int
	Col  int
}
