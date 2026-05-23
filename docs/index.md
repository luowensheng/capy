---
title: Capy
hide:
  - navigation
---

# Capy

> **A transpiler engine with zero default grammar.** Define a tiny
> source language in YAML; the engine turns input into any target
> output you describe. 50 worked demos in the repo.

[Get started in 5 minutes :material-rocket-launch:](getting-started.md){ .md-button .md-button--primary }
[For AI agents :material-robot:](ai-agents.md){ .md-button }
[Browse 50 demos :material-folder-open:](https://github.com/luowensheng/capy/tree/main/samples){ .md-button }

---

## What Capy gives you

<div class="grid cards" markdown>

- :material-shield-lock: **Sandbox AI output**

    The library *is* the grammar. An agent can only emit shapes
    you defined. No way to write `DROP TABLE`, no way to call
    unauthorised hosts, no way to escape into arbitrary code.

- :material-coins: **5–10× fewer output tokens**

    Agents emit short structured DSL; the engine deterministically
    expands it into long boilerplate-heavy code. The library is
    reusable across hundreds of calls.

- :material-brain: **Lower task complexity**

    The agent reasons about *intent*, not syntax. Imports,
    indentation, types, scaffolding, framework idioms — all
    encoded in the library, hidden from the agent.

- :material-shield-check: **Fewer failure points**

    Parser rejects malformed input before the agent's output
    reaches any system. Type validation catches semantic errors
    at parse time. Output is by construction syntactically valid.

</div>

[Read the full AI agents guide →](ai-agents.md)

---

## 60-second tour: nine ways to use Capy

Each tab shows the **Capy source** and the **generated target**.
Source is short and declarative; targets are real, runnable artifacts.

=== "Python"

    **`lib.yaml`** declares a few functions; **`script.capy`** uses them:

    ```
    import json
    import os
    say "hello, world"

    if x
        say "x is set"
    end

    loop n in [1, 2, 3]
        say n
    end
    ```

    Generated **`out.py`**:

    ```python
    import json
    import os
    print("hello, world")
    if x:
        print("x is set")

    for n in [1, 2, 3]:
        print(n)
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-py)

=== "Canvas game"

    12 lines of game-DSL produce a runnable HTML5 canvas page with
    sprites, key handlers, and a `requestAnimationFrame` loop:

    ```
    game "Block Hopper" 480 320

    sprite player "#4dd" 220 280 40 20
    sprite enemy  "#f64" 100 100 30 30

    on_key "ArrowLeft"  player -4 0
    on_key "ArrowRight" player  4 0

    tick enemy_bounce "sprites.enemy.x += 1; if (sprites.enemy.x > 450) sprites.enemy.x = 0;"
    ```

    Generated **`game.html`** (excerpt):

    ```javascript
    const sprites = {
      player: { x: 220, y: 280, w: 40, h: 20, color: "#4dd" },
      enemy:  { x: 100, y: 100, w: 30, h: 30, color: "#f64" },
    };
    function update() {
      if (keys["ArrowLeft"])  { sprites.player.x += -4; }
      if (keys["ArrowRight"]) { sprites.player.x +=  4; }
      sprites.enemy.x += 1; if (sprites.enemy.x > 450) sprites.enemy.x = 0;
    }
    function loop() { update(); draw(); requestAnimationFrame(loop); }
    loop();
    ```

    **12 lines → 67 lines of runnable HTML5 (5.5×).**
    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-canvas-game)

=== "Postgres schema"

    ```
    table users
        pk     id
        unique email "varchar(255)"
        col    name  "varchar(255) NOT NULL"
    end

    table posts
        pk     id
        fk     author_id -> users
        col    title "varchar(255) NOT NULL"
    end

    index posts author_id
    ```

    Generated **`schema.sql`**:

    ```sql
    CREATE TABLE users (
      id bigserial PRIMARY KEY,
      email varchar(255) UNIQUE NOT NULL,
      name varchar(255) NOT NULL
    );
    CREATE TABLE posts (
      id bigserial PRIMARY KEY,
      author_id bigint NOT NULL REFERENCES users(id),
      title varchar(255) NOT NULL
    );

    CREATE INDEX ix_posts_author_id ON posts(author_id);
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-postgres-schema)

