package capy_test

import (
	"strings"
	"testing"

	"github.com/olivierdevelops/capy"
)

// (bt — a single backtick string — is declared in capy_test.go.)

// multitokenCloserLib is a minimal angle-bracket HTML grammar built on
// `block_close_seq`: each tag opens on its literal `<tag>` sequence and
// closes on the multi-token sequence `</tag>`. It exercises the
// sequence-closed block mode end to end — matched pairs, attributes on
// the opener, free-flowing (non-indented) bodies, and adjacent close
// tags (`</p></div>`) that the greedy-punct lexer merges into one token.
const multitokenCloserLib = `
extension html

function div
    arg literal "<"
    arg literal "div"
    arg literal "class"
    arg literal "="
    arg capture cls string
    arg literal ">"
    block_close_seq "</div>"
    write ` + bt + `<div class="${decoded cls}">${body}</div>` + bt + `
end

function p
    arg literal "<"
    arg literal "p"
    arg literal ">"
    block_close_seq "</p>"
    write ` + bt + `<p>${body}</p>` + bt + `
end

function b
    arg literal "<"
    arg literal "b"
    arg literal ">"
    block_close_seq "</b>"
    write ` + bt + `<b>${body}</b>` + bt + `
end

function text
    bare
    arg capture s raw
    write ` + bt + `${escapeHtml s}` + bt + `
end
`

func TestMultiTokenCloserMatchedPairs(t *testing.T) {
	lib, err := capy.NewLibrary(multitokenCloserLib)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}
	cases := map[string]string{
		// Simple matched pair.
		`<p>"hi"</p>`: "<p>hi</p>",
		// Opener with an attribute capture.
		`<div class="card">"x"</div>`: `<div class="card">x</div>`,
		// Nested tags + adjacent close tags (`</p></div>` merges in the lexer).
		`<div class="card"><p>"x"</p></div>`: `<div class="card"><p>x</p></div>`,
		// Mixed text and inline tags, the canonical webview_gui example.
		`<div class="card"><p>"Hello, "<b>"world"</b>"."</p></div>`: `<div class="card"><p>Hello, <b>world</b>.</p></div>`,
		// Escaping still runs on text nodes.
		`<p>"a < b & c"</p>`: "<p>a &lt; b &amp; c</p>",
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

func TestMultiTokenCloserMismatchDetection(t *testing.T) {
	lib, err := capy.NewLibrary(multitokenCloserLib)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}
	// Each block demands its OWN closing sequence, so a stray or wrong
	// closer cannot satisfy a different opener — mismatched nesting is a
	// hard parse error, exactly HTML's contract.
	bad := []string{
		`<div class="card"><p>"hi"</div>`, // missing </p>
		`<p>"hi"</b>`,                      // wrong closer
		`<p>"hi"`,                          // unclosed (EOF)
	}
	for _, src := range bad {
		if _, err := lib.Run(src); err == nil {
			t.Errorf("Run(%q): expected mismatch error, got success", src)
		}
	}
}
