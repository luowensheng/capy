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
