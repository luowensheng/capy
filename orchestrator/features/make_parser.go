package orchfeatures

import (
	"fmt"
	"sort"
	"strings"

	"github.com/luowensheng/capy/domain"
	"github.com/luowensheng/capy/features"
)

// MakeParser builds the outer pattern matcher. It walks the token stream and
// at each statement boundary tries each library function's compiled Elements
// in priority order; the first complete match wins.
func MakeParser() features.Parser {
	return features.Parser{
		Parse: func(toks []domain.Token, lib domain.Library) (domain.Block, error) {
			fns := make([]*domain.FuncDef, 0, len(lib.Functions))
			for _, f := range lib.Functions {
				fns = append(fns, f)
			}
			sort.SliceStable(fns, func(i, j int) bool {
				if fns[i].Priority != fns[j].Priority {
					return fns[i].Priority > fns[j].Priority
				}
				li := startsWithLiteral(fns[i])
				lj := startsWithLiteral(fns[j])
				if li != lj {
					return li
				}
				return literalLength(fns[i]) > literalLength(fns[j])
			})
			pp := &outerP{toks: toks, fns: fns, byName: lib.Functions, types: lib.Types}
			return pp.parseProgram(false, "")
		},
	}
}

func startsWithLiteral(p *domain.FuncDef) bool {
	return len(p.Elements) > 0 && !p.Elements[0].IsCapture
}
func literalLength(p *domain.FuncDef) int {
	n := 0
	for _, e := range p.Elements {
		if !e.IsCapture {
			n++
		} else {
			break
		}
	}
	return n
}

type outerP struct {
	toks   []domain.Token
	pos    int
	fns    []*domain.FuncDef
	byName map[string]*domain.FuncDef
	types  map[string]domain.TypeDef
}

func (p *outerP) Peek() domain.Token    { return p.toks[p.pos] }
func (p *outerP) Advance() domain.Token { t := p.toks[p.pos]; p.pos++; return t }
func (p *outerP) Save() int             { return p.pos }
func (p *outerP) Restore(s int)         { p.pos = s }

func (p *outerP) parseProgram(inBlock bool, closerName string) (domain.Block, error) {
	var stmts []domain.FuncCall
	for {
		k := p.Peek().Kind
		if k == domain.TokEOF {
			break
		}
		if k == domain.TokNewline {
			p.Advance()
			continue
		}
		if k == domain.TokDedent {
			break
		}
		if inBlock && p.atCloser(closerName) {
			break
		}
		s, err := p.parseStmt()
		if err != nil {
			return domain.Block{}, err
		}
		stmts = append(stmts, s)
	}
	return domain.Block{Stmts: stmts}, nil
}

func (p *outerP) atCloser(name string) bool {
	if name == "" {
		return false
	}
	cp, ok := p.byName[name]
	if !ok {
		return false
	}
	saved := p.Save()
	_, err := p.tryMatch(cp)
	p.Restore(saved)
	return err == nil
}

