# Host capabilities — env, args, read_file

This sample shows the three inner-DSL primitives a library can use to
pull values from outside the source file:

| Primitive             | Returns                          | Backed by (CLI)      |
|-----------------------|----------------------------------|----------------------|
| `(env "NAME")`        | OS env var, or `""` if unset     | `os.Getenv`          |
| `(arg N)`             | Nth positional CLI arg           | `os.Args`            |
| `(arg_count)`         | how many args were supplied      | `len(os.Args)`       |
| `(args)`              | full args list                   | `os.Args`            |
| `(read_file "PATH")`  | file contents (errors abort)     | `os.ReadFile`        |

Paths in `read_file` are resolved relative to the script's directory.

## Run it

```sh
ENV=production DATABASE_URL=postgres://db.internal/prod \
  capy run lib.capy script.capy v2.3.1 us-west-2
```

The 5-line script combines with:
- `ENV` → `environment: production`
- `DATABASE_URL` → injected as a container env var
- positional arg 0 (`v2.3.1`) → `version` and image tag
- positional arg 1 (`us-west-2`) → `region`
- the sibling `api-keys.txt` → expanded into `--key=...` container args

…to produce a complete Kubernetes Deployment manifest.

## Security note

The CLI uses `infra.OSHost` (real `os.Getenv`, `os.ReadFile`). The
**wasm playground** and the **default embedded Go API** use
`domain.NoOpHost` — `env` returns `""`, `read_file` errors with a
clear message. This is deliberate: a library author can't smuggle
your filesystem or env into a sandboxed transpilation.

If you embed Capy in your own Go program and want the library to see
your env/files, opt in explicitly:

```go
import (
    "github.com/olivierdevelops/capy"
    "github.com/olivierdevelops/capy/infra"
)
lib, _ := capy.NewLibraryFromFile("lib.capy")
lib.SetHost(infra.OSHost{
    UserArgs: os.Args[1:],
    BaseDir:  ".",
})
```
