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
}

func (e *outerEval) renderBlock(b domain.Block) (string, error) {
	var out strings.Builder
	for _, c := range b.Stmts {
		s, err := e.renderFuncCall(c)
		if err != nil {
			return "", err
		}
		out.WriteString(s)
	}
	return out.String(), nil
}

func (e *outerEval) renderFuncCall(c domain.FuncCall) (string, error) {
	if err := e.validateArgs(c); err != nil {
		return "", err
	}
	var bodyOutput string
	if c.Body != nil {
		s, err := e.renderBlock(*c.Body)
		if err != nil {
			return "", err
		}
		bodyOutput = s
	}
	out, err := e.renderTemplate(c, bodyOutput)
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
		s, err := e.renderFuncCall(*c.Closer)
		if err != nil {
			return "", err
		}
		out += s
	}
	return out, nil
}

func (e *outerEval) renderTemplate(c domain.FuncCall, body string) (string, error) {
	if c.Func.TemplateAST == nil {
		return "", nil
	}
	locals := map[string]any{"body": body}
	for k, v := range c.Captures {
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
	case "", "any", "raw", "ident", "tail":
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
