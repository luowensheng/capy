# transpile-env

Typed config DSL → `.env` file. Demonstrates regex-validated identifier
names.

## Run

```sh
../../capy run lib.yaml script.capy
```

## What this teaches

- A library-defined type (`EnvName`) enforces SCREAMING_SNAKE naming.
- `set lowercase_key` would fail with a clear validation error.
- The captured `value` flows through as source text — strings keep their
  quotes, numbers don't.
