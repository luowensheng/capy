package orchfeatures

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luowensheng/capy/infra"
)

// runYAML loads a library + runs a script and returns (output, err).
func runYAML(t *testing.T, libYAML, script string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	libP := filepath.Join(dir, "lib.yaml")
	if err := os.WriteFile(libP, []byte(libYAML), 0644); err != nil {
		t.Fatal(err)
	}
	lex := MakeLexer()
	parser := MakeParser()
	tpl := MakeTemplateRenderer(infra.TemplateEngine{})
	eval := MakeEvaluator(tpl)
	loader := MakeLibraryLoader(infra.YamlParser{}, lex.Tokenize)
	lib, err := loader.Load(libP)
	if err != nil {
		return "", err
	}
	toks, err := lex.Tokenize(script)
	if err != nil {
		return "", err
	}
	prog, err := parser.Parse(toks, lib)
	if err != nil {
		return "", err
	}
	return eval.Run(prog, lib)
}

func TestEval_BasicTemplateRender(t *testing.T) {
	lib := `
functions:
  greet:
    args: [{ kind: capture, name: n, type: any }]
    template: "hi {{ .n }}\n"
`
	out, err := runYAML(t, lib, `greet "Alice"`+"\n")
	if err != nil {
		t.Fatal(err)
	}
	if out != "hi \"Alice\"\n" {
		t.Fatalf("got %q", out)
	}
}

func TestEval_ContextAccumulation(t *testing.T) {
	lib := `
context:
  imports: []
functions:
  import:
    args:
      - { kind: literal, value: "import" }
      - { kind: capture, name: n, type: ident }
    template: ""
    run: |
      append context.imports n
file_template: |
  {{- range .context.imports }}import {{ . }}
  {{ end }}
`
	out, err := runYAML(t, lib, "import json\nimport os\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "import json") || !strings.Contains(out, "import os") {
		t.Fatalf("missing accumulated imports in output: %q", out)
	}
}

func TestEval_TypeValidation_Failure(t *testing.T) {
	lib := `
types:
  Email:
    pattern: "^[^@]+@[^@]+\\.[^@]+$"
functions:
  set_email:
    args: [{ kind: capture, name: e, type: Email }]
    template: "ok\n"
`
	_, err := runYAML(t, lib, `set_email "not-an-email"`+"\n")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "Email") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEval_TypeValidation_Success(t *testing.T) {
	lib := `
types:
  Email:
    pattern: "^[^@]+@[^@]+\\.[^@]+$"
functions:
  set_email:
    args: [{ kind: capture, name: e, type: Email }]
    template: "email={{ .e }}\n"
`
	out, err := runYAML(t, lib, `set_email "alice@example.com"`+"\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "alice@example.com") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestEval_BlockBodyRendering(t *testing.T) {
	lib := `
functions:
  end: {}
  if:
    args:
      - { kind: literal, value: "if" }
      - { kind: capture, name: c, type: any }
    block: { closer: end }
    template: |
      if {{ .c }}:
      {{ .body | indent 2 }}
  say:
    args: [{ kind: capture, name: m, type: any }]
    template: "  say {{ .m }}\n"
`
	out, err := runYAML(t, lib, `if x
    say "hello"
end
`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "if x:") {
		t.Fatalf("missing 'if x:' in output: %q", out)
	}
	if !strings.Contains(out, "say \"hello\"") {
		t.Fatalf("missing body in output: %q", out)
	}
}

func TestEval_DelimBlock(t *testing.T) {
	lib := `
functions:
  for:
    args:
      - { kind: literal, value: "for" }
      - { kind: capture, name: v, type: ident }
      - { kind: literal, value: "in" }
      - { kind: capture, name: i, type: any }
    block: { open: "{", close: "}" }
    template: "for {{ .v }} in {{ .i }} { {{ .body }} }\n"
  say:
    args: [{ kind: capture, name: m, type: any }]
    template: "  say {{ .m }}\n"
`
	out, err := runYAML(t, lib, "for x in 40 {\n    say x\n}\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "for x in 40") {
		t.Fatalf("missing header in output: %q", out)
	}
	if !strings.Contains(out, "say x") {
		t.Fatalf("missing body in output: %q", out)
	}
}

func TestEval_UnknownStatement(t *testing.T) {
	lib := `functions: {}`
	_, err := runYAML(t, lib, "z = 9\n")
	if err == nil || !strings.Contains(err.Error(), "no library function matches") {
		t.Fatalf("expected no-match error, got %v", err)
	}
}

func TestEval_RegexMatch(t *testing.T) {
	lib := `
context:
  ok: false
functions:
  check:
    args: [{ kind: capture, name: v, type: any }]
    template: ""
    run: |
      if (regex_match v "^[a-z]+$")
          set context.ok true
      end
file_template: "{{ .context.ok }}\n"
`
	out, err := runYAML(t, lib, `check "hello"`+"\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "true" {
		t.Fatalf("expected true, got %q", out)
	}
}

func TestEval_StatusEnumType(t *testing.T) {
	lib := `
types:
  Status:
    options: ["todo", "done"]
functions:
  set:
    args: [{ kind: capture, name: s, type: Status }]
    template: "{{ .s }}\n"
`
	_, err := runYAML(t, lib, `set "in-progress"`+"\n")
	if err == nil || !strings.Contains(err.Error(), "options") {
		t.Fatalf("expected options error, got %v", err)
	}
}
