# transpile-postgres-schema

Schema DSL → PostgreSQL DDL with tables, columns, indexes, and FKs.

```sh
../../capy run lib.yaml script.capy > schema.sql
psql < schema.sql
```
