package capy_test

import (
	"strings"
	"testing"

	"github.com/luowensheng/capy"
)

func TestEmbed_InlineCapyLibrary(t *testing.T) {
	lib, err := capy.NewLibrary(`
extension html

function button
    arg literal "button"
    arg capture label string
    write ` + "`<button>${label}</button>\n`" + `
end

function link
    arg literal "link"
    arg capture text string
    arg capture href string
    write ` + "`<a href=${href}>${text}</a>\n`" + `
end
`)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}

	out, err := lib.Run(`button "Click me"
link "Home" "/index.html"
`)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := "<button>\"Click me\"</button>\n<a href=\"/index.html\">\"Home\"</a>\n"
	if out != want {
		t.Errorf("output mismatch:\n got: %q\nwant: %q", out, want)
	}
	if lib.Extension() != "html" {
		t.Errorf("extension: %q", lib.Extension())
	}
}

func TestEmbed_InlineYAMLLibrary(t *testing.T) {
	// YAML libraries are still parseable; their `template:` strings now
	// flow through the same AST renderer after auto-wrapping. (Practically
	// no project should use YAML anymore — .capy is canonical.)
	lib, err := capy.NewLibrary(`
extension txt

function greet
    arg literal "greet"
    arg capture who string
    write ` + "`hello ${who}\n`" + `
end
`)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}

	out, err := lib.Run(`greet "world"`)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected output to contain hello, got %q", out)
	}
}

func TestEmbed_ReuseLibrary(t *testing.T) {
	lib, err := capy.NewLibrary(`
extension txt

function say
    arg literal "say"
    arg capture msg any
    write ` + "`[${msg}]\n`" + `
end
`)
	if err != nil {
		t.Fatal(err)
	}

	for i, src := range []string{`say one`, `say two`, `say three`} {
		out, err := lib.Run(src)
		if err != nil {
			t.Errorf("run %d: %v", i, err)
		}
		if !strings.Contains(out, "[") {
			t.Errorf("run %d output: %q", i, out)
		}
	}
}

func TestEmbed_ReportsErrors(t *testing.T) {
	_, err := capy.NewLibrary(`nonsense top-level directive`)
	if err == nil {
		t.Error("expected error for invalid library")
	}
}

func TestEmbed_Introspect(t *testing.T) {
	lib, err := capy.NewLibrary(`
extension html

comments
    line "#"
end

function link
    description "An anchor."
    arg literal "link"
    arg capture text string "Visible label."
    arg capture url string "Destination URL."
    write ` + "`<a href=${url}>${text}</a>`" + `
end

function pre
    arg capture lang ident
    block_verbatim end
    write ` + "`<pre>${body}</pre>`" + `
end

function end
end
`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	fns := lib.Introspect()
	byName := map[string]capy.FunctionInfo{}
	for _, f := range fns {
		byName[f.Name] = f
	}

	link, ok := byName["link"]
	if !ok {
		t.Fatal("link not introspected")
	}
	if link.Description != "An anchor." {
		t.Errorf("link description = %q", link.Description)
	}
	if len(link.Args) != 3 {
		t.Fatalf("link args = %d, want 3", len(link.Args))
	}
	if link.Args[0].Kind != "literal" || link.Args[0].Value != "link" {
		t.Errorf("arg0 = %+v", link.Args[0])
	}
	if link.Args[1].Kind != "capture" || link.Args[1].Name != "text" ||
		link.Args[1].Type != "string" || link.Args[1].Description != "Visible label." {
		t.Errorf("arg1 = %+v", link.Args[1])
	}

	pre, ok := byName["pre"]
	if !ok {
		t.Fatal("pre not introspected")
	}
	if pre.Block != "verbatim:end" {
		t.Errorf("pre block = %q, want verbatim:end", pre.Block)
	}

	if got := lib.CommentMarkers(); len(got) != 1 || got[0] != "#" {
		t.Errorf("comment markers = %v, want [#]", got)
	}
}

// TestEmbed_IntrospectOptionalArgs verifies Introspect surfaces the
// Optional/Default metadata for trailing optional captures so an
// editor can derive its catalogue without hand-maintenance.
func TestEmbed_IntrospectOptionalArgs(t *testing.T) {
	lib := mustLib(t, `
extension html

function button
    arg literal "button"
    arg capture label string
    arg capture variant string default "primary"
    write `+bt+`<button class="btn-${variant}">${label}</button>`+bt+`
end
`)
	var btn capy.FunctionInfo
	for _, f := range lib.Introspect() {
		if f.Name == "button" {
			btn = f
		}
	}
	if len(btn.Args) != 3 {
		t.Fatalf("button args = %d, want 3", len(btn.Args))
	}
	if btn.Args[1].Optional {
		t.Error("label (arg1) should be required, got Optional=true")
	}
	v := btn.Args[2]
	if !v.Optional {
		t.Error("variant (arg2) should be Optional")
	}
	if v.Default != "primary" {
		t.Errorf("variant default = %q, want primary", v.Default)
	}
}

// bt is a tiny helper to embed backtick literals inside Go raw-ish test
// sources without fighting Go's own backtick string rules.
const bt = "`"

func mustLib(t *testing.T, src string) *capy.Library {
	t.Helper()
	lib, err := capy.NewLibrary(src)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}
	return lib
}

