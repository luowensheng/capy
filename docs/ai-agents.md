# Capy for AI agents

Capy was designed to be useful to humans, but its model unlocks two
properties that are quietly enormous for AI workflows:

1. **Token compression** — agents emit short structured source instead
   of long boilerplate-heavy code. The library does the rest.
2. **Sandboxing** — the library is a contract. Whatever the agent
   produces is constrained to shapes the library defined. No prompt
   injection, no arbitrary shell escape, no malformed output.

This page explains how to use Capy with LLMs and agentic frameworks to
get both.

---

## Why the model fits LLMs

When an LLM generates code, it spends most of its tokens on boilerplate
the surrounding context already implies: function signatures, import
statements, repeated patterns, error handling scaffolding, type
declarations the user didn't ask for.

Capy turns that boilerplate into **library data** — written once, reused
forever. The agent emits the **semantic content**, which is typically
5–10× smaller, and the engine deterministically expands it into the
target language.

| What the LLM emits             | What gets produced               | Compression |
|--------------------------------|----------------------------------|-------------|
| 12 lines of Capy game spec     | 67 lines of runnable HTML5 game  | 5.5×        |
| 9 lines of landing-page DSL    | 54 lines of responsive HTML+CSS  | 6.0×        |
| 4 lines of Kubernetes spec     | 13 lines of multi-section YAML   | 3.2×        |
| 8 lines of Express route DSL   | 24 lines of Node server          | 3.0×        |
| 5 lines of schema DSL          | 16 lines of PostgreSQL DDL       | 3.2×        |

The ratio is biggest for boilerplate-heavy targets (UI, infra,
plumbing). For dense, semantically-rich targets (Python with control
flow) the savings are smaller — but the agent still benefits from the
**determinism** (no off-by-one bugs, no missing imports) and
**validation** (the library type-checks args at parse time).

These numbers are for one-shot generation. **In agentic loops the gap
compounds**: the same library is reused across hundreds of calls; the
per-call cost approaches the size of the source the agent emits.

---

## Why sandboxing emerges for free

The library is **the complete grammar** of the source language. Anything
not declared as a function is a parse error. This makes Capy a natural
sandbox for AI code generation in domains where you can't trust the
model to emit safe text.

### Example 1: a restricted SQL builder

Define a tiny SQL DSL that **only** supports `SELECT … FROM … WHERE …`
on a whitelist of tables. The model cannot emit `DROP TABLE` — the
library doesn't define that shape.

```
type TableName
    options "users" "posts" "comments"          # ← whitelist
end

function query
    arg literal "select"
    arg capture cols any
    arg literal "from"
    arg capture tbl TableName                    # ← validated
    arg literal "where"
    arg capture cond any
    write `SELECT ${cols} FROM ${tbl} WHERE ${cond};
`
end
```

The LLM can write:

```
select id from users where active
```

…and gets parameterised SQL. The LLM **cannot** write:

```
DROP TABLE users; --
```

…because there is no `DROP` pattern, and even `select * from secrets …`
fails because `secrets` is not in the `TableName` options. The agent is
constrained by data, not by an after-the-fact filter.

### Example 2: a sandboxed shell-script writer

Define commands the agent is allowed to invoke; nothing else exists.

```
type Command
    options "ls" "cat" "grep" "head" "tail"        # ← read-only commands
end

function run
    arg literal "run"
    arg capture cmd Command
    arg capture args any
    write `${cmd} ${args}
`
end
```

Even if the agent emits `run rm -rf /`, the type check fails: `rm` is
not in `options`. The output never leaves the engine in a runnable
form.

### Example 3: a typed API client

```
type Method
    options "GET" "POST"
end

type Host
    pattern "^https://api\\.mycompany\\.com/"
end

function api
    arg literal "api"
    arg capture method Method
    arg capture url Host
    arg capture body any
    write `fetch(${url | toQuoted}, {
  method: ${method | toQuoted},
  body: JSON.stringify(${body}),
});
`
end
```

The model can hit your API. It can't hit `evil-corp.example.com`,
because the URL must match the host regex. It can't `DELETE`, because
the method enum forbids it.

### Why this matters for agents

For an agent that has tool-use access to your codebase, the standard
risk story is "what if the model emits something nasty?" With raw
codegen you need post-hoc filtering, sandboxes, or human review.

With Capy, the boundary is **declarative and static**. You hand the
agent a library; whatever it emits is by construction within that
contract. The agent never sees a way out — there is no way out — so
you don't spend prompt budget trying to fence it off.

---

## Three workflow patterns

### Pattern A: "Design the library once, agent emits source forever"

1. A human (or one initial LLM pass) writes the `lib.capy`.
2. From then on, the agent only emits Capy source.
3. The transpiler does the heavy lifting deterministically.

Cost profile:

- One-time library design: ~2000–5000 tokens.
- Per-invocation source: ~50–200 tokens.
- Output complexity is unbounded.

Best for: scaffolding tools, CRUD generators, anything where the same
target shape is produced many times.

### Pattern B: "Two-pass code generation"

