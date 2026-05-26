package orchfeatures

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/luowensheng/capy/domain"
	"github.com/luowensheng/capy/features"
)

// The lexer is purely lexical: it identifies words, numbers, strings, brackets,
// punctuation, and indentation. It does NOT classify any word as a keyword —
// "if", "loop", "end", "true", etc. are all just TokIdent. The library's
// patterns decide what those words mean. This is the core of "0 default grammar".
// defaultCommentMarkers is the marker set used when no explicit
// list is supplied. The engine has NO predefined user-script
// comment syntax — but Capy's own manifest/inner-DSL format uses
// `#`, so the no-args entry point keeps `#` to lex manifests and
// inner-DSL run bodies. User scripts go through TokenizeWith and
// pass lib.Comments instead.
var defaultCommentMarkers = []string{"#"}

func MakeLexer() features.Lexer {
	return features.Lexer{
		Tokenize:     tokenize,
		TokenizeWith: tokenizeWith,
	}
}

func tokenize(source string) ([]domain.Token, error) {
	return tokenizeWith(source, defaultCommentMarkers)
}

func tokenizeWith(source string, commentMarkers []string) ([]domain.Token, error) {
	var toks []domain.Token
	indents := []int{0}
	bracket := 0

	lines := splitLines(source)
	for li, raw := range lines {
		line := raw
		if bracket == 0 {
			indent := 0
			i := 0
			for i < len(line) {
				c := line[i]
				if c == ' ' {
					indent++
					i++
				} else if c == '\t' {
					indent += 4
					i++
				} else {
					break
				}
			}
			rest := strings.TrimSpace(line[i:])
			if rest == "" || hasCommentPrefix(rest, commentMarkers) {
				continue
			}
			if indent%4 != 0 {
				return nil, fmt.Errorf("line %d: indentation must be 4 spaces or 1 tab per level", li+1)
			}
			level := indent / 4
			top := indents[len(indents)-1]
			if level > top {
				if level != top+1 {
					return nil, fmt.Errorf("line %d: unexpected indent jump", li+1)
				}
				indents = append(indents, level)
				toks = append(toks, domain.Token{Kind: domain.TokIndent, Line: li + 1})
			}
			for level < indents[len(indents)-1] {
				indents = indents[:len(indents)-1]
				toks = append(toks, domain.Token{Kind: domain.TokDedent, Line: li + 1})
			}
			line = line[i:]
		}

		newToks, openDelta, err := tokenizeLine(line, li+1, commentMarkers)
		if err != nil {
			return nil, err
		}
		toks = append(toks, newToks...)
		bracket += openDelta
		if bracket < 0 {
			return nil, fmt.Errorf("line %d: unmatched closing bracket", li+1)
		}
		// Always emit NEWLINE at end of a logical line. Inside brackets, the
		// value parsers (parseObjLit / parseListLit / paren sub-call) skip
		// them; block parsers (for `{...}` style blocks) use them as
		// statement boundaries.
		toks = append(toks, domain.Token{Kind: domain.TokNewline, Line: li + 1})
	}
	for len(indents) > 1 {
		indents = indents[:len(indents)-1]
		toks = append(toks, domain.Token{Kind: domain.TokDedent})
	}
	toks = append(toks, domain.Token{Kind: domain.TokEOF})
	return toks, nil
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

// punctChars are accepted as parts of a TokPunct token. We greedily consume
// runs of these to form multi-char operators like ==, !=, <=, >=, ->, |>.
var punctChars = "=<>!+-*/%&|^~?:,.;@$"

func isPunct(b byte) bool { return strings.IndexByte(punctChars, b) >= 0 }

// hasCommentPrefix reports whether s begins with any of the
// supplied comment markers. Empty markers list → never a comment.
func hasCommentPrefix(s string, markers []string) bool {
	for _, m := range markers {
		if m != "" && strings.HasPrefix(s, m) {
			return true
		}
	}
	return false
}

func tokenizeLine(line string, lineNo int, commentMarkers []string) ([]domain.Token, int, error) {
	var toks []domain.Token
	open := 0
	i := 0
	col := 1
	for i < len(line) {
		// Decode the next rune so non-ASCII letters / symbols / emoji
		// don't get bit-truncated to garbage. ASCII chars return w == 1
		// and the existing single-byte fast paths below stay correct.
		r, w := utf8.DecodeRuneInString(line[i:])
		switch {
		case r == ' ' || r == '\t':
			i++
			col++
		case hasCommentPrefix(line[i:], commentMarkers):
			i = len(line)
		case r == '"' || r == '\'':
			s, n, err := readString(line[i:], byte(r))
			if err != nil {
				return nil, 0, fmt.Errorf("line %d: %v", lineNo, err)
			}
			toks = append(toks, domain.Token{Kind: domain.TokString, Text: s, Line: lineNo, Col: col})
			i += n
			col += n
		case r == '`':
			s, n, err := readString(line[i:], '`')
			if err != nil {
				return nil, 0, fmt.Errorf("line %d: %v", lineNo, err)
			}
			toks = append(toks, domain.Token{Kind: domain.TokTemplate, Text: s, Line: lineNo, Col: col})
			i += n
			col += n
		case r == '(':
			toks = append(toks, domain.Token{Kind: domain.TokLParen, Text: "(", Line: lineNo, Col: col})
			i++
			col++
			open++
		case r == ')':
			toks = append(toks, domain.Token{Kind: domain.TokRParen, Text: ")", Line: lineNo, Col: col})
			i++
			col++
			open--
		case r == '[':
			toks = append(toks, domain.Token{Kind: domain.TokLBrack, Text: "[", Line: lineNo, Col: col})
			i++
			col++
			open++
		case r == ']':
			toks = append(toks, domain.Token{Kind: domain.TokRBrack, Text: "]", Line: lineNo, Col: col})
			i++
			col++
			open--
		case r == '{':
			toks = append(toks, domain.Token{Kind: domain.TokLBrace, Text: "{", Line: lineNo, Col: col})
			i++
			col++
			open++
		case r == '}':
			toks = append(toks, domain.Token{Kind: domain.TokRBrace, Text: "}", Line: lineNo, Col: col})
			i++
			col++
			open--
		case unicode.IsDigit(r) || (r == '-' && i+1 < len(line) && unicode.IsDigit(rune(line[i+1])) && (i == 0 || !isPunct(line[i-1]) && line[i-1] != ' ' && line[i-1] != '\t' || true)):
			n := readNumber(line[i:])
			toks = append(toks, domain.Token{Kind: domain.TokNumber, Text: line[i : i+n], Line: lineNo, Col: col})
			i += n
			col += n
		case isIdentStart(r):
			n := readIdent(line[i:])
			text := line[i : i+n]
			toks = append(toks, domain.Token{Kind: domain.TokIdent, Text: text, Line: lineNo, Col: col})
			i += n
			col += n
		case r < 0x80 && isPunct(byte(r)):
			n := readPunct(line[i:])
			toks = append(toks, domain.Token{Kind: domain.TokPunct, Text: line[i : i+n], Line: lineNo, Col: col})
			i += n
			col += n
		default:
			return nil, 0, fmt.Errorf("line %d col %d: unexpected character %q", lineNo, col, r)
		}
		_ = w // most arms advance by single byte anyway; ident/number/string arms compute their own width
	}
	return toks, open, nil
}

func readString(s string, quote byte) (string, int, error) {
	if s[0] != quote {
		return "", 0, fmt.Errorf("expected quote")
	}
	var b strings.Builder
	i := 1
	for i < len(s) {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			b.WriteByte(s[i])
			b.WriteByte(s[i+1])
			i += 2
			continue
		}
		if c == quote {
			return b.String(), i + 1, nil
		}
		b.WriteByte(c)
		i++
	}
	return "", 0, fmt.Errorf("unterminated string")
}

