package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/luowensheng/capy/domain"
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
