package capy_test

import (
	"strings"
	"testing"
)

// Source-level metaprogramming: a script may declare its own DSL
// primitives via top-level `define NAME ... end` blocks, then use them.
// The embedded API (NewLibrary + Run/RunMulti) extracts and merges those
// defines before evaluation — the same behavior the CLI has.
//
// These tests pin commit 05175a6 ("support source-level metaprogramming
// via capy.Library.Run/RunMulti"), which previously had no coverage. The
// original bug: the wasm playground rejected the metaprogramming sample
// with `unexpected character ''` because an em-dash inside an
// un-extracted define-block template reached the lexer.

// TestEmbed_Metaprogramming mirrors the deployed playground sample: a
// deliberately minimal library (just `print`) plus a source that defines
// `heading` and `quote` inline, then uses both.
func TestEmbed_Metaprogramming(t *testing.T) {
	lib := mustLib(t, `
extension md

comments
    line "#"
end

function print
    arg literal "print"
    arg capture text string
    write `+bt+`${unquote text}
`+bt+`
end
`)

	out := mustRun(t, lib, `# This source extends the grammar inline.
define heading
    arg literal "heading"
    arg capture text string
    write `+bt+`# ${unquote text}
`+bt+`
end

define quote
    arg literal "quote"
    arg capture text string
    arg capture who string
    write `+bt+`> ${unquote text}
> — ${unquote who}
`+bt+`
end

heading "Metaprogramming"
print "A library primitive still works."
quote "Get out of the way." "an enthusiast"`)

	want := "# Metaprogramming\n" +
		"A library primitive still works.\n" +
		"> Get out of the way.\n> — an enthusiast\n"
	if out != want {
		t.Errorf("metaprogramming mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// TestEmbed_MetaprogrammingEmDash is the focused regression for the
// `unexpected character ''` lexer failure: an em-dash (and other non-ASCII
// prose) living inside a define-block template must survive extraction and
// render byte-for-byte. Before 05175a6 the embedded path never extracted
// the define block, so the lexer met the em-dash as a bare statement and
// errored.
func TestEmbed_MetaprogrammingEmDash(t *testing.T) {
	lib := mustLib(t, `
extension md

function print
    arg literal "print"
    arg capture text string
    write `+bt+`${unquote text}
`+bt+`
end
`)

	out, err := lib.Run(`define note
    arg literal "note"
    arg capture text string
    write `+bt+`— ${unquote text} — Café 北京 🎉
`+bt+`
end

note "em-dash prose"`)
	if err != nil {
		t.Fatalf("metaprogramming with non-ASCII template errored (the '' lexer bug): %v", err)
	}
	want := "— em-dash prose — Café 北京 🎉\n"
	if out != want {
		t.Errorf("em-dash template mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// TestEmbed_MetaprogrammingSourceWins verifies that a source `define`
// overrides a library function of the same name (source defines WIN on
// conflict, matching CLI behavior).
func TestEmbed_MetaprogrammingSourceWins(t *testing.T) {
	lib := mustLib(t, `
extension txt

function greet
    arg literal "greet"
    arg capture who string
    write `+bt+`LIB: ${unquote who}
`+bt+`
end
`)

	out := mustRun(t, lib, `define greet
    arg literal "greet"
    arg capture who string
    write `+bt+`SRC: ${unquote who}
`+bt+`
end

greet "world"`)
	if strings.TrimSpace(out) != "SRC: world" {
		t.Errorf("source define should override library function, got %q", out)
	}
}

// TestEmbed_MetaprogrammingRunMulti verifies the multi-file entry point
// also extracts defines (Run delegates to RunMulti, but exercise it
// directly so the file-map return path is covered too).
func TestEmbed_MetaprogrammingRunMulti(t *testing.T) {
	lib := mustLib(t, `
extension txt

function print
    arg literal "print"
    arg capture text string
    write `+bt+`${unquote text}
`+bt+`
end
`)

	out, files, err := lib.RunMulti(`define shout
    arg literal "shout"
    arg capture text string
    write `+bt+`${unquote text}!
`+bt+`
end

shout "hi"
print "bye"`)
	if err != nil {
		t.Fatalf("RunMulti: %v", err)
	}
	if want := "hi!\nbye\n"; out != want {
		t.Errorf("RunMulti output:\n got: %q\nwant: %q", out, want)
	}
	// This single-file library declares no `file "...":` blocks, so the
	// multi-file map must be empty.
	if len(files) != 0 {
		t.Errorf("expected empty file map, got %d entries: %v", len(files), files)
	}
}

// TestEmbed_NoDefinesUnchanged guards the common path: a script with no
// `define` blocks must behave exactly as before (extraction is a no-op).
func TestEmbed_NoDefinesUnchanged(t *testing.T) {
	lib := mustLib(t, `
extension txt

function print
    arg literal "print"
    arg capture text string
    write `+bt+`${unquote text}
`+bt+`
end
`)
	out := mustRun(t, lib, `print "plain"`)
	if strings.TrimSpace(out) != "plain" {
		t.Errorf("no-defines path changed, got %q", out)
	}
}

// TestEmbed_MalformedDefineErrors verifies a broken define block surfaces
// a load error rather than silently degrading.
func TestEmbed_MalformedDefineErrors(t *testing.T) {
	lib := mustLib(t, `
extension txt

function print
    arg literal "print"
    arg capture text string
    write `+bt+`${unquote text}
`+bt+`
end
`)
	_, err := lib.Run(`define broken
    arg capture x nonsense_type
    write `+bt+`${x}
`+bt+`
end

broken "y"`)
	if err == nil {
		t.Fatal("expected error from malformed define block, got nil")
	}
}
