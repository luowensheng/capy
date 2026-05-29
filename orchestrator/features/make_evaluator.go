package orchfeatures

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/luowensheng/capy/domain"
	"github.com/luowensheng/capy/features"
)

// MakeEvaluator builds the transpiler-driver outer evaluator. It walks the
// parsed program top-down. For each FuncCall:
//
//  1. Validate captured args against their declared types.
//  2. If the function opens a block, recursively render the body block first
//     so the rendered string is available as `body` in the function's
//     write-style template.
//  3. Render the function's TemplateAST and append to the parent block's
//     body string.
//  4. Run the function's run snippet (mutates context only).
//  5. After the body completes, render+run the closer FuncCall the same way.
//
// Once the program block is fully rendered, the orchestrator (RunScript)
// renders FileTemplateAST with `body`=(top-level body) and `context`=(final).
//
// Rendering walks the inner-DSL AST directly via InnerEvaluator.RenderAST —
// no Go-template detour. Templates and inner-DSL share one grammar.
func MakeEvaluator() features.Evaluator {
	return MakeEvaluatorWithHost(domain.NoOpHost{})
}

// MakeEvaluatorWithHost is like MakeEvaluator but accepts a host providing
// env / arg / read_file capabilities. The CLI uses infra.OSHost; embedded
// callers and the wasm playground default to NoOpHost so library
// `env`/`read_file` primitives return empty values instead of touching the
// embedder's process state.
func MakeEvaluatorWithHost(host domain.Host) features.Evaluator {
	if host == nil {
		host = domain.NoOpHost{}
	}
	runMulti := func(program domain.Block, lib domain.Library) (string, map[string]string, error) {
		ctx := deepCopyMap(lib.Context)
		ev := &outerEval{
			lib:   lib,
			inner: &InnerEvaluator{Context: ctx, Host: host},
		}
		body, err := ev.renderBlock(program)
		if err != nil {
			return "", nil, err
		}
		// Render the top-level file_template via the AST walker.
		// Libraries with no file_template emit `body` verbatim.
		var out string
		if lib.FileTemplateAST != nil {
			out, err = ev.inner.RenderAST(*lib.FileTemplateAST, map[string]any{"body": body})
			if err != nil {
				return "", nil, fmt.Errorf("file_template: %v", err)
			}
		} else {
			out = body
		}
		files := map[string]string{}
		for path, ast := range lib.FilesAST {
			renderedPath, err := ev.inner.RenderPath(path, map[string]any{"body": body})
			if err != nil {
				return "", nil, fmt.Errorf("file path %q: %v", path, err)
			}
			rendered, err := ev.inner.RenderAST(*ast, map[string]any{"body": body})
			if err != nil {
				return "", nil, fmt.Errorf("file %q: %v", renderedPath, err)
			}
			files[renderedPath] = rendered
		}
		return out, files, nil
	}
	return features.Evaluator{
		Run: func(program domain.Block, lib domain.Library) (string, error) {
			out, _, err := runMulti(program, lib)
			return out, err
		},
		RunMulti: runMulti,
	}
}

type outerEval struct {
	lib   domain.Library
	inner *InnerEvaluator
	// depth is the AST depth at which the next renderFuncCall will run.
	// Used as a fallback when the explicit-depth helpers aren't on the
	// call path; renderBlockAt / renderFuncCallAt thread the value
	// explicitly through recursion.
	depth int
}

func (e *outerEval) renderBlock(b domain.Block) (string, error) {
	return e.renderBlockAt(b, e.depth)
}

func (e *outerEval) renderBlockAt(b domain.Block, depth int) (string, error) {
	var out strings.Builder
	for _, c := range b.Stmts {
		s, err := e.renderFuncCallAt(c, depth)
		if err != nil {
			return "", err
		}
		out.WriteString(s)
	}
	return out.String(), nil
}

func (e *outerEval) renderFuncCall(c domain.FuncCall) (string, error) {
	return e.renderFuncCallAt(c, e.depth)
}

