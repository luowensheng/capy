---
title: Library commands + library path
hide:
  - toc
---

<div class="capy-hero" markdown>

<span class="capy-eyebrow">v0.19 — LIBRARIES AS CLIs</span>

# Install libraries. Run them by name. Ship commands.

Libraries live on a search path. Reference them by name from any
directory. Declare your own commands (`run`, `build`, `serve`, …)
that can shell out, write files, and produce real artifacts —
not just stdout.

</div>

## The big picture

Three new things in v0.19, each useful on its own and great
together:

1. **`CAPY_LIBS` search path.** Install a library once; reference
   it by name from anywhere.
2. **Library commands.** A library can declare custom verbs in
   its `capy.capy` manifest. Each command body runs against a
   small shell-flavoured inner DSL — `let`, `exec`, `write_file`,
   `mktemp`, `print`, `cd`, plus the `compile` primitive that
   runs the library on a script.
3. **`capy new`.** Scaffold a project from a library. If the
   library declares a `new` command, it gets called with the
   project directory; otherwise a tiny default scaffold lands.

## A 60-second tour

```sh
# 1. Set the search path (default: ~/.config/capy/libs/ + ~/.capy/libs/).
export CAPY_LIBS=~/.capy/libs

# 2. Scaffold a new library.
capy lib new recipe
# ✓ created library "recipe" at ~/.capy/libs/recipe
#   capy recipe run ~/.capy/libs/recipe/examples/hello.recipe

# 3. Use it.
capy recipe run ~/.capy/libs/recipe/examples/hello.recipe
# Hello from recipe, world!

# 4. Build (uses the library's `compile` command).
capy recipe compile ~/.capy/libs/recipe/examples/hello.recipe
# wrote ~/.capy/libs/recipe/examples/hello.recipe.txt

# 5. Auto-detect by file extension.
echo 'greet "ext"' > /tmp/cake.recipe
capy /tmp/cake.recipe
# Hello from recipe, ext!

# 6. Scaffold a project that uses the library.
capy new my-app --using recipe
# ✓ created project "my-app" using library "recipe"
```

## `CAPY_LIBS` search path

`capy` resolves library names against a colon-separated
(semicolon on Windows) path list. Defaults:

| Platform | Default |
|---|---|
| Linux | `$XDG_CONFIG_HOME/capy/libs/` (and `~/.capy/libs/` as a fallback) |
| macOS | `~/Library/Application Support/Capy/libs/` |
| Windows | `%APPDATA%\Capy\libs\` |

Override with `CAPY_LIBS`:

```sh
export CAPY_LIBS="$HOME/.capy/libs:/usr/local/share/capy/libs"
```

Resolution looks for:

1. `<dir>/<name>.capy` (bare file)
2. `<dir>/<name>/<name>.capy` (library directory, file matches name)
3. `<dir>/<name>/lib.capy` (library directory, generic name)

First match wins.

### `capy lib` subcommands

```sh
capy lib list                  # list every library on CAPY_LIBS
capy lib which recipe          # show the resolved path
capy lib new my-recipe         # scaffold a new library
capy lib path                  # print the search path entries
```

## Library commands

A library declares commands in its manifest (also a `.capy` file
— no separate config format). Example from the
`capy lib new` scaffold:

```
name        "recipe"
version     "0.1.0"
description "A recipe DSL."

extension   "txt"

function greet
    arg literal "greet"
    arg capture who string
    write `Hello from recipe, ${unquote who}!
`
end

command "run"
    description "Compile and print to stdout."
    let out = (compile context.arg0)
    print out
end

command "compile"
    description "Compile and write to a .txt file."
    let out    = (compile context.arg0)
    let target = "${context.arg0}.txt"
    write_file target out
    print "wrote ${target}"
