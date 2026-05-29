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
	// strayIndents counts INDENT tokens that appeared mid-body without a
	// block-opener directive in front of them — i.e. the user nested
	// content deeper than the block's anchor purely for visual styling.
	// Each stray INDENT must be paired with a matching DEDENT before the
	// real body-closing DEDENT is reached. If the matching DEDENT is
	// immediately followed by the body's closer keyword, that closer is
	// also part of the cosmetic nesting and is consumed as a no-op.
	strayIndents := 0
	for {
		k := p.Peek().Kind
		if k == domain.TokEOF {
			break
		}
		if k == domain.TokNewline {
			p.Advance()
			continue
		}
		if k == domain.TokIndent {
			// A stray indent — content deeper than the surrounding block
			// without a real opener. Treat as a no-op so user-side
			// indentation is purely cosmetic.
			p.Advance()
			strayIndents++
			continue
		}
		if k == domain.TokDedent {
			if strayIndents > 0 {
				p.Advance()
				strayIndents--
				// If the user mirrored their stray INDENT with a stray
				// `end` keyword, consume it (and its newline) so the
				// real block closer is reached.
				if inBlock && p.atCloser(closerName) {
					cp := p.byName[closerName]
					_, _ = p.tryMatch(cp)
					p.consumeNewlines()
				}
				continue
			}
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
		// Stamp the statement's source position so templates can read
		// it via the `line` / `col` render locals.
		inst.Line = startLine
		inst.Col = startCol
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
			if fn.Block.IsVerbatim {
				text, closer, err := p.parseVerbatimBody(fn.Block.Closer)
				if err != nil {
					return domain.FuncCall{}, err
				}
				// Stash the raw body bytes on the FuncCall via a
				// synthetic single-statement block whose VerbatimText
				// field carries the text. The renderer surfaces this
				// via `${body}` exactly like a parsed block body.
				inst.Body = &domain.Block{VerbatimText: text, IsVerbatim: true}
				inst.Closer = closer
			} else {
				body, closer, err := p.parseBlockBody(fn.Block.Closer)
				if err != nil {
					return domain.FuncCall{}, err
				}
				inst.Body = &body
				inst.Closer = closer
			}
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
	case "tail":
		// Capture every remaining token on the current statement as a
		// single source-text string, reconstructed using the original
		// column positions so `20px` (no source whitespace) stays
		// joined as `20px` while `1px solid red` keeps its spaces.
		// Useful for free-form trailing values like CSS property
		// values.
		var b strings.Builder
		prevEnd := -1
		for {
			k := p.Peek().Kind
			if k == domain.TokNewline || k == domain.TokEOF || k == domain.TokDedent || k == domain.TokIndent {
				break
			}
			tok := p.Advance()
			if prevEnd >= 0 && tok.Col > prevEnd {
				// Preserve only as many spaces as appeared in source.
				b.WriteString(strings.Repeat(" ", tok.Col-prevEnd))
			}
			b.WriteString(tok.Text)
			prevEnd = tok.Col + len(tok.Text)
		}
		if b.Len() == 0 {
			return domain.CaptureValue{}, fmt.Errorf("expected tail value, got end of line")
		}
		return domain.CaptureValue{Text: b.String()}, nil
	}
	// Library-defined types. Three sub-cases:
	//   * `group_open`/`group_close` set → delimited capture: walk
	//     tokens between the open and close delimiters (with balanced
	//     nesting) and return the joined source-form text.
	//   * `base:` set → dispatch to the base type's token-capture
	//     rules (lets `type Port { base: int }` accept `port 8443`).
	//   * Otherwise → behave like `raw`: accept one ident / string /
	//     number token and defer all validation to evaluation time.
	if td, isLibType := p.types[t]; isLibType {
		if td.GroupOpen != "" {
			return p.captureGroup(td)
		}
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

// captureGroup walks tokens between a group type's open and close
// delimiters, joining the in-between source text with column-
// preserved spacing (same algorithm as `tail`). Balanced nesting:
// each occurrence of `GroupOpen` increments depth, each
// `GroupClose` decrements; the capture only terminates when depth
// returns to zero. Supports multi-line groups — the scanner just
// keeps walking past NEWLINE / INDENT / DEDENT until it finds the
// matching close.
func (p *outerP) captureGroup(td domain.TypeDef) (domain.CaptureValue, error) {
	if !p.consumeLiteral(td.GroupOpen) {
		return domain.CaptureValue{}, fmt.Errorf("expected group open %q, got %q", td.GroupOpen, p.Peek().Text)
	}
	var b strings.Builder
	depth := 1
	prevEnd := -1
	prevLine := -1
	for {
		tok := p.Peek()
		switch tok.Kind {
		case domain.TokEOF:
			return domain.CaptureValue{}, fmt.Errorf("unterminated group: expected %q before end of input", td.GroupClose)
		case domain.TokNewline:
			// Multi-line groups: keep newlines as literal newline
			// characters in the captured text. Reset intra-line state.
			b.WriteByte('\n')
			p.Advance()
			prevEnd = -1
			continue
		case domain.TokIndent, domain.TokDedent:
			// These structural tokens contribute no characters to
			// the captured text; their effect is encoded in the next
			// token's Col on the new line.
			p.Advance()
			continue
		}
		if tok.Text == td.GroupClose {
			depth--
			if depth == 0 {
				p.Advance()
				return domain.CaptureValue{Text: b.String()}, nil
			}
			// A nested close: include it in the captured text.
		} else if tok.Text == td.GroupOpen {
			depth++
		}
		// Inter-token spacing: if we're still on the same line as
		// the previous token, pad to the source column. After a
		// newline reset prevEnd so we start fresh at column 0.
		if prevLine == tok.Line && prevEnd >= 0 && tok.Col > prevEnd {
			b.WriteString(strings.Repeat(" ", tok.Col-prevEnd))
		} else if prevLine != tok.Line && tok.Col > 1 && b.Len() > 0 {
			// Cross-line: leading indent on a continuation line.
			b.WriteString(strings.Repeat(" ", tok.Col-1))
		}
		text := tokenSourceForm(tok)
		b.WriteString(text)
		prevEnd = tok.Col + len(text)
		prevLine = tok.Line
		p.Advance()
	}
}

// consumeLiteral peeks at the current token and, if its Text matches
// `lit` (over the token kinds the outer parser considers literal-
// matchable), advances. Mirrors the existing matchLiteral but is
// also tolerant of TokString/TokNumber whose text might appear as
// a group delimiter in unusual DSLs.
func (p *outerP) consumeLiteral(lit string) bool {
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

// parseVerbatimBody captures the body of a `block_verbatim`-declared
// function as raw source text. INDENT-balanced nested blocks are
// counted (so `pre … { something … } … end` works) but the body is
// NEVER re-parsed against the library's functions — every token
// between the opener and the matching closer is reconstructed back
// into source-form text using column-position arithmetic, the same
// way the `tail` capture rebuilds free-form values.
//
// The indent baseline is the leading column of the first body line;
// every line's leading whitespace is shifted left by that amount so
// the captured body starts at column 1. This makes block-verbatim
// pleasant for code blocks: the user indents `pre go … end` four
// spaces in their script and the captured body still starts at
// column 1, ready for whatever the library's template does with it.
func (p *outerP) parseVerbatimBody(closerName string) (string, *domain.FuncCall, error) {
	if p.Peek().Kind != domain.TokIndent {
		return "", nil, fmt.Errorf("line %d: expected indented block (verbatim, closer=%s)", p.Peek().Line, closerName)
	}
	p.Advance()

	// Collect every body token. Track INDENT/DEDENT balance so nested
	// indentation doesn't terminate the verbatim block early.
	var bodyToks []domain.Token
	depth := 0
	for {
		k := p.Peek().Kind
		switch k {
		case domain.TokEOF:
			return "", nil, fmt.Errorf("line %d: unterminated verbatim block, expected closer %q", p.Peek().Line, closerName)
		case domain.TokIndent:
			depth++
			bodyToks = append(bodyToks, p.Advance())
			continue
		case domain.TokDedent:
			if depth == 0 {
				// This is OUR closing DEDENT — body ends here.
				p.Advance()
				goto done
			}
			depth--
			bodyToks = append(bodyToks, p.Advance())
			continue
		}
		bodyToks = append(bodyToks, p.Advance())
	}
done:

	text := reconstructVerbatim(bodyToks)

	cp, ok := p.byName[closerName]
	if !ok {
		return "", nil, fmt.Errorf("library function %q (verbatim closer) not found", closerName)
	}
	saved := p.Save()
	closerInst, err := p.tryMatch(cp)
	if err != nil {
		p.Restore(saved)
		return "", nil, fmt.Errorf("line %d: expected closer %q", p.Peek().Line, closerName)
	}
	if p.Peek().Kind != domain.TokNewline && p.Peek().Kind != domain.TokEOF {
		return "", nil, fmt.Errorf("line %d: closer must end the line", p.Peek().Line)
	}
	p.consumeNewlines()
	return text, &closerInst, nil
}

// reconstructVerbatim rebuilds source-like text from a token slice
// captured by `parseVerbatimBody`. With the lexer's source-absolute
// `Col` tracking (see startCol in tokenizeWith), the algorithm is
// straightforward: find the minimum first-token column across body
// lines (= the body's indent baseline) and shift every line left by
// that amount so the captured text starts at column 1. Within a
// line, gap arithmetic between consecutive tokens uses the same
// column-preserved spacing logic that `tail` already relies on.
// INDENT/DEDENT tokens carry no visible text and are skipped — col
// values already encode the right whitespace.
func reconstructVerbatim(toks []domain.Token) string {
	if len(toks) == 0 {
		return ""
	}
	// Baseline: the leftmost source column of any text-bearing
	// token. We shift every line left by (baseline - 1) so the
	// captured body starts at column 1.
	baseline := -1
	for _, t := range toks {
		switch t.Kind {
		case domain.TokNewline, domain.TokIndent, domain.TokDedent:
			continue
		}
		if baseline < 0 || t.Col < baseline {
			baseline = t.Col
		}
	}
	if baseline < 1 {
		baseline = 1
	}

	var lines []string
	var cur strings.Builder
	onLineStart := true
	prevEnd := -1

	flushLine := func() {
		lines = append(lines, cur.String())
		cur.Reset()
		onLineStart = true
		prevEnd = -1
	}

	for _, t := range toks {
		switch t.Kind {
		case domain.TokIndent, domain.TokDedent:
			continue
		case domain.TokNewline:
			flushLine()
			continue
		}
		// Effective column after shifting the body to start at col 1.
		effCol := t.Col - (baseline - 1)
		if effCol < 1 {
			effCol = 1
		}
		if onLineStart {
			if effCol > 1 {
				cur.WriteString(strings.Repeat(" ", effCol-1))
			}
			onLineStart = false
		} else if effCol > prevEnd {
			cur.WriteString(strings.Repeat(" ", effCol-prevEnd))
		}
		text := tokenSourceForm(t)
		cur.WriteString(text)
		prevEnd = effCol + len(text)
	}
	if !onLineStart {
		flushLine()
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n") + "\n"
}

// tokenSourceForm reproduces a token's source-text representation.
// For most kinds the Text field is exact; for strings we re-add the
// quote characters that the lexer stripped.
func tokenSourceForm(t domain.Token) string {
	switch t.Kind {
	case domain.TokString:
		return "\"" + t.Text + "\""
	case domain.TokTemplate:
		return "`" + t.Text + "`"
	}
	return t.Text
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
	// Dedent-only block: no closer keyword to match. Useful for
	// CSS-style selectors and other DSLs where a body is delimited
	// purely by indentation.
	if closerName == "" {
		return body, nil, nil
	}
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
