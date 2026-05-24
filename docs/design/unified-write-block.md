# Design — unified `write:` block

> Status: **proposed, not implemented**. Captured here so the
> rationale, edge cases, and migration story aren't lost.
> Implementation is a multi-week swing; merging this doc commits us
> only to the shape, not the schedule.

## Motivation

Today a library author writes **two** blocks per function:

```
function import
    arg literal "import"
    arg capture name ident
    template_str ""               # ← Go text/template (output)
    run:                          # ← Capy inner DSL (state)
        append context.imports name
end
```

Two surface languages for one conceptual rule:

| Block       | What it expresses                              | Syntax  |
|-------------|------------------------------------------------|---------|
| `template:` | Body text contribution                         | Go      |
| `run:`      | Context mutation                               | Capy    |

This works, but newcomers consistently hit the same friction:

1. "Which block does what?" — the names don't both name what they do
   ("run what?").
2. "Why are the syntaxes different?" — `{{ .name | indent 4 }}` vs.
   `append context.imports name` for things that feel related.
3. Template errors surface as `template: capy:27: function "split"
   not defined` — pointing at the wrong file because Go's template
   engine is the source.

The deeper problem: the per-function rule "when source matches X,
contribute Y to the body AND maybe update state Z" is conceptually
one thing. Splitting it across two blocks is an implementation
artefact, not a design choice the author asked for.

## Proposal — one `write:` block

```
function import
    arg literal "import"
    arg capture name ident
    write `
        ${append context.imports name}
    `
end

function greet
    arg capture name any
    write `
        Hello, ${name}!
    `
end
```

Inside the backticks:

- **Literal text** is emitted verbatim into the body.
- **`${...}`** is the only escape. Two flavours, distinguished by the
  syntactic shape of what's inside:
  - **Value interpolation** — a path or pure expression
    (`${name}`, `${context.title}`, `${pascalCase name}`,
    `${indent 4 body}`). Result is spliced into the output.
  - **Statement interpolation** — starts with a reserved
    statement keyword (`set`, `append`, `prepend`, `merge`,
    `delete`, `if`, `for`, `loop`, `error`, `emit`). Runs as a
    side effect; produces no visible output.

Reserved keywords are the same finite set the current inner DSL
already uses; nothing new for an existing library author to learn.

### Why this works as ONE block

The objection to a unified block was: "how do you express 'do
something stateful, emit nothing' without ambiguity?" Answer:
wrap it in `${…}`. Side effects look like interpolations.

| Need                              | How                                                |
|-----------------------------------|----------------------------------------------------|
| Emit literal text                 | Type it inside the backticks                       |
| Interpolate a value               | `${name}` / `${context.title}` / `${pascalCase x}` |
| Update context                    | `${append context.imports name}` (no visible output) |
| Loop                              | `${for x in xs}` … `${end}`                        |
| Conditional                       | `${if cond}` … `${else}` … `${end}`                |
| Call a template helper            | `${pascalCase name}` / `${indent 4 body}`          |
| The inner block's rendered body   | `${body}`                                          |

No "which lines are statements, which are output?" rule. *Every*
non-literal is `${…}`. That's the whole clarity claim.

### Control flow

```
function loop
    arg literal "loop"
    arg capture v ident
    arg literal "in"
    arg capture i any
    block_closer end
    write `
        for ${v} in ${i}:
            ${body}
    `
end

file_template `
    ${for imp in context.imports}
    import ${imp}
    ${end}
    ${body}
`
```

`${for X in Y}` … `${end}` and `${if cond}` … `${else}` … `${end}`
are block-form statement interpolations. Same `${…}` sigil; nothing
else is special.

`${body}` is a reserved name meaning "the inner block's rendered
body" (in function `write:`) or "the concatenated top-level body"
(in `file_template`).

### Whitespace

`${…}` consumes nothing. `${-` strips trailing whitespace from the
preceding literal; `-}` strips leading whitespace from the
following literal. (Same idea as Go's `{{- -}}`, applied to the
new sigil.)

```
${- for imp in context.imports }
import ${imp}
${- end }
```

…produces no blank lines between iterations.

### Template helpers

Existing template helpers (`indent`, `pascalCase`, `camelCase`,
`snakeCase`, `dasherize`, `unquote`, `toQuoted`, `toPyLit`,
`toJSON`, `toJSONIndent`, `join`, `split`, `nonEmpty`, `lower`,
`upper`, `trimSuffix`, `trimPrefix`, `unescape`, `add`, `sub`,
`mul`, `percent`, `stars`) become callable from inside `${…}`
exactly like inner-DSL functions:

```
${indent 4 body}
${join ", " context.tags}
${pascalCase name}
```

One vocabulary across templates and state code.

### Errors

Every error points at a `.capy` line:col, not at a synthetic Go
template position. `${foo bar}` where `foo` isn't a helper or a
reserved keyword produces:

```
script.capy:14:8: unknown identifier "foo"
hint: did you mean "for"? (statement) or one of: pascalCase,
indent, join, … (helper)
```

## What this replaces

