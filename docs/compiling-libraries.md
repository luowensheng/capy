---
title: Compiling a Capy library to a standalone binary
---

# Compiling a Capy library

Capy ships a `build` subcommand that turns a `.capy` library into a
**standalone executable**. The binary has the library source baked
in as a string constant and dispatches to its commands at runtime —
nobody needs `capy` installed to run the resulting tool.

The same machinery works as a cross-compiler: a single `capy build`
on macOS can produce binaries for Linux, Windows, ARM devices, and
WebAssembly. Under the hood it shells out to `go build`, so any Go
toolchain target is reachable.

This page is a walkthrough — author a tiny library, build it for the
host, then cross-compile it for four other targets, with concrete
size numbers and tips at each step.

---

## Prerequisites

- The `capy` CLI ([install](getting-started.md#1-install)).
- A Go toolchain (1.22+). `capy build` runs `go build` under the
  hood. **Only the developer needs Go — the output binaries don't
  require it.**

Check:

```sh
capy version
go version
```

---

## Step 1 — author a library

A library is just a `.capy` file. The minimal shape that
`capy build` accepts has a manifest (`name`, `version`,
`description`) plus at least one `command "..."` block — that's the
verb the resulting binary will dispatch to.

Save the following as `greet.capy`:

```
name        "greet"
version     "0.1.0"
description "A tiny greet DSL."
extension   "txt"

function greet
    arg literal "greet"
    arg capture who string
    write `Hello from greet, ${unquote who}!
`
end

command "run"
    description "Print the greeting."
    let out = (compile context.arg0)
    print out
end
```

…and a sample script `hello.greet`:

```
greet "world"
```

Sanity-check with the in-tree CLI before building:

```sh
capy greet run hello.greet
# Hello from greet, world!
```

---

## Step 2 — build for the host

```sh
capy build greet -o greet
```

Output:

```
building greet (this needs the Go toolchain)…
✓ wrote greet (5.4 MB)
  try:  greet --help
```

The resulting binary is **self-contained** — copy it anywhere on the
same OS / arch and it'll dispatch the library's commands:

```sh
./greet run hello.greet
# Hello from greet, world!
```

`greet --help` lists every declared command with its auto-generated
arg/flag help, exactly the way `capy <library> --help` does when
running from source.

---

## Step 3 — cross-compile

`capy build` honours Go's `GOOS` / `GOARCH` environment variables.
One developer machine produces binaries for every common deployment
target:

| Target | Command | Output size (greet example) |
|---|---|---|
| **macOS (host arm64)** | `capy build greet -o greet` | 5.4 MB |
| **Linux x86-64** | `GOOS=linux GOARCH=amd64 capy build greet -o greet-linux` | 5.7 MB |
| **Linux ARM64** (Raspberry Pi 4, AWS Graviton…) | `GOOS=linux GOARCH=arm64 capy build greet -o greet-arm` | 5.4 MB |
| **Windows x86-64** | `GOOS=windows GOARCH=amd64 capy build greet -o greet.exe` | 5.8 MB |
| **WebAssembly (browser)** | `GOOS=js GOARCH=wasm capy build greet -o greet.wasm` | 7.0 MB |

The output of every cross-compile is a true target-format binary:

```sh
$ file greet-linux
greet-linux: ELF 64-bit LSB executable, x86-64, statically linked

$ file greet.exe
greet.exe:   PE32+ executable (console) x86-64, for MS Windows
```

Statically linked = drop on any matching kernel and it just runs.
No glibc compatibility dance, no `LD_LIBRARY_PATH`, no DLLs to
collect.

For the **complete Go target matrix** see `go tool dist list` —
freebsd, openbsd, illumos, aix, dragonfly, plan9, every `arm`
revision, riscv64 — Capy inherits all of them because the build
step is plain `go build`.

---

## Step 4 — WebAssembly walkthrough

Wasm needs slightly more wiring because the binary doesn't run
standalone — it needs a JS host to feed it stdin and route stdout.
The pattern is the same one the [playground](playground.md) uses
for the engine itself.

```sh
GOOS=js GOARCH=wasm capy build greet -o greet.wasm
```

Copy Go's wasm shim (it's in your Go install — same one capy-wasm
uses):

```sh
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" .
# older Go versions:
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" .
```

Minimal `index.html` host:

```html
<!doctype html>
<html>
<body>
  <textarea id="src">greet "browser"</textarea>
  <pre id="out"></pre>
  <script src="wasm_exec.js"></script>
  <script>
    const go = new Go();
    go.argv = ["greet", "run", "/dev/stdin"];   // command + script
    WebAssembly.instantiateStreaming(fetch("greet.wasm"), go.importObject)
      .then(r => go.run(r.instance));
  </script>
</body>
</html>
```

The binary expects a real stdin/stdout — the simplest host wraps it
the same way `capy-wasm` does. See
[`cmd/capy-wasm/main.go`](https://github.com/luowensheng/capy/blob/main/cmd/capy-wasm/main.go)
in the repo for a complete browser-facing entry point that exposes
`capyRun(libSrc, scriptSrc)` as a JS function — easier to integrate
into a real page than wiring stdin by hand.

---

## Tips & tricks

### Shrink the binary — `-s -w` strips debug info

```sh
GOFLAGS='-ldflags=-s -w' capy build greet -o greet-min
```

`greet-min` ends up at **~3.7 MB instead of 5.4 MB** (roughly 30%
smaller). Strips DWARF debug info and the symbol table; harmless for
production but you lose pretty stack traces on a crash.

For another ~10% shrink, run `upx --best` on the output. UPX-packed
binaries start a touch slower but ship smaller.

### Reproducible builds — `-trimpath`

```sh
GOFLAGS='-trimpath -ldflags=-s -w' capy build greet -o greet
```

`-trimpath` removes absolute file paths from the binary so the same
input source always produces the same bytes regardless of which
machine compiled it. Useful for release artefacts that you publish a
checksum for.

### Pin a version into the binary

```sh
GOFLAGS='-ldflags=-X main.version=v1.4.2' capy build greet -o greet
```

The generated `main.go` declares `var version = "dev"`; the linker's
`-X` flag overrides it. Your library's `--version` will then print
`v1.4.2`.

### Bundle multiple targets in one tarball

A common release recipe — produce binaries for every supported
target plus checksums:

```sh
for t in \
  "linux amd64" "linux arm64" \
  "darwin amd64" "darwin arm64" \
  "windows amd64"
do
  set -- $t   # split into $1=os $2=arch
  out="greet-$1-$2"
  [ "$1" = "windows" ] && out="$out.exe"
  GOOS=$1 GOARCH=$2 GOFLAGS='-trimpath -ldflags=-s -w' \
    capy build greet -o "dist/$out"
done
(cd dist && shasum -a 256 greet-* > SHA256SUMS)
```

You now have a `dist/` folder with five binaries + a checksum file —
upload that to a GitHub release and your library is `curl`-installable
on any of them.

### `capy build` does NOT pull commands from disk at runtime

The library source is **embedded** at build time. Once the binary
exists, editing `greet.capy` won't change the binary's behaviour —
you have to rebuild. That's a feature: the binary is a snapshot, so
shipping `greet-v1.0` is a meaningful artefact you can pin and
reproduce.

### Build directly from a local `.capy` path

`capy build` accepts a path, not just a library name:

```sh
capy build ./libs/greet.capy -o greet
capy build /tmp/scratch/draft.capy -o draft
```

Useful for project-local libraries you haven't installed on
`CAPY_LIBS`.

### Keep the temp build directory for inspection

```sh
capy build greet --keep-temp
```

Prints the path of the temp dir holding the generated `main.go`,
`go.mod`, `go.sum`. Helpful when a `go build` failure is mysterious —
go look at what was generated and run `go build` on it directly to
get the full Go compiler diagnostics.

### Build cache makes repeated builds fast

The first `capy build greet` does a full `go mod tidy` and pulls
dependencies. Subsequent rebuilds (even after editing `greet.capy`)
reuse Go's build cache and finish in 1–2 seconds. CI matrices that
build every target in parallel share the cache too.

### Building inside the Capy source tree vs. from a release

If you're working inside a clone of `github.com/luowensheng/capy`,
`capy build` automatically detects the local module and uses a
`replace` directive — your changes to the engine flow into the
output binary. If you installed `capy` via `go install` or downloaded
a release, the build pulls the published module version from the
proxy instead.

### What if the agent / user wants the binary INSIDE the browser?

For "I want a Capy library that runs in a browser tab" the cleanest
path is to compile the **engine** ([`cmd/capy-wasm`](https://github.com/luowensheng/capy/tree/main/cmd/capy-wasm))
to WASM and load your library source dynamically. That's exactly the
playground's setup. `capy build greet -o greet.wasm` (i.e. embedding
the library) also works but the entry point currently expects an
`os.Args`-style invocation, so a JS host is required to feed it.

---

## Caveats

| Caveat | Mitigation |
|---|---|
| **Go toolchain required to build** (not to run) | One-time install. `go install golang.org/dl/go1.22@latest`. |
| **Cross-compiling to `js/wasm` produces a binary, not a webpage** | Pair with `wasm_exec.js` + a small HTML host, OR use the playground-style entry in `cmd/capy-wasm` instead. |
| **No `--target` flag yet** | Use `GOOS` / `GOARCH` env vars (table above). Vote with an issue if you want a flag. |
| **Library `command` bodies that `exec` external tools** | Those tools must exist on the *target* machine, not the build machine. `exec "pandoc" …` in a library command will fail on a host that doesn't have pandoc installed. |
| **Library `read_file` / `write_file` paths** | Run with the right working directory or pass absolute paths. The binary uses the host filesystem like any other process. |
| **Binary size** | 5–6 MB is the Go runtime baseline. `-s -w` + UPX gets you to ~2 MB. The library source itself contributes a few KB at most. |

---

## Comparison: `capy build` vs. the alternatives

| Approach | Pros | Cons |
|---|---|---|
| **`capy build greet`** | Self-contained, statically linked, cross-compiles for free, version-pinnable, embeds the library | Needs Go to build; ~5 MB minimum binary size |
| **Ship `.capy` + require `capy install`** | Tiny artifact (a `.capy` file is a few KB) | Every user needs `capy` installed; library updates need redistribution |
| **Ship as a Go library** ([embedding](embedding.md)) | No CLI binary; integrate Capy into a larger Go program | Only useful when your distribution surface is already Go code |
| **Ship as `cmd/capy-wasm` + lib** | Runs in any browser, no install | Two files (wasm + HTML host); only browser context |

For most "I built a DSL, I want to give it to teammates" use cases,
`capy build` is the right answer — one command produces five binaries
your colleagues can `curl` and run.

---

## Next steps

- [Library commands + `CAPY_LIBS`](library-commands.md) — design the
  commands that go inside your library before you ship it.
- [Embedding](embedding.md) — alternative path: link Capy into your
  Go program instead of producing a CLI.
- [Auto-generated library docs](library-documentation.md) — produce
  a reference `README.md` from your library to bundle with the
  release.
