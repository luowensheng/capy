//go:build js && wasm

// capy-wasm is the browser entry point. It exposes a single global
// function `capyRun(libSrc, format, scriptSrc)` returning an object:
//
//	{ ok: true,  output: "...", files: { "path": "...", ... } }
//	{ ok: false, error: "...", hint: "..." }
//
// The playground at docs/assets/playground/index.html loads this module
// via `wasm_exec.js` and calls capyRun on the user's input.
//
// Build:
//
//	GOOS=js GOARCH=wasm go build -o docs/assets/playground/capy.wasm ./cmd/capy-wasm
package main

import (
	"strings"
	"syscall/js"

	"github.com/olivierdevelops/capy"
	"github.com/olivierdevelops/capy/domain"
)

// version is set at build time via:
//
//	go build -ldflags="-X main.version=v0.12.0" ...
//
// scripts/build-playground.sh and the docs CI workflow pass this so the
// playground label tracks the current engine. Falls back to "dev" for
// local one-off builds.
var version = "dev"

func main() {
	js.Global().Set("capyRun", js.FuncOf(capyRun))
	js.Global().Set("capyDocs", js.FuncOf(capyDocs))
	js.Global().Set("capyIntrospect", js.FuncOf(capyIntrospect))
	js.Global().Set("capyVersion", js.FuncOf(func(this js.Value, args []js.Value) any {
		return version
	}))
	// Block forever so the module stays alive for callbacks.
	<-make(chan struct{})
}

// capyIntrospect(libSrc) → { ok, functions:[…], comments:[…] }
//
// Returns the declared functions (name / description / args / block /
// priority) and comment markers of the supplied library, so an editor
// can derive autocomplete / hover-docs / highlighting instead of
// hand-maintaining a parallel catalogue.
func capyIntrospect(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errResult("capyIntrospect expects (libSrc)", "")
	}
	libSrc := args[0].String()
	lib, err := capy.NewLibrary(libSrc)
	if err != nil {
		return errResultFromErr(err, libSrc)
	}
	fns := lib.Introspect()
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
	markers := lib.CommentMarkers()
	jsMarkers := make([]any, len(markers))
	for i, m := range markers {
		jsMarkers[i] = m
	}
	return js.ValueOf(map[string]any{
		"ok":        true,
		"functions": out,
		"comments":  jsMarkers,
	})
}

// capyDocs(libSrc, format) → { ok, docs } or { ok:false, error, hint }
//
// Renders Markdown reference documentation for the supplied library.
// Same engine path as `capy docs <library>` on the CLI.
func capyDocs(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errResult("capyDocs expects (libSrc)", "")
	}
	libSrc := args[0].String()
	lib, err := capy.NewLibrary(libSrc)
	if err != nil {
		return errResultFromErr(err, libSrc)
	}
	return js.ValueOf(map[string]any{
		"ok":   true,
		"docs": capy.RenderLibraryDocs(lib),
	})
}

// capyRun(libSrc, _format, scriptSrc) → result object.
//
// Arguments (positional):
//
//	0: library source (string) — .capy native syntax.
//	1: format (string) — ignored, kept for JS-side compatibility.
//	2: script source (string).
//
// File-import directives (`@import "..."`) are NOT expanded in the
// browser — there's no filesystem. Use a single-file script.
func capyRun(this js.Value, args []js.Value) any {
	if len(args) < 3 {
		return errResult("capyRun expects (libSrc, format, scriptSrc)", "")
	}
	libSrc := args[0].String()
	scriptSrc := args[2].String()

	lib, err := capy.NewLibrary(libSrc)
	if err != nil {
		return errResultFromErr(err, scriptSrc)
	}

	out, files, err := lib.RunMulti(scriptSrc)
	if err != nil {
		return errResultFromErr(err, scriptSrc)
	}

	// Convert map[string]string → map[string]any for js.ValueOf.
	jsFiles := make(map[string]any, len(files))
	for k, v := range files {
		jsFiles[k] = v
	}

	return js.ValueOf(map[string]any{
		"ok":        true,
		"output":    out,
		"files":     jsFiles,
		"extension": lib.Extension(),
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
		// Render with the caret-pointed source view in the `pretty`
		// field so the playground can show it.
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

// sniffCapy is duplicated from cmd/capy-mcp to avoid a dependency-graph
// detour. Kept small on purpose.
func sniffCapy(src string) bool {
	for _, line := range strings.Split(src, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if strings.HasPrefix(s, "function ") || strings.HasPrefix(s, "type ") ||
			strings.HasPrefix(s, "extension ") || strings.HasPrefix(s, "output_file ") ||
			strings.HasPrefix(s, "file_template:") || strings.HasPrefix(s, "context") {
			return true
		}
		return false
	}
	return false
}
