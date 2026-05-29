package capy_test

import (
	"strings"
	"testing"

	"github.com/luowensheng/capy"
)

// Tests for the round-4 grammar features that closed the last two gaps in
// .ignore/missing.md: context-sensitive lookahead (§5) and multi-section
// blocks for try/rescue/finally (§8).

// §5 — `when_followed_by indent` / `when_not_followed_by indent` let a
// flat function and a block function share a leading keyword and
// disambiguate purely by whether an indented body follows.
func TestLookaheadKeywordSharing(t *testing.T) {
	lib := mustLib(t, `
extension txt

function os_block
    arg literal "os"
    arg capture name string
    when_followed_by indent
    block_closer endos
    write `+bt+`BLOCK ${unquote name}:
${body}`+bt+`
end

function endos
end

function os_flat
    arg literal "os"
    arg capture name string
    when_not_followed_by indent
    write `+bt+`FLAT ${unquote name}
`+bt+`
end

function note
    arg literal "note"
    arg capture t string
    write `+bt+`  - ${unquote t}
`+bt+`
end
`)
	out := mustRun(t, lib, `os "linux"
os "darwin"
    note "only here"
endos`)
	want := "FLAT linux\nBLOCK darwin:\n  - only here\n"
	if out != want {
		t.Errorf("lookahead disambiguation mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// §5 — running the same ambiguous source many times must give the same
// parse (the lookahead gate must be order-independent, like the §2
// determinism guarantee).
func TestLookaheadDeterministic(t *testing.T) {
	src := `
extension txt
function k_block
    arg literal "k"
    when_followed_by indent
    block_closer endk
    write `+bt+`B
${body}`+bt+`
end
function endk
end
function k_flat
    arg literal "k"
    when_not_followed_by indent
    write `+bt+`F
`+bt+`
end
function item
    arg literal "item"
    write `+bt+`.`+bt+`
end
`
	for i := 0; i < 50; i++ {
		lib := mustLib(t, src)
		flat := mustRun(t, lib, `k`)
		if flat != "F\n" {
			t.Fatalf("iter %d flat = %q, want F", i, flat)
		}
		block := mustRun(t, lib, "k\n    item\nendk")
		if block != "B\n." {
			t.Fatalf("iter %d block = %q, want B.", i, block)
		}
	}
}

// §5 — a library cannot declare both predicates on one function.
func TestLookaheadContradictionRejected(t *testing.T) {
	_, err := capy.NewLibrary(`
extension txt
function bad
    arg literal "bad"
    when_followed_by indent
    when_not_followed_by indent
    write `+bt+`x
`+bt+`
end
`)
	if err == nil {
		t.Fatal("expected error for contradictory lookahead, got nil")
	}
	if !strings.Contains(err.Error(), "when_followed_by") {
		t.Errorf("error should mention the contradiction, got: %v", err)
	}
}

// §8 — a `block_sections` function captures a main body plus named
// section sub-bodies (rescue / finally), each rendered independently and
// exposed to the template as `${body}` / `${rescue}` / `${finally}`.
func TestSectionedBlockAllSections(t *testing.T) {
	lib := mustLib(t, sectionLib)
	out := mustRun(t, lib, `try
    step "a"
    step "b"
rescue
    step "handle"
finally
    step "cleanup"
end`)
	want := "TRY{a;b;}RESCUE{handle;}FINALLY{cleanup;}\n"
	if out != want {
		t.Errorf("sectioned block mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// §8 — omitted sections render as empty locals (not undefined-var errors).
func TestSectionedBlockOmittedSection(t *testing.T) {
	lib := mustLib(t, sectionLib)
	out := mustRun(t, lib, `try
    step "a"
finally
    step "cleanup"
end`)
	want := "TRY{a;}RESCUE{}FINALLY{cleanup;}\n"
	if out != want {
		t.Errorf("omitted-section mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// §8 — sections may be omitted entirely; a bare try/end still works.
func TestSectionedBlockBareBody(t *testing.T) {
	lib := mustLib(t, sectionLib)
	out := mustRun(t, lib, `try
    step "only"
end`)
	want := "TRY{only;}RESCUE{}FINALLY{}\n"
	if out != want {
		t.Errorf("bare-body mismatch:\n got: %q\nwant: %q", out, want)
	}
}

// §8 — a duplicated section is a parse error.
func TestSectionedBlockDuplicateSection(t *testing.T) {
	lib := mustLib(t, sectionLib)
	_, err := lib.Run(`try
    step "a"
rescue
    step "x"
rescue
    step "y"
end`)
	if err == nil {
		t.Fatal("expected duplicate-section error, got nil")
	}
}

// sectionLib is a try/rescue/finally library shared by the §8 tests.
const sectionLib = `
extension txt

function try
    arg literal "try"
    block_sections rescue finally closer end
    write ` + bt + `TRY{${body}}RESCUE{${rescue}}FINALLY{${finally}}
` + bt + `
end

function end
end

function step
    arg literal "step"
    arg capture s string
    write ` + bt + `${unquote s};` + bt + `
end
`
