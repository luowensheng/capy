# Feature reference

A flat list of everything Capy ships with. For deep dives follow the
links; this page is a quick reference for "is X supported?".

---

## Engine model

| Feature | Status | Where |
|---------|--------|-------|
| Zero default grammar — every shape is library-defined | ✅ | [language-reference](language-reference.md) |
| Single-binary Go engine, no runtime deps | ✅ | `go install github.com/luowensheng/capy/cmd/capy@latest` |
| Programmatic Go API | ✅ | `orchestrator.Run(libPath, scriptPath)` and `orchestrator.RunStrings(libYAML, libPath, scriptSrc)` |
| Caret-pointed error messages with line:col | ✅ | [`domain.CapyError`](https://github.com/luowensheng/capy/blob/main/domain/errors.go), `FormatWithSource` |

## Library schema

| Section | Required? | Purpose |
|---------|-----------|---------|
| `extension` | optional | Suggested output file extension (informational). |
| `output_file` | optional | If set, capy writes here instead of stdout. |
| `types` | optional | Library-defined argument types with `base`, `pattern`, `options`. |
| `context` | optional | Initial schema for the accumulated context (lists/maps/scalars). |
| `functions` | required | Every recognised source-language construct. |
| `file_template` | optional | Top-level assembler block. Defaults to emitting `body` verbatim. |

## Function definitions

| Field | Purpose |
|-------|---------|
| `args` | Ordered list of `{kind: literal, value}` and `{kind: capture, name, type}` entries. |
| `template` | Go `text/template` rendered into the body per match. |
| `run` | Inner-DSL snippet that mutates `context`. Does NOT execute user code. |
| `block.closer` | Mode-A block opener (INDENT/DEDENT body + named closer function). |
| `block.open` + `block.close` | Mode-B block opener (explicit delimiter pair). |
| `priority` | Higher wins on ambiguous matches; default 0. |

### Auto-name-prepend rule

If `args:` contains zero `kind: literal` entries, the engine prepends a
literal of the function's key. Add any literal and the function name
disappears from the surface — that's how operator-style patterns
(`<var> = <value>`, `<a> -> <b>`) work.

## Built-in capture types

| Type | What it captures |
|------|------------------|
| `any` | Any single value expression (numbers, strings, lists, objects, dotted paths, paren sub-calls, comparisons). |
| `ident` | A single identifier token; bound as a string. |
| `raw` | One identifier OR string token. |
| `string` | A quoted string literal — OR a bare identifier. |
| `int` | An integer literal — OR a bare identifier. |
| `float` | A float literal — OR a bare identifier. |
| `bool` | `true`/`false` — OR a bare identifier. |
| `<library-type>` | Any name defined under `types:`. |

Bare identifiers always pass primitive type checks because the target
language could resolve them as variables of any type at its own
runtime.

## Library-defined types

Declared under `types:` with three optional fields **applied in order**:

| Field | What it does |
|-------|--------------|
| `base` | Built-in kind check first (`string`/`int`/`float`/`bool`/`any`). |
| `pattern` | Regex applied to the value's string form. |
| `options` | Enum membership. |

Common patterns shipped with samples: `Email`, `Slug`, `EnvName`,
`SemVer`, `Status`, `Identifier`.

## Lexical features

| Feature | Notes |
|---------|-------|
| Identifiers | `[A-Za-z_][A-Za-z0-9_]*` |
| Numbers | Integers and floats, signed. |
| Strings | `"..."`, `'...'`, and `` `...` `` (template), all with `${expr}` interpolation |
| Multi-character punctuation | Greedy run: `==`, `!=`, `<=`, `>=`, `:=`, `->`, `=>`, `|>`, etc. each lex as one token. |
| Comments | `#` to end of line. |
| Indentation | 4 spaces or 1 tab per level. Mixed/odd indent is a parse error. |
| Multi-line literals | `{...}`, `[...]`, `(...)` allow newlines inside; value parsers skip them. |
| Object literal keys | Quoted strings OR bare identifiers (`{name: "x", "id": 1}`). |

## Inner DSL (`run:` field)

All operations available inside a `run:` snippet:

| Statement | Purpose |
|-----------|---------|
| `set <path> <value>` | Bind a field at a dotted/indexed path on `context`. |
| `append <path> <value>` | Push to a list. Creates the list if it doesn't exist. |
| `prepend <path> <value>` | Prepend to a list. |
| `merge <path> <map>` | Shallow-merge into a map. |
| `delete <path>` | Remove a field/key. |
| `if <expr>` ... `end` | Library-side conditional update. |
| `loop <var> in <expr>` ... `end` | Library-side iteration (NOT user-script iteration). |
| `regex_match value pattern` | Boolean expression value. |
| `error <message>` | Abort transpilation with a clear message. |

Paths root at `context` (or at a `loop` local). Bracket indexing
supported: `context.scripts[name]` evaluates `name` at runtime.

Expressions inside the function body support comparisons (`==`, `!=`,
`<`, `<=`, `>`, `>=`), unary `not`, lists `[...]`, objects `{...}`,
dotted paths.

## Interpolation helpers

Available inside any `write \`...\`` literal via `${expr | helper}`
or `${helper arg expr}`, in both per-function bodies and the top-level
`file_template`.

| Helper | Effect |
|--------|--------|
| `indent N` | Indent every line of a string by N spaces. |
| `lower` / `upper` | Case conversion. |
| `join SEP <list>` | Join a list of strings. |
| `unquote` | Strip one layer of surrounding `"..."`, `'...'`, or `` `...` ``. |
| `dasherize` | Replace underscores with hyphens (for CSS, etc.). |
| `toQuoted` | Wrap a string in JSON-style double quotes. |
| `toPyLit` | Format a Go value as a Python literal (`True`/`False`/`None`/lists/dicts). |
| `toJSON` | Marshal any value to compact JSON. |
| `toJSONIndent` | Marshal any value to pretty JSON. |

Control flow (`if`, `for`) lives in the function body itself —
not inside `${...}` interpolations.

## Block functions

Two modes; declare exactly one per opener:

### Mode A — indent + named closer

```
function if
    arg literal "if"
    arg capture cond any
    block_closer end
    write `if ${cond}:
${indent 4 body}
`
end
```

Body delimited by INDENT/DEDENT. Closer is itself a library function
(may have a template, or be silent).

### Mode B — explicit delimiters

```
block_open "{"
block_close "}"
```

Body delimited by the named tokens. No closer function needed.

Nesting works freely in both modes, including mixed (Mode-A inside
Mode-B and vice versa).

## CLI

| Subcommand | Effect |
|------------|--------|
| `capy run <lib.capy> <script.capy>` | Transpile a script. Output to stdout unless `--out` or library `output_file`. |
| `capy check <lib.capy>` | Validate a library; report functions and types. Exit 0 if valid. |
| `capy docs <lib.capy>` | Render a Markdown reference doc from the library's `description` annotations. |
| `capy init [<dir>]` | Scaffold a new library project. |
| `capy version` | Print build version. |
| `capy help [<command>]` | Inline help. |

Flags for `run`:

| Flag | Effect |
|------|--------|
| `--out <path>` | Override `output_file`. |
| `--no-color` | Disable ANSI escape codes (reserved). |
| `--debug` | Verbose engine tracing (reserved). |
| `-lib <path>` | Legacy library path (use the positional form instead). |

## Output assembly

Each library function's body `write`s text into a per-block **body**
string. Block functions reference `${body}` (or `${indent N body}`)
to get the rendered output of their children. The top-level
program's body, plus the accumulated `context`, are passed to
`file_template` for the final output.

Inside any function body / `write` literal:

| Reference | What it is |
|-----------|------------|
| `${<capture>}` | Source-text form of a capture (with quotes for string literals). |
| `${body}` | Inner block's rendered output (only inside block-opener functions). |
| `${context.X}` | Read-only snapshot of the accumulated context at this point. |
| `${func arg arg}` | Call a helper inline (`indent`, `pascalCase`, `toQuoted`, …). |

Inside `file_template`:

| Reference | What it is |
|-----------|------------|
| `${body}` / `write body` | Concatenation of all top-level statements' rendered output. |
| `${context.X}` | Final accumulated context. |

## Source vs evaluated captures

A capture has two faces:

- **Inside `write \`...\`` interpolation** — captures resolve to
  **source text** (with quotes for string literals). So `if x > 0`
  exposes `cond` as the literal text `x > 0` and a Python emitter
  can write `if ${cond}:` unchanged.
- **In `set` / `append` / `if` expressions** — captures resolve to
  **evaluated Go values**. Strings become Go strings without quotes,
  numbers become int64/float64, lists become `[]any`, objects become
  `map[string]any`. Unresolved bare identifiers fall back to their
  literal name.

This dual model means one capture serves both render-by-text (templates)
and structured accumulation (context) without any explicit conversion.

## AI / agent ecosystem

| Asset | Purpose |
|-------|---------|
| [Claude Code skill](https://github.com/luowensheng/capy/tree/main/skills/capy-author) | Full skill with `SKILL.md` + instructions + 5 reference docs |
| [Slash commands](https://github.com/luowensheng/capy/tree/main/commands/capy) | `/capy-new`, `/capy-add-function`, `/capy-add-type`, `/capy-explain`, `/capy-debug` |
| [`CAPY_FOR_LLMS.md`](CAPY_FOR_LLMS.md) | Single-page brief paste-able into any model |
| [Cursor rule](https://github.com/luowensheng/capy/tree/main/editors/cursor) | Drop-in `.cursor/rules/capy.md` |
| [Continue config](https://github.com/luowensheng/capy/tree/main/editors/continue) | Adds LLM brief to context |
| [Aider read](https://github.com/luowensheng/capy/tree/main/editors/aider) | `aider --read docs/CAPY_FOR_LLMS.md` |
| [Agent system prompt](https://github.com/luowensheng/capy/blob/main/agents/capy-system-prompt.md) | Drop-in for any tool |

See [Capy for AI agents](ai-agents.md) for token-savings math and
sandboxing patterns.

## Editor support

| Editor | Where |
|--------|-------|
| VS Code | [`editors/vscode/capy/`](https://github.com/luowensheng/capy/tree/main/editors/vscode/capy) — syntax highlighting for `.capy` source and libraries |

## Distribution

| Method | Command |
|--------|---------|
| Go | `go install github.com/luowensheng/capy/cmd/capy@latest` |
| Binary release | [Releases page](https://github.com/luowensheng/capy/releases) |
| Install script | `curl -fsSL https://raw.githubusercontent.com/luowensheng/capy/main/scripts/install.sh \| sh` |
| Docker | `docker build -t capy .` (Dockerfile in repo root) |
| Homebrew | Coming when the `luowensheng/homebrew-tap` repo exists |

## Pre-1.0 status

The library schema may evolve between minor versions before 1.0.
Each breaking change appears in [CHANGELOG](https://github.com/luowensheng/capy/blob/main/CHANGELOG.md)
with a migration note. The engine itself is stable enough to use in
production for code generation; just pin a specific version.

Planned features: see [roadmap](roadmap.md).
