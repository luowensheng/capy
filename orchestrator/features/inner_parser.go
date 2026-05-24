package orchfeatures

import (
	"fmt"

	"github.com/luowensheng/capy/domain"
)

// ParseInner parses an inner-DSL `run:` snippet (already lexed via the outer
// lexer) into an InnerBlock AST. The inner DSL is hardcoded — it knows fixed
// statement forms (set, append, ..., if, loop, plain call) and uses the
// shared value-expression parser.
func ParseInner(toks []domain.Token) (domain.InnerBlock, error) {
	p := &innerP{toks: toks}
	return p.parseProgram(false)
}

type innerP struct {
	toks []domain.Token
	pos  int
}

func (p *innerP) Peek() domain.Token    { return p.toks[p.pos] }
func (p *innerP) Advance() domain.Token { t := p.toks[p.pos]; p.pos++; return t }
func (p *innerP) Save() int             { return p.pos }
func (p *innerP) Restore(s int)         { p.pos = s }

func (p *innerP) parseProgram(inBlock bool) (domain.InnerBlock, error) {
	var stmts []domain.InnerStmt
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
		if inBlock && (p.atKeyword("end") || p.atKeyword("else")) {
			break
		}
		s, err := p.parseStmt()
		if err != nil {
			return domain.InnerBlock{}, err
		}
		stmts = append(stmts, s)
	}
	return domain.InnerBlock{Stmts: stmts}, nil
}

func (p *innerP) atKeyword(w string) bool {
	t := p.Peek()
	return t.Kind == domain.TokIdent && t.Text == w
}

func (p *innerP) parseStmt() (domain.InnerStmt, error) {
	t := p.Peek()
	if t.Kind != domain.TokIdent {
		return nil, fmt.Errorf("line %d: expected statement, got %q", t.Line, t.Text)
	}
	switch t.Text {
	case "set":
		p.Advance()
		path, err := p.parsePath()
		if err != nil {
			return nil, err
		}
		val, err := parseValue(p, nil)
		if err != nil {
			return nil, err
		}
		p.consumeNewline()
		return domain.SetStmt{Target: path, Value: val}, nil
	case "append":
		p.Advance()
		path, err := p.parsePath()
		if err != nil {
			return nil, err
		}
		val, err := parseValue(p, nil)
		if err != nil {
			return nil, err
		}
		p.consumeNewline()
		return domain.AppendStmt{Target: path, Value: val}, nil
	case "prepend":
		p.Advance()
		path, err := p.parsePath()
		if err != nil {
			return nil, err
		}
		val, err := parseValue(p, nil)
		if err != nil {
			return nil, err
		}
		p.consumeNewline()
		return domain.PrependStmt{Target: path, Value: val}, nil
	case "merge":
		p.Advance()
		path, err := p.parsePath()
		if err != nil {
			return nil, err
		}
		val, err := parseValue(p, nil)
		if err != nil {
			return nil, err
		}
		p.consumeNewline()
		return domain.MergeStmt{Target: path, Value: val}, nil
	case "delete":
		p.Advance()
		path, err := p.parsePath()
		if err != nil {
			return nil, err
		}
		p.consumeNewline()
		return domain.DeleteStmt{Target: path}, nil
	case "if":
		return p.parseIf()
	case "loop", "for":
		return p.parseLoop()
	case "write":
		p.Advance()
		val, err := parseValue(p, nil)
		if err != nil {
			return nil, err
		}
		p.consumeNewline()
		return domain.WriteStmt{Value: val}, nil
	case "let":
		// `let NAME = EXPR` — bind a local variable (in command bodies).
		p.Advance()
		nameTok := p.Peek()
		if nameTok.Kind != domain.TokIdent {
			return nil, fmt.Errorf("line %d: let requires an identifier name", nameTok.Line)
		}
		p.Advance()
		eq := p.Peek()
		if !(eq.Kind == domain.TokPunct && eq.Text == "=") {
			return nil, fmt.Errorf("line %d: let requires `= EXPR`", eq.Line)
		}
		p.Advance()
		val, err := parseValue(p, nil)
		if err != nil {
			return nil, err
		}
		p.consumeNewline()
		// Reuse SetStmt with a synthetic path rooted at `locals`.
		// The evaluator treats `locals.X` writes as local-scope binds.
		return domain.SetStmt{
			Target: domain.Path{Root: "locals", Steps: []domain.PathStep{{Field: nameTok.Text}}},
			Value:  val,
		}, nil
	}
	// fallthrough: a generic call (e.g. `regex_match x y`, `error "msg"`,
	// or a recursive call to another library function — though for now the
	// inner DSL only supports primitives at this position).
	return p.parseCall()
}

func (p *innerP) consumeNewline() {
	for p.Peek().Kind == domain.TokNewline {
		p.Advance()
	}
}

