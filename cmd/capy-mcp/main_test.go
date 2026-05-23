package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// callTool drives a single tools/call through handle() and returns the text
// content (or error text).
func callTool(t *testing.T, name string, args map[string]any) (string, bool) {
	t.Helper()
	params, _ := json.Marshal(map[string]any{"name": name, "arguments": args})
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params:  params,
	}
	resp := handle(req)
	if resp.Error != nil {
		t.Fatalf("rpc error: %s", resp.Error.Message)
	}
	res, ok := resp.Result.(toolResult)
	if !ok {
		t.Fatalf("expected toolResult, got %T", resp.Result)
	}
	if len(res.Content) == 0 {
		return "", res.IsError
	}
	return res.Content[0].Text, res.IsError
}

func TestMCP_Initialize(t *testing.T) {
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
	}
	resp := handle(req)
	if resp.Error != nil {
		t.Fatalf("error: %v", resp.Error)
	}
	r := resp.Result.(map[string]any)
	if r["protocolVersion"] != protocolVersion {
		t.Errorf("protocol: %v", r["protocolVersion"])
	}
}

func TestMCP_ToolsList(t *testing.T) {
	resp := handle(rpcRequest{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "tools/list"})
	r := resp.Result.(map[string]any)
	tools := r["tools"].([]toolDef)
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}
	names := map[string]bool{}
	for _, tl := range tools {
		names[tl.Name] = true
	}
	for _, want := range []string{"capy_run", "capy_run_file", "capy_check"} {
		if !names[want] {
			t.Errorf("missing tool: %s", want)
		}
	}
}

func TestMCP_CapyRun_Inline(t *testing.T) {
	lib := `
extension html

function h1
    arg literal "h1"
    arg capture t string
    template_str "<h1>{{ .t }}</h1>\n"
end
`
	out, isErr := callTool(t, "capy_run", map[string]any{
		"library": lib,
		"script":  `h1 "Hi"` + "\n",
	})
	if isErr {
		t.Fatalf("unexpected error: %s", out)
	}
	if !strings.Contains(out, "<h1>") {
		t.Errorf("output: %q", out)
	}
}

func TestMCP_CapyCheck_Valid(t *testing.T) {
	lib := `
extension txt
function greet
    arg literal "greet"
    arg capture who any
    template_str "hello {{ .who }}\n"
end
`
	out, isErr := callTool(t, "capy_check", map[string]any{"library": lib})
	if isErr {
		t.Fatalf("isError set: %s", out)
	}
	if !strings.Contains(out, `"valid": true`) {
		t.Errorf("expected valid:true, got %s", out)
	}
	if !strings.Contains(out, `"greet"`) {
		t.Errorf("expected greet in functions: %s", out)
	}
}

func TestMCP_CapyCheck_Invalid(t *testing.T) {
	out, _ := callTool(t, "capy_check", map[string]any{
		"library": "this is not a library",
	})
	if !strings.Contains(out, `"valid": false`) {
		t.Errorf("expected valid:false, got %s", out)
	}
}

func TestMCP_CapyRun_YAMLFormat(t *testing.T) {
	yamlLib := `
extension: txt
functions:
  shout:
    args:
      - { kind: literal, value: "shout" }
      - { kind: capture, name: msg, type: any }
    template: "{{ .msg }}!\n"
`
	out, isErr := callTool(t, "capy_run", map[string]any{
		"library": yamlLib,
		"script":  `shout "hey"` + "\n",
		"format":  "yaml",
	})
	if isErr {
		t.Fatalf("error: %s", out)
	}
	if !strings.Contains(out, "!") {
		t.Errorf("output: %q", out)
	}
}

func TestSniffCapy(t *testing.T) {
	cases := []struct {
		src  string
		want bool
	}{
		{"# comment\nfunction foo\n    arg literal \"foo\"\nend\n", true},
		{"extension py\n", true},
		{"extension: py\nfunctions:\n  foo: {}\n", false},
		{"functions:\n  foo: {}\n", false},
		{"", false},
	}
	for i, c := range cases {
		if got := sniffCapy(c.src); got != c.want {
			t.Errorf("case %d: sniffCapy = %v, want %v\nsrc: %q", i, got, c.want, c.src)
		}
	}
}