end
```

### Command body — the shell-flavoured inner DSL

Inside a `command` body you can use everything the regular inner
DSL gives you (`set`, `append`, `if`, `for`, …) plus:

| Form | Effect |
|---|---|
| `let X = (EXPR)` | Bind a local. |
| `(compile script_path)` | Run the library on a script. Returns the output. |
| `print EXPR` | Print to stdout. |
| `write_file PATH CONTENTS` | Create or overwrite the file (parent dirs made). |
| `mkdir PATH` | Create a directory (with parents). |
| `(mktemp ".ext")` | Fresh temp file path. |
| `(mktemp_dir)` | Fresh temp directory path. |
| `exec CMD ARGS...` | Run a subprocess; stream stdout/stderr to the user. |
| `(exec_capture CMD ARGS...)` | Run a subprocess; capture combined output. |
| `cd PATH` | Change working directory. |

### Positional args + context

When you run `capy recipe build my-script.recipe extra`:

- `context.arg0` = `"my-script.recipe"`
- `context.arg1` = `"extra"`
- `context.args` = `["my-script.recipe", "extra"]`
- `context.lib_path` = path to the resolved library
- `context.lib_dir` = directory containing the library
- `context.lib_name` = name declared in the manifest

`${context.arg0}` interpolation works inside backtick literals
exactly like everywhere else in Capy.

### Built-in `run`

If your library doesn't declare a `command "run"`, calling
`capy <lib> run <script>` falls back to the legacy "render to
stdout" behaviour — no commands declared = no behaviour change.

## `capy new`

Scaffold a project that USES a library:

```sh
capy new my-app --using recipe
```

If the library declares a `command "new"`, it's called with the
project directory as `context.arg0`. That lets a library author
ship rich scaffolding (a whole Vite app, an Android Studio
project, etc.).

Otherwise, the CLI drops a minimal `hello.<lib>` script + README
into the new directory.

## File-extension convention

A script named `X.<libname>` is auto-resolved when you pass it
alone:

```sh
echo 'greet "world"' > cake.recipe
capy cake.recipe         # resolves library `recipe`, runs `run`
```

The library must be on `CAPY_LIBS` for this to work.

## Walkthrough: a Python library with multi-step build

```
# ~/.capy/libs/python/python.capy
name "python"
version "0.18.0"
extension "py"

function say
    arg literal "say"
    arg capture msg any
    write `print(${msg})
`
end

command "run"
    description "Generate the Python and run python3 on it."
    let out = (compile context.arg0)
    let tmp = (mktemp ".py")
    write_file tmp out
    exec "python3" tmp
end

command "build"
    description "Generate the Python file next to the script."
    let out    = (compile context.arg0)
    let target = "${context.arg0}.py"
    write_file target out
    print "wrote ${target}"
end
```

Use:

```sh
$ echo 'say "hi"' > hello.py.capy
$ capy python run hello.py.capy
hi
$ capy python build hello.py.capy
wrote hello.py.capy.py
```

## Security note

Library commands can do arbitrary work — `exec`, write files,
read environment. Today `capy` trusts every library on
`CAPY_LIBS` by default; a richer trust model
(`~/.capy/trusted.capy`, `--trust` flag, per-library SHA pins)
is on the roadmap (see
[future-features § 6.5](https://github.com/luowensheng/capy/blob/main/docs/design/future-features.md)).

For now: only put libraries on `CAPY_LIBS` that you'd trust to
run as yourself. Library files cloned from random URLs should
land elsewhere first; review them, then move into the search
path.

## What's still on the roadmap

Per the [comprehensive design](https://github.com/luowensheng/capy/blob/main/docs/design/future-features.md):

- Argument / flag parsing with auto-generated `--help` per command.
- Trust model: `~/.capy/trusted.capy`, `--trust` / `--dry-run`.
- Cross-command composition (`call <cmd>`, `last_command.X`).
- Multiple implementations of one library.
- Library version + lockfile.
- Git-based `capy lib add github.com/...`.
- WASM packaging + single-binary compiler.

What ships in v0.19 is the substrate: search path + manifest +
commands + the project-scaffolding workflow. Everything else
builds on top.
