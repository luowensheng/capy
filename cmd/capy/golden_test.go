package main

import (
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
//   go test ./cmd/capy/... -update
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
				if output != want {
					t.Fatalf("output mismatch:\nwant:\n%s\ngot:\n%s", want, output)
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
