# multi-target-demo

A single Capy source file (`script.capy`) compiled by three different
libraries into three completely different artifacts:

- `lib_sql.yaml`  → SQL `INSERT` statements (`sql.expected.txt`)
- `lib_json.yaml` → A JSON document (`json.expected.txt`)
- `lib_md.yaml`   → A Markdown table (`md.expected.txt`)

The point: **the library is the grammar.** Add a new target by writing
a new library; never touch the source.

```sh
../../capy run lib_sql.yaml  script.capy
../../capy run lib_json.yaml script.capy
../../capy run lib_md.yaml   script.capy
```
