---
title: Embed Capy in Go
---

# Embed Capy in a Go program

Capy is a Go library. You don't have to ship the `capy` binary or maintain
separate `lib.capy` files — your program can carry its own grammar inline
and transpile user input at runtime, all in pure Go.

```go
import "github.com/luowensheng/capy"

lib, _ := capy.NewLibrary(`
    extension html

    function button
        arg literal "button"
        arg capture label string
        write `<button>${label}</button>
`
    end
`)

out, _ := lib.Run(`button "Click me"`)
// → <button>"Click me"</button>
```

That's the entire API surface for the common case.

## When to embed Capy

- **Your CLI takes a config file** in a friendlier-than-YAML DSL. Write
  the parser in 50 lines of Capy instead of 500 of `encoding/yaml` +
  string interpolation.
- **Your tool generates code** (think Prisma's `model User { ... }` →
  SQL migrations). Embedding Capy lets users write that DSL natively
  while your Go code consumes the generated output.
- **You want hot-swappable grammars** — read a library file at startup,
  let users contribute new ones without recompiling.
- **You want a sandboxed scripting surface** for an AI agent — let it
  emit Capy DSL, your binary transpiles to whatever target you trust.

## Install

```sh
go get github.com/luowensheng/capy
```

That's it. No CLI dependency, no `capy` binary required at runtime.

## The full API

The `capy` package exposes a tiny, intentionally-stable surface:

```go
// Compile a library from an in-memory string.
func NewLibrary(librarySrc string) (*Library, error)        // .capy native syntax
func NewLibraryFromFile(path string) (*Library, error)      // disk

// Run a source script through the library.
func (l *Library) Run(scriptSrc string) (string, error)

// Diagnostic helpers.
func (l *Library) Extension() string         // declared `extension:` field
func (l *Library) OutputFile() string        // declared `output_file:` field
func (l *Library) FunctionNames() []string   // declared function keys

// Introspection — the library describes itself (see below).
func (l *Library) Introspect() []FunctionInfo // every declared function
func (l *Library) CommentMarkers() []string   // declared comment markers
```

That's the whole package. Everything else is convention.

`*Library` is safe to reuse across many `Run` calls (each call gets a
fresh accumulating context).

## Introspection — the library describes itself

`Introspect()` returns the full declared shape of every function —
name, doc string, argument list (literal vs. capture, capture type,
per-arg description, and whether the arg is **optional** with a
**default**), block kind, and priority. The data comes straight from
the compiled library, so a tool can derive its metadata instead of
hand-maintaining a parallel catalogue that silently drifts.

This is what powers a live editor's autocomplete, hover-docs, syntax
highlighting, and reference panel — all from one source of truth (the
`.capy` library itself).

```go
type FunctionInfo struct {
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    Args        []ArgInfo `json:"args"`
    Block       string    `json:"block,omitempty"`     // e.g. "verbatim:end", "dedent", "closer:end"
    Priority    int       `json:"priority,omitempty"`
}

type ArgInfo struct {
    Kind        string `json:"kind"`                  // "literal" or "capture"
    Value       string `json:"value,omitempty"`       // literal token text
    Name        string `json:"name,omitempty"`        // capture's bound name
    Type        string `json:"type,omitempty"`        // capture's declared type
    Description string `json:"description,omitempty"` // trailing doc string
    Optional    bool   `json:"optional,omitempty"`    // trailing arg with a default
    Default     string `json:"default,omitempty"`     // value bound when omitted
}
```

Example — introspecting a one-function library:

```go
lib, _ := capy.NewLibrary(`
extension html

comments
    line "#"
end

function button
    description "A clickable button."
    arg literal "button"
    arg capture label   string "Visible text."
    arg capture variant string default "primary"   "Style variant."
    write ` + "`<button class=\"btn-${variant}\">${label}</button>`" + `
end
`)

for _, fn := range lib.Introspect() {
    fmt.Println(fn.Name, "-", fn.Description)
    for _, a := range fn.Args {
        if a.Kind == "capture" {
            opt := ""
            if a.Optional {
                opt = fmt.Sprintf(" (optional, default %q)", a.Default)
            }
            fmt.Printf("  %s: %s%s — %s\n", a.Name, a.Type, opt, a.Description)
        }
    }
}
fmt.Println("comment markers:", lib.CommentMarkers())
```

Output:

```
button - A clickable button.
  label: string — Visible text.
  variant: string (optional, default "primary") — Style variant.
