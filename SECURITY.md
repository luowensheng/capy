# Security Policy

## Supported Versions

While Capy is pre-1.0, only the latest released minor version is supported with security updates. Once 1.0 lands, this policy will be updated to keep the latest two minor versions patched.

| Version | Supported |
|---------|-----------|
| 0.1.x   | ✅        |
| < 0.1.0 | ❌        |

## Reporting a Vulnerability

**Do not file a public GitHub issue for a security report.**

Instead, email `the maintainer via a private GitHub Security Advisory at https://github.com/luowensheng/capy/security/advisories/new` with:

- A description of the vulnerability.
- Steps to reproduce, or a minimal `lib.yaml` + `script.capy` that triggers it.
- The version (`capy --version`) and your OS.
- Your name + GitHub handle for credit (optional).

We aim to acknowledge reports within 72 hours and to ship a fix within 14 days for confirmed high-severity issues. Lower-severity issues may be batched into the next regular release.

## Threat model

Capy reads two untrusted inputs:

1. **Library YAML** (`lib.yaml`). Treated as code by the engine — only run libraries you trust.
2. **Source script** (`.capy`). Treated as data and matched against library patterns; the engine does not execute the source.

Reasonable bug-bounty-worthy findings include:

- A crafted `lib.yaml` that crashes the engine (panic, infinite loop, resource exhaustion).
- A crafted `.capy` source that bypasses type validation declared in the library.
- A crafted `run:` snippet that escapes the inner DSL and runs arbitrary code on the host.
- Path traversal or arbitrary file write via `output_file:` (current scope: writes only to the path provided).

Not in scope (these are expected behaviors):

- A library that intentionally renders malicious target-language output. Capy is a code generator — the library decides what to emit.
- A library that intentionally consumes CPU/memory in `run:` snippets (no sandboxing).