=== "Express server"

    ```
    port 8080

    use "morgan('combined')"
    get  "/health" "res.json({ok: true})"
    post "/users"  "const u = req.body; res.status(201).json({id: 42, ...u})"
    ```

    Generated **`server.js`**:

    ```javascript
    const express = require("express");
    const app = express();

    app.use(express.json());
    app.use(morgan('combined'));

    app.get("/health", (req, res) => {
      res.json({ok: true})
    });

    app.post("/users", (req, res) => {
      const u = req.body; res.status(201).json({id: 42, ...u})
    });

    app.listen(8080, () => { console.log("listening on", 8080); });
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-express-server)

=== "Kubernetes"

    ```
    deployment capy_api
    image    "ghcr.io/luowensheng/capy:0.1.0"
    replicas 3
    port     8080
    ```

    Generated **`deployment.yaml`**:

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: capy_api
    spec:
      replicas: 3
      template:
        spec:
          containers:
            - name: capy_api
              image: ghcr.io/luowensheng/capy:0.1.0
              ports:
                - containerPort: 8080
    ```

    **4 lines → 13-line manifest (3.2×).**
    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-kubernetes)

=== "GraphQL schema"

    ```
    type User
        required id : "ID"
        required name : "String"
        field    role : "UserRole"
    end

    enum UserRole
        variant ADMIN
        variant MEMBER
    end
    ```

    Generated **`schema.graphql`**:

    ```graphql
    type User {
      id: ID!
      name: String!
      role: UserRole
    }
    enum UserRole {
      ADMIN
      MEMBER
    }
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-graphql)

=== "Slack message"

    ```
    header  "📦 Build complete"
    section "Branch *main* built in *4m 12s* and is ready to deploy."
    divider
    section "Tests: 124/124 passing"
    button  "View build" "https://ci.example.com/build/1234"
    ```

    Generated **Slack Block Kit JSON** (POST to a webhook):

    ```json
    {
      "blocks": [
        { "type": "header", "text": { "type": "plain_text", "text": "📦 Build complete" } },
        { "type": "section", "text": { "type": "mrkdwn", "text": "Branch *main* built..." } },
        { "type": "divider" },
        { "type": "section", "text": { "type": "mrkdwn", "text": "Tests: 124/124 passing" } },
        { "type": "actions", "elements": [{ "type": "button", "text": "...", "url": "..." }] }
      ]
    }
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-slack-blocks)

