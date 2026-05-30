package orchfeatures

import (
	"testing"

	"github.com/olivierdevelops/capy/domain"
)

// idxStep builds a `[<int>]` index PathStep with a literal integer.
func idxStep(i int64) domain.PathStep {
	return domain.PathStep{IsIndex: true, Index: domain.NumberLit{IsInt: true, I: i}}
}

func fieldStep(name string) domain.PathStep {
	return domain.PathStep{Field: name}
}

// TestWritePathListIndex covers the in-place overwrite of a list element
// by index — `set context.buf[i] value` — including negative indices and
// out-of-range errors. This is the affordance that lets a library
// retroactively rewrite buffered output (e.g. null out a dead store).
func TestWritePathListIndex(t *testing.T) {
	e := &InnerEvaluator{Context: map[string]any{
		"buf": []any{"a", "b", "c"},
	}}
	caps := map[string]domain.CaptureValue{}
	locals := map[string]any{}
	p := domain.Path{Root: "context", Steps: []domain.PathStep{fieldStep("buf"), idxStep(1)}}

	if err := e.writePath(p, "B", caps, locals, "set"); err != nil {
		t.Fatalf("set buf[1]: %v", err)
	}
	buf := e.Context["buf"].([]any)
	if buf[1] != "B" {
		t.Fatalf("buf[1] = %v, want B", buf[1])
	}
	// Untouched neighbours.
	if buf[0] != "a" || buf[2] != "c" {
		t.Fatalf("neighbours changed: %v", buf)
	}

	// Negative index: -1 is the last element.
	pNeg := domain.Path{Root: "context", Steps: []domain.PathStep{fieldStep("buf"), idxStep(-1)}}
	if err := e.writePath(pNeg, "Z", caps, locals, "set"); err != nil {
		t.Fatalf("set buf[-1]: %v", err)
	}
	if got := e.Context["buf"].([]any)[2]; got != "Z" {
		t.Fatalf("buf[-1] write: buf[2] = %v, want Z", got)
	}

	// Out of range errors rather than panicking or growing the list.
	pOOR := domain.Path{Root: "context", Steps: []domain.PathStep{fieldStep("buf"), idxStep(9)}}
	if err := e.writePath(pOOR, "x", caps, locals, "set"); err == nil {
		t.Fatalf("set buf[9] should error (len=3)")
	}
}

// TestWritePathListIndexAppend covers append/prepend against a nested
// list stored at an element index.
func TestWritePathListIndexAppend(t *testing.T) {
	e := &InnerEvaluator{Context: map[string]any{
		"rows": []any{[]any{"x"}, []any{"y"}},
	}}
	caps := map[string]domain.CaptureValue{}
	locals := map[string]any{}

	pApp := domain.Path{Root: "context", Steps: []domain.PathStep{fieldStep("rows"), idxStep(0)}}
	if err := e.writePath(pApp, "x2", caps, locals, "append"); err != nil {
		t.Fatalf("append rows[0]: %v", err)
	}
	row0 := e.Context["rows"].([]any)[0].([]any)
	if len(row0) != 2 || row0[1] != "x2" {
		t.Fatalf("rows[0] = %v, want [x x2]", row0)
	}

	pPre := domain.Path{Root: "context", Steps: []domain.PathStep{fieldStep("rows"), idxStep(1)}}
	if err := e.writePath(pPre, "y0", caps, locals, "prepend"); err != nil {
		t.Fatalf("prepend rows[1]: %v", err)
	}
	row1 := e.Context["rows"].([]any)[1].([]any)
	if len(row1) != 2 || row1[0] != "y0" {
		t.Fatalf("rows[1] = %v, want [y0 y]", row1)
	}
}
