# transpile-gh-actions

Workflow DSL → GitHub Actions YAML. Jobs accumulate into a list with
their body strings; the file template renders them with their steps.

```sh
../../capy run lib.yaml script.capy
```
