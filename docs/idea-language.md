---
title: Capy as an idea language
---

# Capy as an idea language

The most ambitious claim in this docs site: **Capy is a language for
describing ideas. Libraries are implementers of the idea.**

The implication: when you outgrow an implementation — when the Go
server isn't fast enough and you want Rust, when the React app
needs to also exist as SwiftUI, when the SQL needs to also be a
GraphQL schema — you **swap the library, not the source**.

You can continually improve the implementation without ever
rewriting the idea.

## What changes with this framing

Conventional software:

```
Idea (in someone's head)
  → coded in Go
  → ...
  → "we should rewrite this in Rust for performance"
  → 3 months of work, business value during which: zero
  → bugs reintroduced, behaviour drift, feature freeze
```

With Capy:

```
Idea (in script.capy)
  → lib_go.capy   → Go implementation
  → lib_rust.capy → Rust implementation (added when needed)
  → run both, diff outputs, switch over, retire lib_go.capy
```

The idea — what the system does, end to end — never changed. Only
the artifact changed. The team didn't have to re-design anything;
they re-targeted a stable contract.

## Already shipped: a small example

[`samples/multi-language-demo/`](https://github.com/luowensheng/capy/tree/main/samples/multi-language-demo)
is the canonical demonstration. One 10-line source:

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

Five libraries, one per target language — Python, JavaScript, Go,
Rust, C. Each emits real, runnable code in its target's idioms:

- Python: `def add(a, b):` / `if __name__ == "__main__":`
- JavaScript: `function add(a, b)` / IIFE wrapping main
- Go: `package main` + `import fmt` + `:=` short declarations
- Rust: `i32` types + `println!` macro
- C: `#include <stdio.h>` + `int main(void)` + `return 0`

Verified: Python and JS outputs both `print 12` when actually
executed.

The source never changed; the implementation language did.

## The deeper claim

Programming languages already let us separate **what** from **how**
within one paradigm — that's what compilers do. Capy extends the
separation **across paradigms, runtimes, and stacks**.

Concretely:

1. **The idea is structured text** (the `script.capy`).
2. **Each library is one implementation of that idea** — a function
   from idea to a specific target's source code.
3. **Functions compose, libraries can be swapped, and the idea
   stays the same.**

This is closer in spirit to "interface and implementation" than to
traditional code generation. The library IS an implementation; the
script IS an interface call.

## Where this actually pays off

Real workflows that fit:

- **A research script wants a production version.** Your team has a
  Python prototype that an analyst uses. Performance becomes an
  issue. With Capy: declare the algorithm once in a Capy DSL, write
  `lib_python.capy` and `lib_cpp.capy`. The analyst keeps using the
  Python output; production uses the C++ output. They never drift
  because they're the same source.

- **A client library ships in 6 languages.** Stripe, Twilio, every
  API company maintains 6 SDKs. They drift constantly. The DSL is
  the API spec; each library targets one host language. New endpoint
  in the spec → all 6 SDKs regenerate cleanly.

- **A platform team wants Rust without a rewrite.** Performance
  pressure on a Go service. With Capy, the team adds `lib_rust.capy`
  alongside `lib_go.capy`, runs benchmarks, switches over a service
  at a time. The DSL stays the same; the runtime moves.

- **Cross-platform mobile.** [`samples/android-app/`](https://github.com/luowensheng/capy/tree/main/samples/android-app)
  and [`samples/ios-app/`](https://github.com/luowensheng/capy/tree/main/samples/ios-app)
  use the **same source shape**. One declaration, two platforms.
  Same idea ("a Habit Tracker with two screens"), two
  implementations.

- **Same backend, two stacks** — the design-system pattern from
  [`samples/design-system-components/`](https://github.com/luowensheng/capy/tree/main/samples/design-system-components)
  works for React, Vue, and Svelte from the same UI declaration.
  Identical visual semantics across three frameworks.

## What this isn't

- **Not a replacement for the implementing language.** You still
  need to know Rust to write a good Rust library. The library is
  *your team's expertise in Rust*, expressed once and reused
  forever.
- **Not zero-cost.** Designing a library that targets two stacks
  cleanly takes thought — you have to find the abstraction that
  fits both without leaking host-language quirks into the DSL.
- **Not universal.** Some ideas don't compress well into a
  declarative shape (anything fundamentally control-flow-heavy,
  anything with rich runtime semantics). Capy is for declarative
  ideas: APIs, schemas, configs, scenes, components, models, plans,
  documents. Most software, but not all.

## What it asks of you

The thing you have to design once, and only once, is **the DSL
itself**. It's the contract. If the contract is wrong, no library
can save you. If the contract is right, you can keep adding
implementations forever.

This is why [`docs/grammar-as-contract.md`](grammar-as-contract.md)
and [`docs/library-authoring.md`](library-authoring.md) matter so
much. They're not optional skills — they're what separates a Capy
project that scales from one that bit-rots.

## A short philosophical aside

Most rewrites aren't about ideas changing. They're about hosts
changing — the language, the framework, the runtime, the cloud,
the company's "we use Kotlin now" decision. The IDEAS are usually
fine. Capy gives you a vocabulary in which to write *ideas* — and
hosts become details.

That's the bet. The samples in this repo are evidence; the docs
linked above are the playbook.
