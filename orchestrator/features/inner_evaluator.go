package orchfeatures

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/luowensheng/capy/domain"
)

// InnerEvaluator runs a `run:` snippet against:
//   - captures: the bindings the outer match produced (read-only).
//   - context:  the live accumulator (mutated by set/append/...).
//   - locals:   the inner scope (loop variables; library-private bindings).
//   - host:     the embedder-provided host capability surface (env/arg/read_file).
//
// It does NOT execute user-script code. It only updates `context`.
type InnerEvaluator struct {
	Context map[string]any
	Host    domain.Host
}

// host returns a non-nil Host. The zero value of InnerEvaluator gets a
// NoOpHost so test-style construction without explicit wiring keeps
// working — every host primitive just returns the empty zero value.
func (e *InnerEvaluator) host() domain.Host {
	if e.Host == nil {
		return domain.NoOpHost{}
	}
	return e.Host
}

func (e *InnerEvaluator) Exec(prog domain.InnerBlock, captures map[string]domain.CaptureValue) error {
	locals := map[string]any{}
	return e.execBlock(prog, captures, locals)
}

func (e *InnerEvaluator) execBlock(b domain.InnerBlock, caps map[string]domain.CaptureValue, locals map[string]any) error {
	for _, s := range b.Stmts {
		if err := e.execStmt(s, caps, locals); err != nil {
			return err
		}
	}
	return nil
}

func (e *InnerEvaluator) execStmt(s domain.InnerStmt, caps map[string]domain.CaptureValue, locals map[string]any) error {
	switch n := s.(type) {
	case domain.SetStmt:
		v, err := e.eval(n.Value, caps, locals)
		if err != nil {
			return err
		}
		return e.writePath(n.Target, v, caps, locals, "set")
	case domain.AppendStmt:
		v, err := e.eval(n.Value, caps, locals)
		if err != nil {
			return err
		}
		return e.writePath(n.Target, v, caps, locals, "append")
	case domain.PrependStmt:
		v, err := e.eval(n.Value, caps, locals)
		if err != nil {
			return err
		}
		return e.writePath(n.Target, v, caps, locals, "prepend")
	case domain.MergeStmt:
		v, err := e.eval(n.Value, caps, locals)
		if err != nil {
			return err
		}
		m, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("merge: value must be a map")
		}
		return e.writePath(n.Target, m, caps, locals, "merge")
	case domain.DeleteStmt:
		return e.writePath(n.Target, nil, caps, locals, "delete")
	case domain.IfStmt:
		v, err := e.eval(n.Cond, caps, locals)
		if err != nil {
			return err
		}
		if truthy(v) {
			return e.execBlock(n.Body, caps, locals)
		}
		return nil
	case domain.LoopStmt:
		v, err := e.eval(n.Iter, caps, locals)
		if err != nil {
			return err
		}
		list, ok := v.([]any)
		if !ok {
			return fmt.Errorf("loop iterable must be a list")
		}
		for _, item := range list {
			child := copyMap(locals)
			child[n.Var] = item
			if err := e.execBlock(n.Body, caps, child); err != nil {
				return err
			}
		}
		return nil
	case domain.CallStmt:
		return e.runPrimitive(n.Call, caps, locals)
	}
	return fmt.Errorf("unknown inner stmt")
}

func (e *InnerEvaluator) runPrimitive(c domain.CallExpr, caps map[string]domain.CaptureValue, locals map[string]any) error {
	name := strings.Join(c.Name, ".")
	switch name {
	case "error":
		if len(c.Args) == 0 {
			return fmt.Errorf("error")
		}
		v, err := e.eval(c.Args[0], caps, locals)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", toString(v))
	}
	return fmt.Errorf("unknown inner call %q", name)
}

// writePath performs op on context (or locals if root is "locals"). The path's
// root must be either "context" or a name in `locals`/`caps`.
func (e *InnerEvaluator) writePath(p domain.Path, value any, caps map[string]domain.CaptureValue, locals map[string]any, op string) error {
	if p.Root != "context" {
		return fmt.Errorf("%s: only `context.*` paths are writable, got root %q", op, p.Root)
	}
	if len(p.Steps) == 0 {
		return fmt.Errorf("%s: path must have at least one step under context", op)
	}
	// Walk to parent, mutate last.
	parent := any(e.Context)
	for i, step := range p.Steps {
		isLast := i == len(p.Steps)-1
		if isLast {
			return e.applyOp(parent, step, value, caps, locals, op)
		}
		next, err := e.descend(parent, step, caps, locals)
		if err != nil {
			return err
		}
		parent = next
	}
	return nil
}