=== "React component"

    ```
    component Counter
        prop  label : "string"
        state count : "number" = 0
        effect "count" "document.title = 'count: ' + count"
        render "<div><h1>{label}: {count}</h1>..."
    end
    ```

    Generated **`Counter.tsx`**:

    ```tsx
    import React, { useState, useEffect } from "react";

    type CounterProps = { label: string };

    export function Counter(props: CounterProps) {
      const [count, setCount] = useState<number>(0);
      useEffect(() => {
        document.title = 'count: ' + count
      }, [count]);

      return (<div><h1>{label}: {count}</h1>...);
    }
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-react-component)

=== "Assembly (x86-64)"

    ```
    program "sum-demo"
        var x = 5
        var y = 7
        add x y
        store result
        exit 0
    end
    ```

    Generated **`demo.asm`** (real NASM, assembles with `nasm -felf64`):

    ```asm
    section .data
        x: dq 0
        y: dq 0

    section .text
        global _start

    _start:
        mov rax, 5
        mov [x], rax
        mov rax, 7
        mov [y], rax
        mov rax, [x]
        add rax, [y]
        mov [result], rax
        mov rdi, 0
        mov rax, 60
        syscall
    ```

    A high-level source language → real assembly with a `.data`
    section auto-built from tracked symbols.
    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/assembly)

---

## Where Capy shines

Capy fits **anywhere you'd hand-roll a tiny parser to drive code
generation**. The pattern shows up in a lot of places once you start
noticing it.

<div class="grid cards" markdown>

- :material-robot: **AI agents & LLM copilots**

    Sandbox + token compression + complexity reduction. The most
    valuable use case Capy enables, and the one that's hard to do
    with any other tool. [Read the full guide →](ai-agents.md)

- :material-package-variant: **Internal scaffolding & generators**

    Replace Yeoman / Plop / Hygen / custom Go binaries that emit
    template files. The library is the conventions; the source is
    five lines. New generators ship by editing one YAML file.

- :material-sync: **One source → many targets**

    Define a user model once; generate Postgres DDL, TypeScript
    types, Pydantic, Zod, GraphQL — from the same source. Drift
    becomes impossible.

- :material-server: **Config-as-code at scale**

    50 services × 3 environments × k8s + Terraform + CI = boilerplate
    explosion. With Capy, each service is a 6-line file; the library
    encodes the policy.

- :material-account-tie: **DSLs for domain experts**

    Give finance / legal / healthcare experts a notation that's
    natural for their domain and compiles to runnable code. The
    grammar becomes an audit boundary.

- :material-file-document-multiple: **Documentation generation**

    Stop letting README, OpenAPI, changelog, and release notes
    drift. One source produces all of them; CI re-runs when source
    changes.

- :material-history: **Migration / refactor tools**

    Old format → new format. Library parses the old, emits the new.
    Self-documenting, type-checked, beats a one-off Python script.

- :material-school: **Education & DSL design**

    Build a calculator language in 30 lines of YAML. Teach grammar
    + semantics + output without inflicting yacc on students.

- :material-shield-check: **Audit & compliance**

    Every artifact has a Capy-source lineage. "What's the policy
    for X?" = read function X. The grammar IS the policy.

</div>

[See all 14 use cases with concrete scenarios → `docs/use-cases.md`](use-cases.md)

---

## How it works

```mermaid
flowchart LR
  S[source.capy] --> L[Lexer]
  L --> P[Pattern matcher]
  P --> T[Per-statement templates]
  P --> C[Context updates via run:]
  T --> A[file_template:]
  C --> A
  A --> O[target output]
  Lib[lib.yaml] -. defines .-> P
  Lib -. defines .-> T
  Lib -. defines .-> C
```

Three things drive output:

1. **`args:`** — what shapes the parser recognises in source.
2. **`template:`** — what each match emits into the body.
3. **`run:`** — how each match updates a shared `context` (lists, maps, scalars).

A top-level **`file_template:`** assembles `body` + `context` into the
final file.

There are **no built-in keywords**. `if`, `loop`, `=`, blocks,
comments — all defined by the library, or not at all if your DSL
doesn't need them.

---

## Why Capy fits AI workflows

The same model that makes Capy a good DSL substrate makes it
genuinely useful for AI agents and copilots. Four properties:

### 1. Sandboxing — the grammar is the boundary

Anything not declared in the library is a **parse error**. Whatever
an agent emits is, by construction, within the library's contract.

```yaml
# A restricted SQL DSL with a table whitelist.
types:
  TableName:
    options: ["users", "posts", "comments"]

functions:
  query:
    args:
      - { kind: literal, value: "select" }
      - { kind: capture, name: cols, type: any }
      - { kind: literal, value: "from" }
      - { kind: capture, name: tbl,  type: TableName }   # ← enforced
      - { kind: literal, value: "where" }
      - { kind: capture, name: cond, type: any }
    template: "SELECT {{ .cols }} FROM {{ .tbl }} WHERE {{ .cond }};\n"
