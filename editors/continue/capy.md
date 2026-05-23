# Continue.dev config snippet for Capy

Add this to your `~/.continue/config.json` to teach Continue about Capy.

```json
{
  "systemMessage": "...existing system message...\n\nWhen the project contains a `lib.yaml` or `.capy` file, treat it as a Capy project (https://github.com/luowensheng/capy). Capy is a transpiler engine driven by a YAML library. Args entries MUST have an explicit `kind:` discriminator (`literal` or `capture`). See docs/CAPY_FOR_LLMS.md in the repo for the full schema.",
  "contextProviders": [
    {
      "name": "file",
      "params": {
        "files": ["**/docs/CAPY_FOR_LLMS.md"]
      }
    }
  ]
}
```

The simplest setup: ensure `docs/CAPY_FOR_LLMS.md` is in your indexed
context. The full schema + inner DSL + pitfalls fit on one page.