1. Agent first emits a high-level Capy plan.
2. The transpiler expands it into the target language.
3. Optionally, a second agent reviews the target output.

This decouples *intent* from *expression*. The first agent worries
about what the program should do. The library worries about how to say
it correctly.

Best for: cases where the target language has many footguns (memory
safety, async correctness, security) but the semantic structure is
simple.

### Pattern C: "Validate user-provided code against a Capy DSL"

If users (or another agent) submit code in your custom DSL, Capy is
also the validator: load the library, try to parse the input, surface
caret-pointed errors. The same artifact is grammar + validator +
generator.

Best for: user-facing scripting (admin DSLs, low-code platforms,
shareable snippets).

### Pattern D: "Human defines the syntax, AI fits a library to it"

This one is the most underrated for non-programmers and the most
collaborative for everyone else. The human starts by writing the
**source** in whatever way feels natural — the keywords they wish
existed, the indentation they prefer, the verbs that map onto how
they think about the domain. The AI's job is to make Capy parse
that source by iterating on the **library**.

The split of responsibilities:

| Who | Owns |
|---|---|
| **Human** | The DSL surface — `main.<ext>`. The words, the shapes, the structure they want to write in. |
| **AI** | The library — `<ext>.capy`. Function shapes, captures, `write` templates, commands. Whatever it takes to make the human's source compile to the target output. |

A typical loop:

1. Human writes a few lines of their imagined DSL.
2. They paste it to the AI with a one-sentence goal: *"this should
   produce an HTML calculator"* / *"this should generate a Kubernetes
   manifest"* / *"this should compile to a SQL view."*
3. AI iterates a library (adding `function`, `arg`, `write`, `block_*`,
   commands…) until `capy main.<ext>` produces the right output.
4. AI may also iterate Capy itself — if the human's source style
   reveals a gap (`bare` for keyword-less rows, `tail` for free-form
   values, `block_dedent` for indent-only blocks, `top_level` for
   position-aware emission), the engine grows to fit. **The user's
   syntax is the spec; the toolchain adapts.**
5. Once the library is good, the human edits `main.<ext>` freely
   without AI help — they're writing in their own vocabulary. They
   only return to the AI when they want to grow the DSL itself.

Why this matters for non-programmers:

- The artifact they read and edit is **the DSL they designed** —
  not React, not Tailwind, not Terraform. The vocabulary fits the
  domain.
- The library + Capy engine + AI prompt are the "compiler stack",
  but the user never has to look at them. They just write.
- Edits stay deterministic. The library is fixed; the same DSL
  source always produces the same output. No "the AI wrote
  something different this time."
- The AI is on tap for *grammar-level* changes (new keywords,
  new constructs) but doesn't need to be in the loop for
  *content-level* edits.

Why it matters for AI workflows:

- Less prompt context burned on re-explaining the target language
  on every call.
- The AI's outputs are tiny (`50–200` tokens of DSL) instead of
  large (`800+` tokens of HTML/YAML/SQL). Cost and latency both
  drop.
- Reviewability: a human can read `main.<ext>` and audit it in
  seconds. They can't audit `index.html` plus inline `<script>`
  plus inline `<style>` as fast.

Concrete shape of a session:

```
# Round 1: human-authored source
USER (paste):

    TodoList "Today"
        DO "Buy groceries"
        DO "Finish report" priority high
        DONE "Email client"
    end

USER (goal): "Generate a clean HTML page with a checkbox list."

# AI iteration: add `TodoList`, `DO`, `DONE`, and a `priority`
# attribute function; wire a file_template. Commit `todolist.capy`.

# Round 2: human iterates the source freely
USER (edit main.todolist):

    TodoList "Today"
        DO "Buy groceries" priority high
        DO "Pick up dry cleaning"
        DONE "Email client"

    TodoList "This week"
        DO "Tax filing"
        DO "Annual review prep"
    end

# Re-run `capy main.todolist` — works without an AI call.

# Round 3: human wants something new
USER: "Add a `due TOMORROW` annotation that renders as a yellow badge."

# AI adds one function to `todolist.capy`. Human keeps writing.
```

Best for:
- Domain-specific tools authored by domain experts (recipe writers,
  game designers, ops engineers, marketers, teachers).
- Long-lived projects where the SAME human edits the source many
  times, and only occasionally needs grammar growth.
- "I want a `lib.capy`-as-IDE" workflows where the DSL becomes a
  living tool tailored to one person or team.

---

## Integrations shipped in this repo

