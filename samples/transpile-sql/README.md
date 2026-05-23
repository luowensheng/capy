# transpile-sql

Tiny query DSL → SQL.

## Run

```sh
../../capy run lib.yaml script.capy
```

## Expected output

```sql
-- table: users
SELECT id FROM users WHERE active;
SELECT name FROM users WHERE verified;
INSERT INTO users VALUES [1, "alice", "alice@example.com"];
```

## What this teaches

- Multi-literal patterns (`select ... from ... where ...`) — five tokens,
  three captures, three literals.
- A pure rendering library: no `context`, no `run:` — every function just
  has a `template`.
- The captured `cond` is source text — so `active` flows through as the
  identifier `active`, not as a string `"active"`.
