package orchfeatures

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/luowensheng/capy/domain"
)

// translateNewShape walks an inner-DSL block written in the
// proposed unified shape (write calls intermixed with state-mutation
// statements) and splits it into:
//   - a Go text/template string that the existing template renderer
//     can consume (built from the WriteStmt values and the
//     write-only control-flow that wraps them)
//   - a residual InnerBlock containing only state-mutation
//     statements that the inner evaluator runs after the template
//     renders
//
// The split lets the unified shape ride on top of the existing
// engine without inventing a new template runtime. Control flow
// (for / if) that contains a mix of writes AND state mutations is
// duplicated: one copy goes into the template (for the writes,
// state stripped), one copy goes into the run block (for the
// mutations, writes stripped). That guarantees both phases see
// the same iteration / branching shape.
func translateNewShape(b domain.InnerBlock) (template string, run domain.InnerBlock, err error) {
	var tpl strings.Builder
	var runStmts []domain.InnerStmt
	// Track loop-variable names so the body translator emits
	// `$name` references (not `.name`) for them.
	scope := map[string]bool{}
	for _, s := range b.Stmts {
		if err := translateStmt(s, &tpl, &runStmts, scope); err != nil {
			return "", domain.InnerBlock{}, err
		}
	}
	return tpl.String(), domain.InnerBlock{Stmts: runStmts}, nil
}

