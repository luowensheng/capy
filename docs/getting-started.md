# Getting Started

A five-minute tour of Capy. By the end you'll have run the canonical
Python-transpiling sample and understand the four things every Capy library
controls: `functions`, `types`, `context`, and `file_template`.

## 1. Install

```sh
# Go users
go install github.com/luowensheng/capy/cmd/capy@latest

# Or download a binary from
# https://github.com/luowensheng/capy/releases
```

Verify:

```sh
capy version
```

## 2. Run a sample

Clone the repo (samples live there) and run the Python transpiler:

```sh
git clone https://github.com/luowensheng/capy
cd capy
capy run samples/transpile-py/lib.yaml samples/transpile-py/script.capy
```

Expected output:

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

## 3. What just happened

Capy read **two files**:

1. `lib.yaml` — the library. It declares **functions** (each function has an `args:` shape, a `template:` for output, and optional `run:` for state updates).
2. `script.capy` — the input source.

Capy matched each line of `script.capy` against the library's function shapes. For each match:

- The function's `template:` was rendered into the **output body** with the captured values substituted.
- The function's `run:` (if any) updated the **accumulated context** (lists, maps, scalars).

At the end, the library's `file_template:` produced the final output from `.body` (concatenated rendered fragments) and `.context` (final accumulated state).

Concretely, `import json` matches:

```yaml
import:
  args:
    - { kind: literal, value: "import" }
    - { kind: capture, name: name, type: ident }
  template: ""              # contributes nothing to body
  run: |
    append context.imports name
```

So `import json` adds `"json"` to `context.imports`. The file template at the top of the output emits `import json` from there.

## 4. Make your own

```sh
mkdir my-dsl
cd my-dsl
capy init .
capy run lib.yaml script.capy
```

`capy init` scaffolds a starter library + script. Edit `lib.yaml` to add your own functions, types, and file template.

## 5. Where to go next

- [Library Authoring](library-authoring.md) — the reference walkthrough.
- [Cookbook](cookbook.md) — short answers for common patterns.
- [Tutorials](tutorials/01-hello-world.md) — four progressive walkthroughs.
- The [samples/](https://github.com/luowensheng/capy/tree/main/samples) directory on GitHub — 50 end-to-end demos.
