package domain

// Host is the small, intentional surface a Capy library can ask of its
// embedder. It lets a library author pull values from outside the source
// file — environment variables, CLI arguments, sibling files — and weave
// them into the accumulating context.
//
// Capy is still a transpiler: nothing here executes user-script code.
// `env`, `arg`, `read_file` are READ-ONLY primitives that surface
// already-existing host data so the library can incorporate it.
//
// Implementations:
//   - infra.OSHost — backed by os.Getenv / os.Args / os.ReadFile. Used by
//     the CLI. Lets you build configs that depend on the deployment
//     environment.
//   - domain.NoOpHost — every method returns the zero value or an error.
//     Default for embedded callers (capy.NewLibrary) and the wasm
//     playground, where exposing the embedder's filesystem/env would be
//     a security hazard.
//   - tests can stub a Host to make builds deterministic.
type Host interface {
	// Env returns the OS environment variable named NAME, or "" if unset.
	Env(name string) string

	// Arg returns the i-th positional CLI argument (zero-indexed), or
	// "" if i is out of range. Index 0 is the first user-supplied arg
	// AFTER the script path — `capy run lib.capy script.capy a b c`
	// gives Arg(0)="a", Arg(1)="b", Arg(2)="c".
	Arg(i int) string

	// ArgCount returns how many positional CLI args were supplied.
	ArgCount() int

	// Args returns a copy of the full positional CLI args slice.
	Args() []string

	// ReadFile returns the contents of the file at PATH. Relative paths
	// are resolved by the implementation (usually against the script's
	// directory). An error aborts the transpilation with a clear message.
	ReadFile(path string) (string, error)

	// OS returns the lowercase host operating-system identifier:
	// "linux", "darwin", "windows", "freebsd", "js" (wasm), etc.
	// Matches Go's runtime.GOOS so libraries can branch on it.
	OS() string

	// Arch returns the lowercase host architecture: "amd64", "arm64",
	// "wasm", etc. Matches Go's runtime.GOARCH.
	Arch() string

	// Cwd returns the host's current working directory at the time
	// transpilation started.
	Cwd() (string, error)

	// HomeDir returns the host user's home directory ($HOME on POSIX,
	// %USERPROFILE% on Windows).
	HomeDir() (string, error)

	// --- Side-effecting primitives (library commands only) ---

	// WriteFile creates (or overwrites) the file at PATH with the
	// given contents. Parent directories are created as needed.
	WriteFile(path, contents string) error

	// Mkdir creates the directory at PATH (and parents). Idempotent.
	Mkdir(path string) error

	// MkTemp returns the path to a freshly-created temp file with
	// the given suffix (e.g. ".py"). The caller is responsible for
	// removing it.
	MkTemp(suffix string) (string, error)

	// MkTempDir returns the path to a freshly-created temp directory.
	MkTempDir() (string, error)

	// Exec runs the named command with args, streaming its stdout
	// and stderr to the calling process's stdout/stderr. Returns
	// non-nil on non-zero exit.
	Exec(name string, args ...string) error

	// ExecCapture runs the named command and returns its combined
	// output.
	ExecCapture(name string, args ...string) (string, error)
}

// NoOpHost satisfies Host with empty/zero results everywhere. It's the
// safe default for sandboxed embedders (wasm playground, third-party Go
// programs that don't want their host filesystem exposed to library
// authors).
type NoOpHost struct{}

func (NoOpHost) Env(name string) string             { return "" }
func (NoOpHost) Arg(i int) string                   { return "" }
func (NoOpHost) ArgCount() int                      { return 0 }
func (NoOpHost) Args() []string                     { return nil }
func (NoOpHost) ReadFile(path string) (string, error) {
	return "", &CapyError{Msg: "read_file: host has no filesystem access", Hint: "this Capy build runs in a sandbox; pass --allow-fs on the CLI or supply an OSHost when embedding"}
}
func (NoOpHost) OS() string                 { return "" }
func (NoOpHost) Arch() string               { return "" }
func (NoOpHost) Cwd() (string, error)       { return "", nil }
func (NoOpHost) HomeDir() (string, error)   { return "", nil }

// Side-effecting primitives all refuse in the no-op host.
var errNoSandboxedFS = &CapyError{Msg: "filesystem / exec not available in this runtime", Hint: "library commands require infra.OSHost; the WASM / embedded sandbox doesn't provide it"}

func (NoOpHost) WriteFile(path, contents string) error           { return errNoSandboxedFS }
func (NoOpHost) Mkdir(path string) error                         { return errNoSandboxedFS }
func (NoOpHost) MkTemp(suffix string) (string, error)            { return "", errNoSandboxedFS }
func (NoOpHost) MkTempDir() (string, error)                      { return "", errNoSandboxedFS }
func (NoOpHost) Exec(name string, args ...string) error          { return errNoSandboxedFS }
func (NoOpHost) ExecCapture(name string, args ...string) (string, error) {
	return "", errNoSandboxedFS
}
