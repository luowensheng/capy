package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luowensheng/capy/orchestrator"
)

// TestGolden walks samples/ and runs each (lib.yaml, *.capy) pair through the
// orchestrator. Each script.capy pairs with <basename>.expected.txt (for
// successful runs) or <basename>.expected-error.txt (for runs that must error).
//
// To regenerate goldens after intentional behavior changes:
//
//	go test ./cmd/capy/... -update
func TestGolden(t *testing.T) {
	root := findSamplesRoot(t)
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read samples dir: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		runSampleGoldens(t, dir)
	}
}

func runSampleGoldens(t *testing.T, dir string) {
	libPath := filepath.Join(dir, "lib.yaml")
	if _, err := os.Stat(libPath); err != nil {
		return
	}
	scripts, err := filepath.Glob(filepath.Join(dir, "*.capy"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, scriptPath := range scripts {
		scriptPath := scriptPath
		base := strings.TrimSuffix(filepath.Base(scriptPath), ".capy")
		name := filepath.Base(dir) + "/" + base
		t.Run(name, func(t *testing.T) {
			expectOkPath := filepath.Join(dir, base+".expected.txt")
			expectErrPath := filepath.Join(dir, base+".expected-error.txt")
			output, runErr := runCapy(libPath, scriptPath)
			switch {
			case fileExists(expectErrPath):
				if runErr == nil {
					t.Fatalf("expected error per %s, got success: %s", expectErrPath, output)
				}
				want := readFileTrim(t, expectErrPath)
				got := strings.TrimSpace(runErr.Error())
				if *update {
					writeFile(t, expectErrPath, got+"\n")
					return
				}
				if got != want {
					t.Fatalf("error mismatch:\nwant: %q\ngot:  %q", want, got)
				}
			case fileExists(expectOkPath):
				if runErr != nil {
					t.Fatalf("expected success, got error: %v", runErr)
				}
				want := readFile(t, expectOkPath)
				if *update {
					writeFile(t, expectOkPath, output)
					return
				}
				// Normalise CRLF and any UTF-8 BOM so cross-platform CI
				// (Windows in particular) doesn't fail on invisible
				// byte differences. `.gitattributes` already enforces LF
				// in the repo; this is belt-and-suspenders.
				gotNorm := normalize(output)
				wantNorm := normalize(want)
				if gotNorm != wantNorm {
					t.Fatalf("output mismatch:\n%s", diffSummary(wantNorm, gotNorm))
				}
			default:
				t.Skipf("no golden file for %s", scriptPath)
			}
		})
	}
}

func runCapy(libPath, scriptPath string) (string, error) {
	return orchestrator.Run(libPath, scriptPath)
}

func findSamplesRoot(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cur := wd
	for {
		p := filepath.Join(cur, "samples")
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			t.Fatalf("samples dir not found from %s", wd)
		}
		cur = parent
	}
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

func readFile(t *testing.T, p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func readFileTrim(t *testing.T, p string) string {
	return strings.TrimSpace(readFile(t, p))
}

func writeFile(t *testing.T, p, content string) {
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func normalize(s string) string {
	// strip UTF-8 BOM if present (0xEF 0xBB 0xBF)
	s = strings.TrimPrefix(s, "\ufeff")
	// CRLF and lone CR → LF
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// Strip trailing whitespace. A golden file may have a final LF
	// that an in-memory output lacks (or vice versa). We do not care.
	s = strings.TrimRight(s, " \t\n")
	return s
}

// diffSummary returns a short description of the first byte that differs
// between want and got, plus a few bytes of surrounding context. This is
// what makes invisible-byte mismatches debuggable on CI.
func diffSummary(want, got string) string {
	var b strings.Builder
	for i := 0; i < len(want) && i < len(got); i++ {
		if want[i] != got[i] {
			lo := i - 20
			if lo < 0 {
				lo = 0
			}
			hi := i + 20
			if hi > len(want) {
				hi = len(want)
			}
			fmt.Fprintf(&b, "first diff at byte %d (line %d): want %q got %q\n",
				i, lineOf(want, i), want[lo:hi], got[lo:min(hi, len(got))])
			break
		}
	}
	if len(want) != len(got) {
		fmt.Fprintf(&b, "length: want=%d got=%d\n", len(want), len(got))
	}
	fmt.Fprintf(&b, "\n--- want (%d bytes) ---\n%s\n--- got (%d bytes) ---\n%s\n",
		len(want), want, len(got), got)
	return b.String()
}

func lineOf(s string, i int) int {
	n := 1
	for j := 0; j < i && j < len(s); j++ {
		if s[j] == '\n' {
			n++
		}
	}
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
