# transpile-canvas-game

Compact game DSL → a **complete runnable HTML5 canvas page** with
sprites, key handlers, and a `requestAnimationFrame` game loop.

```sh
../../capy run lib.yaml script.capy > game.html
open game.html   # or xdg-open on Linux
```

10 lines of source produce ~50 lines of HTML+CSS+JS that boots a
playable demo. The arrow keys move the player; an enemy sprite auto-
moves via a `tick` raw-JS escape hatch.

## What this teaches

- **Compact DSL → substantial output.** The whole game spec is 10
  lines; the generated HTML is a self-contained, runnable page.
- **Multi-section accumulation.** Sprites, key handlers, and tick
  callbacks each live in their own context list. The file template
  weaves them into the right places of the JS.
- **Raw-JS escape hatch via `tick`.** When the DSL doesn't cover a
  behavior, the user drops to inline JS that the file template
  embeds verbatim.
- **Captures route into JS literals.** Strings become JS strings via
  `toQuoted`; numbers stay as numbers.
