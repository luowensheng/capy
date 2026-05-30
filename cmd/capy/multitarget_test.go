package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/olivierdevelops/capy/orchestrator"
)

// TestMultiTargetSamples runs every "one source → many files" sample and
// diffs against the committed expected/ tree.
func TestMultiTargetSamples(t *testing.T) {
	root := findSamplesRoot(t)
	cases := []string{"webapp-trio", "android-app", "ios-app", "libtorch-train", "backend-with-tests"}
	for _, name := range cases {
		name := name
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(root, name)
			lib := filepath.Join(dir, "lib.capy")
			script := filepath.Join(dir, "script.capy")
			_, files, err := orchestrator.RunMulti(lib, script)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			for rel, got := range files {
				expectedPath := filepath.Join(dir, "expected", rel)
				want, err := os.ReadFile(expectedPath)
				if err != nil {
					t.Errorf("missing expected %s: %v", rel, err)
					continue
				}
				if string(want) != got {
					t.Errorf("%s/%s drift", name, rel)
				}
			}
		})
	}
}

// TestDesignSystem runs each of the three framework libraries against the
// same source and diffs against expected-<framework>/.
func TestDesignSystem(t *testing.T) {
	root := findSamplesRoot(t)
	dir := filepath.Join(root, "design-system-components")
	script := filepath.Join(dir, "script.capy")
	for _, fw := range []string{"react", "vue", "svelte"} {
		fw := fw
		t.Run(fw, func(t *testing.T) {
			lib := filepath.Join(dir, "lib_"+fw+".capy")
			_, files, err := orchestrator.RunMulti(lib, script)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			for rel, got := range files {
				expectedPath := filepath.Join(dir, "expected-"+fw, rel)
				want, err := os.ReadFile(expectedPath)
				if err != nil {
					t.Errorf("missing expected %s: %v", rel, err)
					continue
				}
				if string(want) != got {
					t.Errorf("%s/%s drift", fw, rel)
				}
			}
		})
	}
}

// TestZipOutput exercises the --zip path via the orchestrator + CLI helper.
// We don't shell out — we use the same writeZip the CLI uses by reading the
// archive back and confirming every file is present with matching content.
func TestZipOutput(t *testing.T) {
	root := findSamplesRoot(t)
	dir := filepath.Join(root, "multi-file-project")
	lib := filepath.Join(dir, "lib.capy")
	script := filepath.Join(dir, "script.capy")

	_, files, err := orchestrator.RunMulti(lib, script)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected multi-file output")
	}

	zipPath := filepath.Join(t.TempDir(), "out.zip")
	if err := writeZip(zipPath, files); err != nil {
		t.Fatalf("writeZip: %v", err)
	}

	// Read it back and confirm every original file is in the archive.
	rd, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer rd.Close()

	got := map[string]string{}
	for _, f := range rd.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open zip entry: %v", err)
		}
		b, err := readAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		got[f.Name] = string(b)
	}

	for rel, want := range files {
		// writeZip normalizes paths to forward slashes; compare in that form.
		key := filepath.ToSlash(rel)
		if g, ok := got[key]; !ok || g != want {
			t.Errorf("zip entry %s drift\nwant: %d bytes\ngot:  %d bytes (present=%v)", key, len(want), len(g), ok)
		}
	}
}

func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 1024)
	for {
		n, err := r.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, err
		}
	}
}
