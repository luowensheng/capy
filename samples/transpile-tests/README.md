# transpile-tests

Test-case DSL → Go test functions. Each `test` is a block; assertions
inside emit standard `t.Errorf` checks.

```sh
../../capy run lib.yaml script.capy
```
