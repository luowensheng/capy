package capy_test

import (
	"testing"

	"github.com/luowensheng/capy"
)

// (bt — a single backtick string — is declared in capy_test.go.)

// Tests for the round-3 features that closed gaps reported in
// .ignore/missing.md: block backtracking (§1), deterministic candidate
// ordering (§2), the asString interpolation verb (§3), the `word` capture
// type (§4), and the `dotted_ident` capture type (§9).

// §4 — `word` captures a shell-style bare word (a maximal run of adjacent
// tokens with no source whitespace), so hyphens / slashes / `=` / globs
// survive even though the lexer splits them.
func TestWordCapture(t *testing.T) {
	lib, err := capy.NewLibrary(`
extension txt
function exec
    arg literal "exec"
    arg capture cmd word
    write ` + bt + `[${cmd}]
` + bt + `
end
`)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}
	cases := map[string]string{
		"exec restart-api":     "[restart-api]\n",
		"exec k8s/deploy.yaml": "[k8s/deploy.yaml]\n",
		"exec name=^web$":      "[name=^web$]\n",
		"exec --oneline":       "[--oneline]\n",
	}
	for src, want := range cases {
		out, err := lib.Run(src)
		if err != nil {
			t.Fatalf("Run(%q): %v", src, err)
		}
		if out != want {
			t.Errorf("Run(%q) = %q, want %q", src, out, want)
		}
	}
}

// §9 — `dotted_ident` consumes an IDENT(.IDENT)* chain as one value, so
// `match err.kind` works bare instead of needing the string workaround.
func TestDottedIdentCapture(t *testing.T) {
	lib, err := capy.NewLibrary(`
extension txt
function match
    arg literal "match"
    arg capture path dotted_ident
    write ` + bt + `<${path}>
` + bt + `
end
`)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}
	for src, want := range map[string]string{
		"match err.kind": "<err.kind>\n",
		"match a.b.c.d":  "<a.b.c.d>\n",
		"match plain":    "<plain>\n",
	} {
		out, err := lib.Run(src)
		if err != nil {
			t.Fatalf("Run(%q): %v", src, err)
		}
		if out != want {
			t.Errorf("Run(%q) = %q, want %q", src, out, want)
		}
	}
}

// §3 — asString normalises both a bare token and a quoted string to one
// valid JSON string. Both forms must produce identical output.
func TestAsStringHelper(t *testing.T) {
	lib, err := capy.NewLibrary(`
extension txt
function emit
    arg literal "emit"
    arg capture v raw
    write ` + bt + `${asString v}
` + bt + `
end
`)
	if err != nil {
		t.Fatalf("NewLibrary: %v", err)
	}
	for src, want := range map[string]string{
		`emit foo`:              "\"foo\"\n",
		`emit "foo"`:            "\"foo\"\n",
		`emit "he said \"hi\""`: "\"he said \\\"hi\\\"\"\n",
	} {
		out, err := lib.Run(src)
		if err != nil {
			t.Fatalf("Run(%q): %v", src, err)
		}
		if out != want {
			t.Errorf("Run(%q) = %q, want %q", src, out, want)
		}
	}
}

// §1 + §2 — a block function and a flat function share the leading `os`
// keyword. The block form is tried first (deterministic name order); when
// no indented body follows, the parser must BACKTRACK and match the flat
// form instead of erroring. Running many times must give the SAME result
// every time (no map-order heisenbug).
func TestBlockBacktrackingAndDeterminism(t *testing.T) {
	src := `
extension txt
function osblock
    arg literal "os"
    arg capture name string
    block_closer endos
    write ` + bt + `BLOCK ${name}
` + bt + `
end
function endos
    arg literal "endos"
    write ` + bt + `` + bt + `
end
function osflat
    arg literal "os"
    arg capture name string
    write ` + bt + `FLAT ${name}
` + bt + `
end
`
	// Flat fall-through: no indented body, so osblock's header matches but
	// its body parse fails → backtrack → osflat matches.
	flat := `os "linux"`
	// Real block: indented body + closer.
	block := "os \"linux\"\n    os \"inner\"\nendos"

	for i := 0; i < 50; i++ {
		lib, err := capy.NewLibrary(src)
		if err != nil {
			t.Fatalf("NewLibrary: %v", err)
		}
		out, err := lib.Run(flat)
		if err != nil {
			t.Fatalf("iter %d Run(flat): %v", i, err)
		}
		if out != "FLAT \"linux\"\n" {
			t.Fatalf("iter %d flat = %q, want FLAT (nondeterministic backtrack?)", i, out)
		}
		out, err = lib.Run(block)
		if err != nil {
			t.Fatalf("iter %d Run(block): %v", i, err)
		}
		// osblock's template has no ${body}, so the (successfully parsed)
		// inner body isn't emitted — we only assert the block form matched
		// without error and rendered its own line.
		if out != "BLOCK \"linux\"\n" {
			t.Fatalf("iter %d block = %q", i, out)
		}
	}
}
