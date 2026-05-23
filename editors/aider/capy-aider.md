# Aider integration for Capy

Add to your `.aider.conf.yml` in a Capy project:

```yaml
read:
  - docs/CAPY_FOR_LLMS.md
  - schemas/library.schema.json
```

Or invoke aider with the brief inline:

```sh
aider --read docs/CAPY_FOR_LLMS.md
```

This gives aider the full schema, inner-DSL reference, and pitfalls list
in its context. After that, prompts like "add a function that..." or
"this isn't matching, what's wrong" produce accurate edits.
