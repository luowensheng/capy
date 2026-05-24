package infra

import (
	"strings"
	"testing"
)

func TestExtractDefines_None(t *testing.T) {
	src := "foo bar\nbaz\n"
	out, lib, err := ExtractDefines(src)
	if err != nil {
		t.Fatal(err)
	}
	if out != src {
		t.Errorf("source should be unchanged when no defines:\n got %q", out)
	}
	if lib != "" {
		t.Errorf("expected empty library, got: %s", lib)
	}
}

func TestExtractDefines_OneBlock(t *testing.T) {
	src := `define greet
    arg literal "greet"
    arg capture name string
    template_str "Hello, {{ .name | unquote }}!\n"
end

greet "World"
greet "Alice"
`
	cleaned, lib, err := ExtractDefines(src)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cleaned, "define greet") {
		t.Errorf("define block was not removed from source:\n%s", cleaned)
	}
	if !strings.Contains(cleaned, `greet "World"`) {
		t.Errorf("calls were stripped along with the define:\n%s", cleaned)
	}
	if !strings.Contains(lib, "function greet") {
		t.Errorf("synthetic library missing the function:\n%s", lib)
	}
}

func TestExtractDefines_MultipleBlocks(t *testing.T) {
	src := `define a
    arg literal "a"
    template_str "a\n"
end

define b
    arg literal "b"
    template_str "b\n"
end

a
b
`
	cleaned, lib, err := ExtractDefines(src)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cleaned, "define ") {
		t.Errorf("not all defines stripped: %s", cleaned)
	}
	if !strings.Contains(lib, "function a") || !strings.Contains(lib, "function b") {
		t.Errorf("library missing one of the functions:\n%s", lib)
	}
}

func TestExtractDefines_UnclosedBlock(t *testing.T) {
	src := `define greet
    arg literal "greet"
    template_str ""
`
	_, _, err := ExtractDefines(src)
	if err == nil || !strings.Contains(err.Error(), "missing matching `end`") {
		t.Errorf("expected missing-end error, got %v", err)
	}
}

func TestExtractDefines_MalformedBody(t *testing.T) {
	src := `define greet
    arg whatever "x"
end
`
	_, _, err := ExtractDefines(src)
	if err == nil {
		t.Errorf("expected error for malformed body")
	}
}

func TestExtractDefines_BadName(t *testing.T) {
	src := `define "not-an-ident"
    template_str ""
end
`
	// "define" followed by a string literal isn't a valid name; the
	// extractor should refuse it rather than passing garbage to the
	// .capy library parser.
	_, _, err := ExtractDefines(src)
	if err == nil {
		t.Errorf("expected error for non-identifier define name")
	}
}

func TestExtractDefines_IgnoresIndentedDefine(t *testing.T) {
	// A `define` that's indented (inside a block body) is part of the
	// surrounding template — not a new top-level define.
	src := `function outer
    template:
        define this should not be parsed as a block
end
`
	out, lib, err := ExtractDefines(src)
	if err != nil {
		t.Fatal(err)
	}
	if lib != "" {
		t.Errorf("expected no defines extracted, got: %s", lib)
	}
	if !strings.Contains(out, "define this") {
		t.Errorf("indented `define` was incorrectly stripped:\n%s", out)
	}
}