func (e *InnerEvaluator) descend(parent any, step domain.PathStep, caps map[string]domain.CaptureValue, locals map[string]any) (any, error) {
	if step.IsIndex {
		idx, err := e.eval(step.Index, caps, locals)
		if err != nil {
			return nil, err
		}
		switch p := parent.(type) {
		case map[string]any:
			key := toString(idx)
			return p[key], nil
		case []any:
			i, ok := idx.(int64)
			if !ok {
				return nil, fmt.Errorf("list index must be int")
			}
			// Negative indices count from the end: -1 → last element.
			// Useful for "append to the last appended item" patterns
			// inside nested blocks.
			n := int64(len(p))
			if i < 0 {
				i += n
			}
			if i < 0 || i >= n {
				return nil, fmt.Errorf("list index %d out of range (len=%d)", i, n)
			}
			return p[int(i)], nil
		}
		return nil, fmt.Errorf("cannot index into %T", parent)
	}
	switch p := parent.(type) {
	case map[string]any:
		return p[step.Field], nil
	}
	return nil, fmt.Errorf("cannot descend %q on %T", step.Field, parent)
}

func (e *InnerEvaluator) applyOp(parent any, step domain.PathStep, value any, caps map[string]domain.CaptureValue, locals map[string]any, op string) error {
	m, mok := parent.(map[string]any)
	var key string
	if step.IsIndex {
		idx, err := e.eval(step.Index, caps, locals)
		if err != nil {
			return err
		}
		key = toString(idx)
	} else {
		key = step.Field
	}
	if !mok {
		return fmt.Errorf("%s: target parent is not a map", op)
	}
	switch op {
	case "set":
		m[key] = value
	case "delete":
		delete(m, key)
	case "append":
		list, _ := m[key].([]any)
		m[key] = append(list, value)
	case "prepend":
		list, _ := m[key].([]any)
		m[key] = append([]any{value}, list...)
	case "merge":
		existing, _ := m[key].(map[string]any)
		if existing == nil {
			existing = map[string]any{}
		}
		for k, v := range value.(map[string]any) {
			existing[k] = v
		}
		m[key] = existing
	default:
		return fmt.Errorf("unknown op %q", op)
	}
	return nil
}

// --- expression evaluator (independent of outer scope) ---

