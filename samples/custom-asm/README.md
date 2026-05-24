# Custom assembly DSL — many architectures, one source

A tiny architecture-neutral assembly DSL (`data`, `func`, `write`,
`exit`, `end`). Swap the library to retarget:

| Library                       | Target           | Syscall trap | Args                |
|-------------------------------|------------------|--------------|---------------------|
| `lib-x86_64-linux.capy`       | x86_64 Linux     | `syscall`    | %rdi %rsi %rdx %rax |
| `lib-arm64-linux.capy`        | AArch64 Linux    | `svc #0`     | x0 x1 x2 x8         |
| `lib-riscv64-linux.capy`      | RV64I Linux      | `ecall`      | a0 a1 a2 a7         |

## Run

```sh
capy run lib-x86_64-linux.capy  script.capy > hello.s
capy run lib-arm64-linux.capy   script.capy > hello.s
capy run lib-riscv64-linux.capy script.capy > hello.s
```

Then on a matching host (or via a cross-toolchain):

```sh
# x86_64
gcc -nostdlib -static hello.s -o hello && ./hello

# arm64 from arm64 Linux
aarch64-linux-gnu-gcc -nostdlib -static hello.s -o hello && ./hello

# riscv64 from riscv64 Linux
riscv64-linux-gnu-gcc -nostdlib -static hello.s -o hello && ./hello
```

All three output:

```
Hello, world
```

## What this demonstrates

- **The source IS the spec.** 5 lines describe the program intent.
  The library decides ABI, register conventions, syscall numbers,
  instruction syntax.
- **Adding a new target = adding a library.** Want POWER? MIPS?
  s390x? FreeBSD x86_64 (different syscall numbers)? Just write a
  new `lib-<arch>-<os>.capy` — the script doesn't change.
- **You can branch on `(os)` and `(arch)`** if you want a single
  smart library that handles multiple targets internally. The
  [host capabilities](../../docs/host-capabilities.md) page covers
  the OS/arch introspection primitives.

## Why not just write the assembly directly?

For a hello world, you don't gain much. The pattern shines when:

- You're emitting code-generation TEMPLATES, not one-off programs.
  E.g., a JIT-style compiler that generates inlined per-arch code.
- You're cross-compiling and want byte-identical output for
  identical source intent across architectures.
- You're teaching assembly and want students to write a single
  algorithmic description that materialises in their host's ISA
  without dragging in a giant compiler.
