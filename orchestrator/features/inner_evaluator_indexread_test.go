package orchfeatures

import (
	"testing"

	"github.com/olivierdevelops/capy/domain"
)

// TestDescendRead covers the shared read-descend helper over both
// container kinds: map key, list int index (positive / negative /
// out-of-range), wrong-type index, and a scalar parent.
func TestDescendRead(t *testing.T) {
	m := map[string]any{"a": "x", "1": "one"}
	if v, ok := descendRead(m, "a"); !ok || v != "x" {
		t.Fatalf("map[a] = %v,%v want x,true", v, ok)
	}
	// Map key is the stringified index — int64(1) hits the "1" key.
	if v, ok := descendRead(m, int64(1)); !ok || v != "one" {
		t.Fatalf("map[1] = %v,%v want one,true", v, ok)
	}
	// Missing map key: container, but nil value.
	if v, ok := descendRead(m, "nope"); !ok || v != nil {
		t.Fatalf("map[nope] = %v,%v want nil,true", v, ok)
	}

	l := []any{"a", "b", "c"}
	if v, ok := descendRead(l, int64(1)); !ok || v != "b" {
		t.Fatalf("list[1] = %v,%v want b,true", v, ok)
	}
	// Negative index counts from the end.
	if v, ok := descendRead(l, int64(-1)); !ok || v != "c" {
		t.Fatalf("list[-1] = %v,%v want c,true", v, ok)
	}
	// A Go int (the form a for-loop index binds as) indexes too.
	if v, ok := descendRead(l, 0); !ok || v != "a" {
		t.Fatalf("list[int 0] = %v,%v want a,true", v, ok)
	}
	// Out of range: container, nil value (tolerant, no panic).
	if v, ok := descendRead(l, int64(9)); !ok || v != nil {
		t.Fatalf("list[9] = %v,%v want nil,true", v, ok)
	}
	// Wrong-type index into a list: miss, but still a container.
	if v, ok := descendRead(l, "x"); !ok || v != nil {
		t.Fatalf("list[\"x\"] = %v,%v want nil,true", v, ok)
	}
	// Scalar parent: not a container.
	if _, ok := descendRead("scalar", int64(0)); ok {
		t.Fatalf("descendRead on scalar should report not-a-container")
	}
}

// TestResolvePathStepsIndex covers value-position index reads through the
// inner-DSL resolver: list by index and map by key, including a computed
// index expression and the round-trip with a write.
func TestResolvePathStepsIndex(t *testing.T) {
	e := &InnerEvaluator{Context: map[string]any{
		"buf":   []any{"a", "b", "c"},
		"known": map[string]any{"x": "1", "y": "2"},
	}}
	caps := map[string]domain.CaptureValue{}
	locals := map[string]any{"i": int64(2), "name": "y"}

	// context.buf[i] with local i=2 → "c".
	steps := []domain.PathStep{
		{Field: "context"}, {Field: "buf"},
		{IsIndex: true, Index: domain.VarRef{Steps: []domain.PathStep{{Field: "i"}}}},
	}
	v, err := e.resolvePathSteps(steps, caps, locals)
	if err != nil || v != "c" {
		t.Fatalf("context.buf[i] = %v,%v want c,nil", v, err)
	}

	// context.known[name] with local name="y" → "2".
	steps = []domain.PathStep{
		{Field: "context"}, {Field: "known"},
		{IsIndex: true, Index: domain.VarRef{Steps: []domain.PathStep{{Field: "name"}}}},
	}
	v, err = e.resolvePathSteps(steps, caps, locals)
	if err != nil || v != "2" {
		t.Fatalf("context.known[name] = %v,%v want 2,nil", v, err)
	}

	// Negative literal index: context.buf[-1] → "c".
	steps = []domain.PathStep{
		{Field: "context"}, {Field: "buf"},
		{IsIndex: true, Index: domain.NumberLit{IsInt: true, I: -1}},
	}
	v, err = e.resolvePathSteps(steps, caps, locals)
	if err != nil || v != "c" {
		t.Fatalf("context.buf[-1] = %v,%v want c,nil", v, err)
	}

	// Round-trip: write buf[0] then read it back by index.
	wp := domain.Path{Root: "context", Steps: []domain.PathStep{
		{Field: "buf"}, {IsIndex: true, Index: domain.NumberLit{IsInt: true, I: 0}},
	}}
	if err := e.writePath(wp, "A", caps, locals, "set"); err != nil {
		t.Fatalf("write buf[0]: %v", err)
	}
	steps = []domain.PathStep{
		{Field: "context"}, {Field: "buf"},
		{IsIndex: true, Index: domain.NumberLit{IsInt: true, I: 0}},
	}
	v, err = e.resolvePathSteps(steps, caps, locals)
	if err != nil || v != "A" {
		t.Fatalf("round-trip buf[0] = %v,%v want A,nil", v, err)
	}
}

// TestEvalInterpPathIndex covers template-position `${…[…]}` reads:
// list by local index, map by captured key, nested grid[i][j], and the
// tolerant empty on out-of-range.
func TestEvalInterpPathIndex(t *testing.T) {
	e := &InnerEvaluator{Context: map[string]any{
		"buf":   []any{"a", "b", "c"},
		"known": map[string]any{"k": "kv"},
		"grid":  []any{[]any{"r0c0", "r0c1"}, []any{"r1c0", "r1c1"}},
	}}
	locals := map[string]any{"i": int64(1), "j": int64(0), "key": "k"}

	cases := []struct{ atom, want string }{
		{"context.buf[i]", "b"},          // list by local index
		{"context.buf[-1]", "c"},         // negative literal
		{"context.known[key]", "kv"},     // map by captured key
		{"context.grid[i][j]", "r1c0"},   // nested index
		{"context.buf[(sub i 1)]", "a"},  // computed index expression
		{"context.buf[99]", ""},          // out of range → tolerant empty
	}
	for _, c := range cases {
		got, err := e.evalInterpAtom(c.atom, locals)
		if err != nil {
			t.Fatalf("%s errored: %v", c.atom, err)
		}
		if toString(got) != c.want {
			t.Fatalf("%s = %q, want %q", c.atom, toString(got), c.want)
		}
	}
}
