package infra

import (
	"strings"
	"testing"
)

func TestCapyLib_Minimal(t *testing.T) {
	src := "\n" +
		"extension py\n" +
		"\n" +
		"function greet\n" +
		"    arg literal \"greet\"\n" +
		"    arg capture name ident\n" +
		"    write `print('hi ${name}')\n`\n" +
		"end\n"
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
	if !strings.Contains(g.Body, "print('hi") {
		t.Errorf("body: %q", g.Body)
	}
}

func TestCapyLib_FunctionBlock(t *testing.T) {
	src := "\n" +
		"extension c\n" +
		"\n" +
		"function fn\n" +
		"    arg literal \"fn\"\n" +
		"    arg capture name ident\n" +
		"    block_closer end\n" +
		"    write `int ${name}() {\n" +
		"${indent 4 body}\n" +
		"}\n`\n" +
		"end\n" +
		"\n" +
		"function end\n" +
		"end\n"
	lib, err := parseCapyLib(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	fn := lib.Functions["fn"]
	if fn.Block == nil || fn.Block.Closer != "end" {
		t.Errorf("block: %+v", fn.Block)
	}
	if !strings.Contains(fn.Body, "int ${name}()") {
		t.Errorf("body missing function header: %q", fn.Body)
	}
}

func TestCapyLib_FileTemplate(t *testing.T) {
	src := "\n" +
		"extension go\n" +
		"\n" +
		"function noop\n" +
		"    arg literal \"noop\"\n" +
		"end\n" +
		"\n" +
		"file_template\n" +
		"    write `package main\n" +
		"\n" +
		"${body}`\n" +
		"end\n"
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
	toks, err := tokenizeLibLine(`    arg capture name "hello \"world\""`)
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 4 || toks[2] != "name" || toks[3] != `hello "world"` {
		t.Errorf("tokens: %#v", toks)
	}
}

func TestCapyLib_TypeBlocks(t *testing.T) {
	src := `
extension cfg

type Email
    pattern "^[^@]+@[^@]+\\.[^@]+$"
end

type LogLevel
    options "debug" "info" "warn" "error"
end

type Port
    base int
end

function noop
    arg literal "noop"
end
`
	lib, err := parseCapyLib(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(lib.Types) != 3 {
		t.Fatalf("expected 3 types, got %d", len(lib.Types))
	}
	if lib.Types["Email"].Pattern == "" {
		t.Errorf("Email pattern missing: %+v", lib.Types["Email"])
	}
	if got := lib.Types["LogLevel"].Options; len(got) != 4 || got[0] != "debug" {
		t.Errorf("LogLevel options: %+v", got)
	}
	if lib.Types["Port"].Base != "int" {
		t.Errorf("Port base: %q", lib.Types["Port"].Base)
	}
}

func TestCapyLib_ContextBlock(t *testing.T) {
	src := `
extension py

context
    imports []
    counter 0
    name "default"
    flag true
end

function noop
    arg literal "noop"
end
`
	lib, err := parseCapyLib(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got, ok := lib.Context["imports"].([]any); !ok || len(got) != 0 {
		t.Errorf("imports: %T %v", lib.Context["imports"], lib.Context["imports"])
	}
	if lib.Context["counter"] != int64(0) {
		t.Errorf("counter: %#v", lib.Context["counter"])
	}
	if lib.Context["name"] != "default" {
		t.Errorf("name: %#v", lib.Context["name"])
	}
	if lib.Context["flag"] != true {
		t.Errorf("flag: %#v", lib.Context["flag"])
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
