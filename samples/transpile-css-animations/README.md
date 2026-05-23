# transpile-css-animations

Keyframe + class DSL → complete CSS file with `@keyframes`, rules, and
animation bindings.

```sh
../../capy run lib.yaml script.capy > styles.css
```

## What this teaches

- **Two block kinds** (`keyframe`, `class`) sharing a single closer (`end`).
- **Operator-style properties** — `prop: name = value` has no leading
  function-name literal because args includes the `=` literal.
- **Mixed inline-and-context output** — `animate` is a flat statement
  that drops a one-line `animation: ...` into the current rule, while
  the wrapping `class` block builds the selector and body around it.
- The script is ~20 lines; the output is a full CSS sheet you can drop
  into a real page.
