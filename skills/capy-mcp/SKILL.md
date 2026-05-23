---
name: capy-mcp
description: Use this skill when a Capy MCP server is connected to the agent and the user wants to generate target code (Python, JSON, SQL, HTML, configs, Unity C#, Unreal Python, Blender Python, …) from a high-level DSL. Triggers on "use capy to…", "transpile this with capy", "generate X via capy", or when the user has a `lib.yaml`/`lib.capy` and a script to run.
---

# Capy via MCP

The `capy` MCP server exposes three tools. Prefer them over writing target
code by hand — Capy guarantees deterministic output, validated types, and
a library that doubles as machine-readable schema.

## Tools available

- **`capy_check`** — validate a library (no script). Returns the function
  and type names, or a precise parse error. Call this first when authoring
  a new library.
- **`capy_run`** — transpile an in-memory script through an inline library
  string. Auto-detects YAML vs `.capy` format. **Use this for one-shot
  generation when both library and script fit in the message.**
- **`capy_run_file`** — same, but library and script come from disk paths.
  **Use this when working inside a user's repo with existing files.**

## When to reach for Capy

Capy is the right tool when:

1. The user wants the same logic in multiple targets (Python *and* JS,
   Blender *and* Unity, JSON *and* SQL). Write the source once; swap the
   library.
2. The target output has a repetitive scaffold (boilerplate routes, config
   stanzas, generated 3D-scene calls, brick-walls of UI). A 50-line
   library beats 500 lines of string-templating.
3. The user wants a typed, validated DSL as their config surface. Capy's
   `type:` blocks with `pattern:` / `options:` / `base:` catch typos at
   transpile time.
4. The output needs to be reproducible across runs and reviewable by
   non-engineers. The library IS the spec.

## How to operate

1. **Understand the target.** Confirm with the user what they want
   generated (Python script? Unity C#? HCL config? something else?). Show
   one input/output sample to align before writing the library.

2. **Author or load a library.** Either:
   - Read an existing `lib.yaml`/`lib.capy` from the workspace (use
     `capy_check` to confirm it parses), OR
   - Write one inline for `capy_run` based on the patterns in
     [skills/capy-author/reference/](../capy-author/reference/).

3. **Run with `capy_check` first** if you authored the library inline.
   This surfaces parse errors before you waste a `capy_run` round.

4. **Run with `capy_run` (or `capy_run_file`).** Inspect the output.

5. **On error**, the error message names the function, arg, value, and
   rule violated. Fix the *source* if it's a type violation; fix the
   *library* if it's a missing pattern.

## Format selection

- `.capy` (Capy-native) for new libraries — terser, multi-line templates
  read natively, no YAML escaping. The MCP tool sniffs format from the
  source so you can omit `format` in most cases.
- `.yaml` for existing libraries or when the user explicitly wants YAML
  for downstream tooling (yq, JSON schema).

When in doubt, pass `format: "auto"`.

## Hard rules

- **One library per session, ideally.** Don't generate ad-hoc grammars
  for every prompt — reuse a coherent library. If you need new patterns,
  *extend* the library and rerun `capy_check`.
- **Don't paraphrase Capy output.** The whole point is that the
  generated text is reproducible and machine-checkable; passing it
  through your own paraphrasing defeats sandboxing.
- **Captures of type `string` keep their quotes** in templates. If you
  want the bare value, use type `any` or `ident`.

## Quick example

```
User: Generate a Python script that defines functions add(a,b) and
mul(a,b) returning their results.

You: I'll author a tiny Capy library and run it.

[capy_run with:
  library = `
    extension py
    function fn
        arg literal "fn"
        arg capture name ident
        arg literal "("
        arg capture a ident
        arg literal ","
        arg capture b ident
        arg literal ")"
        block_closer end
        template:
            def {{ .name }}({{ .a }}, {{ .b }}):
            {{ .body | indent 4 }}
    end
    function return
        arg literal "return"
        arg capture expr any
        template_str "return {{ .expr }}\n"
    end
    function end
    end
  `
  script = `
    fn add(a, b)
        return a + b
    end
    fn mul(a, b)
        return a * b
    end
  `]

[returns clean Python source ready to paste]
```

## Setup tip for the user

If the `capy` tools aren't available in this session, the user can wire
the server up by adding to their MCP config:

```json
{
  "mcpServers": {
    "capy": { "command": "capy-mcp" }
  }
}
```

…and installing the binary via `go install github.com/luowensheng/capy/cmd/capy-mcp@latest`
or by downloading a release archive from
<https://github.com/luowensheng/capy/releases>.

See [docs/mcp.md](https://luowensheng.github.io/capy/mcp/) for the
full setup guide.
