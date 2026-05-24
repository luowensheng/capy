---
title: Host capabilities
hide:
  - toc
---

<div class="capy-hero" markdown>

<span class="capy-eyebrow">ENV · ARGS · READ_FILE</span>

# Pull host values into the transpilation

Capy stays a transpiler — but the values that flow into the
accumulated context can come from outside the source: environment
variables, positional CLI args, and sibling files. Four inner-DSL
primitives expose them, gated by a sandboxed default so the
playground stays safe.

</div>

---

## The four primitives

| Primitive             | Returns                          | Backed by (CLI)      |
|-----------------------|----------------------------------|----------------------|
| `(env "NAME")`        | OS env var, or `""` if unset     | `os.Getenv`          |
| `(arg N)`             | Nth positional CLI arg (0-indexed)  | `os.Args` (post script-path)  |
| `(arg_count)`         | how many positional args were supplied | `len(os.Args[2:])` |
| `(args)`              | full positional args list        | `os.Args[2:]`        |
| `(read_file "PATH")`  | file contents (errors abort)     | `os.ReadFile`        |
| `(os)`                | host OS identifier ("linux", "darwin", "windows", …) | `runtime.GOOS` |
| `(arch)`              | host arch ("amd64", "arm64", …)  | `runtime.GOARCH`     |
| `(cwd)`               | current working directory        | `os.Getwd`           |
| `(home_dir)`          | user's home directory            | `os.UserHomeDir`     |

Use them as expressions in any function body:

```
function service
    arg literal "service"
    arg capture name string
    set context.service name
    set context.environment (env "ENV")
    set context.version (arg 0)
    set context.api_keys (read_file "keys.txt")
end
```

## The Host abstraction

Capy doesn't call `os.Getenv` directly from the evaluator. Instead the
inner evaluator holds a `domain.Host` interface:

```go
type Host interface {
    Env(name string) string
    Arg(i int) string
    ArgCount() int
    Args() []string
    ReadFile(path string) (string, error)
}
```

Three implementations ship in the engine:

- **`infra.OSHost`** — real `os.Getenv` / `os.Args` / `os.ReadFile`. The
  CLI installs this so `capy run lib.capy script.capy a b c` works.
  `ReadFile` resolves relative paths against the script's directory.
- **`domain.NoOpHost`** — every method returns the zero value (or an
  error for `ReadFile`). The **default** for embedded Go callers and
  the **only** host the wasm playground uses. Sandboxed by design.
- **anything else you implement.** Test mocks, in-memory file maps,
  feature-flag stores, secret-manager backends — anything that
  satisfies the four methods.

## Opting into real host access from Go

`capy.NewLibrary` defaults to `NoOpHost`. If you're embedding Capy in a
trusted environment and want libraries to see your env/files, opt in:

```go
import (
    "os"
    "github.com/luowensheng/capy"
    "github.com/luowensheng/capy/infra"
)

lib, _ := capy.NewLibraryFromFile("lib.capy")
lib.SetHost(infra.OSHost{
    UserArgs: os.Args[1:],
    BaseDir:  ".",
})
out, _ := lib.Run(scriptSrc)
```

The opt-in is explicit on purpose: when you compose Capy with
user-supplied libraries, leaving the default `NoOpHost` means the
library cannot exfiltrate your environment.

## CLI

The CLI installs `OSHost` automatically. Positional args after the
script path are visible to `(arg N)`:

```sh
ENV=production DATABASE_URL=postgres://... \
  capy run lib.capy script.capy v2.3.1 us-west-2 eu-west
#                                  ↑ arg 0     ↑ arg 1   ↑ arg 2
```

`read_file` paths resolve relative to the script's directory, so:

```
set context.api_keys (read_file "api-keys.txt")
```

reads `dirname(scriptPath)/api-keys.txt`.

## Playground

The wasm playground exposes the **same primitives** but they're
permanently bound to `NoOpHost`. A sample library that calls
`(env "FOO")` will see `""`. This is by design — the playground runs
arbitrary user-supplied libraries and a malicious one shouldn't be able
to exfiltrate your env, list your local files, or read your
`~/.aws/credentials`.

The `host-capabilities` sample includes a "graceful degradation" note
in its description so newcomers understand why the playground output
differs from the CLI run.

## Worked example

The full sample lives at
[`samples/host-capabilities/`](https://github.com/luowensheng/capy/tree/main/samples/host-capabilities).
A 5-line source produces a complete Kubernetes Deployment manifest
that incorporates env vars, CLI args, and the contents of a sibling
secrets file at transpile time.

## When NOT to use this

These primitives let a library reach into the host. That's powerful
but also a tight coupling — your library now depends on something
external being set up correctly. Lean toward solving the same problem
with declarative input in the source first:

```
# Prefer this
deploy environment="production" version="v2.3.1"
```

over

```
# Over this
deploy
```

…with the library silently reading `ENV` and `arg 0`. Use the host
primitives when the values genuinely belong to the deployment context
(secrets, environment, infrastructure metadata) rather than the
"shape" of what's being generated.
