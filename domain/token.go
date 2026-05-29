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
	// Width is the raw byte width of the lexeme as it appeared in source,
	// including any surrounding quotes for string/template tokens. Text
	// for a string strips the quotes, so Width (not len(Text)) is the
	// authoritative source span — `tail` uses it to compute inter-token
	// spacing and to know a token was quoted. Zero means "unset"; consumers
	// should fall back to len(Text).
	Width int
}
