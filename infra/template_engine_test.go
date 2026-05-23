package infra

import (
	"strings"
	"testing"
)

func TestTemplate_Indent(t *testing.T) {
	out, err := TemplateEngine{}.Render(`{{ "a\nb\nc" | indent 2 }}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "  a\n  b\n  c" {
		t.Fatalf("got %q", out)
	}
}

func TestTemplate_LowerUpper(t *testing.T) {
	out, err := TemplateEngine{}.Render(`{{ .s | upper }}-{{ .s | lower }}`, map[string]any{"s": "Hello"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "HELLO-hello" {
		t.Fatalf("got %q", out)
	}
}

func TestTemplate_ToQuoted(t *testing.T) {
	out, err := TemplateEngine{}.Render(`{{ .s | toQuoted }}`, map[string]any{"s": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if out != `"hi"` {
		t.Fatalf("got %q", out)
	}
}

func TestTemplate_ToPyLit(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, "None"},
		{true, "True"},
		{false, "False"},
		{"hi", `"hi"`},
		{[]any{int64(1), int64(2)}, "[1, 2]"},
	}
	for _, c := range cases {
		out, err := TemplateEngine{}.Render(`{{ .v | toPyLit }}`, map[string]any{"v": c.in})
		if err != nil {
			t.Fatalf("%v: err %v", c.in, err)
		}
		// Numbers route through the catchall path; integers default to "%v".
		if c.want != "" && !strings.Contains(out, c.want) && out != c.want {
			t.Fatalf("toPyLit(%v) = %q want %q", c.in, out, c.want)
		}
	}
}

func TestTemplate_ToJSONIndent(t *testing.T) {
	out, err := TemplateEngine{}.Render(`{{ .v | toJSONIndent }}`, map[string]any{
		"v": map[string]any{"k": "v", "n": float64(2)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"k": "v"`) {
		t.Fatalf("got %q", out)
	}
}

func TestTemplate_Join(t *testing.T) {
	out, err := TemplateEngine{}.Render(`{{ join "," .xs }}`, map[string]any{
		"xs": []any{"a", "b", "c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "a,b,c" {
		t.Fatalf("got %q", out)
	}
}