| Today                          | Tomorrow                                |
|--------------------------------|-----------------------------------------|
| `template:` (Go text/template) | `write:` (Capy template engine)         |
| `template_str "…"`             | `write \`…\`` (single-line form)        |
| `run:`                         | merged into `write:` via `${stmt}`      |
| `file_template:`               | `file_template \`…\``                   |
| `{{ .name }}`                  | `${name}`                               |
| `{{ .name \| pascalCase }}`    | `${pascalCase name}`                    |
| `{{- range .xs }}` … `{{ end }}` | `${- for x in xs}` … `${end}`         |
| `{{ if .cond }}` … `{{ end }}` | `${if cond}` … `${end}`                 |
| `{{ template "x" . }}`         | `${include "x"}` (sub-templates)        |
| `{{ define "x" }}`             | `template "x" \`…\`` block at top level  |

## Anti-goals (we are NOT doing these)

- **No new control structures.** Just `for` / `if` / `else`. No
  `while`, no `switch`, no `try`. Anything more is the host
  language's job, not the template's.
- **No expression DSL beyond what the inner DSL already has.**
  `${a + b}` does NOT become valid; arithmetic stays inside helper
  calls (`${add a b}`). Templates describe shape, not computation.
- **No partial-template inheritance / overrides.** `${include "x"}`
  is a function call by name; no `extends` / `block` machinery.
- **No "raw" Go template escape hatch.** If a library needs
  something the unified engine can't express, that's a bug in the
  engine, not a feature flag.

## Open questions

1. **Backtick string literals across `.capy`?** Today `.capy` uses
   double-quoted strings only. `write` blocks introduce a new
   string form. Decide: backticks only inside `write` / `template`
   / `file_template` headers, or globally? (Lean: locally — a
   library-author surface, not user-facing.)

2. **Indentation handling inside backticks.** The current `template:`
   block strips the deepest common leading indent so authors can
   write naturally-indented templates. Backticks should do the
   same. Spec it explicitly.

3. **Closing-backtick ambiguity.** If a template emits a literal
   backtick (a Markdown sample, a shell snippet, an asm directive),
   you need an escape. Options: `\\\`` inside the body, or use
   triple-backtick fencing for templates that need backticks.
   (Lean: triple-backtick variant.)

4. **`${include "x"}` semantics.** Compile-time inlining, or
   render-time call with the current `context` + `body` scope?
   Render-time is more flexible but slower; inlining matches what
   Go's `define` / `template` does. (Lean: render-time; perf hasn't
   been a bottleneck.)

5. **What happens to the inner DSL's `regex_match`?** Inside
   `${if regex_match name "^x"}` works fine as a statement
   interpolation; inside a value position `${regex_match name "^x"}`
   should evaluate as bool and splice "true"/"false". Specify this
   coercion explicitly.

6. **Locals.** Inner DSL has `loop x in y` exposing `x`; `${for x
   in y}` does the same. But what about user-introduced locals
   inside the template — `${let total add a b}`? Lean: don't add
   it. Helpers compose well enough; the temptation to compute in
   templates is what makes them ugly.

## Migration

Implementation phases:

1. **Build the new engine alongside the old.** A new `write:`
   block-shaped parser in `infra/`; a new template runtime in
   `orchestrator/features/`. Existing `template:` + `run:` keep
   working untouched. Engine selects per-function based on which
   block the library declared.

2. **Convert 3–5 reference samples.** `transpile-py`,
   `recipe-card`, `source-imports`, `host-capabilities`,
   `multi-target-ws-server`. Diff outputs; everything must be
   byte-identical to today.

3. **Convert remaining samples in waves.** ~60 libraries. Bulk
   conversion is mechanical for the simple ones; the gnarly
   `file_template:` blocks in
   `samples/cross-platform-installer/`, `samples/multi-language-demo/`,
   etc. need manual review.

4. **Deprecate `template:` + `run:`.** Two minor versions of
   warning; remove in v1.0.

5. **Update docs + showcase + playground bundles** at each
   conversion wave.

Estimated effort:
- Engine: 2–3 weeks
- Sample conversion: 1 week
- Docs + playground: 1 week
- Total: ~5 weeks at one-developer-full-time pace

## Decision check

Three things have to be true for this to be worth doing:

1. **The unified shape is genuinely clearer to newcomers.**
   Test: hand someone the `import` example in both forms; ask them
   to add a `from X import Y` variant. Time them.
2. **No regressions in expressive power.** Every existing library
   must port cleanly with byte-identical output.
3. **Errors get better, not just different.** New engine must
   surface `.capy` line:col on every failure path that currently
   says `template:`.

If any of these wobbles in prototyping, abandon.

## Not yet decided

- Should `write:` REPLACE `template:` (one name) or coexist
  briefly under both? Coexistence is gentler; one-name is cleaner.
  Default: coexist for one minor version, then `template:` becomes
  a deprecated alias.
- Should the inner-DSL's `set` keyword remain for context updates,
  or be replaced by `${context.x = value}` assignment syntax?
  Assignment-syntax is shorter; keyword-syntax is consistent with
  the existing inner DSL. Default: keep `set` for parity.
