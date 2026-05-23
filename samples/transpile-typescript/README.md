# transpile-typescript

Declarative model definitions → TypeScript interfaces.

## Run

```sh
../../capy run lib.yaml script.capy
```

## What this teaches

- Mode A blocks (`model ... end`) for indented bodies.
- Both `template:` and `run:` on the same function: emit the interface,
  AND track the model name for a comment summary in `file_template:`.
- A model body's children (`field ... : ...`) render in source order
  into the body string.