| Tool | Where | What it gives you |
|------|-------|-------------------|
| **Claude Code skill** | [`skills/capy-author/`](https://github.com/luowensheng/capy/tree/main/skills/capy-author) | A full skill with `SKILL.md` + instructions + 5 reference docs the model loads on demand. Triggers on "write a Capy library for …" or any `.capy` file in context. |
| **Slash commands** | [`commands/capy/`](https://github.com/luowensheng/capy/tree/main/commands/capy) | `/capy-new <target>`, `/capy-add-function`, `/capy-add-type`, `/capy-explain`, `/capy-debug` |
| **One-page LLM brief** | [`CAPY_FOR_LLMS.md`](CAPY_FOR_LLMS.md) | Self-contained prompt for any model. Paste into Cursor/Continue/Aider/raw-API system message. |
| **Cursor rule** | [`editors/cursor/`](https://github.com/luowensheng/capy/tree/main/editors/cursor) | Drop in `.cursor/rules/capy.md` |
| **Continue config** | [`editors/continue/`](https://github.com/luowensheng/capy/tree/main/editors/continue) | Adds the LLM brief to context |
| **Aider read** | [`editors/aider/`](https://github.com/luowensheng/capy/tree/main/editors/aider) | `aider --read docs/CAPY_FOR_LLMS.md` |
| **Generic system prompt** | [`agents/capy-system-prompt.md`](https://github.com/luowensheng/capy/blob/main/agents/capy-system-prompt.md) | Drop-in for any tool not listed above |

---

## Token math, in detail

Most LLM cost (and most latency) is in **output tokens**. The naive way
to use an LLM for code generation:

```
User prompt: "build a Python Flask app with these 3 routes ..."
LLM output: ~800 tokens of Python (imports + Flask boilerplate + routes)
```

The Capy way:

```
Library `.capy` (once, in context or a file): ~400 tokens
User prompt: "build a Flask app with these 3 routes ..."
LLM output: ~50 tokens of Capy source
Engine output (deterministic, free): ~800 tokens of Python
```

The model emits **16× fewer tokens** for the same target. Over an
agentic loop of 100 generations, that's the difference between a
$2 task and a $0.10 task.

The library is amortised over its lifetime. Once it exists, every
subsequent call benefits. Many libraries are also reusable across
projects (e.g. a Cobra CLI library, a Drizzle schema library, a
Kubernetes manifest library).

---

## Determinism: an underrated win

LLM output is non-deterministic by design. Capy output is
deterministic by construction:

| Property | Raw LLM codegen | Capy + LLM |
|----------|-----------------|------------|
| Same input → same output | ❌ (varies per sample) | ✅ |
| Output passes a grammar check | ⚠️ (usually) | ✅ (always) |
| Output passes a type check | ⚠️ (sometimes) | ✅ (always, if library has types) |
| Output is well-formed JSON / YAML / SQL | ⚠️ | ✅ |
| Output uses only allowed APIs/tables/hosts | ⚠️ | ✅ |
| Token cost grows with target complexity | ✅ | ❌ (constant w.r.t. boilerplate) |
| Easy to audit what the agent can do | ❌ | ✅ (`capy check lib.capy`) |

The right-hand column is what makes Capy a genuinely useful primitive
in agent toolchains, not just a templating engine.

---

## Putting it together: a minimal agent loop

```python
# Pseudocode for an agent that emits Capy source.

LIBRARY = open("lib.capy").read()          # ~400 tokens
SYSTEM = (
    "You are an agent that emits Capy source code. "
    "Here is the only language you may use:\n\n" + LIBRARY +
    "\n\nReply with ONLY Capy source. The transpiler will run it."
)

for task in tasks:
    resp = llm.complete(system=SYSTEM, user=task)   # ~50 tokens out
    source = resp.text
    target = subprocess.run(
        ["capy", "run", "lib.capy", "/dev/stdin"],
        input=source,
        check=True,                                  # ← rejects malformed
        capture_output=True,
    ).stdout
    deploy(target)
```

Three properties this gives you out of the box:

1. **Cost**: ~50 output tokens per task no matter how complex the
   target.
2. **Safety**: malformed or out-of-spec output is rejected at parse
   time. The agent can never deploy something that doesn't match the
   library's contract.
3. **Auditability**: you can read `lib.capy` in 5 minutes and know
   exactly what the agent is capable of producing.

---

## When Capy is NOT the right fit

Be honest about the trade-offs:

- **One-off generation** where the target shape changes every time.
  The library design cost won't amortise. Use raw LLM codegen.
- **When you actually need arbitrary code.** If you want the agent to
  invent novel algorithms, the constrained grammar is in your way.
  Use raw codegen with sandboxed execution.
- **When the library would be huge.** If your target language has
  100+ distinct shapes the agent might need, expressing them all in
  Capy is a chore. Use a real parser generator.

The sweet spot is **constrained, repetitive, boilerplate-heavy
output** — CRUD apps, infrastructure manifests, schemas, config
files, dashboards, components, scaffolding. Most agent work falls
here.

---

## Next steps

- Read [CAPY_FOR_LLMS.md](CAPY_FOR_LLMS.md) — the single-page brief
  you can paste into any model's context.
- Browse [the 50 sample demos](https://github.com/luowensheng/capy/tree/main/samples)
  — each is a tiny grammar producing a complete target file.
- Install the [Claude Code skill](https://github.com/luowensheng/capy/tree/main/skills/capy-author)
  if you use Claude Code.
- See [getting started](getting-started.md) and
  [library authoring](library-authoring.md) to design your own library.
