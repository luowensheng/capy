# backend-with-tests

**Generate backend handler code AND matching tests from one source.**

Every `handler X method M path P returns R` line in `script.capy`
produces both a Go handler stub AND a smoke test for it. The team's
directory layout and the "every handler must have a test" rule are
baked into the library.

## The source (4 lines)

```
handler ListUsers   method GET     path "/users"        returns "[]User"
handler GetUser     method GET     path "/users/{id}"   returns "User"
handler CreateUser  method POST    path "/users"        accepts "UserCreateRequest"  returns "User"
handler DeleteUser  method DELETE  path "/users/{id}"   returns "void"
```

## What you get

```
out/
├── README.md                              ← route catalog
└── internal/handlers/
    ├── handlers.go                        ← Mount() + 4 handler stubs
    └── handlers_test.go                   ← 4 Test_X_RouteRegistered functions
```

The `handlers.go` includes a `Mount(mux *http.ServeMux)` function
that registers every declared route. Each handler returns
`http.StatusNotImplemented` until you fill it in.

`handlers_test.go` has one test per handler. Each builds a fresh
mux, mounts the routes, hits the declared path with the declared
method, and asserts the 501 contract. As you implement each
handler, the test starts failing — at which point you replace it
with real assertions.

## Run + verify

```sh
capy run --out-dir out lib.capy script.capy
cd out
echo "module example/backend\ngo 1.22" > go.mod
go test ./...
# PASS: every route is registered; every stub returns 501 as expected.
```

The generated Go is real and compiles immediately.

## What the library enforces

Read [`lib.capy`](lib.capy) to see every team convention:

- **Directory layout** — `file "internal/handlers/handlers.go":`
  hard-codes the path. New handlers can't drift to the wrong place.
- **Test-per-handler** — the test file ALWAYS emits one test per
  handler. There's no way to declare a handler without also
  declaring its test.
- **Router placement** — same file as the handlers.
- **Stub return code** — 501 for un-implemented handlers; the test
  expects 501; the contract is mechanical.
- **A route catalog** — `README.md` is the third output, auto-
  updated whenever a handler is added.

A new contributor reading the library sees the team's conventions
in ~30 seconds. They literally cannot violate them — adding a
handler regenerates everything in the right place.

[Pattern docs → `docs/backend-codegen.md`](../../docs/backend-codegen.md)