func (e *InnerEvaluator) eval(x domain.Expr, caps map[string]domain.CaptureValue, locals map[string]any) (any, error) {
	switch n := x.(type) {
	case domain.NumberLit:
		if n.IsInt {
			return n.I, nil
		}
		return n.F, nil
	case domain.StringLit:
		return interpolateGeneric(n.Value, func(path []string) (any, error) {
			return e.resolvePath(path, caps, locals)
		})
	case domain.BoolLit:
		return n.Value, nil
	case domain.NullLit:
		return nil, nil
	case domain.VarRef:
		return e.resolvePath(n.Path, caps, locals)
	case domain.NotExpr:
		v, err := e.eval(n.X, caps, locals)
		if err != nil {
			return nil, err
		}
		return !truthy(v), nil
	case domain.CompareExpr:
		l, err := e.eval(n.Left, caps, locals)
		if err != nil {
			return nil, err
		}
		r, err := e.eval(n.Right, caps, locals)
		if err != nil {
			return nil, err
		}
		return cmp(n.Op, l, r)
	case domain.ListLit:
		out := make([]any, 0, len(n.Items))
		for _, it := range n.Items {
			v, err := e.eval(it, caps, locals)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case domain.ObjLit:
		out := map[string]any{}
		for i, k := range n.Keys {
			v, err := e.eval(n.Vals[i], caps, locals)
			if err != nil {
				return nil, err
			}
			out[k] = v
		}
		return out, nil
	case domain.CallExpr:
		// Inline function calls: only built-ins like regex_match.
		name := strings.Join(n.Name, ".")
		switch name {
		case "regex_match":
			if len(n.Args) != 2 {
				return nil, fmt.Errorf("regex_match expects 2 args")
			}
			s, err := e.eval(n.Args[0], caps, locals)
			if err != nil {
				return nil, err
			}
			p, err := e.eval(n.Args[1], caps, locals)
			if err != nil {
				return nil, err
			}
			rx, err := regexp.Compile(toString(p))
			if err != nil {
				return nil, err
			}
			return rx.MatchString(toString(s)), nil
		case "env":
			// env "NAME" → string. Returns the OS env var, or "" if unset.
			// Lets libraries weave deployment-time values (DATABASE_URL,
			// PORT, FEATURE_FLAGS) into the accumulating context.
			if len(n.Args) != 1 {
				return nil, fmt.Errorf("env expects 1 arg: env \"NAME\"")
			}
			name, err := e.eval(n.Args[0], caps, locals)
			if err != nil {
				return nil, err
			}
			return e.host().Env(toString(name)), nil
		case "arg":
			// arg N → string. Returns the N-th positional CLI arg (zero-
			// indexed) AFTER the library+script paths, or "" if missing.
			if len(n.Args) != 1 {
				return nil, fmt.Errorf("arg expects 1 arg: arg INDEX")
			}
			idxV, err := e.eval(n.Args[0], caps, locals)
			if err != nil {
				return nil, err
			}
			idx, ok := toInt(idxV)
			if !ok {
				return nil, fmt.Errorf("arg: index must be a number, got %T", idxV)
			}
			return e.host().Arg(idx), nil
		case "arg_count":
			if len(n.Args) != 0 {
				return nil, fmt.Errorf("arg_count takes no args")
			}
			return e.host().ArgCount(), nil
		case "args":
			// args → []string. Useful for `loop a in args` patterns.
			if len(n.Args) != 0 {
				return nil, fmt.Errorf("args takes no args")
			}
			raw := e.host().Args()
			out := make([]any, len(raw))
			for i, s := range raw {
				out[i] = s
			}
			return out, nil
		case "os":
			// os → string (e.g. "linux", "darwin", "windows"). Matches
			// runtime.GOOS so libraries can branch their output by host.
			if len(n.Args) != 0 {
				return nil, fmt.Errorf("os takes no args")
			}
			return e.host().OS(), nil
		case "arch":
			// arch → string (e.g. "amd64", "arm64"). Matches runtime.GOARCH.
			if len(n.Args) != 0 {
				return nil, fmt.Errorf("arch takes no args")
			}
			return e.host().Arch(), nil
		case "cwd":
			if len(n.Args) != 0 {
				return nil, fmt.Errorf("cwd takes no args")
			}
			v, err := e.host().Cwd()
			if err != nil {
				return nil, err
			}
			return v, nil
		case "home_dir":
			if len(n.Args) != 0 {
				return nil, fmt.Errorf("home_dir takes no args")
			}
			v, err := e.host().HomeDir()
			if err != nil {
				return nil, err
			}
			return v, nil
		case "read_file":
			// read_file "path" → string. Path resolves relative to the
			// script directory. Errors abort the transpilation.
			if len(n.Args) != 1 {
				return nil, fmt.Errorf("read_file expects 1 arg: read_file \"PATH\"")
			}
			p, err := e.eval(n.Args[0], caps, locals)
			if err != nil {
				return nil, err
			}
			content, err := e.host().ReadFile(toString(p))
			if err != nil {
				return nil, err
			}
			return content, nil
		}
		return nil, fmt.Errorf("inner call %q not allowed in expression", name)
	}
	return nil, fmt.Errorf("unknown inner expression")
}

// evalExprFallback is eval() with one extra rule: an unresolved single-path
// VarRef (e.g. `name` referring to nothing defined) returns the identifier as
// a string. That matches transpile semantics — the source name flows through
// to the output unchanged.
func (e *InnerEvaluator) evalExprFallback(x domain.Expr, caps map[string]domain.CaptureValue, locals map[string]any) (any, error) {
	if vr, ok := x.(domain.VarRef); ok {
		if _, found := locals[vr.Path[0]]; !found {
			if _, found := caps[vr.Path[0]]; !found {
				if vr.Path[0] != "context" {
					return strings.Join(vr.Path, "."), nil
				}
			}
		}
	}
	return e.eval(x, caps, locals)
}

// resolvePath walks `path` against locals → captures → context.
func (e *InnerEvaluator) resolvePath(path []string, caps map[string]domain.CaptureValue, locals map[string]any) (any, error) {
	root := path[0]
	rest := path[1:]
	var cur any
	if v, ok := locals[root]; ok {
		cur = v
	} else if v, ok := caps[root]; ok {
		// Captures resolve to evaluated values when the inner DSL needs them.
		// String literals become Go strings (without source quotes), numbers
		// become int64/float64, etc. Unresolved bare identifiers become their
		// literal name as a string (the transpile-mode convention). Templates,
		// in contrast, see the raw source text — see make_evaluator.go.
		if v.IsExpr {
			rv, err := e.evalExprFallback(v.Expr, caps, locals)
			if err != nil {
				return nil, err
			}
			cur = rv
		} else {
			cur = v.Text
		}
	} else if root == "context" {
		cur = e.Context
	} else {
		return nil, fmt.Errorf("undefined %q", root)
	}
	for _, step := range rest {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot access %q on non-map", step)
		}
		cur = m[step]
	}
	return cur, nil
}

