# transpile-py

The canonical Capy example: a tiny source language that transpiles to **Python**.

## Files

- `lib.yaml` — defines `import`, `say`, `assign` (`x = ...`), `if`, `loop`, plus the `end` closer.
- `script.capy` — a small program with imports, assignments, a conditional, and a loop.

## Run

```sh
../../capy -lib lib.yaml script.capy
```

## Expected output

```python
import json
import os
print("starting")
x = 42
items = ["a", "b", "c"]
if x:
    print("x is truthy")

for item in items:
    print(item)
```

## What this teaches

- **Templates emit target-language text.** Each function's `template:` contributes a fragment to the body. Captures appear as their source text — so `x = 42` in the source emits literally `x = 42` in the output.
- **`run:` updates context without running user code.** The `import` function's `template:` is empty (no body contribution); its `run:` block calls `append context.imports name`. The file template later renders the accumulated `context.imports` at the top of the output.
- **Block functions use `body`.** The `if` and `loop` functions are block openers. Their templates reference `{{ .body }}` — the already-rendered string concatenation of all child statements. The `| indent 4` helper indents that body for Python's syntax.
- **`file_template:` is the final assembler.** It receives `.body` (the full rendered program) and `.context` (final accumulated state) and produces the output file.

## Source-text vs evaluated values

- In **templates**, captures resolve to source text. `say "hello"` exposes `msg` as the literal text `"hello"` (with quotes), so the Python template `print({{ .msg }})` produces `print("hello")` naturally.
- In **`run:` snippets**, captures resolve to evaluated values. `append context.imports name` appends the Go string `"json"` (no surrounding quotes) so `range .context.imports` yields each import unwrapped.

This dual model lets one capture serve both render-by-text (templates) and structured-accumulation (context) without ceremony.