```

The LLM can emit `select id from users where active`. It **cannot** emit:

- `DROP TABLE users` — there's no `DROP` pattern in the library.
- `select * from secrets where ...` — `secrets` isn't in `options`.
- `'; rm -rf /'` — wouldn't even tokenize as a Capy statement.

No prompt-injection class of attack works here. No post-hoc output
filtering. **The grammar is the boundary.**

Same shape for:

- **Shell DSLs** that whitelist `Command` to `ls`/`cat`/`grep` —
  agent can't `rm`.
- **HTTP DSLs** with `Host` regex'd to your own domain — agent can't
  hit `evil-corp.example.com`.
- **Code-gen DSLs** that only define safe JSX patterns — no XSS
  via raw HTML.

### 2. Reduced token usage

| Mode             | LLM emits | Engine produces |
|------------------|-----------|-----------------|
| Naive codegen    | ~800 tokens of Python (Flask app) | (none) |
| Capy             | ~50 tokens of Capy source | ~800 tokens of Python (deterministic) |

The library is in context **once** (or loaded on demand from a file).
After that, every generation emits short structured source.

In a single call the savings are 5–10×. **In an agent loop the gap
compounds** — same library, hundreds of generations, per-call cost
approaches the source size.

Concrete ratios from samples:

| Demo | Source lines | Output lines | Ratio |
|------|--------------|--------------|-------|
| Canvas game     | 12 | 67 | **5.5×** |
| Landing page    | 9  | 54 | **6.0×** |
| Express server  | 8  | 24 | **3.0×** |
| Kubernetes      | 4  | 13 | **3.2×** |
| Postgres schema | 18 | 21 | 1.2× (mostly structure) |
| XState machine  | 9  | 30 | **3.3×** |

The ratios understate the win. The interesting cost isn't lines —
it's tokens, and target code has high token-per-line density
(boilerplate, repeated identifiers, type annotations).

### 3. Reduced task complexity for the agent

Without Capy, generating a Flask app means the agent has to remember:
imports, Flask app instantiation, `jsonify`/`request` wiring, route
decorator syntax, response codes, JSON shapes. Each detail is a
place to make a mistake.

With Capy, the agent reasons at the level of **`route post "/users"
create_user "..."`** — the shape of the API it's building. Everything
syntactic lives in the library.

The agent gets:

- **No import bookkeeping.** Library tracks imports via `run:`.
- **No indentation worry.** Block templates handle nesting.
- **No framework idioms.** Library encodes Flask, Express, FastAPI,
  etc., once.
- **No boilerplate review.** Output is template-driven.

The agent's job collapses to: *which functions, with which arguments,
in what order?*

### 4. Reduced failure points

| Property | Raw LLM codegen | Capy + LLM |
|----------|-----------------|------------|
| Output passes a grammar check | ⚠️ usually | ✅ always |
| Output passes type validation | ⚠️ sometimes | ✅ always (when types declared) |
| Output uses only allowed APIs / tables / hosts | ⚠️ depends on prompt | ✅ enforced by library |
| Same input produces same output | ❌ | ✅ |
| Easy to audit what the agent can produce | ❌ | ✅ (`capy check lib.yaml`) |
| Single point of fix when target changes | ❌ | ✅ (edit one library) |

The combined effect: fewer retries, fewer guardrails to write,
fewer review cycles. The agent contributes content; the library
contributes correctness.

[Full AI agents guide → token math, three workflow patterns,
integrations with Claude Code / Cursor / Continue / Aider](ai-agents.md)

---

## Install

```sh
# Go users
go install github.com/luowensheng/capy/cmd/capy@latest

