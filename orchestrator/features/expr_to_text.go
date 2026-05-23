package orchfeatures

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/luowensheng/capy/domain"
)

// ExprToText converts a parsed value expression back into a source-like text
// representation. Capy is a TRANSPILER: a `cond:any` capture in `if x > 0`
// should appear in the target template as the literal text `x > 0`, not as
// an evaluated boolean. That's what this function provides.
func ExprToText(x domain.Expr) string {
	switch n := x.(type) {
	case domain.NumberLit:
		if n.IsInt {
			return strconv.FormatInt(n.I, 10)
		}
		return strconv.FormatFloat(n.F, 'g', -1, 64)
	case domain.StringLit:
		// Re-quote so the surface looks like a string literal.
		// The source token already had quotes stripped by the lexer.
		return strconv.Quote(n.Value)
	case domain.BoolLit:
		if n.Value {
			return "true"
		}
		return "false"
	case domain.NullLit:
		return "null"
	case domain.VarRef:
		return strings.Join(n.Path, ".")
	case domain.CompareExpr:
		return ExprToText(n.Left) + " " + n.Op + " " + ExprToText(n.Right)
	case domain.NotExpr:
		return "not " + ExprToText(n.X)
	case domain.ListLit:
		parts := make([]string, 0, len(n.Items))
		for _, it := range n.Items {
			parts = append(parts, ExprToText(it))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case domain.ObjLit:
		parts := []string{}
		for i, k := range n.Keys {
			parts = append(parts, strconv.Quote(k)+": "+ExprToText(n.Vals[i]))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case domain.CallExpr:
		args := []string{}
		for _, a := range n.Args {
			args = append(args, ExprToText(a))
		}
		return "(" + strings.Join(n.Name, ".") + " " + strings.Join(args, " ") + ")"
	}
	return fmt.Sprintf("%v", x)
}
