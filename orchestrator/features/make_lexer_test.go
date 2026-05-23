package orchfeatures

import (
	"testing"

	"github.com/luowensheng/capy/domain"
)

func TestLexer_IndentDedent(t *testing.T) {
	src := "a\n    b\n        c\n    d\ne\n"
	toks, err := tokenize(src)
	if err != nil {
		t.Fatal(err)
	}
	// Build a comparable trace of token kinds.
	got := traceKinds(toks)
	want := []string{
		"Ident:a", "NL",
		"Indent", "Ident:b", "NL",
		"Indent", "Ident:c", "NL",
		"Dedent", "Ident:d", "NL",
		"Dedent", "Ident:e", "NL",
		"EOF",
	}
	assertTrace(t, got, want)
}

func TestLexer_PunctRuns(t *testing.T) {
	toks, err := tokenize("x == y != z >= w <= v := u")
	if err != nil {
		t.Fatal(err)
	}
	puncts := []string{}
	for _, tk := range toks {
		if tk.Kind == domain.TokPunct {
			puncts = append(puncts, tk.Text)
		}
	}
	want := []string{"==", "!=", ">=", "<=", ":="}
	if len(puncts) != len(want) {
		t.Fatalf("got %v want %v", puncts, want)
	}
	for i := range want {
		if puncts[i] != want[i] {
			t.Fatalf("punct[%d] = %q, want %q", i, puncts[i], want[i])
		}
	}
}

func TestLexer_StringsAndTemplates(t *testing.T) {
	toks, err := tokenize(`a "double" b 'single' c ` + "`template`" + ` d`)
	if err != nil {
		t.Fatal(err)
	}
	kinds := []domain.TokenKind{}
	for _, tk := range toks {
		if tk.Kind == domain.TokString || tk.Kind == domain.TokTemplate {
			kinds = append(kinds, tk.Kind)
		}
	}
	want := []domain.TokenKind{domain.TokString, domain.TokString, domain.TokTemplate}
	if len(kinds) != len(want) {
		t.Fatalf("got %v want %v", kinds, want)
	}
}

func TestLexer_MultiLineObjectBracket(t *testing.T) {
	// Bracket counting allows multi-line literals, but newlines are emitted —
	// value parsers must skip them.
	src := "send {\n  \"k\": 1\n}\n"
	toks, err := tokenize(src)
	if err != nil {
		t.Fatal(err)
	}
	// We should have a TokRBrace at some point, no error.
	saw := false
	for _, tk := range toks {
		if tk.Kind == domain.TokRBrace {
			saw = true
		}
	}
	if !saw {
		t.Fatalf("no closing brace token in output")
	}
}

func TestLexer_BadIndent(t *testing.T) {
	// 3 spaces is not a valid indent
	_, err := tokenize("a\n   b\n")
	if err == nil {
		t.Fatal("expected indent error")
	}
}

func TestLexer_NestedIndentJump(t *testing.T) {
	// Jumping two levels in one step should error.
	_, err := tokenize("a\n        b\n")
	if err == nil {
		t.Fatal("expected indent-jump error")
	}
}

func traceKinds(toks []domain.Token) []string {
	out := []string{}
	for _, tk := range toks {
		switch tk.Kind {
		case domain.TokIdent:
			out = append(out, "Ident:"+tk.Text)
		case domain.TokNewline:
			out = append(out, "NL")
		case domain.TokIndent:
			out = append(out, "Indent")
		case domain.TokDedent:
			out = append(out, "Dedent")
		case domain.TokEOF:
			out = append(out, "EOF")
		case domain.TokNumber:
			out = append(out, "Num:"+tk.Text)
		case domain.TokString:
			out = append(out, "Str:"+tk.Text)
		case domain.TokPunct:
			out = append(out, "Punct:"+tk.Text)
		}
	}
	return out
}

func assertTrace(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("trace length mismatch:\ngot:  %v\nwant: %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("trace[%d] = %q, want %q\nfull got: %v", i, got[i], want[i], got)
		}
	}
}