func (p *outerP) parseStmt() (domain.FuncCall, error) {
	startTok := p.Peek()
	startLine := startTok.Line
	startCol := startTok.Col
	for _, fn := range p.fns {
		saved := p.Save()
		inst, err := p.tryMatch(fn)
		if err != nil {
			p.Restore(saved)
			continue
		}
		// Delimiter-mode block: opener is followed directly by `open` (no newline).
		// Named-closer block: opener is followed by NEWLINE then INDENT.
		// Non-block: opener is followed by NEWLINE.
		if fn.Block != nil && fn.Block.Open != "" {
			// Expect the open delimiter on the same line.
			if !(p.matchesDelim(fn.Block.Open)) {
				p.Restore(saved)
				continue
			}
			body, err := p.parseDelimBlock(fn.Block.Open, fn.Block.Close)
			if err != nil {
				return domain.FuncCall{}, err
			}
			inst.Body = &body
			// Expect NEWLINE after the closing delimiter.
			if p.Peek().Kind != domain.TokNewline && p.Peek().Kind != domain.TokEOF {
				return domain.FuncCall{}, fmt.Errorf("line %d: expected newline after block close", p.Peek().Line)
			}
			p.consumeNewlines()
			return inst, nil
		}
		if p.Peek().Kind != domain.TokNewline && p.Peek().Kind != domain.TokEOF {
			p.Restore(saved)
			continue
		}
		p.consumeNewlines()
		if fn.Block != nil {
			body, closer, err := p.parseBlockBody(fn.Block.Closer)
			if err != nil {
				return domain.FuncCall{}, err
			}
			inst.Body = &body
			inst.Closer = closer
		}
		return inst, nil
	}
	// Build a "did you mean…?" hint: the closest literal-starting
	// function name to the unrecognized token.
	err := domain.NewError(startLine, startCol, "no library function matches token %q", startTok.Text)
	literals := make([]string, 0, len(p.fns))
	for _, fn := range p.fns {
		if len(fn.Elements) > 0 && !fn.Elements[0].IsCapture {
			literals = append(literals, fn.Elements[0].Literal)
		}
	}
	if best := domain.SuggestClosest(startTok.Text, literals, 2); best != "" {
		err.Hint = fmt.Sprintf("did you mean %q?", best)
	} else if len(literals) > 0 {
		// Show what IS valid as a fallback hint.
		shown := literals
		if len(shown) > 6 {
			shown = shown[:6]
		}
		err.Hint = fmt.Sprintf("library functions start with one of: %s", strings.Join(shown, ", "))
	}
	return domain.FuncCall{}, err
}

// matchesDelim peeks-and-consumes a single-token delimiter (typically `{`).
func (p *outerP) matchesDelim(d string) bool {
	t := p.Peek()
	if t.Text == d {
		p.Advance()
		return true
	}
	return false
}

// parseDelimBlock parses statements until `close` is reached, then consumes it.
// Newlines inside are statement boundaries.
func (p *outerP) parseDelimBlock(open, close string) (domain.Block, error) {
	p.consumeNewlines()
	var stmts []domain.FuncCall
	for {
		if p.Peek().Text == close {
			p.Advance()
			return domain.Block{Stmts: stmts}, nil
		}
		if p.Peek().Kind == domain.TokEOF {
			return domain.Block{}, fmt.Errorf("unexpected EOF inside %q...%q block", open, close)
		}
		if p.Peek().Kind == domain.TokNewline {
			p.Advance()
			continue
		}
		s, err := p.parseStmt()
		if err != nil {
			return domain.Block{}, err
		}
		stmts = append(stmts, s)
	}
}

func (p *outerP) consumeNewlines() {
	for p.Peek().Kind == domain.TokNewline {
		p.Advance()
	}
}

func (p *outerP) tryMatch(fn *domain.FuncDef) (domain.FuncCall, error) {
	caps := map[string]domain.CaptureValue{}
	for i, el := range fn.Elements {
		// Auto-skip optional comma between consecutive captures.
		if i > 0 && el.IsCapture && fn.Elements[i-1].IsCapture &&
			p.Peek().Kind == domain.TokPunct && p.Peek().Text == "," {
			p.Advance()
		}
		if !el.IsCapture {
			if !p.matchLiteral(el.Literal) {
				return domain.FuncCall{}, fmt.Errorf("expected %q", el.Literal)
			}
			continue
		}
		stop := nextLiterals(fn.Elements[i+1:])
		val, err := p.captureValue(el.CapType, stop)
		if err != nil {
			return domain.FuncCall{}, err
		}
		caps[el.Name] = val
	}
	return domain.FuncCall{Func: fn, Captures: caps}, nil
}

func nextLiterals(rest []domain.PatternElement) []string {
	var out []string
	for _, e := range rest {
		if !e.IsCapture {
			out = append(out, e.Literal)
		} else {
			break
		}
	}
	return out
}

