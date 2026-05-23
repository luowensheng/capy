# Migration Guide

Per-release notes for breaking changes. Each section gives the old vs new
shape, plus a brief rationale.

## Upgrading to 0.1.0

There's nothing to migrate from — this is the initial public release. All
future breaking changes will appear here with explicit before/after
snippets and (where possible) automated migration steps.

## Template

For future entries, follow this template:

```
## 0.X.0 → 0.Y.0

### Field renamed: `foo:` → `bar:`

**Before**

```yaml
functions:
  greet:
    foo: ...
```

**After**

```yaml
functions:
  greet:
    bar: ...
```

**Why**

(rationale)

**Migration**

(automated script or one-line sed, when possible)
```
