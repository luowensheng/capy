# supercharge-sql

**Take SQL DDL and add high-level macros — output is plain Postgres
DDL that any database accepts.**

Same pattern as `supercharge-markdown`: Capy sits ON TOP of an
existing format, giving authors a richer surface while preserving
the host as the deployment target.

## The DSL

```
table users
    pk id
    col email "varchar(255) UNIQUE NOT NULL"
    col name  "varchar(100)"
    timestamps
end

table posts
    pk id
    fk author_id -> users
    col title    "varchar(255) NOT NULL"
    col body     "text"
    timestamps
    soft_delete
end

index posts author_id
```

The Capy library defines seven macros (`table`, `pk`, `fk`, `col`,
`timestamps`, `soft_delete`, `index`). Each expands to standard
Postgres syntax:

| DSL              | Expands to                                                    |
|------------------|---------------------------------------------------------------|
| `pk id`          | `id bigserial PRIMARY KEY`                                    |
| `fk x -> users`  | `x bigint NOT NULL REFERENCES users(id)`                      |
| `timestamps`     | `created_at` + `updated_at` columns with `now()` defaults     |
| `soft_delete`    | `deleted_at timestamptz` (NULL = active)                      |
| `index t c`      | `CREATE INDEX ix_t_c ON t(c);`                                |

## What you write vs. what gets emitted

32 lines of DSL → 32 lines of polished SQL (including comments and
formatting). The interesting compression is in *cognitive load*:

```
timestamps      # 1 word of DSL
```

vs.

```sql
created_at timestamptz NOT NULL DEFAULT now(),
updated_at timestamptz NOT NULL DEFAULT now()
```

Multiply across every table in a schema with 20+ tables and you've
removed every audit-column typo from the codebase.

## Run

```sh
../../capy run lib.capy script.capy > schema.sql
psql -f schema.sql
```

The output is plain Postgres DDL. No runtime dependency on Capy —
`schema.sql` is committed to the repo, applied via your normal
migration tooling.

## Why "supercharge" not "replace"

The conventional alternatives — Prisma, sqlc, custom Go/Python
generators — replace SQL with a different surface and pay the cost
of maintaining a translator forever. Capy's approach:

- **The output is the SQL** you'd have written by hand. No magic
  ORM layer.
- **Library is yours**, in this repo. Doesn't bind you to a vendor.
- **Targets are swappable.** Write `lib_mysql.capy` for MySQL,
  `lib_sqlite.capy` for SQLite. Same `script.capy`.

## Add a new macro

Want `audit_columns` (created_by, updated_by, version)? Add 5 lines
to `lib.capy`:

```
function audit_columns
    arg literal "audit_columns"
    template_str "  created_by bigint REFERENCES users(id),\n  updated_by bigint REFERENCES users(id),\n  version integer NOT NULL DEFAULT 1,\n"
end
```

Now `audit_columns` works in every table. The library is the
extension mechanism.
