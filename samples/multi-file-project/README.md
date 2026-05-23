# multi-file-project

**One Capy source → a complete multi-file project tree.**

This sample demonstrates Capy's `file "path":` blocks. The library
declares six output files; the engine writes them all to disk under
the `--out-dir` you pass to `capy run`.

## What you write

`script.capy` (9 lines):

```
project "todo-api"
    description "A tiny TODO REST service"
    author "you@example.com"

    route GET    "/health"     health_check
    route GET    "/todos"      list_todos
    route POST   "/todos"      create_todo
    route GET    "/todos/{id}" get_todo
    route DELETE "/todos/{id}" delete_todo
end
```

## What you get

```
generated/
├── README.md
├── pyproject.toml
├── .gitignore
├── src/
│   ├── main.py            ← FastAPI app with all 5 routes mounted
│   └── handlers.py        ← Handler stubs ready to implement
└── tests/
    └── test_smoke.py      ← Smoke tests for every route
```

## Run

```sh
../../capy run --out-dir generated lib.capy script.capy
```

Output:

```
wrote generated/.gitignore (48 bytes)
wrote generated/README.md (209 bytes)
wrote generated/pyproject.toml (277 bytes)
wrote generated/src/handlers.py (735 bytes)
wrote generated/src/main.py (757 bytes)
wrote generated/tests/test_smoke.py (1117 bytes)
```

Open `generated/src/main.py`:

```python
"""Generated FastAPI app for todo-api."""
from fastapi import FastAPI
from . import handlers

app = FastAPI(title="todo-api")

@app.get("/health")
async def health_check_endpoint(*args, **kwargs):
    return await handlers.health_check(*args, **kwargs)

@app.get("/todos")
async def list_todos_endpoint(*args, **kwargs):
    return await handlers.list_todos(*args, **kwargs)
# ...
```

## How it works

The library has six `file "..."` blocks at the top level. Each
declares an output path and a Go template. The engine renders every
block against the same final context + body and writes each result
to its declared path (creating subdirectories as needed).

```
file "README.md":
    # {{ .context.name | unquote }}
    ...

file "src/main.py":
    """Generated FastAPI app for {{ .context.name | unquote }}."""
    from fastapi import FastAPI
    {{ range .context.routes -}}
    @app.{{ .method | lower }}({{ .path | toQuoted }})
    ...
    {{ end }}

file "tests/test_smoke.py":
    ...
```

Adding a route line in `script.capy` regenerates all five files
that mention routes. **The library is your project scaffold;
script.capy is the truth.**

## Why this matters

A typical microservice has 8-15 files of scaffolding plus a handful
of real handler implementations. Editing the scaffold (renaming the
project, switching from FastAPI to Flask, changing the test
framework) means touching every file individually — and they drift
on every refactor.

With Capy:

- The scaffold lives in `lib.capy`. Change it once; regenerate.
- The intent lives in `script.capy`. Add a route, regenerate.
- The output is a normal project tree. No runtime dependency on
  Capy.

This pattern scales to any project type: TypeScript libraries, Rust
crates, Go modules, Helm charts, Terraform modules. Each has its
own characteristic file tree; each is a `file "...":` block in a
single Capy library.
