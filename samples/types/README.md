# types

Library-defined argument types with `pattern:` (regex) and `options:` (enum). Demonstrates both passing validation and failing validation.

## Files

- `lib.yaml` — declares `Email` (regex), `Status` (enum), `Slug` (regex) and three functions using them.
- `script.capy` — calls each function with a valid value.
- `script-invalid.capy` — calls `set_email` with a value that fails the regex.

## Run

```sh
../../capy -lib lib.yaml script.capy
../../capy -lib lib.yaml script-invalid.capy
```

## Expected output

```
email = alice@example.com
status = todo
slug = hello-world
```

Invalid script:

```
capy: function "set_email" arg "e": value "not-an-email" does not match pattern for type "Email"
```

(exit code 1)

## What this teaches

- A `types:` entry has three optional fields applied in order: `base`, `pattern`, `options`. Any of them can fail and abort transpilation with a clear message.
- Type names referenced from a function's `args[].type` must resolve at load time — typos are caught early.
- Type validation runs against the **source text** of the captured argument (with surrounding quotes stripped for string literals).
