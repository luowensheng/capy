# transpile-prisma-schema

Schema DSL → Prisma `schema.prisma`. Run `prisma generate` against the output.

```sh
../../capy run lib.yaml script.capy > schema.prisma
npx prisma format
```
