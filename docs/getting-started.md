# Getting Started

A five-minute tour of Capy. By the end you'll have run the canonical
Python-transpiling sample and understand the four things every Capy library
controls: `functions`, `types`, `context`, and `file_template`.

A Capy library is **a file written in Capy's own syntax** (`.capy`).
YAML is also accepted as a secondary format for teams that need
yq / JSON-schema tooling — see [§ Also supported: YAML](library-authoring.md#also-supported-yaml).
Examples in this doc are all `.capy`.

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
capy run samples/transpile-py/lib.capy samples/transpile-py/script.capy
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

1. The library (`lib.capy`). It declares **functions** (each function
   has an `arg` shape, a `template:` for output, and an optional
   `run:` for state updates).
2. `script.capy` — the input source.

Capy matched each line of the source against the library's function
shapes. For each match:

- The function's `template:` was rendered into the **output body**
  with the captured values substituted.
- The function's `run:` (if any) updated the **accumulated context**
  (lists, maps, scalars).

At the end, the library's `file_template:` produced the final output
from `.body` (concatenated rendered fragments) and `.context` (final
accumulated state).

Concretely, `import json` matches a function that looks like this:

```
function import
    arg literal "import"
    arg capture name ident
    template_str ""              # contributes nothing to body
    run:
        append context.imports name
end
```

`import json` adds `"json"` to `context.imports`. The file template
at the top of the output emits `import json` from there.

## 4. Make your own

```sh
mkdir my-dsl
cd my-dsl
capy init .
capy run lib.capy script.capy
```

`capy init` scaffolds a starter library + script. Edit `lib.capy`
to add your own functions, types, and file template.

## 5. Where to go next

- [Library Authoring](library-authoring.md) — the reference walkthrough.
- [Cookbook](cookbook.md) — short answers for common patterns.
- [Tutorials](tutorials/01-hello-world.md) — four progressive walkthroughs.
- The [samples/](https://github.com/luowensheng/capy/tree/main/samples) directory on GitHub — 50 end-to-end demos.
