package domain

import (
	"strings"
	"testing"
)

func TestSuggestClosest_PicksCloseMatch(t *testing.T) {
	got := SuggestClosest("endpiont", []string{"endpoint", "param", "returns"}, 2)
	if got != "endpoint" {
		t.Errorf("expected endpoint, got %q", got)
	}
}

func TestSuggestClosest_NoMatchWhenTooFar(t *testing.T) {
	got := SuggestClosest("xxxxxx", []string{"endpoint", "param"}, 2)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestFormatWithSource_RendersHintAndCaret(t *testing.T) {
	err := NewError(3, 5, "no library function matches token %q", "endpiont")
	err.Hint = "did you mean \"endpoint\"?"
	source := "line one\nline two\n    endpiont GET /users\nline four"
	out := FormatWithSource(err, source)

	if !strings.Contains(out, "error: no library function matches token") {
		t.Errorf("missing error line:\n%s", out)
	}
	if !strings.Contains(out, "hint: did you mean") {
		t.Errorf("missing hint:\n%s", out)
	}
	if !strings.Contains(out, "  3 │     endpiont GET /users") {
		t.Errorf("missing source line:\n%s", out)
	}
	if !strings.Contains(out, "    │     ^") {
		t.Errorf("missing caret:\n%s", out)
	}
}

func TestFormatWithSource_WithoutHint(t *testing.T) {
	err := NewError(1, 1, "boom")
	out := FormatWithSource(err, "x = 1\n")
	if strings.Contains(out, "hint:") {
		t.Errorf("unexpected hint line:\n%s", out)
	}
	if !strings.Contains(out, "  1 │ x = 1") {
		t.Errorf("missing source:\n%s", out)
	}
}

func TestCapyError_FileLineCol(t *testing.T) {
	err := &CapyError{File: "script.capy", Line: 12, Col: 3, Msg: "boom"}
	if err.Error() != "script.capy:12:3: boom" {
		t.Errorf("got %q", err.Error())
	}
}

func TestCapyError_NoPosition(t *testing.T) {
	err := &CapyError{Msg: "boom"}
	if err.Error() != "boom" {
		t.Errorf("got %q", err.Error())
	}
}