// renderFuncCallAt renders one FuncCall, tracking the AST depth so
// templates can branch on whether they're being rendered at the
// top level (depth == 0) or inside another block's body (depth > 0).
// The depth is exposed to the inner DSL as the integer local `depth`
// plus the boolean convenience `top_level` (depth == 0).
func (e *outerEval) renderFuncCallAt(c domain.FuncCall, depth int) (string, error) {
	if err := e.validateArgs(c); err != nil {
		return "", err
	}
	var bodyOutput string
	if c.Body != nil {
		if c.Body.IsVerbatim {
			// Verbatim blocks bypass nested rendering — their raw text
			// IS the body output.
			bodyOutput = c.Body.VerbatimText
		} else {
			s, err := e.renderBlockAt(*c.Body, depth+1)
			if err != nil {
				return "", err
			}
			bodyOutput = s
		}
	}
	// Multi-section blocks (try/rescue/finally): render each parsed
	// section sub-body independently so the template can place them via a
	// local named after the section keyword (${rescue}, ${finally}).
	var sectionOutputs map[string]string
	if len(c.Sections) > 0 {
		sectionOutputs = make(map[string]string, len(c.Sections))
		for name, blk := range c.Sections {
			if blk == nil {
				continue
			}
			s, err := e.renderBlockAt(*blk, depth+1)
			if err != nil {
				return "", err
			}
			sectionOutputs[name] = s
		}
	}
	out, err := e.renderTemplateAt(c, bodyOutput, sectionOutputs, depth)
	if err != nil {
		return "", err
	}
	if c.Func.RunAST != nil {
		// Expose the rendered inner-block output to the run pass as
		// `body` so state-mutation statements can stash the rendered
		// text into context (e.g. CSS-rule accumulation). Do NOT
		// shadow a user-defined capture also named `body` — captures
		// take precedence.
		runLocals := map[string]any{}
		if _, shadowed := c.Captures["body"]; !shadowed {
			runLocals["body"] = bodyOutput
		}
		if err := e.inner.ExecWithLocals(*c.Func.RunAST, c.Captures, runLocals); err != nil {
			return "", fmt.Errorf("function %q run: %v", c.Func.Name, err)
		}
	}
	if c.Closer != nil {
		s, err := e.renderFuncCallAt(*c.Closer, depth)
		if err != nil {
			return "", err
		}
		out += s
	}
	return out, nil
}

func (e *outerEval) renderTemplate(c domain.FuncCall, body string) (string, error) {
	return e.renderTemplateAt(c, body, nil, e.depth)
}

func (e *outerEval) renderTemplateAt(c domain.FuncCall, body string, sections map[string]string, depth int) (string, error) {
	if c.Func.TemplateAST == nil {
		return "", nil
	}
	locals := map[string]any{
		"body":      body,
		"depth":     int64(depth),
		"top_level": depth == 0,
		"line":      int64(c.Line),
		"col":       int64(c.Col),
	}
	// Seed every declared section local to "" so a template referencing
	// `${rescue}` renders empty (not undefined) when that section is
	// omitted at the call site, then overlay the rendered sub-bodies.
	if c.Func.Block != nil {
		for _, name := range c.Func.Block.Sections {
			locals[name] = ""
		}
	}
	for k, v := range sections {
		locals[k] = v
	}
	for k, v := range c.Captures {
		// Function-typed captures (named nonterminals): render each
		// matched sub-FuncCall and concatenate. The result is the
		// target text the sub-construct produces.
		if v.Sub != nil {
			// An optional `join "X"` on the capture inserts X between the
			// rendered sub-results (default: no separator).
			var join string
			for _, a := range c.Func.Args {
				if a.Kind == "capture" && a.Name == k {
					join = a.Join
					break
				}
			}
			var sb strings.Builder
			for i, sub := range v.Sub {
				s, err := e.renderFuncCallAt(sub, depth)
				if err != nil {
					return "", err
				}
				if i > 0 {
					sb.WriteString(join)
				}
				sb.WriteString(s)
			}
			locals[k] = sb.String()
			continue
		}
		// Templates always see the source-text form of a capture.
		// This is the transpiler model: what the user wrote appears
		// in the target unless a helper transforms it.
		locals[k] = v.Text
	}
	return e.inner.RenderAST(*c.Func.TemplateAST, locals)
}