// --- helpers ---

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case int64:
		return x != 0
	case int:
		return x != 0
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

// toInt coerces a runtime value to int. Returns (i, true) on success.
// Accepts int / int64 / float64 (truncated) / numeric strings.
func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(x))
		if err == nil {
			return n, true
		}
	}
	return 0, false
}

func toString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case bool:
		if x {
			return "true"
		}
		return "false"
	}
	return fmt.Sprintf("%v", v)
}

func cmp(op string, l, r any) (bool, error) {
	eq := func() bool {
		ls, lOk := l.(string)
		rs, rOk := r.(string)
		if lOk && rOk {
			return ls == rs
		}
		ln, lN := numAny(l)
		rn, rN := numAny(r)
		if lN && rN {
			return ln == rn
		}
		lb, lOk2 := l.(bool)
		rb, rOk2 := r.(bool)
		if lOk2 && rOk2 {
			return lb == rb
		}
		if l == nil && r == nil {
			return true
		}
		return false
	}
	cmpNum := func() (int, error) {
		ln, lN := numAny(l)
		rn, rN := numAny(r)
		if lN && rN {
			if ln < rn {
				return -1, nil
			}
			if ln > rn {
				return 1, nil
			}
			return 0, nil
		}
		ls, lOk := l.(string)
		rs, rOk := r.(string)
		if lOk && rOk {
			if ls < rs {
				return -1, nil
			}
			if ls > rs {
				return 1, nil
			}
			return 0, nil
		}
		return 0, fmt.Errorf("incomparable values")
	}
	switch op {
	case "==":
		return eq(), nil
	case "!=":
		return !eq(), nil
	}
	c, err := cmpNum()
	if err != nil {
		return false, err
	}
	switch op {
	case "<":
		return c < 0, nil
	case ">":
		return c > 0, nil
	case "<=":
		return c <= 0, nil
	case ">=":
		return c >= 0, nil
	}
	return false, fmt.Errorf("unknown comparator %q", op)
}

func numAny(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// interpolateGeneric: ${path} substitution; resolver is provided.
func interpolateGeneric(s string, resolve func(path []string) (any, error)) (string, error) {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '$' && s[i+1] == '{' {
			j := i + 2
			depth := 1
			for j < len(s) && depth > 0 {
				if s[j] == '{' {
					depth++
				} else if s[j] == '}' {
					depth--
					if depth == 0 {
						break
					}
				}
				j++
			}
			if j >= len(s) {
				return "", fmt.Errorf("unterminated ${...}")
			}
			expr := s[i+2 : j]
			path := strings.Split(strings.TrimSpace(expr), ".")
			v, err := resolve(path)
			if err != nil {
				return "", err
			}
			b.WriteString(toString(v))
			i = j + 1
		} else if s[i] == '\\' && i+1 < len(s) {
			// One layer of `\X` → X. Library authors use this to embed
			// literal quotes (`\"`) and dollar signs (`\$`) in interpolated
			// strings. To pass a target-language escape sequence through
			// to the generated output (assembler `\n`, regex `\d`, etc.)
			// double the backslash in source: `\\n` → `\n`.
			b.WriteByte(s[i+1])
			i += 2
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String(), nil
}
