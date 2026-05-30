---
title: Backend code-gen with auto-wired tests
---

# Backend code with auto-wired tests

Most backend teams have unspoken rules that nobody enforces:

- "Handlers go in `internal/handlers/`."
- "Every handler has a corresponding `_test.go` test."
- "The router lives next to the handlers."
- "Stub handlers return 501."

New contributors break these rules; reviewers point them out;
existing handlers slowly drift from the conventions. Linters help
but only catch what you've encoded. Capy goes further: **the
library is the conventions**.

## The pattern

```
script.capy                lib.capy
  4 handler                  • Encodes directory layout
  declarations               • Encodes "every handler has a test"
       │                     • Encodes "router lives here"
       │                     • Encodes "stubs return 501"
       │                     • Encodes 501 → test expects 501
       │
       ▼
  capy run --out-dir .
       │
       ▼
  internal/handlers/handlers.go        ← stubs + router
  internal/handlers/handlers_test.go   ← matching tests
  README.md                            ← route catalog
```

One source declaration. Three files. The handler stub returns 501
until the developer fills it in; the auto-generated test asserts
the 501 contract. Once the developer implements the handler (and
the response becomes 200), the test starts failing — at which
point they replace it with real assertions.

This is a *contract* between code-gen and the developer: "I gave
you a stub + a smoke test; replace both as you implement."

## The worked sample

[`samples/backend-with-tests/`](https://github.com/olivierdevelops/capy/tree/main/samples/backend-with-tests)

Source (4 lines, one per handler):

```
handler ListUsers   method GET     path "/users"        returns "[]User"
handler GetUser     method GET     path "/users/{id}"   returns "User"
handler CreateUser  method POST    path "/users"        accepts "UserCreateRequest"  returns "User"
handler DeleteUser  method DELETE  path "/users/{id}"   returns "void"
```

Generated `internal/handlers/handlers.go`:

```go
package handlers

import "net/http"

func Mount(mux *http.ServeMux) {
    mux.HandleFunc("GET /users", ListUsers)
    mux.HandleFunc("GET /users/{id}", GetUser)
    mux.HandleFunc("POST /users", CreateUser)
    mux.HandleFunc("DELETE /users/{id}", DeleteUser)
}

// GET /users — returns []User
func ListUsers(w http.ResponseWriter, r *http.Request) {
    http.Error(w, "ListUsers not implemented", http.StatusNotImplemented)
}
// ... three more ...
```

Generated `internal/handlers/handlers_test.go`:

```go
package handlers

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func Test_ListUsers_RouteRegistered(t *testing.T) {
    mux := http.NewServeMux()
    Mount(mux)
    req := httptest.NewRequest("GET", "/users", nil)
    rr := httptest.NewRecorder()
    mux.ServeHTTP(rr, req)
    if rr.Code != http.StatusNotImplemented {
        t.Errorf("expected 501, got %d", rr.Code)
    }
}
// ... three more ...
```

Plus a `README.md` that lists every declared route — a catalog
that's never out of date because it's regenerated from the same
declaration.

**`go test` against this output passes.** The generated code is
real, compileable, runnable Go.

## What the library is enforcing

Read [`samples/backend-with-tests/lib.capy`](https://github.com/olivierdevelops/capy/tree/main/samples/backend-with-tests/lib.capy)
and you can see every convention:

- **Directory layout** — `file "internal/handlers/handlers.go":`
  hard-codes the path. New handlers always land in the right place.
- **Test-per-handler rule** — `file
  "internal/handlers/handlers_test.go":` ALWAYS emits one test per
  handler. There's no way to declare a handler without also
  declaring its test.
- **Router placement** — same file as the handlers. The team's
  preference.
- **Status code contract** — stub returns 501; test expects 501.
  The contract is mechanical.
- **Documentation** — `README.md` is a third output, listing every
  route. Auto-updated whenever a handler is added.

A reviewer who reads the library knows the conventions in 30
seconds. A new contributor can't deviate from them.

## Adapt to your team's conventions

The shape of the library is the customization surface. Some
variations:

- **gRPC, not REST**: replace the `method GET path "..."` directive
  with `service users rpc CreateUser ...` and emit `.proto` files
  alongside Go stubs. The Capy DSL adapts.
- **Different framework**: emit FastAPI handlers + pytest tests,
  Express + Mocha tests, Actix + integration tests. One library per
  stack; same script.capy.
- **Different layout**: maybe `cmd/server/handlers/...` is your
  team's convention. Change the `file:` block path; everything
  rewires.

## How it composes with the rest of Capy

- **`@import` for shared declarations** — extract common handler
  groups into separate `.capy` files and import them.
- **`type` validation** — `arg capture method Method` plus
  `type Method { options "GET" "POST" "PUT" "DELETE" "PATCH" }`
  rejects typos like `method GTE` at transpile time.
- **The MCP server** — an AI agent designing a new feature can
  emit `handler` lines into your `script.capy` and trust the
  library to produce conformant code + tests.

## When this is the wrong tool

- **Highly dynamic dispatch**. Capy is for declarative shapes. If
  your handlers are built from a dataclass of complex policy
  decisions, the DSL becomes awkward.
- **Single-handler projects**. The conventions-as-library pattern
  pays off when there are enough handlers (and enough turnover) for
  the rules to matter. For a 3-route hobby project, just write
  Go.

For everything in between — internal APIs, CRUD services, gateways,
admin tooling, the bread-and-butter backend work that absorbs most
engineering time — this pattern eliminates a category of review
nitpicks and bus-factor risk.