func readNumber(s string) int {
	i := 0
	if s[0] == '-' {
		i++
	}
	dot := false
	for i < len(s) {
		c := s[i]
		if c == '.' && !dot {
			// only consume a dot if the next char is a digit (avoid `a.b` confusion)
			if i+1 < len(s) && unicode.IsDigit(rune(s[i+1])) {
				dot = true
				i++
				continue
			}
			break
		}
		if !unicode.IsDigit(rune(c)) {
			break
		}
		i++
	}
	return i
}

func readPunct(s string) int {
	i := 0
	for i < len(s) && isPunct(s[i]) {
		i++
	}
	return i
}

// isIdentStart accepts ASCII letters, underscore, AND any non-ASCII
// rune. The lexer is purely lexical and Capy makes no assumptions
// about how users write content — em-dashes, smart quotes, accented
// Latin, CJK, and emoji must all flow through bare-prose positions
// without erroring. ASCII punctuation that participates in literal
// matches (`punctChars` — `=<>!+-*/%&|^~?:,.;@$`) is checked
// separately in the switch BEFORE this predicate, so we don't have
// to exclude them here.
func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || r >= 0x80
}

func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || r >= 0x80
}

// readIdent walks runes (not bytes) so a multi-byte sequence like
// `é` (0xC3 0xA9) is consumed as one rune and the returned byte
// length covers the whole encoded form.
func readIdent(s string) int {
	i := 0
	for i < len(s) {
		r, w := utf8.DecodeRuneInString(s[i:])
		if !isIdentPart(r) {
			break
		}
		i += w
	}
	return i
}
