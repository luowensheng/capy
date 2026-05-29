package capy_test

import (
	"strings"
	"testing"

	"github.com/luowensheng/capy"
)

// (bt — a single backtick string — is declared in capy_test.go.)

// genericHTMLLib is a GENERIC angle-bracket HTML grammar: a single
// `element` function opens on `<NAME ...>` (NAME captured as an ident),
// matches zero-or-more `attribute` nonterminals via a function-typed
// repetition (`attribute*`), and closes on the capture-bound multi-token
// sequence `</NAME>` (`block_close_seq "</" name ">"`). One function thus
// covers every well-formed paired tag — `<div>`, `<p>`, `<span class="x">`
// — and a stray `</p>` inside a `<div>` is a hard parse error.
//
// Attribute VALUES use the `raw` capture type (one ident/string token),
// not `string`: a `string` capture runs through the expression parser,
// which would treat the tag-closing `>` as a comparison operator and
// swallow it. `raw` consumes exactly one token, leaving `>` for the
// opener's literal — the right primitive for delimited nonterminal pieces.
const genericHTMLLib = `
extension html

function element
    arg literal "<"
    arg capture name ident
    arg capture attrs attribute*
    arg literal ">"
    block_close_seq "</" name ">"
    write ` + bt + `<${name}${attrs}>${body}</${name}>` + bt + `
end

function attribute
    arg capture key ident
    arg literal "="
    arg capture val raw
    write ` + bt + ` ${key}="${val}"` + bt + `
end

function text
    bare
    arg capture s raw
    write ` + bt + `${escapeHtml s}` + bt + `
end
`

func TestFuncTypeGenericHTML(t *testing.T) {
	lib, err := capy.NewLibrary(genericHTMLLib)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}
	cases := map[string]string{
		// Bare paired tag, generic opener + capture-bound closer.
		`<p>"hi"</p>`: "<p>hi</p>",
		// One attribute via the attribute* nonterminal.
		`<div class="card">"x"</div>`: `<div class="card">x</div>`,
		// Multiple attributes (repetition matches each in turn).
		`<input type="text" name="email">"v"</input>`: `<input type="text" name="email">v</input>`,
		// Nested tags, each closed by its own capture-bound sequence.
		`<div class="card"><p>"x"</p></div>`: `<div class="card"><p>x</p></div>`,
		// Mixed text + inline tags.
		`<div class="card"><p>"Hello, "<b>"world"</b>"."</p></div>`: `<div class="card"><p>Hello, <b>world</b>.</p></div>`,
	}
	for src, want := range cases {
		out, err := lib.Run(src)
		if err != nil {
			t.Fatalf("Run(%q): %v", src, err)
		}
		if got := strings.TrimSpace(out); got != want {
			t.Errorf("Run(%q) = %q, want %q", src, got, want)
		}
	}
}

func TestFuncTypeGenericHTMLMismatch(t *testing.T) {
	lib, err := capy.NewLibrary(genericHTMLLib)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}
	// The capture-bound closer means `<div>` is closed ONLY by `</div>`:
	// a wrong or missing closer is a hard parse error (mismatched nesting).
	bad := []string{
		`<div class="card"><p>"hi"</div>`, // inner <p> never closed
		`<p>"hi"</b>`,                      // wrong closer tag name
		`<p>"hi"`,                          // unclosed at EOF
	}
	for _, src := range bad {
		if _, err := lib.Run(src); err == nil {
			t.Errorf("Run(%q): expected mismatch error, got success", src)
		}
	}
}

// cellLib's `cell` nonterminal carries a `#` literal prefix so it can only
// match a deliberate `#N` token pair — never a stray identifier. That keeps
// the error-path tests honest: when a `+`/mandatory match fails there is no
// bare fallback to silently absorb the leftover token.
const repetitionLib = `
extension txt

function row
    arg literal "row"
    arg capture items cell+ sep ","
    write ` + bt + `[${items}]` + bt + `
end

function cell
    arg literal "#"
    arg capture v int
    write ` + bt + `(${v})` + bt + `
end
`

func TestFuncTypeRepetitionWithSep(t *testing.T) {
	lib, err := capy.NewLibrary(repetitionLib)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}
	cases := map[string]string{
		`row #1`:         `[(1)]`,
		`row #1, #2, #3`: `[(1)(2)(3)]`,
	}
	for src, want := range cases {
		out, err := lib.Run(src)
		if err != nil {
			t.Fatalf("Run(%q): %v", src, err)
		}
		if got := strings.TrimSpace(out); got != want {
			t.Errorf("Run(%q) = %q, want %q", src, got, want)
		}
	}
}

// onePlusRequiresOne: a `+` quantifier demands at least one occurrence.
const onePlusRequiresOneLib = `
extension txt

function row
    arg literal "row"
    arg capture items cell+
    write ` + bt + `[${items}]` + bt + `
end

function cell
    arg literal "#"
    arg capture v int
    write ` + bt + `(${v})` + bt + `
end
`

func TestFuncTypePlusRequiresOne(t *testing.T) {
	lib, err := capy.NewLibrary(onePlusRequiresOneLib)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}
	if _, err := lib.Run(`row`); err == nil {
		t.Errorf("Run(%q): expected error for empty `+` repetition, got success", "row")
	}
}

// starAllowsZero: a `*` quantifier matches zero occurrences cleanly.
const starAllowsZeroLib = `
extension txt

function row
    arg literal "row"
    arg capture items cell*
    write ` + bt + `[${items}]` + bt + `
end

function cell
    arg literal "#"
    arg capture v int
    write ` + bt + `(${v})` + bt + `
end
`

func TestFuncTypeStarAllowsZero(t *testing.T) {
	lib, err := capy.NewLibrary(starAllowsZeroLib)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}
	out, err := lib.Run(`row`)
	if err != nil {
		t.Fatalf("Run(row): %v", err)
	}
	if got := strings.TrimSpace(out); got != `[]` {
		t.Errorf("Run(row) = %q, want %q", got, `[]`)
	}
}

// exactlyOneLib exercises a non-repeated function-typed capture (exactly
// one occurrence, mandatory).
const exactlyOneLib = `
extension txt

function wrap
    arg literal "wrap"
    arg capture inner cell
    write ` + bt + `<${inner}>` + bt + `
end

function cell
    arg literal "#"
    arg capture v int
    write ` + bt + `(${v})` + bt + `
end
`

func TestFuncTypeExactlyOne(t *testing.T) {
	lib, err := capy.NewLibrary(exactlyOneLib)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}
	out, err := lib.Run(`wrap #7`)
	if err != nil {
		t.Fatalf("Run(wrap #7): %v", err)
	}
	if got := strings.TrimSpace(out); got != `<(7)>` {
		t.Errorf("Run(wrap #7) = %q, want %q", got, `<(7)>`)
	}
	// A missing mandatory nonterminal is an error.
	if _, err := lib.Run(`wrap`); err == nil {
		t.Errorf("Run(wrap): expected error for missing mandatory nonterminal")
	}
}
