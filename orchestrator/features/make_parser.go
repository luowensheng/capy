package orchfeatures

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/olivierdevelops/capy/domain"
	"github.com/olivierdevelops/capy/features"
)

// MakeParser builds the outer pattern matcher. It walks the token stream and
// at each statement boundary tries each library function's compiled Elements
// in priority order; the first complete match wins.
func MakeParser() features.Parser {
	return features.Parser{
		Parse: func(toks []domain.Token, src string, lib domain.Library) (domain.Block, error) {
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
				if a, b := literalLength(fns[i]), literalLength(fns[j]); a != b {
					return a > b
				}
				// Final tiebreaker on name so candidate ordering is TOTAL
				// and deterministic. Without this, functions that tie on
				// (priority, literal-start, literal-length) kept Go's
				// randomized map-iteration order — making any keyword
				// collision a run-to-run heisenbug (missing.md §2).
				return fns[i].Name < fns[j].Name
			})
			pp := &outerP{toks: toks, fns: fns, byName: lib.Functions, types: lib.Types, srcLines: splitSourceLines(src)}
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
	// srcLines is the original source split into lines (1-indexed via
	// srcLines[line-1]). Used by parseVerbatimBody to capture a
	// `block_verbatim` body as the raw byte range — preserving blank
	// lines and comment-marker lines that produce no tokens.
	srcLines []string
	// seqDepth counts how many sequence-closed blocks (block_close_seq)
	// are currently open. Inside such a block, statements are packed on a
	// line with no newline separators (`<p>"hi"</p>`), so a flat function
	// need not be followed by a NEWLINE/EOF — leftover tokens simply
	// become the next statement. >0 relaxes that requirement in parseStmt.
	seqDepth int
}

