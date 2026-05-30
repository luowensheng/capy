package infra

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/olivierdevelops/capy/domain"
)

// OSHost is the domain.Host implementation backed by real OS primitives.
// Used by the CLI so libraries can read env vars, positional CLI args,
// and sibling files. NOT used by the wasm playground or default embedded
// callers — exposing the embedder's filesystem from a library would be a
// security hazard.
//
// Construction notes:
//   - Args carries the positional CLI args AFTER the library + script
//     paths have been stripped (so Arg(0) is the first user arg, not
//     the binary name).
//   - BaseDir is the script's directory; ReadFile resolves relative
//     paths against it so libraries can ask for `read_file "config.toml"`
//     and get the file next to the script, not next to the binary.
type OSHost struct {
	UserArgs []string
	BaseDir  string
}

func (h OSHost) Env(name string) string { return os.Getenv(name) }

func (h OSHost) Arg(i int) string {
	if i < 0 || i >= len(h.UserArgs) {
		return ""
	}
	return h.UserArgs[i]
}

func (h OSHost) ArgCount() int { return len(h.UserArgs) }

func (h OSHost) Args() []string {
	out := make([]string, len(h.UserArgs))
	copy(out, h.UserArgs)
	return out
}

func (h OSHost) OS() string   { return runtime.GOOS }
func (h OSHost) Arch() string { return runtime.GOARCH }

func (h OSHost) Cwd() (string, error)     { return os.Getwd() }
func (h OSHost) HomeDir() (string, error) { return os.UserHomeDir() }

func (h OSHost) WriteFile(path, contents string) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return &domain.CapyError{Msg: fmt.Sprintf("write_file %q: mkdir parent: %v", path, err)}
		}
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		return &domain.CapyError{Msg: fmt.Sprintf("write_file %q: %v", path, err)}
	}
	return nil
}

func (h OSHost) Mkdir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return &domain.CapyError{Msg: fmt.Sprintf("mkdir %q: %v", path, err)}
	}
	return nil
}

func (h OSHost) MkTemp(suffix string) (string, error) {
	f, err := os.CreateTemp("", "capy-*"+suffix)
	if err != nil {
		return "", &domain.CapyError{Msg: fmt.Sprintf("mktemp: %v", err)}
	}
	name := f.Name()
	f.Close()
	return name, nil
}

func (h OSHost) MkTempDir() (string, error) {
	dir, err := os.MkdirTemp("", "capy-*")
	if err != nil {
		return "", &domain.CapyError{Msg: fmt.Sprintf("mktemp_dir: %v", err)}
	}
	return dir, nil
}

func (h OSHost) Exec(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return &domain.CapyError{Msg: fmt.Sprintf("exec %s: %v", name, err)}
	}
	return nil
}

func (h OSHost) ExecCapture(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String() + stderr.String(),
			&domain.CapyError{Msg: fmt.Sprintf("exec %s: %v\n%s", name, err, stderr.String())}
	}
	return stdout.String(), nil
}

func (h OSHost) ReadFile(path string) (string, error) {
	resolved := path
	if !filepath.IsAbs(resolved) && h.BaseDir != "" {
		resolved = filepath.Join(h.BaseDir, path)
	}
	b, err := os.ReadFile(resolved)
	if err != nil {
		return "", &domain.CapyError{
			Msg:  fmt.Sprintf("read_file %q: %v", path, err),
			Hint: "path is resolved relative to the script's directory",
		}
	}
	return string(b), nil
}
