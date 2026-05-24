# Tutorial 1: Hello World DSL

Build the smallest possible Capy library: one function, one template,
nothing else. Estimated time: 5 minutes.

## Goal

Write a Capy library so this source:

```
greet "Alice"
greet "Bob"
```

Produces this output:

```
Hello, Alice!
Hello, Bob!
```

## Step 1 — scaffold

```sh
mkdir hello-dsl
cd hello-dsl
capy init .
```

You now have `lib.yaml`, `script.capy`, and `README.md`.

## Step 2 — define the function

Open `lib.yaml`. Replace it with:

```
extension txt

function greet
    arg capture name any
    write `Hello, ${name}!
`
end
```

What this says:

- One function named `greet`.
- One `arg` — a capture named `name`, accepting any value.
- Because there are zero `arg literal` lines, the engine
  **auto-prepends** the function name (`greet`) as a leading literal.
  So the surface shape is `greet <any>`.
- The function body `write`s `Hello, <name>!\n` per match. The
  newline inside the backtick literal is exactly what gets emitted.

## Step 3 — write the source

Replace `script.capy` with:

```
greet "Alice"
greet "Bob"
```

## Step 4 — run it

```sh
capy run lib.capy script.capy
```

Output:

```
Hello, Alice!
Hello, Bob!
```

## Step 5 — what just happened

The lexer tokenized each line into `greet` + `"Alice"`. The parser
matched against the `greet` function shape, captured the name, and the
evaluator rendered the template with `{{ .name }} = "Alice"` (note: the
source text includes the quotes because the input is a quoted string
literal).

## Try it

- Change the write to `` write `hi ${name}\n` ``. Re-run. Output updates.
- Add a second function `shout` that emits uppercase using the
  `upper` helper: `` write `${upper name}\n` ``.
- Try invalid input: `greet` (no argument) → see the structured error.

## Next

[Tutorial 2: Building a config DSL](02-config-dsl.md) introduces
**types**, **context accumulation**, and the **file template**.
