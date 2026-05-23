# interactive-breakout

**4 lines of Capy DSL → 174-line playable HTML5 Breakout.**

The source:

```
game     "Breakout"  480 320
paddle   80 10  7
ball     6 4 4
bricks   5 8 56 14 4
```

That's it. Four numbers configure the paddle, three configure the ball,
five configure the brick wall. The library does the rest:

- Paddle with left/right arrow controls + edge clamping
- Ball with launch (space), wall bounces, paddle bounce with spin
  influenced by where the ball hits the paddle
- 5×8 brick wall with per-row colors and per-row score values
- Collision detection that flips X or Y velocity depending on
  approach angle
- Live HUD with score and lives
- Game-over and win screens with restart (R)

## Run

```sh
../../capy run lib.yaml script.capy > breakout.html
open breakout.html   # or xdg-open / start
```

## Why this matters

The 4-line source contains zero game logic. All of that lives in
`lib.yaml`'s `file_template`. The DSL is **purely configuration** —
proof that Capy can produce real, interactive output, not just toy
"hello world" code.

To change the difficulty: edit the four numbers. To change the look:
edit `lib.yaml`. The source file never has to know about
`requestAnimationFrame`, collision math, or canvas APIs.
