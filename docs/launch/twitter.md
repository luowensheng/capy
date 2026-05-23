# Twitter / Mastodon / Bluesky thread (draft)

Short version (single post):

> 🌱 Capy v0.1.0 — a transpiler engine in Go where the grammar is a YAML
> file. Define your DSL, get a code generator. Zero default keywords.
> Single-binary install. Six worked examples (Python, JSON, SQL,
> Makefile, HTML, TS).
>
> github.com/luowensheng/capy

---

Thread version (5 posts):

1/ 🌱 Just shipped Capy v0.1.0. It's a small Go binary (~1500 LOC) that
turns a YAML file describing a grammar into a working transpiler. No
parser-generator, no code generation, no template engine alone — one
runtime that does all three.

2/ A library declares functions, types, and a file template. Each
function has an args shape (literals + typed captures), a body template
fragment, and an optional `run:` snippet that updates an accumulated
context.

```yaml
greet:
  args: [{ kind: capture, name: name, type: any }]
  template: "Hello, {{ .name }}!\n"
```

3/ The engine has zero default keywords. `if`, `loop`, `=` only exist
when the library defines them. The README example builds a tiny
Python-flavored DSL from scratch — `import`, `say`, `assign`, `if`,
`loop` — that transpiles to runnable Python.

4/ Six samples in the repo covering Python, JSON, SQL, Makefile, HTML
components, and TypeScript interfaces. Install via `go install` or one
of the binary releases. MIT.

5/ Pre-1.0 — the YAML schema will likely evolve. Roadmap includes:
`else` arm, multi-output, configurable surface syntax, `validate:`
snippets in inner Capy. Feedback wanted.
github.com/luowensheng/capy
