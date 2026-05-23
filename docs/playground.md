---
title: Playground (try it in your browser)
hide:
  - navigation
  - toc
---

# Try Capy in your browser

Capy compiles to WebAssembly. The playground below runs the **real
compiler** — no server round-trip — on six curated sample DSLs
you can edit live. Press **Run** (or ⌘/Ctrl + Enter) to regenerate;
press **Download** to grab the result.

<iframe src="assets/playground/index.html"
        width="100%"
        height="720"
        style="border: 1px solid #30363d; border-radius: 8px; background: #0d1117;"
        loading="lazy"
        title="Capy playground"></iframe>

## What you can do here

- **Pick a sample** from the dropdown (recipe / invitation / meal
  plan / reading log / Breakout / Snake).
- **Edit the script** in the left pane. Press Run; the right pane
  updates.
- **Preview HTML output** inline (recipe cards, invitation, games
  all render directly).
- **Download** what you made — single file or zip archive for
  multi-file outputs.
- **Open the library** (click the disclosure at the bottom) to see
  or modify the grammar that powers the sample.

## How it works

```
your browser
    │
    ├─ index.html  ── the playground UI
    ├─ wasm_exec.js ── Go's standard wasm runtime
    └─ capy.wasm   ── the Capy compiler, ~6.2 MB,
                       loaded once, cached forever
```

The `capy.wasm` module is the same engine that ships with the CLI
and the MCP server, compiled with `GOOS=js GOARCH=wasm`. It exposes
a single function `capyRun(lib, format, script)` returning the
rendered output and (for multi-file libraries) the full file map.

Source: [`cmd/capy-wasm/main.go`](https://github.com/luowensheng/capy/blob/main/cmd/capy-wasm/main.go).

## Run the playground locally

```sh
git clone https://github.com/luowensheng/capy
cd capy
scripts/build-playground.sh        # builds capy.wasm + bundles samples.json
python3 -m http.server -d docs/assets/playground 8000
# open http://localhost:8000
```

The `scripts/build-playground.sh` script does three things:

1. Compiles `cmd/capy-wasm` to `capy.wasm` with `GOOS=js GOARCH=wasm`.
2. Copies Go's `wasm_exec.js` runtime alongside it.
3. Runs `cmd/playground-bundle` to bake the six curated samples into
   `samples.json`.

CI re-runs all three on every deploy.

## What if I want my OWN library in the playground?

Edit the library pane (expand the disclosure at the bottom) and the
playground will use it instead. Or fork the repo and add an entry to
the `CURATED` list in `cmd/playground-bundle/main.go`.

## What it can't do (yet)

- **No source `@import "..."`** — the playground has no filesystem.
  Inline everything into one `script.capy`.
- **No file system writes** — multi-file libraries still work
  (you'll see one tab per file in the output), but the files only
  exist in the browser until you click Download.

## Limits

The wasm bundle is ~6 MB. It loads once and is cached by the
browser. After that, every Run is a few milliseconds (the
compilation is pure CPU, no network).

Most libraries run in well under 100 ms even for the largest
samples in the picker. If you're using your own multi-thousand-line
library, expect proportionally longer compile times.
