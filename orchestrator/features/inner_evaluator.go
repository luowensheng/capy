package orchfeatures

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/luowensheng/capy/domain"
	"github.com/luowensheng/capy/infra"
)

// osChdir is a thin wrapper kept here so command bodies can call
// `cd PATH`. The Host interface doesn't expose Chdir because in
// the engine's normal mode (rendering) it'd be a footgun; commands
// run with explicit permission so the cost is acceptable.
func osChdir(path string) error { return os.Chdir(path) }

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

	// OnUnknownCall is invoked when runPrimitive doesn't recognise
	// a CallStmt's name. Returning (val, true, err) means "handled
	// (or errored)"; (nil, false, nil) means "still unknown, raise
	// the normal error." Used by orchestrator/commands.go to add
	// command-only primitives (like `compile script`) without
	// growing the global primitive set.
	OnUnknownCall func(name string, args []any) (any, bool, error)
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
		if n.Else != nil {
			return e.execBlock(*n.Else, caps, locals)
		}
		return nil
	case domain.LoopStmt:
		v, err := e.eval(n.Iter, caps, locals)
		if err != nil {
			return err
		}
		switch coll := v.(type) {
		case []any:
			for i, item := range coll {
				child := copyMap(locals)
				child[n.Var] = item
				if n.KeyVar != "" {
					child[n.KeyVar] = i
				}
				if err := e.execBlock(n.Body, caps, child); err != nil {
					return err
				}
			}
			return nil
		case map[string]any:
			// Map iteration: sort keys for determinism (matches Go
			// template's `range` over maps, which sorts by key).
			keys := make([]string, 0, len(coll))
			for k := range coll {
				keys = append(keys, k)
			}
			// tiny sort.
			for i := 0; i < len(keys); i++ {
				for j := i + 1; j < len(keys); j++ {
					if keys[j] < keys[i] {
						keys[i], keys[j] = keys[j], keys[i]
					}
				}
			}
			for _, k := range keys {
				child := copyMap(locals)
				child[n.Var] = coll[k]
				if n.KeyVar != "" {
					child[n.KeyVar] = k
				}
				if err := e.execBlock(n.Body, caps, child); err != nil {
					return err
				}
			}
			return nil
		default:
			return fmt.Errorf("loop iterable must be a list or map")
		}
	case domain.CallStmt:
		return e.runPrimitive(n.Call, caps, locals)
	case domain.WriteStmt:
		// WriteStmt should be translated away at library-load time
		// (see translateNewShape in make_library_loader.go). If one
		// reaches here, it means the translator missed a case.
		return fmt.Errorf("internal: unexpanded WriteStmt — please file a bug")
	}
	return fmt.Errorf("unknown inner stmt")
}

func (e *InnerEvaluator) runPrimitive(c domain.CallExpr, caps map[string]domain.CaptureValue, locals map[string]any) error {
	name := strings.Join(c.Name, ".")
	// Evaluate all args once up-front.
	argVals := make([]any, len(c.Args))
	for i, a := range c.Args {
		v, err := e.eval(a, caps, locals)
		if err != nil {
			return err
		}
		argVals[i] = v
	}
	switch name {
	case "error":
		if len(argVals) == 0 {
			return fmt.Errorf("error")
		}
		return fmt.Errorf("%s", toString(argVals[0]))
	case "print":
		// `print EXPR` — prints to stdout via host.
		parts := make([]string, len(argVals))
		for i, v := range argVals {
			parts[i] = toString(v)
		}
		fmt.Println(strings.Join(parts, " "))
		return nil
	case "write_file":
		if len(argVals) != 2 {
			return fmt.Errorf("write_file: expected 2 args (path, contents)")
		}
		return e.host().WriteFile(toString(argVals[0]), toString(argVals[1]))
	case "mkdir":
		if len(argVals) != 1 {
			return fmt.Errorf("mkdir: expected 1 arg (path)")
		}
		return e.host().Mkdir(toString(argVals[0]))
	case "exec":
		if len(argVals) == 0 {
			return fmt.Errorf("exec: expected at least 1 arg (cmd)")
		}
		cmd := toString(argVals[0])
		args := make([]string, len(argVals)-1)
		for i, a := range argVals[1:] {
			args[i] = toString(a)
		}
		return e.host().Exec(cmd, args...)
	case "cd":
		if len(argVals) != 1 {
			return fmt.Errorf("cd: expected 1 arg (path)")
		}
		// Implemented via os.Chdir at the inner-DSL level (host
		// abstraction would force a separate primitive; keeping it
		// simple while command bodies are POC).
		return osChdir(toString(argVals[0]))
	}
	// Last resort: hand off to the embedder's hook (used by command
	// runners to add `compile script` etc.).
	if e.OnUnknownCall != nil {
		_, handled, err := e.OnUnknownCall(name, argVals)
		if handled {
			return err
		}
	}
	return fmt.Errorf("unknown inner call %q", name)
}