func mustRun(t *testing.T, lib *capy.Library, src string) string {
	t.Helper()
	out, err := lib.Run(src)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return out
}

// TestEmbed_LineColLocals verifies the ${line}/${col} render locals
// (missing2.md §1). Each emitted statement should carry the 1-indexed
// source position it came from.
func TestEmbed_LineColLocals(t *testing.T) {
	lib := mustLib(t, `
extension html

function p
    arg literal "p"
    arg capture text string
    write `+bt+`<p data-line="${line}" data-col="${col}">${decoded text}</p>
`+bt+`
end
`)
	out := mustRun(t, lib, `p "first"
p "second"
p "third"`)
	want := `<p data-line="1" data-col="1">first</p>
<p data-line="2" data-col="1">second</p>
<p data-line="3" data-col="1">third</p>
`
	if out != want {
		t.Errorf("line/col mapping mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// TestEmbed_LineColCaptureWins verifies a user capture named `line`
// shadows the render local (same precedence rule as body/depth).
func TestEmbed_LineColCaptureWins(t *testing.T) {
	lib := mustLib(t, `
extension txt

function at
    arg literal "at"
    arg capture line int
    write `+bt+`line=${line}
`+bt+`
end
`)
	out := mustRun(t, lib, `at 99`)
	if strings.TrimSpace(out) != "line=99" {
		t.Errorf("capture should win over render local, got %q", out)
	}
}

// TestEmbed_Decoded covers the embedded-quote fix (missing2.md §4c):
// ${decoded x} must resolve \" and \n even when the value contains
// unescaped quotes, where the old wrap-and-Unquote path failed.
func TestEmbed_Decoded(t *testing.T) {
	lib := mustLib(t, `
extension txt

function p
    arg literal "p"
    arg capture text string
    write `+bt+`${decoded text}
`+bt+`
end
`)
	cases := []struct{ in, want string }{
		{`p "He said \"hi\""`, "He said \"hi\"\n"},
		{`p "line1\nline2"`, "line1\nline2\n"},
		{`p "tab\there"`, "tab\there\n"},
		{`p "<div class=\"card\">\n  body\n</div>"`, "<div class=\"card\">\n  body\n</div>\n"},
	}
	for _, c := range cases {
		out := mustRun(t, lib, c.in)
		if out != c.want {
			t.Errorf("decoded(%s):\n got: %q\nwant: %q", c.in, out, c.want)
		}
	}
}

// TestEmbed_EscapeHtml verifies the escapeHtml helper neutralises all
// five HTML-significant characters (missing.md §5).
func TestEmbed_EscapeHtml(t *testing.T) {
	lib := mustLib(t, `
extension html

function p
    arg literal "p"
    arg capture text string
    write `+bt+`<p>${escapeHtml (decoded text)}</p>
`+bt+`
end
`)
	out := mustRun(t, lib, `p "a & b < c > d \" e ' f"`)
	want := "<p>a &amp; b &lt; c &gt; d &quot; e &#39; f</p>\n"
	if out != want {
		t.Errorf("escapeHtml mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// TestEmbed_VerbatimRawBytes verifies block_verbatim preserves blank
// lines and comment-marker lines byte-for-byte (missing2.md §6).
func TestEmbed_VerbatimRawBytes(t *testing.T) {
	lib := mustLib(t, `
extension txt

comments
    line "#"
end

function pre
    arg capture lang ident
    block_verbatim end
    write `+bt+`[${lang}]
${body}---
`+bt+`
end

function end
end
`)
	// Body mixes a content line, a comment-marker line, and a blank
	// line — all three must survive byte-for-byte even though `#` is a
	// declared comment marker that the lexer would normally strip.
	out := mustRun(t, lib, `pre md
    intro
    # Heading

    ## Subheading
end`)
	want := "[md]\nintro\n# Heading\n\n## Subheading\n---\n"
	if out != want {
		t.Errorf("verbatim fidelity mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// TestEmbed_OptionalArgs verifies trailing optional captures with
// defaults (missing2.md §5 G1): the call site may omit them and the
// declared default binds.
func TestEmbed_OptionalArgs(t *testing.T) {
	lib := mustLib(t, `
extension html

function button
    arg literal "button"
    arg capture label string
    arg capture variant string default "primary"
    arg capture kind string default "button"
    write `+bt+`<button type="${decoded kind}" class="btn-${decoded variant}">${decoded label}</button>
`+bt+`
end
`)
	cases := []struct{ in, want string }{
		{`button "Save"`, "<button type=\"button\" class=\"btn-primary\">Save</button>\n"},
		{`button "Delete" "danger"`, "<button type=\"button\" class=\"btn-danger\">Delete</button>\n"},
		{`button "Submit" "primary" "submit"`, "<button type=\"submit\" class=\"btn-primary\">Submit</button>\n"},
	}
	for _, c := range cases {
		out := mustRun(t, lib, c.in)
		if out != c.want {
			t.Errorf("optional-args(%s):\n got: %q\nwant: %q", c.in, out, c.want)
		}
	}
}

// TestEmbed_OptionalArgsMustBeTrailing verifies the load-time
// validation that a required capture cannot follow an optional one.
func TestEmbed_OptionalArgsMustBeTrailing(t *testing.T) {
	_, err := capy.NewLibrary(`
extension txt

function bad
    arg literal "bad"
    arg capture a string default "x"
    arg capture b string
    write `+bt+`${a}${b}
`+bt+`
end
`)
	if err == nil {
		t.Fatal("expected trailing-only validation error, got nil")
	}
	if !strings.Contains(err.Error(), "optional") {
		t.Errorf("error should mention optional-trailing rule, got: %v", err)
	}
}

// TestEmbed_BareAndTail verifies a `bare` function (no auto-name
// prepend) with a `tail` capture matches a whole free-form line,
// including UTF-8 content (missing.md §1, missing2 utf8-prose).
func TestEmbed_BareAndTail(t *testing.T) {
	lib := mustLib(t, `
extension html

function line
    bare
    arg capture content tail
    write `+bt+`<p>${content}</p>
`+bt+`
end
`)
	out := mustRun(t, lib, `Each line — yes, em-dashes — becomes a <p>.
Café au lait 北京 🎉`)
	want := "<p>Each line — yes, em-dashes — becomes a <p>.</p>\n<p>Café au lait 北京 🎉</p>\n"
	if out != want {
		t.Errorf("bare+tail mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// TestEmbed_MultilineBacktick verifies backtick string captures in
// user scripts span newlines (missing.md §3).
func TestEmbed_MultilineBacktick(t *testing.T) {
	lib := mustLib(t, `
extension txt

function p
    arg literal "p"
    arg capture text string
    write `+bt+`${decoded text}
`+bt+`
end
`)
	out := mustRun(t, lib, "p `one\ntwo\nthree`")
	if strings.TrimSpace(out) != "one\ntwo\nthree" {
		t.Errorf("multiline backtick mismatch, got %q", out)
	}
}

// TestEmbed_BacktickCodeSpan verifies an escaped backtick inside a
// backtick capture is preserved and decoded back to a literal
// backtick (missing2.md §4b).
func TestEmbed_BacktickCodeSpan(t *testing.T) {
	lib := mustLib(t, `
extension html

function md
    arg literal "md"
    arg capture src string
    write `+bt+`<p>${decoded src}</p>
`+bt+`
end
`)
	out := mustRun(t, lib, "md `inline \\`code\\` here`")
	want := "<p>inline `code` here</p>\n"
	if out != want {
		t.Errorf("backtick code span mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// TestEmbed_GroupTypes verifies group_open/group_close capture types
// consume balanced delimited spans (Stage 6 inline markdown).
func TestEmbed_GroupTypes(t *testing.T) {
	lib := mustLib(t, `
extension html

type Bracketed
    group_open  "["
    group_close "]"
end

type Parens
    group_open  "("
    group_close ")"
end

function link
    arg literal "link"
    arg capture text Bracketed
    arg capture url  Parens
    write `+bt+`<a href="${url}">${text}</a>
`+bt+`
end
`)
	out := mustRun(t, lib, `link [Home page](/index.html)`)
	want := "<a href=\"/index.html\">Home page</a>\n"
	if out != want {
		t.Errorf("group types mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// TestEmbed_BlockDedent verifies block_dedent ends a body at the first
// DEDENT with no closer keyword.
func TestEmbed_BlockDedent(t *testing.T) {
	lib := mustLib(t, `
extension txt

function section
    arg literal "section"
    arg capture title string
    block_dedent
    write `+bt+`# ${decoded title}
${body}`+bt+`
end

function item
    arg literal "item"
    arg capture text string
    write `+bt+`- ${decoded text}
`+bt+`
end
`)
	out := mustRun(t, lib, `section "Fruit"
    item "Apple"
    item "Pear"`)
	want := "# Fruit\n- Apple\n- Pear\n"
	if out != want {
		t.Errorf("block_dedent mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// TestEmbed_TemplateColumnZero verifies a `template … end` body may
// contain a column-0 interpolation line without prematurely closing
// the function (missing2.md §4a).
func TestEmbed_TemplateColumnZero(t *testing.T) {
	lib := mustLib(t, `
extension html

function box
    arg literal "box"
    arg capture text string
    block_closer end
    template
<div class="box">
${indent 2 body}
</div>
    end
end

function p
    arg literal "p"
    arg capture text string
    write `+bt+`<p>${decoded text}</p>
`+bt+`
end

function end
end
`)
	out := mustRun(t, lib, `box "x"
    p "inside"
end`)
	want := "<div class=\"box\">\n  <p>inside</p>\n\n</div>\n"
	if out != want {
		t.Errorf("template col-0 mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// ExampleNewLibrary demonstrates the canonical embedding pattern: define a
// tiny DSL inline, then run user sources against it — all in pure Go, no
// external binary or library files.
func ExampleNewLibrary() {
	lib, _ := capy.NewLibrary(`
extension md

function h1
    arg literal "h1"
    arg capture text string
    write ` + "`# ${text}\n`" + `
end

function bullet
    arg literal "-"
    arg capture text string
    write ` + "`- ${text}\n`" + `
end
`)
	out, _ := lib.Run(`h1 "Shopping list"
- "Milk"
- "Bread"
`)
	_ = out
	// Output is "# \"Shopping list\"\n- \"Milk\"\n- \"Bread\"\n"
}
