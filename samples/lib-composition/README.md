# lib-composition

**One main library imports two helper libraries.**

This sample shows how Capy libraries can be split across multiple
files and composed via `import` directives. The result: shared
types and shared syntax helpers live in `common/`, while
project-specific stuff stays in the main `lib.capy`.

## File layout

```
lib-composition/
├── lib.capy             ← main: imports + project-specific functions
├── common/
│   ├── types.capy       ← shared types (Email, URL, Semver, Slug)
│   └── syntax.capy      ← shared functions (meta, tag, note)
└── script.capy
```

## The main library

```
import "common/types.capy"
import "common/syntax.capy"

extension md

function post
    arg literal "post"
    arg capture title string
    block_closer end
    template:
        # {{ .title | unquote }}
        ...
end

function para
    ...
end
```

Two `import` lines pull in **all** the types and functions declared
in those files. The main library can then declare its own
project-specific functions (`post`, `para`) on top.

## Why split?

- **Reusable types.** `Email`, `URL`, `Semver` validation lives in
  `common/types.capy`. Any other library can import it instead of
  copy-pasting the regex.
- **Reusable syntax.** `tag`, `note`, `meta` are useful in blog
  posts, API docs, recipes, anything. One canonical definition.
- **Smaller main file.** Authors see only the project-specific
  bits when editing `lib.capy`.

## How conflicts resolve

If `common/syntax.capy` defines a `note` function AND the main
`lib.capy` also defines `note`, **the main library wins**. Imports
fill in defaults; explicit declarations override.

This means you can import a shared library and then specialize one
or two functions without forking the whole thing.

## Cycles are rejected

If `a.capy` imports `b.capy` and `b.capy` imports `a.capy`, the
loader stops with an error. The cycle detector tracks absolute
paths, so symlinks and `../` paths don't fool it.

## Run

```sh
../../capy check lib.capy             # confirm imports resolved
# → ok — 6 function(s), 4 type(s)
../../capy run lib.capy script.capy   # generate the post
```
