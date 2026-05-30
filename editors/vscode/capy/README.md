# Capy VS Code extension

Syntax highlighting for `.capy` files + JSON-schema validation for Capy
library YAML files.

## Features

- Syntax highlighting for `.capy` (numbers, strings, identifiers,
  comments, brackets, operators, `true`/`false`/`null`).
- Auto-closing pairs and bracket matching.
- Validation, autocomplete, and hover documentation for `lib.yaml`,
  `lib.yml`, `capy.yaml`, and `capy.yml` via the published JSON schema.

## Install (local, not yet published)

From this directory:

```sh
# package the extension
npx @vscode/vsce package
# install the resulting .vsix
code --install-extension capy-0.1.0.vsix
```

Or, while developing, copy this folder into `~/.vscode/extensions/olivierdevelops.capy-0.1.0/`.

## Marketplace

We'll publish once the schema URL is stable. Track
[issue #1](https://github.com/olivierdevelops/capy/issues) for status.
