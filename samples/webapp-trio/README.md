# webapp-trio

**One Capy source → three files: `index.html`, `app.js`, `styles.css`.**

A complete browser-ready habit-tracker web app. The HTML structure,
JavaScript behavior, and CSS styling each get their own file, with
the right cross-references (`<link rel="stylesheet">`, `<script src>`).

## What you write

```
app "Habit Tracker"
    description "A tiny daily-habit tracker that persists to localStorage."
    color_primary "#4f46e5"
    color_bg "#0f172a"

    habit drink_water  "Drink 8 glasses of water"
    habit read         "Read 20 pages"
    habit walk         "30 min walk"
    habit code         "Practice coding"
    habit meditate     "Meditate 10 min"
end
```

12 lines.

## What you get

```
out/
├── index.html       ← structure, references styles.css + app.js
├── app.js           ← localStorage persistence + streak counting
└── styles.css       ← theme-aware styles (CSS variables)
```

A real working app:

- Checkboxes for each declared habit
- Persistence per-day in `localStorage`
- Streak counter (consecutive days where every habit was checked)
- Reset button + today's date display
- Theme respects the `color_primary` and `color_bg` you set

## Run

```sh
../../capy run --out-dir out lib.capy script.capy
open out/index.html
```

…or bundle to a zip for sharing:

```sh
../../capy run --zip habit-tracker.zip lib.capy script.capy
```

## Why this matters

The three target files (HTML / JS / CSS) are usually written in three
different paradigms by three different parts of a team — and they
drift. With Capy:

- One DSL declares the data and theme.
- The library knows how to wire them together correctly (the HTML
  references the JS by the right name; the CSS variables get the
  colors from the source).
- Add a habit, regenerate. All three files stay in sync.
