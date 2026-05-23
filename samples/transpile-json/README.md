# transpile-json

Same engine, totally different target: build a JSON config from a small declarative source language.

## Files

- `lib.yaml` — four functions: `set_name`, `set_version`, `depend on …`, `script <ident> = "…"`. Every function has an empty `template:` and only updates the accumulated `context`.
- `script.capy` — a project descriptor with name, version, dependencies, and scripts.

## Run

```sh
../../capy -lib lib.yaml script.capy
```

## Expected output

```json
{
  "dependencies": [
    "express",
    "lodash"
  ],
  "name": "my-project",
  "scripts": {
    "build": "go build ./...",
    "test": "go test ./..."
  },
  "version": "1.2.3"
}
```

## What this teaches

- The transpile model is **target-language-agnostic**. Same engine, same primitives, very different output.
- The body of this transpilation is empty — every function contributes only via `run:`. The `file_template:` is just `{{ .context | toJSONIndent }}`.
- Inner-DSL primitives `set`, `append`, and the indexed form `context.scripts[name]` build a deeply structured object that is then JSON-marshaled in one shot.
- String captures resolve to **unquoted Go strings** inside `run:` snippets (e.g. `n` for `set_name "my-project"` is the Go string `"my-project"`, not the source text `"\"my-project\""`). That's why the JSON output has clean values without double-escaping.