func translateStmt(s domain.InnerStmt, tpl *strings.Builder, run *[]domain.InnerStmt, scope map[string]bool) error {
	switch n := s.(type) {
	case domain.WriteStmt:
		return translateWriteStmt(n, tpl, scope)
	case domain.IfStmt:
		// Emit a template if-block for the writes inside, and a
		// run-block if-block for the state mutations inside.
		var bodyTpl strings.Builder
		var bodyRun []domain.InnerStmt
		for _, child := range n.Body.Stmts {
			if err := translateStmt(child, &bodyTpl, &bodyRun, scope); err != nil {
				return err
			}
		}
		var elseTpl strings.Builder
		var elseRun []domain.InnerStmt
		hasElse := false
		if n.Else != nil {
			hasElse = true
			for _, child := range n.Else.Stmts {
				if err := translateStmt(child, &elseTpl, &elseRun, scope); err != nil {
					return err
				}
			}
		}
		// Template side: only emit if-block if there's template
		// content in either arm.
		if bodyTpl.Len() > 0 || elseTpl.Len() > 0 {
			tpl.WriteString("{{ if ")
			tpl.WriteString(exprToTemplateCond(n.Cond, scope))
			tpl.WriteString(" }}")
			tpl.WriteString(bodyTpl.String())
			if hasElse && elseTpl.Len() > 0 {
				tpl.WriteString("{{ else }}")
				tpl.WriteString(elseTpl.String())
			}
			tpl.WriteString("{{ end }}")
		}
		// Run side: only emit if-block if there's state content.
		if len(bodyRun) > 0 || len(elseRun) > 0 {
			rs := domain.IfStmt{Cond: n.Cond, Body: domain.InnerBlock{Stmts: bodyRun}}
			if hasElse && len(elseRun) > 0 {
				rs.Else = &domain.InnerBlock{Stmts: elseRun}
			}
			*run = append(*run, rs)
		}
		return nil
	case domain.LoopStmt:
		// Add the loop variable(s) to the scope before translating the
		// body so `${var}` and `${var.field}` interpolations emit
		// `$var` / `$var.field` (Go-template variable references)
		// instead of `.var` / `.var.field` (data-tree access).
		inner := make(map[string]bool, len(scope)+2)
		for k, v := range scope {
			inner[k] = v
		}
		inner[n.Var] = true
		if n.KeyVar != "" {
			inner[n.KeyVar] = true
		}
		var bodyTpl strings.Builder
		var bodyRun []domain.InnerStmt
		for _, child := range n.Body.Stmts {
			if err := translateStmt(child, &bodyTpl, &bodyRun, inner); err != nil {
				return err
			}
		}
		if bodyTpl.Len() > 0 {
			tpl.WriteString("{{ range ")
			if n.KeyVar != "" {
				// Two-var form: `{{ range $k, $v := EXPR }}`.
				// Go template's `range` over a map yields key, value;
				// over a list yields index, value.
				tpl.WriteString("$")
				tpl.WriteString(n.KeyVar)
				tpl.WriteString(", $")
				tpl.WriteString(n.Var)
				tpl.WriteString(" := ")
			} else {
				tpl.WriteString("$")
				tpl.WriteString(n.Var)
				tpl.WriteString(" := ")
			}
			tpl.WriteString(exprToTemplateValue(n.Iter, scope))
			tpl.WriteString(" }}")
			tpl.WriteString(bodyTpl.String())
			tpl.WriteString("{{ end }}")
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
		// call/error) are state mutations — pass through to the
		// run block unchanged.
		*run = append(*run, s)
		return nil
	}
}

// translateWriteStmt converts a `write EXPR` call into Go template
// syntax appended to tpl. EXPR is most commonly a backtick-string
// literal with `${name}` / `${func a b}` interpolations; it may
// also be a bare value reference (e.g. `write body`).
func translateWriteStmt(w domain.WriteStmt, tpl *strings.Builder, scope map[string]bool) error {
	switch v := w.Value.(type) {
	case domain.StringLit:
		return translateInterpolatedString(v.Value, tpl, scope)
	case domain.VarRef:
		// `write body` → {{ .body }} (or {{ $body }} if body is a loop var)
		tpl.WriteString("{{ ")
		tpl.WriteString(refPath(v.Path, scope))
		tpl.WriteString(" }}")
		return nil
	case domain.CallExpr:
		// `write indent 4 body` → {{ indent 4 .body }}
		tpl.WriteString("{{ ")
		tpl.WriteString(callToTemplate(v, scope))
		tpl.WriteString(" }}")
		return nil
	default:
		return fmt.Errorf("write: unsupported value type %T", v)
	}
}

// translateInterpolatedString converts a backtick literal body
// (with embedded `${EXPR}` interpolations) into Go template syntax.
// Literal text is preserved; `${X}` becomes `{{ .X }}` (or the
// equivalent helper-call form when X is a call expression).
//
// `\n` two-byte sequences in the input — left there by
// mergeMultilineBackticks — are converted to real newline bytes
// here so the template emits the right output.
func translateInterpolatedString(s string, tpl *strings.Builder, scope map[string]bool) error {
	s = unescapeBacktickBody(s)
	i := 0
	for i < len(s) {
		// Look for `${`. Note: `\$` escapes a literal dollar.
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == '$' {
			tpl.WriteByte('$')
			i += 2
			continue
		}
		if i+1 < len(s) && s[i] == '$' && s[i+1] == '{' {
			// Find the matching `}` (allowing nested braces).
			depth := 1
			j := i + 2
			for j < len(s) && depth > 0 {
				switch s[j] {
				case '{':
					depth++
				case '}':
					depth--
					if depth == 0 {
						break
					}
				}
				if depth > 0 {
					j++
				}
			}
			if j >= len(s) {
				return fmt.Errorf("unterminated ${...}")
			}
			expr := strings.TrimSpace(s[i+2 : j])
			tpl.WriteString("{{ ")
			tpl.WriteString(interpolationToTemplate(expr, scope))
			tpl.WriteString(" }}")
			i = j + 1
			continue
		}
		// `{{` and `}}` are Go template syntax — escape them by
		// emitting `{{"{{"}}` / `{{"}}"}}` so the renderer reproduces
		// the literal pair in the output.
		if i+1 < len(s) && s[i] == '{' && s[i+1] == '{' {
			tpl.WriteString(`{{"{{"}}`)
			i += 2
			continue
		}
		if i+1 < len(s) && s[i] == '}' && s[i+1] == '}' {
			tpl.WriteString(`{{"}}"}}`)
			i += 2
			continue
		}
		tpl.WriteByte(s[i])
		i++
	}
	return nil
}

// interpolationToTemplate translates the inside of a `${...}`
// expression into Go-template form. Recognised shapes:
//
//	name            → .name
//	context.foo     → .context.foo
//	body            → .body
//	func arg1 arg2  → func arg1template arg2template
//	(no operators, no nested ${} — keeps the grammar small)
func interpolationToTemplate(expr string, scope map[string]bool) string {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ""
	}
	parts := tokeniseInterp(expr)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return interpAtomToTemplate(parts[0], scope)
	}
	head := parts[0]
	out := head
	for _, a := range parts[1:] {
		out += " " + interpAtomToTemplate(a, scope)
	}
	return out
}

