# interactive-snake

**3 lines of Capy DSL → 131-line playable HTML5 Snake.**

The source:

```
game  "Snake"  400 400
grid  20 20
speed 110
```

What you get:

- Arrow keys / WASD movement with anti-reverse protection
- 20×20 grid with food spawning that avoids the snake
- Snake body with per-segment color gradient
- Score + best-score in `localStorage` (survives page reload)
- Space to pause/resume, R to restart
- Game-over screen

## Run

```sh
../../capy run lib.yaml script.capy > snake.html
open snake.html
```

## How it works

Three statements; three context fields populated. The whole game lives
in `lib.yaml`'s `file_template`. To make the snake faster, change the
`speed` number (it's milliseconds per tick). To resize the grid,
change `grid`.

The point of the demo: Capy DSLs can be tiny *configuration files*
where each statement updates one field of the accumulated context,
and the file_template emits a complete, polished, interactive
artifact. The DSL doesn't model game logic; the library does.
