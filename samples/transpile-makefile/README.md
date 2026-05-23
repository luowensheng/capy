# transpile-makefile

Declarative tasks → Makefile.

## Run

```sh
../../capy run lib.yaml script.capy
```

## What this teaches

- All functions have empty `template:` — the entire output is built from
  context in `file_template:`.
- Map-keyed context (`context.tasks[name] = cmd`) for dynamic key/value
  pairs.
- `range $key, $val := .context.tasks` iterates a map in the file
  template.