// validateArgs walks the function's args and validates each capture against
// its declared type. Type checks are transpile-aware: a bare identifier (a
// VarRef) is accepted by every primitive type because at the target
// language's runtime it could refer to a value of that type. Library-defined
// types apply pattern/options to the source text.
func (e *outerEval) validateArgs(c domain.FuncCall) error {
	for _, a := range c.Func.Args {
		if a.Kind != "capture" {
			continue
		}
		cap, ok := c.Captures[a.Name]
		if !ok {
			return fmt.Errorf("function %q: missing capture %q", c.Func.Name, a.Name)
		}
		// Function-typed captures (named nonterminals) are structural
		// matches, not flat tokens — their type names a library function,
		// not a built-in/declared type, so the textual type check below
		// does not apply.
		if cap.Sub != nil {
			continue
		}
		if _, isFunc := e.lib.Functions[a.Type]; isFunc {
			continue
		}
		// A bare identifier reference is accepted by any primitive type.
		if cap.IsExpr {
			if _, isVar := cap.Expr.(domain.VarRef); isVar {
				// Library types still enforce their pattern/options.
				if _, isLibType := e.lib.Types[a.Type]; !isLibType {
					continue
				}
			}
		}
		if err := e.checkType(a.Type, cap.Text); err != nil {
			// Preserve the inner CapyError's hint when wrapping with the
			// function + argument context.
			if ce, ok := err.(*domain.CapyError); ok {
				wrap := &domain.CapyError{
					Msg:  fmt.Sprintf("function %q arg %q: %s", c.Func.Name, a.Name, ce.Msg),
					Hint: ce.Hint,
				}
				return wrap
			}
			return fmt.Errorf("function %q arg %q: %v", c.Func.Name, a.Name, err)
		}
	}
	return nil
}

// checkType validates a capture's source text against its declared type.
// Built-in kinds inspect the textual form (e.g. `int` requires the text to
// look like an integer literal); library-defined types apply pattern/options.
func (e *outerEval) checkType(t string, text string) error {
	switch t {
	case "", "any", "raw", "ident", "tail", "word", "dotted_ident":
		// Free-form token captures — the parser already enforced their
		// shape at capture time, so any captured text is valid here.
		return nil
	case "string":
		// String captures parse as StringLit and the source-text form is
		// quoted (e.g. `"alice"`). Accept any quoted string.
		if len(text) >= 2 && (text[0] == '"' || text[0] == '\'' || text[0] == '`') {
			return nil
		}
		return fmt.Errorf("expected string literal, got %q", text)
	case "int":
		if _, err := strconv.ParseInt(text, 10, 64); err == nil {
			return nil
		}
		return fmt.Errorf("expected int literal, got %q", text)
	case "float":
		if _, err := strconv.ParseFloat(text, 64); err == nil {
			return nil
		}
		return fmt.Errorf("expected float literal, got %q", text)
	case "bool":
		if text == "true" || text == "false" {
			return nil
		}
		return fmt.Errorf("expected bool literal, got %q", text)
	}
	// Library-defined type
	td, ok := e.lib.Types[t]
	if !ok {
		return fmt.Errorf("unknown type %q", t)
	}
	// Group types: the parser already enforced the delimiters at
	// capture time, so the captured text is valid by construction.
	if td.GroupOpen != "" {
		return nil
	}
	if td.Base != "" && td.Base != "any" {
		if err := e.checkType(td.Base, text); err != nil {
			return err
		}
	}
	// Apply pattern against the un-quoted form of strings, otherwise against the text.
	probe := text
	if len(text) >= 2 && (text[0] == '"' || text[0] == '\'' || text[0] == '`') {
		if u, err := strconv.Unquote(text); err == nil {
			probe = u
		} else if text[0] == '\'' || text[0] == '`' {
			// strconv.Unquote doesn't accept '/` — strip manually
			probe = text[1 : len(text)-1]
		}
	}
	if td.Pattern != "" {
		rx, err := regexp.Compile(td.Pattern)
		if err != nil {
			return fmt.Errorf("type %q has bad regex: %v", td.Name, err)
		}
		if !rx.MatchString(probe) {
			ce := &domain.CapyError{Msg: fmt.Sprintf("value %q does not match pattern for type %q", probe, td.Name)}
			ce.Hint = fmt.Sprintf("type %q requires the value to match regex /%s/", td.Name, td.Pattern)
			return ce
		}
	}
	if len(td.Options) > 0 {
		ok := false
		for _, opt := range td.Options {
			if probe == opt {
				ok = true
				break
			}
		}
		if !ok {
			ce := &domain.CapyError{Msg: fmt.Sprintf("value %q is not in options for type %q", probe, td.Name)}
			if best := domain.SuggestClosest(probe, td.Options, 2); best != "" {
				ce.Hint = fmt.Sprintf("did you mean %q? valid options: %s", best, strings.Join(td.Options, ", "))
			} else {
				ce.Hint = fmt.Sprintf("valid options: %s", strings.Join(td.Options, ", "))
			}
			return ce
		}
	}
	return nil
}

func deepCopyMap(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = deepCopyAny(v)
	}
	return out
}
func deepCopyAny(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return deepCopyMap(x)
	case []any:
		out := make([]any, len(x))
		for i, it := range x {
			out[i] = deepCopyAny(it)
		}
		return out
	}
	return v
}
