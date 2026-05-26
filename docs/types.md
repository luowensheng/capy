# Types

Capy types validate captured argument values at evaluation time. The library
author declares types with top-level `type NAME ... end` blocks; functions
reference them by name in `arg capture NAME TYPE`.

## Built-in kinds

These are always available and don't need a `type` declaration:

| Kind     | Captures                                                                |
|----------|-------------------------------------------------------------------------|
| `any`    | Any single value expression. No validation.                             |
| `ident`  | A single identifier token; bound as a string.                           |
| `raw`    | An identifier OR a string token; bound as a string (no quotes).         |
| `string` | A quoted string literal — or a bare ident (transpile-mode permissive).  |
| `int`    | An integer literal — or a bare ident.                                   |
| `float`  | A float literal — or a bare ident.                                      |
| `bool`   | `true`/`false` — or a bare ident.                                       |

The "or a bare ident" rule exists because in transpile mode the captured
text flows into the target language; a bare ident might be a variable that
holds a value of the expected type at the target's runtime.

## Library-defined types

Declared with `type NAME ... end` blocks. Three optional fields are
applied in this order: `base`, `pattern`, `options`.

```
type Email
    base string                              # 1. type-kind check
    pattern "^[^@]+@[^@]+\\.[^@]+$"           # 2. regex on the value's string form
end

type Status
    options "todo" "in-progress" "done"      # 3. enum membership
end

type PositiveInt
    base int
    pattern "^[1-9][0-9]*$"
end
```

Each field is optional. The most common patterns:

- `pattern` only — for free-form strings that must match a regex.
- `options` only — for enums.
- `base` + `pattern` — to constrain a primitive kind further.

## How validation runs

For each captured argument with a declared type:

1. If the type is a built-in kind, the engine checks the captured **source
   text** (e.g. `int` requires the text to parse as an integer literal).
   Bare identifiers always pass.
2. If the type is library-defined:
   - If `base` is set, run the built-in check for that base kind first.
   - If `pattern` is set, compile + match against the value's string form
     (with surrounding quotes stripped from string literals).
   - If `options` is set, check membership against the value's string form.

If any check fails, the engine returns a structured error like:

```
function "set_email" arg "e": value "not-an-email" does not match pattern for type "Email"
```

## Using types in arguments

```
function set_email
    arg capture e Email
    write `email = ${e}
`
end
```

The capture's type may be either a built-in kind or any name declared
with a top-level `type` block. The loader validates that every referenced
type resolves — typos are caught at `capy check` time.

## Patterns: practical tips

- Use `^...$` anchors. Without anchors, a regex like `[a-z]+` accepts
  anything that contains lowercase letters.
- Use `\\` to escape backslashes inside double-quoted strings (`"\\d"` is the regex `\d`).
- Test patterns with a quick `capy run` against a sample script before
  shipping them.

## Options: practical tips

- Options match the captured string text after quote-stripping. So:
  - `set_status "todo"` → option string `"todo"` ✓
  - `set_status todo` (bare ident) → also `"todo"` ✓
- Put the most common options first; the engine scans linearly (it's a tiny
  list).

## Group types

A type can also describe a **delimited inline capture** — the
parser walks tokens between an opening delimiter and a matching
closing delimiter and hands you the joined source text. This is
how Capy expresses Markdown / LaTeX-style inline syntax without a
separate scanning pass.

```
type Bracketed
    group_open  "["
    group_close "]"
end

type Parens
    group_open  "("
    group_close ")"
end

function link
    arg literal "link"
    arg capture text Bracketed
    arg capture url  Parens
    write `<a href="${escapeHtml url}">${escapeHtml text}</a>
`
end
```

Source `link [Al the Alien](https://alien.com/1)` produces
`<a href="https://alien.com/1">Al the Alien</a>` — `text` and `url`
each capture their delimited run as plain text.

### Properties

- **Balanced nesting** — `link [nested [inner] brackets](url)`
  captures `nested [inner] brackets` correctly because the parser
  tracks open/close depth.
- **Multi-char delimiters work** — `type Bold { group_open "**"
  group_close "**" }` matches `**bold**`. The lexer's punct-greedy
  rule already collapses `**` into a single token.
- **Multi-line captures** — a group can span newlines; the
  captured text contains the real newlines in between.
- **Mutually exclusive with constraint fields** — a single type
  declares EITHER `group_open`/`group_close` OR
  `base`/`pattern`/`options`, never both. The loader rejects the
  mixed form with a clear error.

### Limitations

- The **backtick** (`` ` ``) collides with Capy's string-literal
  lexing — it can't currently be used as a group delimiter. Pick a
  different character (e.g. `~` for `~~code~~`).
- A **prose run that embeds inline calls** (`This is **important**
  text.`) needs a separate prose-scanner that's not part of the
  group-types primitive. For now, write the inline calls on their
  own lines.

## What's NOT here yet

- **`validate:` snippets** written in inner Capy. The README originally
  envisioned this. We've kept `pattern:` + `options:` + `base:` as the v0.1
  surface; full snippets land in a future version.
- **Type composition / inheritance**. Each type stands alone.
- **Custom built-in kinds**. Only the six above; ask via an issue if you
  need more.
