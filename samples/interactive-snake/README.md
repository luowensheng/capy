# interactive-snake

**Event-driven Snake DSL → 180-line playable HTML5 game.**

The source isn't just config — it declares **key bindings AND event
handlers** that the library compiles into a complete game with input
dispatch and game-state events.

## The DSL

```
game "Snake" 400 400
grid cols 20 rows 20
tick every 110

# Input — arrows AND WASD bound to the same actions
on_key "ArrowUp"    turn_up
on_key "ArrowDown"  turn_down
on_key "ArrowLeft"  turn_left
on_key "ArrowRight" turn_right
on_key "w"          turn_up
on_key "s"          turn_down
on_key "a"          turn_left
on_key "d"          turn_right
on_key " "          pause_toggle
on_key "r"          reset

# Game events
on_event eat_food   grow add_score 10
on_event hit_wall   game_over
on_event hit_self   game_over

snake_color "#9fa"
food_color  "#f44"
save_best   "snake_best"
```

### What you can do without touching the library

- **Remap controls** — change every `on_key` line and re-run.
- **Adjust scoring** — `on_event eat_food grow add_score 25` for double points.
- **Bind a second key to the same action** — see how `w` and `ArrowUp`
  both call `turn_up`.
- **Disable game-over on self-hit** — delete the `on_event hit_self
  game_over` line and the snake passes through itself.

## Run

```sh
../../capy run lib.yaml script.capy > snake.html
open snake.html
```

## How the events wire up

The library's game loop calls `emit("eat_food")` when the snake's head
overlaps food, `emit("hit_wall")` when the head goes off-grid, and
`emit("hit_self")` when the head overlaps the body. Each `emit` looks
up the event in the `EVENT_HANDLERS` table the DSL generated:

```javascript
const EVENT_HANDLERS = {
  eat_food: (arg) => {
    ACTIONS["grow"](arg);
    ACTIONS["add_score"](arg, 10);
  },
  hit_wall: (arg) => { ACTIONS["game_over"](arg); },
  hit_self: (arg) => { ACTIONS["game_over"](arg); },
};
```

`ACTIONS` is a fixed table of named functions defined in the library's
`file_template`. The DSL just glues events to action names.

## Why this design

A pure-config DSL (`speed 110`, `grid 20 20`) is fine for tweaking
parameters but doesn't show off Capy's expressive range. By moving key
bindings and event handlers into the DSL, the source becomes a tiny
**behavior description language** — the kind of thing you'd otherwise
hand-write JS for. The library does the boring parts (canvas, timer,
collision math); the DSL says *what should happen* at each event.