// splitSourceLines mirrors the lexer's splitLines: normalise CRLF,
// then split on \n. Keeps line indexing aligned with token Line
// numbers.
func splitSourceLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
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
	// blockErr remembers the FIRST error from a candidate that matched its
	// header and opened a block but then failed to parse its body. We now
	// backtrack out of such failures (missing.md §1) so a flat function can
	// still match the same leading keyword. But if NO later candidate
	// matches either, this remembered error is more informative than the
	// generic "no library function matches" — so we surface it at the end.
	var blockErr error
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
				if blockErr == nil {
					blockErr = err
				}
				p.Restore(saved)
				continue
			}
			inst.Body = &body
			// Expect NEWLINE after the closing delimiter.
			if p.Peek().Kind != domain.TokNewline && p.Peek().Kind != domain.TokEOF {
				return domain.FuncCall{}, fmt.Errorf("line %d: expected newline after block close", p.Peek().Line)
			}
			p.consumeNewlines()
			return inst, nil
		}
		// Sequence-closed block: opener is followed directly by the body
		// (no required newline), terminated by the multi-token CloseSeq.
		// This is the angle-bracket HTML model — `<p>…</p>`.
		if fn.Block != nil && len(fn.Block.CloseSeq) > 0 {
			// Resolve the closing sequence: literal segments contribute
			// their pre-tokenized tokens; a ref segment substitutes the
			// source text of the opener-bound capture as a single token.
			// This is what makes the closer DEPEND ON the opener — a
			// generic `<NAME>` opener captures `name` and declares
			// `block_close_seq "</" name ">"`, so `<div>` closes only on
			// `</div>` and `<p>` only on `</p>`.
			seq, err := resolveCloseSeq(fn.Block.CloseSeq, inst.Captures)
			if err != nil {
				if blockErr == nil {
					blockErr = err
				}
				p.Restore(saved)
				continue
			}
			body, err := p.parseSeqBlock(seq, startLine)
			if err != nil {
				if blockErr == nil {
					blockErr = err
				}
				p.Restore(saved)
				continue
			}
			inst.Body = &body
			p.consumeNewlines()
			return inst, nil
		}
		// Inside a sequence-closed block, statements are not newline-
		// delimited (`<p>"a"<b>"b"</b></p>`): a flat function may be
		// followed immediately by the next statement's tokens or the
		// block's closing sequence. Outside such a block, require the
		// usual NEWLINE/EOF terminator so partial matches are rejected.
		if p.seqDepth == 0 && p.Peek().Kind != domain.TokNewline && p.Peek().Kind != domain.TokEOF {
			p.Restore(saved)
			continue
		}
		p.consumeNewlines()
		// §5 lookahead: gate the candidate on whether an indented block
		// follows. Lets a flat keyword (when_not_followed_by indent) and a
		// block keyword (when_followed_by indent) share the same literal
		// and disambiguate purely by position.
		if fn.Lookahead != nil {
			isIndent := p.Peek().Kind == domain.TokIndent
			if (fn.Lookahead.RequireIndent && !isIndent) || (fn.Lookahead.ForbidIndent && isIndent) {
				p.Restore(saved)
				continue
			}
		}
		if fn.Block != nil {
			if fn.Block.IsVerbatim {
				text, closer, err := p.parseVerbatimBody(fn.Block.Closer, startLine)
				if err != nil {
					if blockErr == nil {
						blockErr = err
					}
					p.Restore(saved)
					continue
				}
				// Stash the raw body bytes on the FuncCall via a
				// synthetic single-statement block whose VerbatimText
				// field carries the text. The renderer surfaces this
				// via `${body}` exactly like a parsed block body.
				inst.Body = &domain.Block{VerbatimText: text, IsVerbatim: true}
				inst.Closer = closer
			} else if len(fn.Block.Sections) > 0 {
				body, sections, closer, err := p.parseSectionedBody(fn.Block.Sections, fn.Block.Closer)
				if err != nil {
					if blockErr == nil {
						blockErr = err
					}
					p.Restore(saved)
					continue
				}
				inst.Body = body
				inst.Sections = sections
				inst.Closer = closer
			} else {
				body, closer, err := p.parseBlockBody(fn.Block.Closer)
				if err != nil {
					if blockErr == nil {
						blockErr = err
					}
					p.Restore(saved)
					continue
				}
				inst.Body = &body
				inst.Closer = closer
			}
		}
		return inst, nil
	}
	// A candidate matched a block opener but its body failed to parse, and
	// nothing else matched either — surface that error (it points at the
	// real problem inside the body, not the generic "no match").
	if blockErr != nil {
		return domain.FuncCall{}, blockErr
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

// resolveCloseSeq flattens a []CloseSegment into the concrete token
// sequence that closes this particular block instance. Literal segments
// contribute their fixed pre-tokenized tokens; a ref segment substitutes
// the source text of the named opener-bound capture as a single token.
func resolveCloseSeq(segs []domain.CloseSegment, caps map[string]domain.CaptureValue) ([]string, error) {
	var out []string
	for _, seg := range segs {
		if seg.Ref != "" {
			cv, ok := caps[seg.Ref]
			if !ok {
				return nil, fmt.Errorf("block_close_seq references capture %q which is not bound", seg.Ref)
			}
			out = append(out, cv.Text)
			continue
		}
		out = append(out, seg.Tokens...)
	}
	return out, nil
}

// parseSeqBlock parses a free-flowing sequence of statements until the
// exact token sequence `seq` is reached, then consumes it. NEWLINE,
// INDENT, and DEDENT between statements are insignificant (HTML-style:
// the structure comes from the tags, not the indentation). Because each
// block carries its own closing sequence, a stray closer for a DIFFERENT
// tag fails to match here and surfaces as a parse error on the next
// statement — that is the mismatched-nesting detection HTML relies on.
func (p *outerP) parseSeqBlock(seq []string, openerLine int) (domain.Block, error) {
	p.seqDepth++
	defer func() { p.seqDepth-- }()
	closer := strings.Join(seq, "")
	var stmts []domain.FuncCall
	for {
		if p.tryConsumeSeq(seq) {
			return domain.Block{Stmts: stmts}, nil
		}
		k := p.Peek().Kind
		if k == domain.TokEOF {
			return domain.Block{}, fmt.Errorf("line %d: unexpected end of input: expected closing %q", openerLine, closer)
		}
		if k == domain.TokNewline || k == domain.TokIndent || k == domain.TokDedent {
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

// tryConsumeSeq matches the multi-token closing sequence `seq` (e.g.
// ["</","div",">"]) against the upcoming tokens and, on a match, consumes
// exactly those tokens — returning true. On no match it leaves the parser
// position untouched and returns false.
//
// Matching is token-based and whitespace-tolerant (mirroring how the
// opener's literals match), so `</p>` and `</p >` both close a `<p>`.
// The one special case is the FINAL element: the lexer greedily packs
// runs of punctuation into one token, so `</p></div>` lexes as
// `</ p ></ div >` — the closing `>` of `</p>` and the opening `</` of
// `</div>` share a single `></` token. When the last element is a strict
// prefix of such a merged punctuation run, it is matched and the leftover
// suffix is written back as a new token (with adjusted Col/Width) so it
// can open the next tag. (A mid-sequence merge cannot occur: a tag name
// is an identifier, which always breaks the punctuation run.)
func (p *outerP) tryConsumeSeq(seq []string) bool {
	i := p.pos
	splitIdx := -1
	var splitTok domain.Token
	for n, el := range seq {
		if i >= len(p.toks) {
			return false
		}
		tok := p.toks[i]
		switch tok.Kind {
		case domain.TokNewline, domain.TokIndent, domain.TokDedent, domain.TokEOF:
			return false
		}
		if tok.Text == el {
			i++
			continue
		}
		if tok.Kind == domain.TokPunct && n == len(seq)-1 &&
			len(tok.Text) > len(el) && strings.HasPrefix(tok.Text, el) {
			rem := tok
			rem.Text = tok.Text[len(el):]
			rem.Col = tok.Col + len(el)
			rem.Width = len(rem.Text)
			splitIdx = i
			splitTok = rem
			break
		}
		return false
	}
	if splitIdx >= 0 {
		p.toks[splitIdx] = splitTok
		p.pos = splitIdx
		return true
	}
	p.pos = i
	return true
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
		// Function-as-type capture (named nonterminal): the capture's
		// type names another library function. Match that function's
		// shape — possibly repeated (`type*` / `type+`) with an optional
		// separator literal — and store the matched sub-FuncCall(s).
		if el.IsFunc {
			val, err := p.captureFuncType(el)
			if err != nil {
				return domain.FuncCall{}, err
			}
			caps[el.Name] = val
			continue
		}
		// Optional capture: if the statement has ended (no value to
		// consume), bind the declared default and fill any remaining
		// optional captures with theirs. Optional args are validated
		// to be trailing, so once we hit an end-of-statement boundary
		// here every remaining element is an optional capture.
		if el.Optional && p.atStatementEnd() {
			for _, rest := range fn.Elements[i:] {
				if rest.IsCapture {
					caps[rest.Name] = defaultCapture(rest.CapType, rest.Default)
				}
			}
			break
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

// captureFuncType matches a function-as-type capture (named nonterminal).
// el.CapType names a library function; el.Repeat selects the arity:
//
//	""  exactly one occurrence
//	"+" one or more
//	"*" zero or more
//
// el.Sep, when set, is a separator literal required between successive
// occurrences. The matched sub-FuncCall(s) are returned in CaptureValue.Sub.
// A zero-progress guard prevents an infinite loop if a sub-function can
// match while consuming no tokens.
func (p *outerP) captureFuncType(el domain.PatternElement) (domain.CaptureValue, error) {
	target := p.byName[el.CapType]
	if target == nil {
		return domain.CaptureValue{}, fmt.Errorf("internal: function-typed capture %q references unknown function %q", el.Name, el.CapType)
	}
	matchOne := func() (domain.FuncCall, bool) {
		saved := p.Save()
		startPos := p.pos
		fc, err := p.tryMatch(target)
		if err != nil || p.pos == startPos {
			p.Restore(saved)
			return domain.FuncCall{}, false
		}
		fc.Func = target
		return fc, true
	}

	// Exactly-one (no repetition): a single mandatory match.
	if el.Repeat == "" {
		fc, ok := matchOne()
		if !ok {
			return domain.CaptureValue{}, fmt.Errorf("expected %s", el.CapType)
		}
		return domain.CaptureValue{Sub: []domain.FuncCall{fc}}, nil
	}

	// Repeated: gather as many occurrences as match, separated by Sep.
	var subs []domain.FuncCall
	for {
		if len(subs) > 0 && el.Sep != "" {
			// A separator is required between occurrences; if it's not
			// present, the repetition is over.
			sep := p.Save()
			if !p.matchLiteral(el.Sep) {
				p.Restore(sep)
				break
			}
			fc, ok := matchOne()
			if !ok {
				// Separator consumed but no following item — roll back the
				// separator so it isn't lost.
				p.Restore(sep)
				break
			}
			subs = append(subs, fc)
			continue
		}
		fc, ok := matchOne()
		if !ok {
			break
		}
		subs = append(subs, fc)
	}
	if el.Repeat == "+" && len(subs) == 0 {
		return domain.CaptureValue{}, fmt.Errorf("expected at least one %s", el.CapType)
	}
	return domain.CaptureValue{Sub: subs}, nil
}

// atStatementEnd reports whether the next token ends the current
// statement — a NEWLINE, EOF, or a DEDENT/closer boundary. Used to
// decide whether an optional capture should fall back to its default.
func (p *outerP) atStatementEnd() bool {
	switch p.Peek().Kind {
	case domain.TokNewline, domain.TokEOF, domain.TokDedent:
		return true
	}
	return false
}

// defaultCapture builds the CaptureValue bound when an optional arg is
// omitted. The text is stored in the SAME source-text form a real
// capture of that type would carry, so `${x}` and `${decoded x}`
// behave identically whether the arg was supplied or defaulted: a
// `string`-typed default is re-quoted (a real string capture's text
// is the quoted source form); other kinds keep the raw value.
func defaultCapture(capType, def string) domain.CaptureValue {
	if capType == "string" {
		return domain.CaptureValue{Text: strconv.Quote(def)}
	}
	return domain.CaptureValue{Text: def}
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
		// The lexer greedily merges runs of punctuation into one token, so
		// an angle-bracket tag boundary like `><` (a `>` closing one tag
		// immediately followed by the `<` opening the next) arrives as a
		// single `><` punct token. When a punct literal is a strict prefix
		// of such a merged run, match the prefix and write the suffix back
		// as the next token (adjusted Col/Width) so it can be matched next.
		if t.Kind == domain.TokPunct && len(t.Text) > len(lit) && strings.HasPrefix(t.Text, lit) {
			rem := t
			rem.Text = t.Text[len(lit):]
			rem.Col = t.Col + len(lit)
			rem.Width = len(rem.Text)
			p.toks[p.pos] = rem
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
		// values and shell-style argv.
		//
		// Quoted tokens are re-emitted WITH their quotes so a spaced,
		// quoted argument survives as one slot: `git commit -m "fix the
		// bug"` rebuilds as `commit -m "fix the bug"`, not the
		// boundary-losing `commit -m fix the bug`. Spacing is computed
		// from each token's source Width (which counts the quotes),
		// independent of how long the re-emitted text is.
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
			b.WriteString(tokenSourceText(tok))
			prevEnd = tok.Col + tokenWidth(tok)
		}
		if b.Len() == 0 {
			return domain.CaptureValue{}, fmt.Errorf("expected tail value, got end of line")
		}
		return domain.CaptureValue{Text: b.String()}, nil
	case "word":
		// A shell-style bare word: a MAXIMAL run of adjacent tokens with
		// no intervening source whitespace, joined into one value. Lets
		// `--oneline`, `-f`, `k8s/deploy.yaml`, `name=^web$`, and hyphenated
		// names like `restart-api` capture as ONE token even though the
		// lexer splits them on `-`, `/`, `=`, `.` etc. (missing.md §4).
		// Unlike `tail`, it stops at the first whitespace gap, so
		// `exec git --bare` can capture `git` as a word and leave
		// `--bare` for the next element.
		if k := p.Peek().Kind; k == domain.TokNewline || k == domain.TokEOF || k == domain.TokDedent || k == domain.TokIndent {
			return domain.CaptureValue{}, fmt.Errorf("expected word, got end of statement")
		}
		var wb strings.Builder
		prevEnd := -1
		for {
			tok := p.Peek()
			switch tok.Kind {
			case domain.TokNewline, domain.TokEOF, domain.TokDedent, domain.TokIndent:
				return domain.CaptureValue{Text: wb.String()}, nil
			}
			if prevEnd >= 0 && tok.Col > prevEnd {
				// A whitespace gap in the source ends the word.
				return domain.CaptureValue{Text: wb.String()}, nil
			}
			p.Advance()
			wb.WriteString(tok.Text)
			prevEnd = tok.Col + len(tok.Text)
		}
	case "dotted_ident":
		// A dotted identifier path: IDENT ( "." IDENT )*. The lexer treats
		// `.` as punctuation, so a bare `ident` capture stops at the first
		// segment; this type consumes the whole `err.kind` / `a.b.c` chain
		// and returns it joined with dots (missing.md §9). Requires no
		// surrounding whitespace around the dots.
		tok := p.Peek()
		if tok.Kind != domain.TokIdent {
			return domain.CaptureValue{}, fmt.Errorf("expected dotted identifier, got %q", tok.Text)
		}
		var db strings.Builder
		db.WriteString(p.Advance().Text)
		end := tok.Col + len(tok.Text)
		for p.Peek().Kind == domain.TokPunct && p.Peek().Text == "." && p.Peek().Col == end {
			dot := p.Advance()
			seg := p.Peek()
			if seg.Kind != domain.TokIdent || seg.Col != dot.Col+1 {
				return domain.CaptureValue{}, fmt.Errorf("expected identifier after %q in dotted identifier", db.String()+".")
			}
			db.WriteString(".")
			db.WriteString(p.Advance().Text)
			end = seg.Col + len(seg.Text)
		}
		return domain.CaptureValue{Text: db.String()}, nil
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

// tokenWidth returns the raw source span of a token in bytes, falling
// back to len(Text) for tokens a lexer left Width unset on (Width is
// authoritative for quoted strings, where Text has the quotes stripped).
func tokenWidth(tok domain.Token) int {
	if tok.Width > 0 {
		return tok.Width
	}
	return len(tok.Text)
}

// tokenSourceText returns a token's text the way it appeared in source.
// For string/template tokens the lexer strips the surrounding quotes from
// Text; this re-adds them so a `tail` capture preserves the slot boundary
// of a spaced, quoted argument. readString keeps inner escape sequences
// verbatim, so re-wrapping is faithful; any bare quote (possible only if
// the source used the other quote style) is escaped so the result stays
// well-formed.
func tokenSourceText(tok domain.Token) string {
	switch tok.Kind {
	case domain.TokString:
		return requote(tok.Text, '"')
	case domain.TokTemplate:
		return requote(tok.Text, '`')
	default:
		return tok.Text
	}
}

func requote(inner string, quote byte) string {
	var b strings.Builder
	b.WriteByte(quote)
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c == '\\' && i+1 < len(inner) {
			// Preserve an existing escape sequence as-is.
			b.WriteByte(c)
			b.WriteByte(inner[i+1])
			i++
			continue
		}
		if c == quote {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	b.WriteByte(quote)
	return b.String()
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
func (p *outerP) parseVerbatimBody(closerName string, openerLine int) (string, *domain.FuncCall, error) {
	if p.Peek().Kind != domain.TokIndent {
		return "", nil, fmt.Errorf("line %d: expected indented block (verbatim, closer=%s)", p.Peek().Line, closerName)
	}
	p.Advance()

	// Walk tokens to advance the parser position past the body and find
	// the closing DEDENT's line. Track INDENT/DEDENT balance so nested
	// indentation doesn't end the block early. We DON'T reconstruct
	// text from these tokens — the body is sliced from the raw source
	// (below) so blank lines and comment-marker lines, which produce no
	// tokens, are preserved byte-for-byte.
	depth := 0
	closerLine := -1
	for {
		switch p.Peek().Kind {
		case domain.TokEOF:
			return "", nil, fmt.Errorf("line %d: unterminated verbatim block, expected closer %q", p.Peek().Line, closerName)
		case domain.TokIndent:
			depth++
			p.Advance()
			continue
		case domain.TokDedent:
			if depth == 0 {
				closerLine = p.Peek().Line
				p.Advance()
				goto done
			}
			depth--
			p.Advance()
			continue
		}
		p.Advance()
	}
done:

	text := p.verbatimSlice(openerLine, closerLine)

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

// verbatimSlice returns the raw source lines strictly between the
// opener line and the closer line (both 1-indexed), dedented by the
// smallest leading-whitespace run across non-blank body lines so the
// captured text starts flush-left. Because it reads the original
// source — not the token stream — blank lines and comment-marker
// lines are preserved exactly (missing2.md §6).
func (p *outerP) verbatimSlice(openerLine, closerLine int) string {
	n := len(p.srcLines)
	if closerLine <= 0 || closerLine > n+1 {
		// Defensive: trailing-EOF dedent has no line; treat the rest
		// of the source as the body.
		closerLine = n + 1
	}
	// Body = 1-indexed lines (openerLine+1 .. closerLine-1)
	//      = 0-indexed slice [openerLine : closerLine-1].
	start, end := openerLine, closerLine-1
	if start < 0 {
		start = 0
	}
	if end > n {
		end = n
	}
	if start >= end {
		return ""
	}
	body := p.srcLines[start:end]

	minIndent := -1
	for _, ln := range body {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		i := 0
		for i < len(ln) && (ln[i] == ' ' || ln[i] == '\t') {
			i++
		}
		if minIndent < 0 || i < minIndent {
			minIndent = i
		}
	}
	if minIndent < 0 {
		minIndent = 0
	}
	out := make([]string, len(body))
	for i, ln := range body {
		if len(ln) >= minIndent {
			out[i] = ln[minIndent:]
		} else {
			out[i] = ln
		}
	}
	return strings.Join(out, "\n") + "\n"
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

// parseIndentedRegion consumes one INDENT…DEDENT region as a nested
// program. The caller must have already verified the next token is an
// INDENT. Used by sectioned blocks for the main body and each section's
// sub-body.
func (p *outerP) parseIndentedRegion() (*domain.Block, error) {
	if p.Peek().Kind != domain.TokIndent {
		return nil, fmt.Errorf("line %d: expected indented body", p.Peek().Line)
	}
	p.Advance()
	body, err := p.parseProgram(true, "")
	if err != nil {
		return nil, err
	}
	if p.Peek().Kind != domain.TokDedent {
		return nil, fmt.Errorf("line %d: expected end of indented body", p.Peek().Line)
	}
	p.Advance()
	return &body, nil
}

// parseSectionedBody parses a multi-section block (missing.md §8):
//
//	try
//	    <main body>
//	rescue
//	    <rescue body>
//	finally
//	    <finally body>
//	end
//
// The opener's main body (if any) comes first as an indented region, then
// zero or more interior section keywords (each at the opener's indent,
// each introducing its own indented sub-body), then the closer keyword.
// Sections may appear in any order and any subset; each may appear at most
// once. Returns the main body, a map of section keyword → sub-body, and
// the matched closer FuncCall.
func (p *outerP) parseSectionedBody(sections []string, closerName string) (*domain.Block, map[string]*domain.Block, *domain.FuncCall, error) {
	isSection := make(map[string]bool, len(sections))
	for _, s := range sections {
		isSection[s] = true
	}

	var mainBody *domain.Block
	if p.Peek().Kind == domain.TokIndent {
		b, err := p.parseIndentedRegion()
		if err != nil {
			return nil, nil, nil, err
		}
		mainBody = b
	}

	secBodies := map[string]*domain.Block{}
	for {
		tok := p.Peek()
		if tok.Kind == domain.TokIdent && isSection[tok.Text] {
			name := tok.Text
			if _, dup := secBodies[name]; dup {
				return nil, nil, nil, fmt.Errorf("line %d: duplicate section %q", tok.Line, name)
			}
			p.Advance()
			if p.Peek().Kind != domain.TokNewline {
				return nil, nil, nil, fmt.Errorf("line %d: section %q must be alone on its line", tok.Line, name)
			}
			p.consumeNewlines()
			if p.Peek().Kind == domain.TokIndent {
				b, err := p.parseIndentedRegion()
				if err != nil {
					return nil, nil, nil, err
				}
				secBodies[name] = b
			} else {
				secBodies[name] = &domain.Block{}
			}
			continue
		}
		// Not a section — must be the closer.
		cp, ok := p.byName[closerName]
		if !ok {
			return nil, nil, nil, fmt.Errorf("library function %q (closer) not found", closerName)
		}
		saved := p.Save()
		closerInst, err := p.tryMatch(cp)
		if err != nil {
			p.Restore(saved)
			return nil, nil, nil, fmt.Errorf("line %d: expected closer %q or one of sections %v", tok.Line, closerName, sections)
		}
		if p.Peek().Kind != domain.TokNewline && p.Peek().Kind != domain.TokEOF {
			return nil, nil, nil, fmt.Errorf("line %d: closer must end the line", p.Peek().Line)
		}
		p.consumeNewlines()
		return mainBody, secBodies, &closerInst, nil
	}
}
