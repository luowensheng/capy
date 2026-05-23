package orchfeatures

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luowensheng/capy/infra"
)

func loadLibYAML(t *testing.T, body string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "lib.yaml")
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	loader := MakeLibraryLoader(infra.YamlParser{}, MakeLexer().Tokenize)
	_, err := loader.Load(p)
	if err != nil {
		return p, err
	}
	return p, nil
}

func TestLoader_RejectsMissingKind(t *testing.T) {
	_, err := loadLibYAML(t, `
functions:
  greet:
    args:
      - { value: "hi" }
`)
	if err == nil || !strings.Contains(err.Error(), "unknown or missing kind") {
		t.Fatalf("expected kind validation error, got %v", err)
	}
}

func TestLoader_RejectsLiteralWithName(t *testing.T) {
	_, err := loadLibYAML(t, `
functions:
  greet:
    args:
      - { kind: literal, value: "hi", name: x }
`)
	if err == nil || !strings.Contains(err.Error(), "cannot have name/type") {
		t.Fatalf("expected literal validation error, got %v", err)
	}
}

func TestLoader_RejectsCaptureWithValue(t *testing.T) {
	_, err := loadLibYAML(t, `
functions:
  greet:
    args:
      - { kind: capture, name: x, value: "oops" }
`)
	if err == nil || !strings.Contains(err.Error(), "cannot have value") {
		t.Fatalf("expected capture validation error, got %v", err)
	}
}

func TestLoader_AutoNamePrepend(t *testing.T) {
	// Library uses only captures → function name is auto-prepended as literal.
	dir := t.TempDir()
	p := filepath.Join(dir, "lib.yaml")
	yaml := `
functions:
  greet:
    args:
      - { kind: capture, name: n, type: any }
    template: "hi {{ .n }}\n"
`
	if err := os.WriteFile(p, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	loader := MakeLibraryLoader(infra.YamlParser{}, MakeLexer().Tokenize)
	lib, err := loader.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	fn := lib.Functions["greet"]
	if fn == nil {
		t.Fatal("greet not loaded")
	}
	if len(fn.Elements) < 1 || fn.Elements[0].IsCapture || fn.Elements[0].Literal != "greet" {
		t.Fatalf("expected leading literal 'greet', got %+v", fn.Elements)
	}
}

func TestLoader_UnknownType(t *testing.T) {
	_, err := loadLibYAML(t, `
functions:
  greet:
    args:
      - { kind: capture, name: n, type: NoSuchType }
`)
	if err == nil || !strings.Contains(err.Error(), "unknown type") {
		t.Fatalf("expected unknown-type error, got %v", err)
	}
}

func TestLoader_MissingCloser(t *testing.T) {
	_, err := loadLibYAML(t, `
functions:
  if:
    args:
      - { kind: literal, value: "if" }
      - { kind: capture, name: c, type: any }
    block: { closer: never_defined }
`)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected closer-not-found error, got %v", err)
	}
}

func TestLoader_BlockBothCloserAndDelim(t *testing.T) {
	_, err := loadLibYAML(t, `
functions:
  end: {}
  if:
    args:
      - { kind: literal, value: "if" }
      - { kind: capture, name: c, type: any }
    block: { closer: end, open: "{", close: "}" }
`)
	if err == nil || !strings.Contains(err.Error(), "either `closer:` OR both") {
		t.Fatalf("expected mutual-exclusion error, got %v", err)
	}
}
