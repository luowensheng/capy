# multi-language-demo

**One Capy source file compiled to five programming languages.**

The same `script.capy` (10 lines) is parsed by five different libraries
to produce equivalent — and actually runnable — programs in:

- Python  (`lib_python.yaml`     → `python3`)
- JavaScript (`lib_javascript.yaml` → `node`)
- Go      (`lib_go.yaml`         → `go run`)
- Rust    (`lib_rust.yaml`       → `rustc`)
- C       (`lib_c.yaml`          → `gcc`)

Each library compiles into the target's idioms: Python uses
`if __name__ == "__main__":`, JS wraps in an IIFE, Go uses `:=`
short declarations and adds `package main`/`import "fmt"`, Rust types
everything with `i32`, C adds `#include <stdio.h>` and a `return 0` to
`main`.

## Run

```sh
# 1. Python — runs and prints 12
../../capy run lib_python.yaml     script.capy | python3

# 2. JavaScript — runs and prints 12
../../capy run lib_javascript.yaml script.capy | node

# 3. Go — save and `go run`
../../capy run lib_go.yaml         script.capy > demo.go && go run demo.go

# 4. Rust — save and rustc
../../capy run lib_rust.yaml       script.capy > demo.rs && rustc demo.rs -o demo && ./demo

# 5. C — save and gcc
../../capy run lib_c.yaml          script.capy > demo.c  && gcc demo.c -o demo && ./demo
```

All five produce `12`.

## The shared source

```
fn add(a, b)
    return a + b
end

main
    let x = 5
    let y = 7
    let z = add(x, y)
    print z
end
```

This is *Capy syntax*, defined by the libraries. Each library declares
the same `fn`, `return`, `main`, `let`, `let_call`, `print`, and `end`
patterns — so each can parse this source. They differ only in their
`template:` fields.

## Why this is the killer demo

Maintaining the "same logic in N languages" problem is a real one:

- An algorithm needs both a Python research script and a C++
  production implementation.
- A client library ships in 6 languages and they drift.
- A data validator runs in the browser (JS) and the server (Python).

Today the answer is "write it N times and hope they stay in sync."

With Capy, you write the logic **once** in a small DSL. Add a target
language by writing a new ~50-line library. The next time you change
the algorithm, all five outputs regenerate.

This isn't pretending to be a real compiler — addition + variable
assignment + one function is the entire grammar. But the *pattern*
scales: more patterns in the library → more language constructs
supported.

## Add a sixth language

Want Java? Kotlin? Swift? Zig? Write a 50-line `lib_X.yaml` with the
seven patterns above translated to X's idioms, and that target lights
up without touching the source.

That's the point.

## Same library, two formats

Every library in this directory ships **twice**:

```
lib_python.yaml      lib_python.capy
lib_javascript.yaml  lib_javascript.capy
lib_go.yaml          lib_go.capy
lib_rust.yaml        lib_rust.capy
lib_c.yaml           lib_c.capy
```

Both formats produce byte-identical output. Capy's loader dispatches
on file extension:

```sh
# YAML form
../../capy run lib_c.yaml script.capy

# Capy-native form
../../capy run lib_c.capy script.capy

# Diff — empty, they match exactly
diff <(../../capy run lib_c.yaml script.capy) <(../../capy run lib_c.capy script.capy)
```

YAML wins on tooling (yq, JSON schema, every editor). `.capy` wins on
ergonomics — one syntax to learn, native multi-line templates, no
YAML escape gotchas. Pick whichever fits your workflow.
