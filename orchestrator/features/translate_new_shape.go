package orchfeatures

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/luowensheng/capy/domain"
)

// translateNewShape walks an inner-DSL block (write calls intermixed
// with state-mutation statements) and returns the residual run AST
// — the subset of statements that touch state (`set` / `append` /
// `prepend` / `merge` / `delete` / `call` / `error`).
//
// The render-bearing statements (`write`, `if`, `for`) live alongside
// the run-bearing ones in the same source body. At load time the
// renderer keeps the FULL AST and skips the state mutations at render
// time (see InnerEvaluator.RenderAST); the run AST returned here
// drives the SEPARATE per-statement run pass.
//
// Control flow (`if` / `for`) that contains a mix of writes AND
// state mutations is preserved on the run side with the writes
// stripped, so both phases see the same iteration / branching
// shape.
func translateNewShape(b domain.InnerBlock) (domain.InnerBlock, error) {
	var runStmts []domain.InnerStmt
	for _, s := range b.Stmts {
		if err := extractRunStmt(s, &runStmts); err != nil {
			return domain.InnerBlock{}, err
		}
	}
	return domain.InnerBlock{Stmts: runStmts}, nil
}

// extractRunStmt walks one statement and appends any state-mutating
// projection of it to `run`. Pure render statements (write) produce
// no output. Control flow is preserved when it wraps state mutations.
func extractRunStmt(s domain.InnerStmt, run *[]domain.InnerStmt) error {
	switch n := s.(type) {
	case domain.WriteStmt:
		return nil
	case domain.IfStmt:
		var bodyRun []domain.InnerStmt
		for _, child := range n.Body.Stmts {
			if err := extractRunStmt(child, &bodyRun); err != nil {
				return err
			}
		}
		var elseRun []domain.InnerStmt
		hasElse := false
		if n.Else != nil {
			hasElse = true
			for _, child := range n.Else.Stmts {
				if err := extractRunStmt(child, &elseRun); err != nil {
					return err
				}
			}
		}
		if len(bodyRun) > 0 || len(elseRun) > 0 {
			rs := domain.IfStmt{Cond: n.Cond, Body: domain.InnerBlock{Stmts: bodyRun}}
			if hasElse && len(elseRun) > 0 {
				rs.Else = &domain.InnerBlock{Stmts: elseRun}
			}
			*run = append(*run, rs)
		}
		return nil
	case domain.LoopStmt:
		var bodyRun []domain.InnerStmt
		for _, child := range n.Body.Stmts {
			if err := extractRunStmt(child, &bodyRun); err != nil {
				return err
			}
		}
		if len(bodyRun) > 0 {
			*run = append(*run, domain.LoopStmt{
				Var:    n.Var,
				KeyVar: n.KeyVar,
				Iter:   n.Iter,
				Body:   domain.InnerBlock{Stmts: bodyRun},
			})
		}
		return nil
	default:
		// All other statements (set/append/prepend/merge/delete/
		// call/error) are state mutations — pass through unchanged.
		*run = append(*run, s)
		return nil
	}
}

// renderInnerBlock re-serialises an InnerBlock back into inner-DSL
// source text. Used after translateNewShape splits the body — the
// residual run statements flow through the regular
// `Run` → tokenize → ParseInner pipeline so existing tests don't
// have to special-case "pre-parsed AST" inputs.
func renderInnerBlock(b domain.InnerBlock) string {
	var out strings.Builder
	for _, s := range b.Stmts {
		renderInnerStmt(s, &out, 0)
	}
	return out.String()
}

func renderInnerStmt(s domain.InnerStmt, out *strings.Builder, indent int) {
	prefix := strings.Repeat("    ", indent)
	switch n := s.(type) {
	case domain.SetStmt:
		fmt.Fprintf(out, "%sset %s %s\n", prefix, renderPath(n.Target), renderExpr(n.Value))
	case domain.AppendStmt:
		fmt.Fprintf(out, "%sappend %s %s\n", prefix, renderPath(n.Target), renderExpr(n.Value))
	case domain.PrependStmt:
		fmt.Fprintf(out, "%sprepend %s %s\n", prefix, renderPath(n.Target), renderExpr(n.Value))
	case domain.MergeStmt:
		fmt.Fprintf(out, "%smerge %s %s\n", prefix, renderPath(n.Target), renderExpr(n.Value))
	case domain.DeleteStmt:
		fmt.Fprintf(out, "%sdelete %s\n", prefix, renderPath(n.Target))
	case domain.CallStmt:
		// `(name args...)` — lisp-style call shape the inner parser
		// already accepts.
		fmt.Fprintf(out, "%s(%s", prefix, strings.Join(n.Call.Name, "."))
		for _, a := range n.Call.Args {
			out.WriteByte(' ')
			out.WriteString(renderExpr(a))
		}
		out.WriteString(")\n")
	case domain.IfStmt:
		fmt.Fprintf(out, "%sif %s\n", prefix, renderExpr(n.Cond))
		for _, c := range n.Body.Stmts {
			renderInnerStmt(c, out, indent+1)
		}
		if n.Else != nil {
			fmt.Fprintf(out, "%selse\n", prefix)
			for _, c := range n.Else.Stmts {
				renderInnerStmt(c, out, indent+1)
			}
		}
		fmt.Fprintf(out, "%send\n", prefix)
	case domain.LoopStmt:
		if n.KeyVar != "" {
			fmt.Fprintf(out, "%sloop %s, %s in %s\n", prefix, n.KeyVar, n.Var, renderExpr(n.Iter))
		} else {
			fmt.Fprintf(out, "%sloop %s in %s\n", prefix, n.Var, renderExpr(n.Iter))
		}
		for _, c := range n.Body.Stmts {
			renderInnerStmt(c, out, indent+1)
		}
		fmt.Fprintf(out, "%send\n", prefix)
	}
}

func renderPath(p domain.Path) string {
	out := p.Root
	for _, st := range p.Steps {
		if st.IsIndex {
			out += "[" + renderExpr(st.Index) + "]"
		} else {
			out += "." + st.Field
		}
	}
	return out
}

func renderExpr(e domain.Expr) string {
	switch n := e.(type) {
	case domain.StringLit:
		return strconv.Quote(n.Value)
	case domain.NumberLit:
		if n.IsInt {
			return strconv.FormatInt(n.I, 10)
		}
		return strconv.FormatFloat(n.F, 'g', -1, 64)
	case domain.BoolLit:
		if n.Value {
			return "true"
		}
		return "false"
	case domain.NullLit:
		return "null"
	case domain.VarRef:
		return strings.Join(n.Path, ".")
	case domain.CallExpr:
		out := "(" + strings.Join(n.Name, ".")
		for _, a := range n.Args {
			out += " " + renderExpr(a)
		}
		return out + ")"
	case domain.NotExpr:
		return "(not " + renderExpr(n.X) + ")"
	case domain.CompareExpr:
		return renderExpr(n.Left) + " " + n.Op + " " + renderExpr(n.Right)
	case domain.ListLit:
		parts := []string{}
		for _, x := range n.Items {
			parts = append(parts, renderExpr(x))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case domain.ObjLit:
		parts := []string{}
		for i, k := range n.Keys {
			parts = append(parts, strconv.Quote(k)+": "+renderExpr(n.Vals[i]))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return ""
}
