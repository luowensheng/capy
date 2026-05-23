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

	"github.com/luowensheng/capy"
	"github.com/luowensheng/capy/domain"
)

func main() {
	js.Global().Set("capyRun", js.FuncOf(capyRun))
	js.Global().Set("capyVersion", js.FuncOf(func(this js.Value, args []js.Value) any {
		return "0.9.0"
	}))
	// Block forever so the module stays alive for callbacks.
	<-make(chan struct{})
}

// capyRun(libSrc, format, scriptSrc) → result object.
//
// Arguments (positional):
//
//	0: library source (string) — YAML or .capy native syntax.
//	1: format (string) — "auto" | "yaml" | "capy". "auto" sniffs from
//	   the first non-comment line.
//	2: script source (string).
//
// File-import directives (`@import "..."`) are NOT expanded in the
// browser — there's no filesystem. Use a single-file script.
func capyRun(this js.Value, args []js.Value) any {
	if len(args) < 3 {
		return errResult("capyRun expects (libSrc, format, scriptSrc)", "")
	}
	libSrc := args[0].String()
	format := args[1].String()
	scriptSrc := args[2].String()

	if format == "" || format == "auto" {
		if sniffCapy(libSrc) {
			format = "capy"
		} else {
			format = "yaml"
		}
	}

	var lib *capy.Library
	var err error
	switch format {
	case "capy":
		lib, err = capy.NewLibrary(libSrc)
	case "yaml":
		lib, err = capy.NewLibraryYAML(libSrc)
	default:
		return errResult("unknown format: "+format, `use "auto", "yaml", or "capy"`)
	}
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