// writePath performs op on context (or locals if root is "locals"). The path's
// root must be either "context" or "locals".
func (e *InnerEvaluator) writePath(p domain.Path, value any, caps map[string]domain.CaptureValue, locals map[string]any, op string) error {
	if p.Root == "locals" {
		// `let X = …` desugars to `set locals.X …`. Single-step paths
		// are the normal case; nested writes use the standard walker.
		if len(p.Steps) == 1 && !p.Steps[0].IsIndex {
			locals[p.Steps[0].Field] = value
			return nil
		}
		return fmt.Errorf("%s: only single-name `locals.X` writes are supported (got %d steps)", op, len(p.Steps))
	}
	if p.Root != "context" {
		return fmt.Errorf("%s: only `context.*` and `locals.*` paths are writable, got root %q", op, p.Root)
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

// RenderAST walks a write-style AST and emits the output text. It
// replaces the older "transpile to Go template syntax then run it"
// flow — the engine no longer needs text/template.
//
// Render-scope: `locals` carries the iteration variable for the
// current `for`, the special `body` value, and any captures the
// caller pre-stuffed in. `context` resolves to e.Context. Captures
// from a function call appear in `locals` as their source-text
// form (matching the previous template-render behaviour: what the
// user wrote appears verbatim in the output unless a helper
// transforms it).
//
// State-mutation statements (set / append / prepend / merge /
// delete / let / call) are intentionally treated as no-ops here —
// state changes happen on a separate pass via Exec().
func (e *InnerEvaluator) RenderAST(b domain.InnerBlock, locals map[string]any) (string, error) {
	var out strings.Builder
	if err := e.renderBlock(b, locals, &out); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (e *InnerEvaluator) renderBlock(b domain.InnerBlock, locals map[string]any, out *strings.Builder) error {
	for _, s := range b.Stmts {
		if err := e.renderStmt(s, locals, out); err != nil {
			return err
		}
	}
	return nil
}

func (e *InnerEvaluator) renderStmt(s domain.InnerStmt, locals map[string]any, out *strings.Builder) error {
	switch n := s.(type) {
	case domain.WriteStmt:
		v, err := e.evalRender(n.Value, locals)
		if err != nil {
			return err
		}
		out.WriteString(toString(v))
		return nil
	case domain.IfStmt:
		v, err := e.evalRender(n.Cond, locals)
		if err != nil {
			return err
		}
		if truthy(v) {
			return e.renderBlock(n.Body, locals, out)
		}
		if n.Else != nil {
			return e.renderBlock(*n.Else, locals, out)
		}
		return nil
	case domain.LoopStmt:
		v, err := e.evalRender(n.Iter, locals)
		if err != nil {
			return err
		}
		switch coll := v.(type) {
		case []any:
			for i, item := range coll {
				child := copyMap(locals)
				child[n.Var] = item
				if n.KeyVar != "" {
					child[n.KeyVar] = i
				}
				if err := e.renderBlock(n.Body, child, out); err != nil {
					return err
				}
			}
		case []string:
			// Host primitives like `args` return []string. Treat it
			// like []any for iteration purposes.
			for i, item := range coll {
				child := copyMap(locals)
				child[n.Var] = item
				if n.KeyVar != "" {
					child[n.KeyVar] = i
				}
				if err := e.renderBlock(n.Body, child, out); err != nil {
					return err
				}
			}
		case map[string]any:
			keys := make([]string, 0, len(coll))
			for k := range coll {
				keys = append(keys, k)
			}
			for i := 0; i < len(keys); i++ {
				for j := i + 1; j < len(keys); j++ {
					if keys[j] < keys[i] {
						keys[i], keys[j] = keys[j], keys[i]
					}
				}
			}
			for _, k := range keys {
				child := copyMap(locals)
				child[n.Var] = coll[k]
				if n.KeyVar != "" {
					child[n.KeyVar] = k
				}
				if err := e.renderBlock(n.Body, child, out); err != nil {
					return err
				}
			}
		case nil:
			// no-op iteration
		default:
			return fmt.Errorf("render: loop iterable must be a list or map, got %T", v)
		}
		return nil
	}
	// Set/Append/Prepend/Merge/Delete/Call/etc. — render-side no-op.
	// State mutations are handled by Exec() on a separate pass.
	return nil
}

// evalRender evaluates an expression in the render scope. Slightly
// different from eval(): VarRef paths that resolve to nothing return
// an empty string (matching template engine behaviour) instead of
// erroring; CallExpr first tries the helper table (so `${unquote x}`
// works).
func (e *InnerEvaluator) evalRender(x domain.Expr, locals map[string]any) (any, error) {
	switch n := x.(type) {
	case domain.NumberLit:
		if n.IsInt {
			return n.I, nil
		}
		return n.F, nil
	case domain.StringLit:
		// Render-mode interpolation: `${expr}` may be a path, helper
		// call, or pipe chain — not just a path lookup. Delegate to a
		// dedicated walker that parses the inner expression on demand.
		return e.interpolateRender(n.Value, locals)
	case domain.BoolLit:
		return n.Value, nil
	case domain.NullLit:
		return nil, nil
	case domain.VarRef:
		v, err := e.resolveRender(n.Path, locals)
		if err != nil {
			// Templates tolerate missing paths — emit empty.
			return "", nil
		}
		return v, nil
	case domain.NotExpr:
		v, err := e.evalRender(n.X, locals)
		if err != nil {
			return nil, err
		}
		return !truthy(v), nil
	case domain.CompareExpr:
		l, err := e.evalRender(n.Left, locals)
		if err != nil {
			return nil, err
		}
		r, err := e.evalRender(n.Right, locals)
		if err != nil {
			return nil, err
		}
		return cmp(n.Op, l, r)
	case domain.CallExpr:
		name := strings.Join(n.Name, ".")
		argVals := make([]any, len(n.Args))
		for i, a := range n.Args {
			v, err := e.evalRender(a, locals)
			if err != nil {
				return nil, err
			}
			argVals[i] = v
		}
		// Try the template-helper table first (add/upper/unquote/…).
		if v, ok, err := infra.ApplyHelper(name, argVals); ok {
			return v, err
		}
		// `len` is a builtin in templates but not in ApplyHelper.
		if name == "len" && len(argVals) == 1 {
			switch v := argVals[0].(type) {
			case []any:
				return int64(len(v)), nil
			case map[string]any:
				return int64(len(v)), nil
			case string:
				return int64(len(v)), nil
			}
			return int64(0), nil
		}
		// Fall through to the inner evaluator's CallExpr (regex_match, env, …).
		return e.eval(x, map[string]domain.CaptureValue{}, locals)
	}
	return nil, fmt.Errorf("render: unsupported expression %T", x)
}

// RenderPath resolves a file-path string with write-style `${...}`
// interpolations. Same engine as interpolateRender, exposed for the
// outer evaluator to call on file PATHS (bare strings, not part of
// a backtick body).
func (e *InnerEvaluator) RenderPath(path string, locals map[string]any) (string, error) {
	return e.interpolateRender(path, locals)
}

// interpolateRender walks a backtick string body and resolves
// `${expr}` markers in render scope. Unlike interpolateGeneric
// (which only handles dotted paths), this evaluates the full
// inner-expression grammar: helper calls (`unquote name`), pipes
// (`x | upper | toQuoted`), and nested calls (`toQuoted (upper x)`).
//
// Mirrors the parsing behaviour the old translateInterpolatedString
// embedded in the Go-template transpilation path — kept identical
// so all migrated samples render byte-for-byte the same.
func (e *InnerEvaluator) interpolateRender(s string, locals map[string]any) (string, error) {
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
			expr := strings.TrimSpace(s[i+2 : j])
			v, err := e.evalInterp(expr, locals)
			if err != nil {
				return "", err
			}
			b.WriteString(toString(v))
			i = j + 1
		} else if s[i] == '\\' && i+1 < len(s) {
			// Go-style escape: \n / \t / \r / \` / \\, and \X
			// (literal X) for other characters — matches the
			// backtick body unescape in translate_new_shape.go so
			// AST-rendered output is byte-identical.
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '`':
				b.WriteByte('`')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte(s[i+1])
			}
			i += 2
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String(), nil
}

// evalInterp parses + evaluates the body of a single `${...}`.
// Grammar mirrors what the legacy template-transpilation path
// accepted:
//
//	atom         := IDENT (. IDENT | [ EXPR ])*    -- path lookup
//	              | NUMBER | STRING                 -- literal
//	              | ( EXPR )                        -- parens
//	stage        := atom+                           -- call: head=fn, rest=args
//	expr         := stage ( | stage )*              -- pipe chain
//
// Pipes flow left to right with each stage receiving the running
// value as its FINAL argument (matches the Go template `|` form).
func (e *InnerEvaluator) evalInterp(expr string, locals map[string]any) (any, error) {
	stages := splitInterpPipeRuntime(expr)
	var running any
	for idx, stage := range stages {
		atoms := tokeniseInterpRuntime(strings.TrimSpace(stage))
		if len(atoms) == 0 {
			continue
		}
		if idx > 0 {
			// Pipe stage: previous value becomes the last argument.
			atoms = append(atoms, "")
		}
		argVals := make([]any, 0, len(atoms))
		head := atoms[0]
		for ai, a := range atoms[1:] {
			if idx > 0 && ai == len(atoms[1:])-1 {
				// last slot is the piped value
				argVals = append(argVals, running)
				continue
			}
			v, err := e.evalInterpAtom(a, locals)
			if err != nil {
				return nil, err
			}
			argVals = append(argVals, v)
		}
		if len(atoms) == 1 {
			// Single atom: it's a value (path or literal), not a call.
			v, err := e.evalInterpAtom(head, locals)
			if err != nil {
				return nil, err
			}
			running = v
			continue
		}
		// Multi-atom: call head with argVals as a helper / builtin.
		if v, ok, err := infra.ApplyHelper(head, argVals); ok {
			if err != nil {
				return nil, err
			}
			running = v
			continue
		}
		// `len` builtin (templates have it but ApplyHelper doesn't).
		if head == "len" && len(argVals) == 1 {
			switch v := argVals[0].(type) {
			case []any:
				running = int64(len(v))
			case map[string]any:
				running = int64(len(v))
			case string:
				running = int64(len(v))
			default:
				running = int64(0)
			}
			continue
		}
		return nil, fmt.Errorf("interp: unknown helper %q", head)
	}
	return running, nil
}

// evalInterpAtom resolves a single token inside `${...}`. Numbers
// and quoted strings pass through; parenthesised groups recurse;
// everything else is a path lookup.
func (e *InnerEvaluator) evalInterpAtom(s string, locals map[string]any) (any, error) {
	if s == "" {
		return "", nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		// Quoted string literal. The lexer preserved every backslash
		// sequence verbatim from the backtick body, so we need TWO
		// unescape passes to match what the previous Go-template
		// path did (backtick body unescape + Go template string
		// literal unescape). Concretely: source `\\n` (3 chars)
		// → after pass 1: `\n` (2 chars: backslash + n)
		// → after pass 2: real newline (1 char)
		// This is what helpers like `trimSuffix ",\\n" body` need —
		// the body has REAL newlines and we want the suffix to match.
		inner := unescapeStringLitInner(unescapeStringLitInner(s[1 : len(s)-1]))
		return inner, nil
	}
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v, nil
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v, nil
	}
	if len(s) >= 2 && s[0] == '(' && s[len(s)-1] == ')' {
		return e.evalInterp(s[1:len(s)-1], locals)
	}
	// Path lookup: split on `.` and resolve.
	path := strings.Split(s, ".")
	v, err := e.resolveRender(path, locals)
	if err != nil {
		// Missing paths render empty — matches Go template tolerance.
		return "", nil
	}
	return v, nil
}

// unescapeStringLitInner does one pass of Go-style escape decoding
// over the unquoted inner of a string literal:
//
//	\n → newline   \t → tab   \r → CR
//	\" → "         \\ → \
//	\X → X (for any other X)
//
// Run twice to mirror the previous chain (backtick-body unescape
// followed by template string-literal unescape).
func unescapeStringLitInner(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte(s[i+1])
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// splitInterpPipeRuntime splits on top-level `|`. Same logic as the
// translator's splitInterpPipe, kept separate so the runtime doesn't
// import from translate_new_shape.go and stays self-contained.
func splitInterpPipeRuntime(s string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
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
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
			cur.WriteByte(c)
		case '(':
			depth++
			cur.WriteByte(c)
		case ')':
			depth--
			cur.WriteByte(c)
		case '|':
			if depth == 0 {
				out = append(out, cur.String())
				cur.Reset()
				continue
			}
			cur.WriteByte(c)
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// tokeniseInterpRuntime tokenises a single pipe stage. Parens stay
// together as one token; quoted strings stay together; whitespace
// separates atoms. Mirrors the translator's tokeniseInterp.
func tokeniseInterpRuntime(s string) []string {
	var out []string
	var cur strings.Builder
	inStr := false
	depth := 0
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
				if depth == 0 {
					out = append(out, cur.String())
					cur.Reset()
				}
			}
			continue
		}
		if c == '"' {
			inStr = true
			cur.WriteByte(c)
			continue
		}
		if c == '(' {
			depth++
			cur.WriteByte(c)
			continue
		}
		if c == ')' {
			depth--
			cur.WriteByte(c)
			if depth == 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}
		if depth > 0 {
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

// resolveRender looks up a path in render scope: locals → context.
// VarRef paths support map indexing (a.b.c) all the way down.
func (e *InnerEvaluator) resolveRender(path []string, locals map[string]any) (any, error) {
	root := path[0]
	rest := path[1:]
	var cur any
	if v, ok := locals[root]; ok {
		cur = v
	} else if root == "context" {
		cur = e.Context
	} else {
		return nil, fmt.Errorf("undefined %q", root)
	}
	for _, step := range rest {
		switch m := cur.(type) {
		case map[string]any:
			cur = m[step]
		case nil:
			return nil, nil
		default:
			return nil, fmt.Errorf("cannot access %q on non-map (%T)", step, cur)
		}
	}
	return cur, nil
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
		case "mktemp":
			// mktemp ".ext" → string (path to a fresh temp file).
			suffix := ""
			if len(n.Args) == 1 {
				v, err := e.eval(n.Args[0], caps, locals)
				if err != nil {
					return nil, err
				}
				suffix = toString(v)
			}
			return e.host().MkTemp(suffix)
		case "mktemp_dir":
			// mktemp_dir → string (path to a fresh temp directory).
			if len(n.Args) != 0 {
				return nil, fmt.Errorf("mktemp_dir takes no args")
			}
			return e.host().MkTempDir()
		case "exec_capture":
			// exec_capture "cmd" "arg1" "arg2" → string (combined stdout).
			if len(n.Args) == 0 {
				return nil, fmt.Errorf("exec_capture: expected at least 1 arg (cmd)")
			}
			vs := make([]string, len(n.Args))
			for i, a := range n.Args {
				v, err := e.eval(a, caps, locals)
				if err != nil {
					return nil, err
				}
				vs[i] = toString(v)
			}
			return e.host().ExecCapture(vs[0], vs[1:]...)
		}
		// Evaluate arg values once for the remaining lookups.
		argVals := make([]any, len(n.Args))
		for i, a := range n.Args {
			v, err := e.eval(a, caps, locals)
			if err != nil {
				return nil, err
			}
			argVals[i] = v
		}
		// Template-helper bridge: `add`, `upper`, `toQuoted`, etc.
		// are the same helpers the renderer uses. Letting them be
		// called from inner-DSL expression positions means libraries
		// can pre-compute values (`set context.total (add x y)`,
		// `set context.path (toQuoted (upper x))`) instead of having
		// to do the work in template position.
		if v, ok, err := infra.ApplyHelper(name, argVals); ok {
			return v, err
		}
		// Hook fallback (e.g. command-runner adds `compile script`).
		if e.OnUnknownCall != nil {
			v, handled, err := e.OnUnknownCall(name, argVals)
			if handled {
				return v, err
			}
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