comment markers: [#]
```

The same data is available from the browser via the wasm builds —
`capyIntrospect(librarySrc)` in the generic engine, `pagesIntrospect()`
in a library-embedded build — returning the identical JSON shape. An
editor can `JSON.parse` it and build autocomplete with zero
hand-maintenance.

## A real example

The repo ships [`examples/embed-html-dsl/`](https://github.com/luowensheng/capy/tree/main/examples/embed-html-dsl)
— a 50-line Go program that defines its own HTML DSL inline and
transpiles a hardcoded source. Run it:

```sh
go run ./examples/embed-html-dsl
```

Output (real, no escaping omitted):

```html
<!DOCTYPE html>
<html>
  <head><title>"Hello from embedded Capy"</title></head>
  <body>
    <h1>"Welcome!"</h1>
    <p>"This entire page was generated by a Capy library compiled INTO this Go binary."</p>
    <p>"No external lib.capy, no separate capy CLI."</p>
    <a href="https://github.com/luowensheng/capy">"Source"</a>
  </body>
</html>
```

Everything between `<!DOCTYPE html>` and `</html>` came from a Capy
library compiled into the Go binary. No filesystem, no subprocess.

## Patterns

### Pattern 1 — your config is your CLI's input

```go
const configLib = `
extension json

function server
    arg literal "server"
    arg capture name string
    block_closer end
    write `{ "name": ${name}, "routes": [
${indent 4 body}
]}
`
end

function route
    arg literal "route"
    arg capture method ident
    arg capture path string
    write `  { "method": "${method}", "path": ${path} },
`
end

function end
end
`

func loadConfig(userInput string) (*ServerConfig, error) {
    lib, err := capy.NewLibrary(configLib)
    if err != nil { return nil, err }
    jsonStr, err := lib.Run(userInput)
    if err != nil { return nil, err }
    return parseJSON(jsonStr)
}
```

Now your users write:

```
server "api"
    route GET  "/healthz"
    route POST "/v1/users"
end
```

…and your Go binary parses real JSON without you writing any DSL parser
code.

### Pattern 2 — let users extend your tool

Ship a default library compiled into the binary, but allow a user-supplied
override:

```go
func loadLib(path string) (*capy.Library, error) {
    if path != "" {
        return capy.NewLibraryFromFile(path)   // user-provided
    }
    return capy.NewLibrary(builtinLib)         // baked-in default
}
```

The same `*Library` works in both cases.

### Pattern 3 — multiple grammars in one process

Each `*Library` is independent. Run the same source through different
libraries to compare outputs, or pick a grammar at runtime based on
some flag:

```go
htmlLib := mustCompile(htmlGrammar)
mdLib   := mustCompile(markdownGrammar)

switch outputFormat {
case "html": return htmlLib.Run(src)
case "md":   return mdLib.Run(src)
}
```

## Performance notes

- `NewLibrary` compiles the grammar once. Reuse the returned `*Library`
  — don't recompile per request.
- `Run` is allocation-heavy (template rendering, AST walking). For
  hot paths consider caching outputs keyed by source hash.
- No goroutines are spawned. Run is synchronous and CPU-bound.

## Caveats and edge cases

- **String captures preserve their quotes.** When your library declares
  `arg capture text string`, the captured value renders as `"foo"`
  (with quotes) in templates — Capy is a *transpiler*, so input syntax
  is preserved into the output. Use `arg capture text any` if you want
  the bare value, or strip the quotes inline with the `unquote` helper:
  `${text | unquote}`.
- **Errors include line numbers** from the source — wire them through
  to your user-facing error path.
- **There is no eval of user code.** A Capy library defines patterns
  and templates; it cannot execute arbitrary Go from the source
  script. Embedding Capy in a server is safe from that angle.

## What it's not

- Not a Go-side imperative API for building libraries. Libraries are
  always declarative `.capy` text. If you need to construct one
  programmatically, write a Go function that builds the string.
- Not a hot-reload watcher — that's your responsibility. `NewLibrary`
  is cheap to call; just re-call it when the source changes.
- Not concurrent-write-safe. Compile once, then `Run` from many
  goroutines on the same `*Library`.
