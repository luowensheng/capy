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

```yaml
extension: json
output_file: ""

context:
  name: ""
  version: ""
  dependencies: []

functions:
  set_name:
    args:
      - { kind: capture, name: n, type: any }
    template: ""
    run: |
      set context.name n

  set_version:
    args:
      - { kind: capture, name: v, type: any }
    template: ""
    run: |
      set context.version v

  depend:
    args:
      - { kind: literal, value: "depend" }
      - { kind: literal, value: "on" }
      - { kind: capture, name: pkg, type: any }
    template: ""
    run: |
      append context.dependencies pkg

file_template: |
  {{ .context | toJSONIndent }}
```

Key concepts here:

- **`context:`** declares the initial state. Maps to `{}`, lists to `[]`.
- **Every function's `template:` is empty.** The body emits nothing —
  the entire output is the rendered context.
- **`run:`** mutates context: `set`, `append`.
- **`file_template:`** uses `toJSONIndent` to marshal context.
- **`depend on`** is a multi-literal pattern. The function key `depend`
  is NOT auto-prepended because the args list contains literals.

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
capy run lib.yaml script.capy
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

Note that JSON object keys are alphabetised by the marshaller — if you
want a specific order, render the file_template manually:

```yaml
file_template: |
  {
    "name": {{ .context.name | toQuoted }},
    "version": {{ .context.version | toQuoted }},
    "dependencies": {{ .context.dependencies | toJSON }}
  }
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
