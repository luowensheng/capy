# transpile-statemachine

State machine DSL → Mermaid state diagram. The transition function uses
the `<ident> -> <ident> on <event>` shape — five tokens, three captures,
two literals (`->` and `on`).

```sh
../../capy run lib.yaml script.capy
```
