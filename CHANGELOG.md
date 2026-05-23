# Changelog

All notable changes to Capy are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) —
with the important caveat that **while pre-1.0, the library YAML schema
may break between minor versions** (see `CONTRIBUTING.md`).

## [Unreleased]

## [0.1.0] — 2026-05-23

Initial public release.

### Added

- **Zero default grammar** engine: every user-facing token shape is defined
  by the loaded library.
- **`functions:`** with kind-discriminated `args:` (`{kind: literal, value}`
  or `{kind: capture, name, type}`). Auto-name-prepend rule when args has
  no literals.
- **`types:`** with three optional validators applied in order: `base`,
  `pattern` (regex), `options` (enum).
- **`context:`** for the accumulated transpilation state; **`run:`** for
  context-mutation snippets in a small inner DSL (`set`, `append`,
  `prepend`, `merge`, `delete`, `if`, `loop`, `regex_match`, `error`).
- **`template:`** per function for body output; **`file_template:`** for
  final-file assembly using `.body` + `.context`.
- **Two block modes**: indent/dedent + named closer (`block: { closer }`)
  or explicit delimiter pair (`block: { open, close }`).
- **Template helpers**: `indent`, `lower`, `upper`, `join`, `toQuoted`,
  `toPyLit`, `toJSON`, `toJSONIndent`.
- Object-literal keys accept quoted strings OR bare identifiers
  (`{name: "x"}` is valid).
- Six samples: `empty-engine`, `assembly`, `types`, `scene-dsl`,
  `transpile-py`, `transpile-json`.
- VHCO project layout.

### Known limitations

- No `else` arm on inner `if` (single-arm only).
- No argument `default:` values.
- No multi-output: each library produces exactly one file.
- No configurable surface syntax (statement terminator, arg separator)
  — deferred to a future version.
- No `import` between library files.

[Unreleased]: https://github.com/luowensheng/capy/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/luowensheng/capy/releases/tag/v0.1.0
