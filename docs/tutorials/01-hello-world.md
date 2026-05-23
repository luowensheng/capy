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

```yaml
extension: txt

functions:
  greet:
    args:
      - { kind: capture, name: name, type: any }
    template: "Hello, {{ .name }}!\n"
```

What this says:

- One function whose name is `greet`.
- `args:` has one entry — a capture named `name`, accepting any value.
- Because there are zero `kind: literal` entries, the engine
  **auto-prepends** the function key (`greet`) as a leading literal. So
  the surface shape is `greet <any>`.
- The `template:` renders `Hello, <name>!\n` per match.

## Step 3 — write the source

Replace `script.capy` with:

```
greet "Alice"
greet "Bob"
```

## Step 4 — run it

```sh
capy run lib.yaml script.capy
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

- Change the template to `"hi {{ .name }}\n"`. Re-run. Output updates.
- Add a second function `shout` that emits uppercase using the `upper`
  helper: `template: "{{ .name | upper }}\n"`.
- Try invalid input: `greet` (no argument) → see the structured error.

## Next

[Tutorial 2: Building a config DSL](02-config-dsl.md) introduces
**types**, **context accumulation**, and the **file template**.
