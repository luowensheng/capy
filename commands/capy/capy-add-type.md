---
name: capy-add-type
description: Add a library-defined type (pattern/options/base) to an existing Capy library.
---

# /capy-add-type

Use when the user wants to add validation for an arg type.

## Steps

1. Confirm the type's purpose: what values should it accept?
2. Choose the validators:
   - **`pattern:`** for regex-shaped strings (emails, identifiers, semver, …).
   - **`options:`** for enums.
   - **`base:`** to constrain a primitive kind first (often `string`).
3. Edit `lib.yaml` to add an entry under `types:`. Create the section if missing.
4. Update relevant function arg types from `any`/`string` to the new type.
5. Run `capy check lib.yaml` to confirm cross-refs resolve.
6. Run `capy run lib.yaml script.capy` and verify both valid AND invalid inputs.

## Common patterns

| Need | YAML |
|---|---|
| Email | `pattern: "^[^@]+@[^@]+\\.[^@]+$"` |
| Identifier | `pattern: "^[A-Za-z_][A-Za-z0-9_]*$"` |
| SCREAMING_SNAKE | `pattern: "^[A-Z][A-Z0-9_]*$"` |
| Slug | `pattern: "^[a-z][a-z0-9-]*$"` |
| Semver | `pattern: "^[0-9]+\\.[0-9]+\\.[0-9]+(-[A-Za-z0-9.-]+)?$"` |
| Status enum | `options: ["todo", "in-progress", "done"]` |
| Positive int | `base: int`, `pattern: "^[1-9][0-9]*$"` |

## Pitfalls

- Forgetting `^...$` anchors → too-loose matching.
- Forgetting to double-escape `\\` in YAML strings.
- Putting `options:` AND `pattern:` when only one is needed (they're AND-ed; that's usually fine but more constraining than intended).
