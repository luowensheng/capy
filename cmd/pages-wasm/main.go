//go:build js && wasm

// pages-wasm is the WebAssembly engine for the Pages note app.
//
// Unlike cmd/capy-wasm (which takes the library at runtime), pages-wasm
// has the Pages library (pages.capy) compiled in via go:embed. The
// browser only needs to ship script source.
//
// Exposed globals:
//
//	pagesRender(scriptSrc string) -> { ok, html, error?, hint?, line?, col?, pretty? }
//	pagesVersion() -> string
//
// Build:
//
//	# pages.capy must be copied into this directory before building.
//	# scripts/build-pages-wasm.sh in the pages repo handles that.
//	GOOS=js GOARCH=wasm go build -o pages.wasm ./cmd/pages-wasm
package main

import (
	_ "embed"
	"syscall/js"

	"github.com/olivierdevelops/capy"
	"github.com/olivierdevelops/capy/domain"
)

//go:embed pages.capy
var pagesLib string

// version is set at build time via -ldflags="-X main.version=...".
var version = "dev"

// compiledLib is the singleton Library compiled once at startup.
// We keep one library across all render calls; Capy guarantees Library
// is safe to reuse (each Run gets a fresh context).
var compiledLib *capy.Library

func main() {
	lib, err := capy.NewLibrary(pagesLib)
	if err != nil {
		// Library is baked in — a failure here is a build-time bug. Surface
		// it through the render function so the JS side can show it.
		js.Global().Set("pagesRender", js.FuncOf(func(this js.Value, args []js.Value) any {
			return errResultFromErr(err, pagesLib)
		}))
	} else {
		compiledLib = lib
		js.Global().Set("pagesRender", js.FuncOf(pagesRender))
		js.Global().Set("pagesIntrospect", js.FuncOf(pagesIntrospect))
		js.Global().Set("pagesCommentMarkers", js.FuncOf(pagesCommentMarkers))
	}
	js.Global().Set("pagesVersion", js.FuncOf(func(this js.Value, args []js.Value) any {
		return version
	}))
	<-make(chan struct{})
}

// pagesIntrospect returns the embedded library's declared functions
// as a JS array of { name, description, args:[{kind,value,name,type,
// description}], block, priority }. The editor derives its
// autocomplete / hover-docs / highlight metadata from this instead of
// hand-maintaining a parallel catalogue.
func pagesIntrospect(this js.Value, args []js.Value) any {
	if compiledLib == nil {
		return js.ValueOf([]any{})
	}
	fns := compiledLib.Introspect()
	out := make([]any, 0, len(fns))
	for _, fn := range fns {
		argList := make([]any, 0, len(fn.Args))
		for _, a := range fn.Args {
			argList = append(argList, map[string]any{
				"kind":        a.Kind,
				"value":       a.Value,
				"name":        a.Name,
				"type":        a.Type,
				"description": a.Description,
			})
		}
		out = append(out, map[string]any{
			"name":        fn.Name,
			"description": fn.Description,
			"args":        argList,
			"block":       fn.Block,
			"priority":    fn.Priority,
		})
	}
	return js.ValueOf(out)
}

// pagesCommentMarkers returns the library's declared line-comment
// markers so the highlighter doesn't hardcode them.
func pagesCommentMarkers(this js.Value, args []js.Value) any {
	if compiledLib == nil {
		return js.ValueOf([]any{})
	}
	markers := compiledLib.CommentMarkers()
	out := make([]any, len(markers))
	for i, m := range markers {
		out[i] = m
	}
	return js.ValueOf(out)
}

func pagesRender(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errResult("pagesRender expects (scriptSrc)", "")
	}
	scriptSrc := args[0].String()
	if scriptSrc == "" {
		return js.ValueOf(map[string]any{"ok": true, "html": ""})
	}
	out, _, err := compiledLib.RunMulti(scriptSrc)
	if err != nil {
		return errResultFromErr(err, scriptSrc)
	}
	return js.ValueOf(map[string]any{
		"ok":   true,
		"html": out,
	})
}

func errResult(msg, hint string) js.Value {
	return js.ValueOf(map[string]any{
		"ok":    false,
		"error": msg,
		"hint":  hint,
	})
}

func errResultFromErr(err error, source string) js.Value {
	if ce, ok := err.(*domain.CapyError); ok {
		pretty := domain.FormatWithSource(ce, source)
		return js.ValueOf(map[string]any{
			"ok":     false,
			"error":  ce.Msg,
			"hint":   ce.Hint,
			"line":   ce.Line,
			"col":    ce.Col,
			"pretty": pretty,
		})
	}
	return js.ValueOf(map[string]any{
		"ok":    false,
		"error": err.Error(),
	})
}