func (p *innerP) parsePath() (domain.Path, error) {
	t := p.Peek()
	if t.Kind != domain.TokIdent {
		return domain.Path{}, fmt.Errorf("line %d: expected path root identifier", t.Line)
	}
	p.Advance()
	path := domain.Path{Root: t.Text}
	for {
		nt := p.Peek()
		if nt.Kind == domain.TokPunct && nt.Text == "." {
			p.Advance()
			name := p.Peek()
			if name.Kind != domain.TokIdent {
				return path, fmt.Errorf("expected identifier after .")
			}
			p.Advance()
			path.Steps = append(path.Steps, domain.PathStep{Field: name.Text})
			continue
		}
		if nt.Kind == domain.TokLBrack {
			p.Advance()
			idx, err := parseValue(p, nil)
			if err != nil {
				return path, err
			}
			if p.Peek().Kind != domain.TokRBrack {
				return path, fmt.Errorf("expected ]")
			}
			p.Advance()
			path.Steps = append(path.Steps, domain.PathStep{IsIndex: true, Index: idx})
			continue
		}
		break
	}
	return path, nil
}

func (p *innerP) parseIf() (domain.InnerStmt, error) {
	p.Advance() // if
	cond, err := parseValue(p, nil)
	if err != nil {
		return nil, err
	}
	if p.Peek().Kind != domain.TokNewline {
		return nil, fmt.Errorf("line %d: expected newline after if cond", p.Peek().Line)
	}
	p.consumeNewline()
	body, err := p.parseBlockBody()
	if err != nil {
		return nil, err
	}
	var elseBlock *domain.InnerBlock
	if p.atKeyword("else") {
		p.Advance()
		// `else if` chains by re-entering parseIf and wrapping it in
		// a single-statement Else block.
		if p.atKeyword("if") {
			nested, err := p.parseIf()
			if err != nil {
				return nil, err
			}
			elseBlock = &domain.InnerBlock{Stmts: []domain.InnerStmt{nested}}
		} else {
			p.consumeNewline()
			eb, err := p.parseBlockBody()
			if err != nil {
				return nil, err
			}
			elseBlock = &eb
			if !p.atKeyword("end") {
				return nil, fmt.Errorf("line %d: expected `end` to close else", p.Peek().Line)
			}
			p.Advance()
			p.consumeNewline()
		}
		return domain.IfStmt{Cond: cond, Body: body, Else: elseBlock}, nil
	}
	if !p.atKeyword("end") {
		return nil, fmt.Errorf("line %d: expected `end` to close if", p.Peek().Line)
	}
	p.Advance()
	p.consumeNewline()
	return domain.IfStmt{Cond: cond, Body: body}, nil
}

func (p *innerP) parseLoop() (domain.InnerStmt, error) {
	p.Advance() // for / loop
	if p.Peek().Kind != domain.TokIdent {
		return nil, fmt.Errorf("line %d: expected loop variable", p.Peek().Line)
	}
	first := p.Advance().Text

	// Two-var form: `for KEY, VAL in EXPR`.
	// Detected by a `,` punct token right after the first ident.
	keyVar := ""
	v := first
	if p.Peek().Kind == domain.TokPunct && p.Peek().Text == "," {
		p.Advance()
		if p.Peek().Kind != domain.TokIdent {
			return nil, fmt.Errorf("line %d: expected second loop variable after `,`", p.Peek().Line)
		}
		keyVar = first
		v = p.Advance().Text
	}

	if !p.atKeyword("in") {
		return nil, fmt.Errorf("line %d: expected `in`", p.Peek().Line)
	}
	p.Advance()
	iter, err := parseValue(p, nil)
	if err != nil {
		return nil, err
	}
	if p.Peek().Kind != domain.TokNewline {
		return nil, fmt.Errorf("line %d: expected newline after loop", p.Peek().Line)
	}
	p.consumeNewline()
	body, err := p.parseBlockBody()
	if err != nil {
		return nil, err
	}
	if !p.atKeyword("end") {
		return nil, fmt.Errorf("line %d: expected `end` to close loop", p.Peek().Line)
	}
	p.Advance()
	p.consumeNewline()
	return domain.LoopStmt{Var: v, KeyVar: keyVar, Iter: iter, Body: body}, nil
}

func (p *innerP) parseBlockBody() (domain.InnerBlock, error) {
	if p.Peek().Kind == domain.TokIndent {
		p.Advance()
	}
	body, err := p.parseProgram(true)
	if err != nil {
		return domain.InnerBlock{}, err
	}
	if p.Peek().Kind == domain.TokDedent {
		p.Advance()
	}
	return body, nil
}

func (p *innerP) parseCall() (domain.InnerStmt, error) {
	t := p.Advance()
	name := []string{t.Text}
	for p.Peek().Kind == domain.TokPunct && p.Peek().Text == "." {
		p.Advance()
		n := p.Peek()
		if n.Kind != domain.TokIdent {
			return nil, fmt.Errorf("expected identifier after .")
		}
		p.Advance()
		name = append(name, n.Text)
	}
	var args []domain.Expr
	for !p.atStatementEnd() {
		if p.Peek().Kind == domain.TokPunct && p.Peek().Text == "," {
			p.Advance()
			continue
		}
		a, err := parseValue(p, nil)
		if err != nil {
			return nil, err
		}
		args = append(args, a)
	}
	p.consumeNewline()
	return domain.CallStmt{Call: domain.CallExpr{Name: name, Args: args}}, nil
}

func (p *innerP) atStatementEnd() bool {
	k := p.Peek().Kind
	return k == domain.TokNewline || k == domain.TokEOF || k == domain.TokDedent
}
