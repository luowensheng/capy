package orchfeatures

import (
	"fmt"
	"strconv"

	"github.com/olivierdevelops/capy/domain"
)

// tokReader is the read-only token cursor used by the value-expression parser.
// Both the outer pattern matcher and the inner DSL parser implement it.
type tokReader interface {
	Peek() domain.Token
	Advance() domain.Token
	Save() int
	Restore(int)
}

// parseValue parses one value expression with an optional comparison tail.
// It stops at any token in the `stop` set (literal stop tokens) or at
// statement-boundary tokens (NEWLINE, EOF, INDENT, DEDENT).
func parseValue(r tokReader, stop []string) (domain.Expr, error) {
	left, err := parseUnary(r, stop)
	if err != nil {
		return nil, err
	}
	t := r.Peek()
	if t.Kind == domain.TokPunct {
		switch t.Text {
		case "==", "!=", "<", ">", "<=", ">=":
			if !contains(stop, t.Text) {
				r.Advance()
				right, err := parseUnary(r, stop)
				if err != nil {
					return nil, err
				}
				return domain.CompareExpr{Op: t.Text, Left: left, Right: right}, nil
			}
		}
	}
	return left, nil
}

func contains(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}

func parseUnary(r tokReader, stop []string) (domain.Expr, error) {
	t := r.Peek()
	if t.Kind == domain.TokIdent && t.Text == "not" && !contains(stop, "not") {
		r.Advance()
		x, err := parseUnary(r, stop)
		if err != nil {
			return nil, err
		}
		return domain.NotExpr{X: x}, nil
	}
	return parsePrimary(r, stop)
}

func parsePrimary(r tokReader, stop []string) (domain.Expr, error) {
	t := r.Peek()
	switch t.Kind {
	case domain.TokNumber:
		r.Advance()
		if i, err := strconv.ParseInt(t.Text, 10, 64); err == nil {
			return domain.NumberLit{IsInt: true, I: i}, nil
		}
		f, err := strconv.ParseFloat(t.Text, 64)
		if err != nil {
			return nil, fmt.Errorf("bad number %q", t.Text)
		}
		return domain.NumberLit{IsInt: false, F: f}, nil
	case domain.TokString, domain.TokTemplate:
		r.Advance()
		return domain.StringLit{Value: t.Text}, nil
	case domain.TokIdent:
		switch t.Text {
		case "true":
			r.Advance()
			return domain.BoolLit{Value: true}, nil
		case "false":
			r.Advance()
			return domain.BoolLit{Value: false}, nil
		case "null":
			r.Advance()
			return domain.NullLit{}, nil
		}
		r.Advance()
		// Steps[0] is the root name; subsequent `.field` and `[expr]`
		// steps alternate freely so `context.rows[i].name[j]` parses.
		steps := []domain.PathStep{{Field: t.Text}}
		for {
			n := r.Peek()
			if n.Kind == domain.TokPunct && n.Text == "." {
				r.Advance()
				name := r.Peek()
				if name.Kind != domain.TokIdent {
					return nil, fmt.Errorf("expected identifier after .")
				}
				r.Advance()
				steps = append(steps, domain.PathStep{Field: name.Text})
				continue
			}
			if n.Kind == domain.TokLBrack {
				// Postfix index: `[expr]`. `[` is only a list literal
				// when it's the FIRST token of a primary; here it
				// follows an identifier path, so the two never collide.
				r.Advance()
				idx, err := parseValue(r, nil)
				if err != nil {
					return nil, err
				}
				if r.Peek().Kind != domain.TokRBrack {
					return nil, fmt.Errorf("expected ]")
				}
				r.Advance()
				steps = append(steps, domain.PathStep{IsIndex: true, Index: idx})
				continue
			}
			break
		}
		return domain.VarRef{Steps: steps}, nil
	case domain.TokLParen:
		r.Advance()
		skipNewlines(r)
		nameTok := r.Peek()
		if nameTok.Kind != domain.TokIdent {
			return nil, fmt.Errorf("expected identifier inside ( )")
		}
		r.Advance()
		name := []string{nameTok.Text}
		for r.Peek().Kind == domain.TokPunct && r.Peek().Text == "." {
			r.Advance()
			n := r.Peek()
			if n.Kind != domain.TokIdent {
				return nil, fmt.Errorf("expected identifier after .")
			}
			r.Advance()
			name = append(name, n.Text)
		}
		var args []domain.Expr
		for r.Peek().Kind != domain.TokRParen {
			if (r.Peek().Kind == domain.TokPunct && r.Peek().Text == ",") || r.Peek().Kind == domain.TokNewline {
				r.Advance()
				continue
			}
			a, err := parsePrimary(r, nil)
			if err != nil {
				return nil, err
			}
			args = append(args, a)
		}
		r.Advance()
		return domain.CallExpr{Name: name, Args: args}, nil
	case domain.TokLBrack:
		return parseListLit(r)
	case domain.TokLBrace:
		return parseObjLit(r)
	}
	return nil, fmt.Errorf("line %d: unexpected token %q in value", t.Line, t.Text)
}

func parseListLit(r tokReader) (domain.Expr, error) {
	r.Advance()
	var items []domain.Expr
	for r.Peek().Kind != domain.TokRBrack {
		if (r.Peek().Kind == domain.TokPunct && r.Peek().Text == ",") || r.Peek().Kind == domain.TokNewline {
			r.Advance()
			continue
		}
		x, err := parsePrimary(r, nil)
		if err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	r.Advance()
	return domain.ListLit{Items: items}, nil
}

func parseObjLit(r tokReader) (domain.Expr, error) {
	r.Advance()
	var keys []string
	var vals []domain.Expr
	for r.Peek().Kind != domain.TokRBrace {
		if (r.Peek().Kind == domain.TokPunct && r.Peek().Text == ",") || r.Peek().Kind == domain.TokNewline {
			r.Advance()
			continue
		}
		kTok := r.Peek()
		// Accept either a quoted string OR a bare identifier as a key.
		// `{name: "p", "age": 34}` is valid — same as `{"name": "p", "age": 34}`.
		if kTok.Kind != domain.TokString && kTok.Kind != domain.TokIdent {
			return nil, fmt.Errorf("line %d: object keys must be strings or identifiers", kTok.Line)
		}
		r.Advance()
		if !(r.Peek().Kind == domain.TokPunct && r.Peek().Text == ":") {
			return nil, fmt.Errorf("line %d: expected ':'", r.Peek().Line)
		}
		r.Advance()
		skipNewlines(r)
		v, err := parsePrimary(r, nil)
		if err != nil {
			return nil, err
		}
		keys = append(keys, kTok.Text)
		vals = append(vals, v)
	}
	r.Advance()
	return domain.ObjLit{Keys: keys, Vals: vals}, nil
}

func skipNewlines(r tokReader) {
	for r.Peek().Kind == domain.TokNewline {
		r.Advance()
	}
}
