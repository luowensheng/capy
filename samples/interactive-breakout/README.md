# interactive-breakout

**Event-driven Breakout DSL → 226-line playable HTML5 game.**

The source isn't just config — it declares **entities, key bindings,
AND event handlers**. The library compiles those declarations into a
complete game with an action table, input dispatch, and event routing.

## The DSL

```
game "Breakout" 480 320

# Entity declarations
paddle width 80 height 10 color "#3df" speed 7
ball   radius 6 color "#fff" speed 4

bricks rows 5 cols 8 width 56 height 14 gap 4
brick_color 0 "#f55"
brick_color 1 "#fa4"
brick_color 2 "#ff4"
brick_color 3 "#4f6"
brick_color 4 "#4af"

# Input bindings — key → action
on_key "ArrowLeft"  paddle_left
on_key "ArrowRight" paddle_right
on_key " "          launch_ball
on_key "r"          reset

# Event handlers — game event → action(s)
on_event brick_hit   destroy_brick add_score 10
on_event paddle_hit  bounce_with_spin
on_event ball_lost   lose_life
on_event all_cleared win

lives 3
```

### The shape of the language

Three statement kinds carry behavior:

| Statement | Meaning |
|---|---|
| `on_key "KEY" action` | When `KEY` is pressed (or held, for arrows), call `action`. |
| `on_event NAME action` | When game event `NAME` fires, call `action`. |
| `on_event NAME action1 action2 N` | Two-action form, with an integer arg passed to `action2`. |

The library defines a fixed set of **events** (`brick_hit`, `paddle_hit`,
`ball_lost`, `all_cleared`) and **actions** (`paddle_left`,
`paddle_right`, `launch_ball`, `destroy_brick`, `add_score`,
`bounce_with_spin`, `lose_life`, `win`, `reset`). The DSL binds them.

## Run

```sh
../../capy run lib.yaml script.capy > breakout.html
open breakout.html
```

## What the library generates

The library's `file_template` enumerates every `on_key` and `on_event`
declaration into a JS dispatch table:

```javascript
const KEY_BINDINGS = {
  "ArrowLeft": "paddle_left",
  "ArrowRight": "paddle_right",
  " ": "launch_ball",
  "r": "reset",
};

const EVENT_HANDLERS = {
  brick_hit: (arg) => {
    ACTIONS["destroy_brick"](arg);
    ACTIONS["add_score"](arg, 10);
  },
  paddle_hit:  (arg) => { ACTIONS["bounce_with_spin"](arg); },
  ball_lost:   (arg) => { ACTIONS["lose_life"](arg); },
  all_cleared: (arg) => { ACTIONS["win"](arg); },
};
```

Plus an `ACTIONS` table of named functions that implement each action,
and a game loop that calls `emit("brick_hit", brick)` etc. at the right
moments. Change a binding → re-run → new game. Add a key → no engine
code changes.

## Why this matters

The previous version of this demo took 4 numbers and produced the same
game every time. This version's source carries **real behavioral
declarations** — the DSL ships event-handling primitives, not just
parameter knobs.

A user can rebind keys without touching JS. A library author can add a
new event (`combo_break`?) and any DSL using it picks it up. The
declaration *is* the behavior; templates render it into runnable code.