# macOS / Linux (binary, no Go required)
curl -fsSL https://raw.githubusercontent.com/luowensheng/capy/main/scripts/install.sh | sh
```

Verify:

```sh
capy version
capy help
```

---

## Where to go next

<div class="grid cards" markdown>

- :material-rocket-launch: **[Getting started](getting-started.md)**

    Install, run a sample, and understand the four things every
    library controls.

- :material-pencil: **[Library authoring](library-authoring.md)**

    The reference walkthrough for writing your own `lib.yaml`.

- :material-robot: **[Capy for AI agents](ai-agents.md)**

    Token cost math, three sandboxing patterns, integrations with
    Claude Code, Cursor, Continue, and Aider.

- :material-book: **[Tutorials](tutorials/01-hello-world.md)**

    Four progressive lessons: Hello → config DSL → Python transpiler
    → custom operators.

- :material-toolbox: **[Cookbook](cookbook.md)**

    Recipes for common patterns.

- :material-list-box: **[Feature reference](features.md)**

    Flat list of everything Capy ships with.

- :material-help-circle: **[FAQ](faq.md)**

    Common questions answered.

- :material-folder-open: **[50 demos on GitHub](https://github.com/luowensheng/capy/tree/main/samples)**

    Full library + script + verified golden output for every kind
    of target.

</div>

---

## All 50 demos at a glance

The tour above showed 9 demos. Here's the full catalogue:

### Web frontend (7)

`canvas-game` · `css-animations` · `react-component` · `landing-page`
· `html-component` · `transpile-form` · `transpile-email-html`

### Backend (3)

`express-server` (Node) · `flask-app` (Python) · `fastapi-app` (Python)

### Code generation (10)

`transpile-py` · `transpile-typescript` · `transpile-go` ·
`transpile-sql` · `transpile-protobuf` · `transpile-graphql` ·
`transpile-tests` · `transpile-cli` (Cobra Go) · `transpile-bash`
· `assembly` (x86-64 NASM)

### Configuration / IaC (13)

`transpile-json` · `transpile-env` · `transpile-dockerfile` ·
`transpile-makefile` · `transpile-nginx` · `transpile-systemd` ·
`transpile-kubernetes` · `transpile-gh-actions` · `transpile-cron`
· `transpile-terraform` · `transpile-openapi` ·
`transpile-prometheus-alerts` · `transpile-chrome-extension`

### Schemas / models (4)

`transpile-postgres-schema` · `transpile-prisma-schema` ·
`transpile-zod-schema` · `transpile-xstate-machine`

### Docs / data / diagrams (10)

`transpile-markdown-todo` · `transpile-blog` ·
`transpile-changelog` · `transpile-resume` ·
`transpile-api-docs` · `transpile-invoice` · `transpile-csv` ·
`transpile-mermaid` · `transpile-statemachine` ·
`transpile-slack-blocks`

### Concept demos (3)

`empty-engine` (proof of zero default grammar) · `types`
(validation) · `scene-dsl` (declarative)

[Browse all 50 demos on GitHub →](https://github.com/luowensheng/capy/tree/main/samples)

---

## Why Capy, not …?

Not a templating engine (it has a parser). Not a parser generator
(it has a runtime). Something in between: **a configurable
transpiler**, with the configuration written as data.

| Tool | What it does | What Capy adds |
|------|--------------|----------------|
| Jinja, Go templates | Substitute values into text | A real parser + accumulated context + types |
| ANTLR, lark, tree-sitter | Parse a language you defined | Targeted at code generation; ships with a runtime; no Java/Python required |
| Custom Go transpilers | Full control | A YAML schema replaces hundreds of lines of code per project |
| gomplate, ytt | Powerful templating with data | A source language with custom syntax, not just template inputs |
| Raw LLM codegen | Maximum flexibility | Determinism + sandboxing + token compression for agents |

Use Capy when you'd otherwise hand-roll a tiny parser to drive
code-generation: configuration languages, scaffolding tools, DSLs
for domain experts, source-to-source rewrites, and **especially
constrained LLM output**.

---

## Status

**Pre-1.0.** The library YAML schema may change between minor
versions. See
[CHANGELOG](https://github.com/luowensheng/capy/blob/main/CHANGELOG.md)
for what's stable, [roadmap](roadmap.md) for what's planned.

[MIT licensed](https://github.com/luowensheng/capy/blob/main/LICENSE). Built in Go. Single binary, no runtime dependencies.