func (p *outerP) matchLiteral(lit string) bool {
	t := p.Peek()
	switch t.Kind {
	case domain.TokIdent, domain.TokPunct,
		domain.TokLParen, domain.TokRParen,
		domain.TokLBrace, domain.TokRBrace,
		domain.TokLBrack, domain.TokRBrack:
		if t.Text == lit {
			p.Advance()
			return true
		}
	}
	return false
}

func (p *outerP) captureValue(t string, stop []string) (domain.CaptureValue, error) {
	switch t {
	case "ident":
		tok := p.Peek()
		if tok.Kind != domain.TokIdent {
			return domain.CaptureValue{}, fmt.Errorf("expected identifier, got %q", tok.Text)
		}
		p.Advance()
		return domain.CaptureValue{Text: tok.Text}, nil
	case "raw":
		tok := p.Peek()
		if tok.Kind == domain.TokIdent || tok.Kind == domain.TokString {
			p.Advance()
			return domain.CaptureValue{Text: tok.Text}, nil
		}
		return domain.CaptureValue{}, fmt.Errorf("expected raw token, got %q", tok.Text)
	}
	// Library-defined types. If the type declares a `base:` (e.g. base int),
	// recursively dispatch to the base type's token-capture rules — that's
	// what users expect when they write `type Port { base: int }` and then
	// `port 8443`. Without a base, behave like `raw`: accept one ident-or-
	// string token and defer all validation to evaluation time.
	if td, isLibType := p.types[t]; isLibType {
		if td.Base != "" && td.Base != "any" {
			return p.captureValue(td.Base, stop)
		}
		tok := p.Peek()
		if tok.Kind == domain.TokIdent || tok.Kind == domain.TokString || tok.Kind == domain.TokNumber {
			p.Advance()
			return domain.CaptureValue{Text: tok.Text}, nil
		}
		return domain.CaptureValue{}, fmt.Errorf("expected ident, string, or number for type %q, got %q", t, tok.Text)
	}
	// Built-in value kinds: parse a value expression (with comparison tail).
	// We store BOTH the parsed Expr (in case future inner-DSL primitives need
	// structured access) AND the source-text rendering — that's what shows up
	// in templates so a `cond:any` in `if x > 0` emits the literal text "x > 0".
	x, err := parseValue(p, stop)
	if err != nil {
		return domain.CaptureValue{}, err
	}
	return domain.CaptureValue{IsExpr: true, Expr: x, Text: ExprToText(x)}, nil
}

func (p *outerP) parseBlockBody(closerName string) (domain.Block, *domain.FuncCall, error) {
	if p.Peek().Kind != domain.TokIndent {
		return domain.Block{}, nil, fmt.Errorf("line %d: expected indented block (closer=%s)", p.Peek().Line, closerName)
	}
	p.Advance()

	body, err := p.parseProgram(true, closerName)
	if err != nil {
		return domain.Block{}, nil, err
	}
	if p.Peek().Kind != domain.TokDedent {
		return domain.Block{}, nil, fmt.Errorf("line %d: expected end of block", p.Peek().Line)
	}
	p.Advance()
	cp, ok := p.byName[closerName]
	if !ok {
		return domain.Block{}, nil, fmt.Errorf("library function %q (closer) not found", closerName)
	}
	saved := p.Save()
	closerInst, err := p.tryMatch(cp)
	if err != nil {
		p.Restore(saved)
		return domain.Block{}, nil, fmt.Errorf("line %d: expected closer %q", p.Peek().Line, closerName)
	}
	if p.Peek().Kind != domain.TokNewline && p.Peek().Kind != domain.TokEOF {
		return domain.Block{}, nil, fmt.Errorf("line %d: closer must end the line", p.Peek().Line)
	}
	p.consumeNewlines()
	return body, &closerInst, nil
}
