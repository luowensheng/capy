# Tutorial 2: Building a Config DSL

Use Capy to turn a small declarative source language into a JSON
configuration file. Estimated time: 10 minutes.

## Goal

Source:

```
set_name "my-app"
set_version "1.2.0"
depend on "express"
depend on "lodash"
```

Output (a `package.json`-ish file):

```json
{
  "name": "my-app",
  "version": "1.2.0",
  "dependencies": [
    "express",
    "lodash"
  ]
}
```

## Step 1 — initial library

```
extension json

context
    name ""
    version ""
    dependencies []
end

function set_name
    arg capture n any
    set context.name n
end

function set_version
    arg capture v any
    set context.version v
end

function depend
    arg literal "depend"
    arg literal "on"
    arg capture pkg any
    append context.dependencies pkg
end

file_template
    write (toJSONIndent context)
end
```

Key concepts here:

- **`context`** declares the initial state. Maps to `{}`, lists to `[]`.
- **No function calls `write`** — the body emits nothing; the entire
  output is the rendered context.
- **`set` / `append`** mutate context.
- **`file_template`** uses `toJSONIndent` to marshal context.
- **`depend on`** is a multi-literal pattern. The function name `depend`
  is NOT auto-prepended because the args list already contains literals.

## Step 2 — add validation

Add a type for semantic versions:

```yaml
types:
  SemVer:
    pattern: "^[0-9]+\\.[0-9]+\\.[0-9]+(-[A-Za-z0-9.-]+)?$"
```

And change `set_version` to use it:

```yaml
set_version:
  args:
    - { kind: capture, name: v, type: SemVer }
```

Now `set_version "1.2.0"` is valid; `set_version "not-a-version"` fails
with a clear error.

## Step 3 — script

```
set_name "my-app"
set_version "1.2.0"
depend on "express"
depend on "lodash"
```

Run:

```sh
capy run lib.capy script.capy
```

Output:

```json
{
  "dependencies": [
    "express",
    "lodash"
  ],
  "name": "my-app",
  "version": "1.2.0"
}
```

Note that JSON object keys are alphabetised by the marshaller — if
you want a specific order, render the file_template manually:

```
file_template
    write `{
  "name": ${toQuoted context.name},
  "version": ${toQuoted context.version},
  "dependencies": ${toJSON context.dependencies}
}
`
end
```

## Try it

- Add a `script <name> = "<cmd>"` function that accumulates a map
  (`set context.scripts[name] cmd`).
- Add an enum-typed field, e.g. `set_license` with
  `options: ["MIT", "Apache-2.0", "GPL-3.0"]`.
- Add a `description "<text>"` function.

## Next

[Tutorial 3: Transpiling to Python](03-transpile-python.md) introduces
**block functions** and **indented bodies**.
