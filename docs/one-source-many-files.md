---
title: One source → many files
---

# One source → many files

A single `.capy` source file can generate **a complete project tree**.
Not just a single output file with a `file_template:`, but a directory
hierarchy with HTML + JS + CSS, or an Android scaffold of 7 files
across 4 directories, or an iOS SwiftUI project, or a CUDA-capable
libtorch C++ trainer.

This page is the consistent story behind every multi-file sample.

## The two ingredients

1. **`file "path/to/output.ext":` blocks** in the library. Multiple
   blocks may appear; each declares one output file. Paths may
   contain slashes for subdirectories.

2. **`capy run --out-dir DIR`** (or `--zip ARCHIVE.zip`). The engine
   renders every `file:` block against the same final context+body
   and writes them under DIR (mkdir-p as needed) or bundles them
   into a single zip.

```sh
capy run --out-dir generated  lib.capy script.capy
capy run --zip   project.zip   lib.capy script.capy
```

## The pattern in one mental model

```
script.capy                lib.capy
(intent)                    (scaffold)
       \                   /
        \                 /
         ──────► capy ◄──────
                   │
                   ▼
       ┌───────────────────────┐
       │  full project tree    │
       │  README, configs,     │
       │  source code, tests   │
       └───────────────────────┘
```

The **scaffold lives in the library** (your team's house style,
target framework, conventions). The **intent lives in the source**
(which screens, which routes, which model layers). Capy joins them.

## Five worked examples

Five samples in the repo demonstrate the range. They all share the
same shape: declare-the-intent → run-with-out-dir → get-a-project.

### 1. Web app — 3 files

[`samples/webapp-trio/`](https://github.com/luowensheng/capy/tree/main/samples/webapp-trio)

12 lines of DSL → `index.html` + `app.js` + `styles.css`, properly
cross-referenced. A complete browser-ready habit tracker.

### 2. Multi-file Python project — 6 files

[`samples/multi-file-project/`](https://github.com/luowensheng/capy/tree/main/samples/multi-file-project)

9-line route declaration → FastAPI app, handler stubs, smoke tests,
pyproject.toml, .gitignore, README.

### 3. Android app — 7 files

[`samples/android-app/`](https://github.com/luowensheng/capy/tree/main/samples/android-app)

15-line declaration → Kotlin source, layout XML, AndroidManifest,
gradle config, string resources, and README. Drop into Android Studio.

### 4. iOS app — 6 files

[`samples/ios-app/`](https://github.com/luowensheng/capy/tree/main/samples/ios-app)

Same source-shape as the Android sample → SwiftUI App, RootView,
per-screen Views, Info.plist, Package.swift. Open in Xcode or build
via SPM.

### 5. libtorch C++ ML trainer — 5 files

[`samples/libtorch-train/`](https://github.com/luowensheng/capy/tree/main/samples/libtorch-train)

17-line neural-network architecture → `model.h` (with register_module
plumbing) + `main.cpp` (training loop) + CMakeLists + run.sh. Compiles
to a native CUDA-enabled trainer.

## Sample shape, consistent across all five

Every sample follows this directory layout:

```
samples/<name>/
├── README.md             ← how to use it, what it generates
├── lib.capy              ← the library declaring `file "..."` blocks
├── script.capy           ← the intent declaration
└── expected/             ← committed golden project tree (CI-diffed)
    └── ...
```

The `expected/` tree is regenerated on every CI run and diffed —
any drift fails the build. This is what makes the contract trustworthy:
your project scaffold can't drift silently.

## Target a NEW thing in 30 minutes

The pattern scales to anything textual. Pick a target you regenerate
often (a CDK stack, a Terraform module, a Helm chart, a React
component package, a SwiftPM library), then:

1. Define the `script.capy` you'd want to write — usually 5-20 lines
   declaring the intent.
2. Write a `lib.capy` with one `file "..."` block per output file
   in the project.
3. Run `capy run --out-dir out lib.capy script.capy`, iterate on the
   templates until the output compiles/runs.
4. `cp -r out expected/` and you have a golden snapshot.
5. Commit. CI will diff future runs against `expected/` and reject
   drift.

## Useful template helpers for this pattern

| Helper                | Use for…                                               |
|-----------------------|--------------------------------------------------------|
| `pascalCase`          | `"Habit Tracker"` → `HabitTracker` (Swift/Kotlin types)|
| `camelCase`           | `"habit tracker"` → `habitTracker` (JS/Java fields)    |
| `snakeCase`           | `"Habit Tracker"` → `habit_tracker` (file names)       |
| `toQuoted`            | wrap a captured string for embedding in source code    |
| `unquote`             | strip the quotes from a `string`-typed capture         |
| `indent N`            | indent every line of a body block by N spaces          |
| `add` / `percent`     | running totals + progress bars (e.g. reading log demo) |
| `trimSuffix " ,\n"`   | drop trailing commas in generators                     |

Full reference: [Templates](templates.md).

## Zip bundling

When you want to deliver a generated project as a single artifact
(downloadable, sendable in chat, attached to an issue), use `--zip`
instead of `--out-dir`:

```sh
capy run --zip habit-tracker.zip samples/android-app/lib.capy samples/android-app/script.capy
```

The archive uses POSIX (`/`) paths internally — works on every
modern unzipper across operating systems.

## Composability

A library declaring `file:` blocks can also `import` other
libraries (see [`multi-file-and-imports.md`](multi-file-and-imports.md)).
Use that to share scaffolding patterns across projects:

```
# my-project/lib.capy
import "../base/project-scaffold.capy"

# Override one file from the base; everything else inherits.
file "README.md":
    # {{ .context.name | unquote }}
    ## Custom intro just for this project
    ...
```

A team can publish a "company starter" library that every new
project imports. Adding a new file to the starter propagates to
every consuming project at the next `capy run`.
