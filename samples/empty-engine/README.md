# empty-engine

A library with no `functions:` and no `types:`. Proves Capy's central claim: **the engine has zero default grammar**.

## Run

```sh
../../capy -lib lib.yaml script.capy
```

## Expected output

```
capy: line 1: no library function matches token "x"
```

(exit code 1)

## What this teaches

- `x = 1` is not built-in. There's no parser path for it. The only way to make `x = 1` valid is to add a function whose `args:` lists `[{kind: capture, name: name, type: ident}, {kind: literal, value: "="}, {kind: capture, name: value, type: any}]`.
- If you want to see assignment work, look at [transpile-py/lib.yaml](../transpile-py/lib.yaml) — its `assign` function adds exactly that shape.
