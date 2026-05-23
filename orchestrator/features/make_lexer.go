package orchfeatures

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/luowensheng/capy/domain"
	"github.com/luowensheng/capy/features"
)

// The lexer is purely lexical: it identifies words, numbers, strings, brackets,
// punctuation, and indentation. It does NOT classify any word as a keyword —
// "if", "loop", "end", "true", etc. are all just TokIdent. The library's
// patterns decide what those words mean. This is the core of "0 default grammar".
func MakeLexer() features.Lexer {
	return features.Lexer{Tokenize: tokenize}
}

func tokenize(source string) ([]domain.Token, error) {
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
			if rest == "" || strings.HasPrefix(rest, "#") {
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

		newToks, openDelta, err := tokenizeLine(line, li+1)
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

func tokenizeLine(line string, lineNo int) ([]domain.Token, int, error) {
	var toks []domain.Token
	open := 0
	i := 0
	col := 1
	for i < len(line) {
		c := line[i]
		switch {
		case c == ' ' || c == '\t':
			i++
			col++
		case c == '#':
			i = len(line)
		case c == '"' || c == '\'':
			s, n, err := readString(line[i:], c)
			if err != nil {
				return nil, 0, fmt.Errorf("line %d: %v", lineNo, err)
			}
			toks = append(toks, domain.Token{Kind: domain.TokString, Text: s, Line: lineNo, Col: col})
			i += n
			col += n
		case c == '`':
			s, n, err := readString(line[i:], '`')
			if err != nil {
				return nil, 0, fmt.Errorf("line %d: %v", lineNo, err)
			}
			toks = append(toks, domain.Token{Kind: domain.TokTemplate, Text: s, Line: lineNo, Col: col})
			i += n
			col += n
		case c == '(':
			toks = append(toks, domain.Token{Kind: domain.TokLParen, Text: "(", Line: lineNo, Col: col})
			i++
			col++
			open++
		case c == ')':
			toks = append(toks, domain.Token{Kind: domain.TokRParen, Text: ")", Line: lineNo, Col: col})
			i++
			col++
			open--
		case c == '[':
			toks = append(toks, domain.Token{Kind: domain.TokLBrack, Text: "[", Line: lineNo, Col: col})
			i++
			col++
			open++
		case c == ']':
			toks = append(toks, domain.Token{Kind: domain.TokRBrack, Text: "]", Line: lineNo, Col: col})
			i++
			col++
			open--
		case c == '{':
			toks = append(toks, domain.Token{Kind: domain.TokLBrace, Text: "{", Line: lineNo, Col: col})
			i++
			col++
			open++
		case c == '}':
			toks = append(toks, domain.Token{Kind: domain.TokRBrace, Text: "}", Line: lineNo, Col: col})
			i++
			col++
			open--
		case unicode.IsDigit(rune(c)) || (c == '-' && i+1 < len(line) && unicode.IsDigit(rune(line[i+1])) && (i == 0 || !isPunct(line[i-1]) && line[i-1] != ' ' && line[i-1] != '\t' || true)):
			n := readNumber(line[i:])
			toks = append(toks, domain.Token{Kind: domain.TokNumber, Text: line[i : i+n], Line: lineNo, Col: col})
			i += n
			col += n
		case isIdentStart(rune(c)):
			n := readIdent(line[i:])
			text := line[i : i+n]
			toks = append(toks, domain.Token{Kind: domain.TokIdent, Text: text, Line: lineNo, Col: col})
			i += n
			col += n
		case isPunct(c):
			n := readPunct(line[i:])
			toks = append(toks, domain.Token{Kind: domain.TokPunct, Text: line[i : i+n], Line: lineNo, Col: col})
			i += n
			col += n
		default:
			return nil, 0, fmt.Errorf("line %d col %d: unexpected character %q", lineNo, col, c)
		}
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

func isIdentStart(r rune) bool { return r == '_' || unicode.IsLetter(r) }
func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func readIdent(s string) int {
	i := 0
	for i < len(s) && isIdentPart(rune(s[i])) {
		i++
	}
	return i
}
