# assembly

A tiny high-level source DSL → x86-64 NASM assembly. Compiles down to a
real `.asm` file that `nasm` + `ld` can turn into an executable.

```sh
../../capy run lib.yaml script.capy > demo.asm
nasm -felf64 demo.asm -o demo.o
ld demo.o -o demo
./demo; echo "exit=$?"
```

## What this teaches

- **Compile-time symbol tracking.** Each `var x = 5` registers `x` in
  `context.vars` via a `run:` snippet. The `program` block's template
  emits the `.data` section by ranging over the accumulated list — so
  the variable table is built up as a side effect of seeing each
  declaration in source.
- **A real lowering pipeline.** Each source statement lowers to multiple
  assembly instructions. `add x y` emits three lines; `var x = 5` emits
  two; `exit 0` emits the sys_exit prelude + syscall.
- **Mixing `template:` and `run:` on the same function.** `var`'s
  template emits the runtime initialisation while its run snippet
  records the symbol for the data section.
- **Two-pass-style codegen with one pass.** The body is rendered first
  (top-down), then the `program` block's template references both
  `.context.vars` (the accumulated symbols) AND `.body` (the rendered
  statements) — giving you a `.data` section automatically synced with
  what was declared.
- **Escape hatch via `mov`.** A generic `mov <reg> <val>` is exposed so
  programs can drop to raw assembly when they need to.

## Source

```capy
program "sum-demo"
    var x = 5
    var y = 7

    add x y
    store result

    exit 0
end
```

## Output (x86-64 NASM)

```asm
; program: sum-demo

section .data
    x: dq 0
    y: dq 0

section .text
    global _start

_start:
    ; var x = 5
    mov rax, 5
    mov [x], rax
    ; var y = 7
    mov rax, 7
    mov [y], rax

    ; add x y
    mov rax, [x]
    add rax, [y]
    mov [result], rax

    ; exit 0
    mov rdi, 0
    mov rax, 60
    syscall
```

Note `result` is only used by `store` — it's not declared as a `var`,
so it doesn't appear in `.data`. A more elaborate library would lint
for that.