// interpAtomToTemplate translates a single token from the inside of
// `${…}` into the equivalent Go-template form. Numbers and quoted
// strings pass through verbatim; identifiers in scope (loop
// variables) become `$name`; everything else becomes `.name`.
func interpAtomToTemplate(s string, scope map[string]bool) string {
	if s == "" {
		return ""
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return s
	}
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return s
	}
	// Dotted path — check if the root is a loop variable.
	if i := strings.IndexByte(s, '.'); i > 0 {
		root := s[:i]
		if scope[root] {
			return "$" + s
		}
	} else if scope[s] {
		return "$" + s
	}
	return "." + s
}

func tokeniseInterp(s string) []string {
	var out []string
	var cur strings.Builder
	inStr := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			cur.WriteByte(c)
			if c == '\\' && i+1 < len(s) {
				cur.WriteByte(s[i+1])
				i++
				continue
			}
			if c == '"' {
				inStr = false
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}
		if c == '"' {
			inStr = true
			cur.WriteByte(c)
			continue
		}
		if c == ' ' || c == '\t' {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// exprToTemplateCond renders a condition expression (for an `if`)
// in Go-template form. We support the common shapes used in
// existing samples; complex conditions fall back to evaluating in
// the run-block path (which has the full inner-DSL evaluator).
func exprToTemplateCond(e domain.Expr, scope map[string]bool) string {
	switch n := e.(type) {
	case domain.VarRef:
		return refPath(n.Path, scope)
	case domain.NotExpr:
		return "not " + exprToTemplateCond(n.X, scope)
	case domain.CompareExpr:
		op := n.Op
		switch op {
		case "==":
			op = "eq"
		case "!=":
			op = "ne"
		case "<":
			op = "lt"
		case "<=":
			op = "le"
		case ">":
			op = "gt"
		case ">=":
			op = "ge"
		}
		return op + " " + exprToTemplateValue(n.Left, scope) + " " + exprToTemplateValue(n.Right, scope)
	case domain.CallExpr:
		return callToTemplate(n, scope)
	}
	return exprToTemplateValue(e, scope)
}

func exprToTemplateValue(e domain.Expr, scope map[string]bool) string {
	switch n := e.(type) {
	case domain.StringLit:
		// n.Value comes out of the .capy lexer with backslash escapes
		// preserved verbatim (`\n` is two chars). Unescape first so
		// Go-template's quoted-string parser sees the right bytes.
		return strconv.Quote(unescapeBacktickBody(n.Value))
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
		return "nil"
	case domain.VarRef:
		return refPath(n.Path, scope)
	case domain.CallExpr:
		return "(" + callToTemplate(n, scope) + ")"
	}
	return ""
}

func callToTemplate(c domain.CallExpr, scope map[string]bool) string {
	name := strings.Join(c.Name, ".")
	parts := []string{name}
	for _, a := range c.Args {
		parts = append(parts, exprToTemplateValue(a, scope))
	}
	return strings.Join(parts, " ")
}

func refPath(path []string, scope map[string]bool) string {
	// `body` is the inner-block-body reserved name; emit `.body`.
	if len(path) == 1 && path[0] == "body" {
		return ".body"
	}
	// Loop variable? Emit Go-template variable reference `$name`.
	if len(path) > 0 && scope[path[0]] {
		return "$" + strings.Join(path, ".")
	}
	return "." + strings.Join(path, ".")
}

// unescapeBacktickBody converts the `\n` two-byte sequences that
// mergeMultilineBackticks inserted (and any other Go-style escapes
// the lexer preserved verbatim) into their actual characters.
func unescapeBacktickBody(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				out.WriteByte('\n')
			case 't':
				out.WriteByte('\t')
			case 'r':
				out.WriteByte('\r')
			case '`':
				out.WriteByte('`')
			case '\\':
				out.WriteByte('\\')
			default:
				out.WriteByte(s[i])
				out.WriteByte(s[i+1])
			}
			i++
			continue
		}
		out.WriteByte(s[i])
	}
	return out.String()
}

// renderInnerBlock re-serialises an InnerBlock back into inner-DSL
// source text. Used after translateNewShape splits the body — the
// residual run statements still need to flow through the regular
// `Run` → tokenize → ParseInner pipeline so existing tests don't
// have to special-case "pre-parsed AST" inputs.
//
// Round-tripping covers the statement forms our translator emits:
// set / append / prepend / merge / delete / call / if / loop.
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
