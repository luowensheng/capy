package infra

import (
	"strings"
	"testing"
)

func TestCapyLib_Minimal(t *testing.T) {
	src := `
extension py

function greet
    arg literal "greet"
    arg capture name ident
    template_str "print('hi {{ .name }}')\n"
end
`
	lib, err := parseCapyLib(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if lib.Extension != "py" {
		t.Errorf("extension: got %q", lib.Extension)
	}
	g, ok := lib.Functions["greet"]
	if !ok {
		t.Fatal("missing greet function")
	}
	if len(g.Args) != 2 || g.Args[0].Kind != "literal" || g.Args[0].Value != "greet" {
		t.Errorf("args: %+v", g.Args)
	}
	if g.Args[1].Kind != "capture" || g.Args[1].Name != "name" || g.Args[1].Type != "ident" {
		t.Errorf("capture arg: %+v", g.Args[1])
	}
	if !strings.Contains(g.Template, "print('hi") {
		t.Errorf("template: %q", g.Template)
	}
}

func TestCapyLib_TemplateBlock(t *testing.T) {
	src := `
extension c

function fn
    arg literal "fn"
    arg capture name ident
    block_closer end
    template:
        int {{ .name }}() {
        {{ .body | indent 4 }}
        }
end

function end
end
`
	lib, err := parseCapyLib(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	fn := lib.Functions["fn"]
	if fn.Block == nil || fn.Block.Closer != "end" {
		t.Errorf("block: %+v", fn.Block)
	}
	want := "int {{ .name }}() {\n{{ .body | indent 4 }}\n}\n"
	if fn.Template != want {
		t.Errorf("template:\n  got:  %q\n  want: %q", fn.Template, want)
	}
}

func TestCapyLib_FileTemplate(t *testing.T) {
	src := `
extension go

function noop
    arg literal "noop"
    template_str ""
end

file_template:
    package main

    {{ .body }}
`
	lib, err := parseCapyLib(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(lib.FileTemplate, "package main") {
		t.Errorf("file_template: %q", lib.FileTemplate)
	}
}

func TestCapyLib_Priority(t *testing.T) {
	src := `
function f
    priority 100
    arg literal "f"
end
`
	lib, err := parseCapyLib(src)
	if err != nil {
		t.Fatal(err)
	}
	if lib.Functions["f"].Priority != 100 {
		t.Errorf("priority: %d", lib.Functions["f"].Priority)
	}
}

func TestCapyLib_TokenizerQuotes(t *testing.T) {
	toks, err := tokenizeLibLine(`    template_str "hello \"world\"\n"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 2 || toks[0] != "template_str" || toks[1] != "hello \"world\"\n" {
		t.Errorf("tokens: %#v", toks)
	}
}

func TestCapyLib_FileTemplate_KeepsActionAtColumnZero(t *testing.T) {
	// An author writes `{{ .body | indent 4 }}` at column 0 so the
	// rendered output has clean nested indentation. The parser must
	// NOT shift that line right when it strips the file_template's
	// base indent.
	src := `
extension py

function noop
    arg literal "noop"
    template_str ""
end

file_template:
    void Start()
    {
{{ .body | indent 8 }}
    }
`
	lib, err := parseCapyLib(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !strings.Contains(lib.FileTemplate, "\n{{ .body | indent 8 }}\n") {
		t.Errorf("action at col 0 was shifted:\n%s", lib.FileTemplate)
	}
	if !strings.Contains(lib.FileTemplate, "void Start()\n") {
		t.Errorf("base indent was not stripped:\n%s", lib.FileTemplate)
	}
}

func TestCapyLib_RejectsUnknownTop(t *testing.T) {
	_, err := parseCapyLib("nonsense foo\n")
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected unknown directive error, got %v", err)
	}
}

func TestCapyLib_RejectsUnterminatedString(t *testing.T) {
	_, err := parseCapyLib(`
function f
    arg literal "oops
end
`)
	if err == nil || !strings.Contains(err.Error(), "unterminated") {
		t.Errorf("expected unterminated string error, got %v", err)
	}
}
