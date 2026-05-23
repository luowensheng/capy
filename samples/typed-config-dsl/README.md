# typed-config-dsl

**Named variables and type checking in Capy.**

This sample is a service-config DSL where every captured value has a
declared type and Capy validates it at transpile time. Bad input is
a precise error pointing at the offending value — not a silent
mis-render or a runtime surprise downstream.

## The DSL in action

`script.capy`:

```
service "api-gateway" version "2.4.1"
    env prod
    port 8443
    owner "platform@example.com"
    log_level info
    brand_color "#4dd9c0"
    tls true
end
```

`./capy run lib.capy script.capy`:

```hcl
service api-gateway {
  version = 2.4.1
  env = prod
  port = 8443
  owner = platform@example.com
  log_level = info
  brand_color = #4dd9c0
  tls = true
}
```

## The types

The library declares seven types, exercising every kind Capy supports:

```
type Email
    pattern "^[^@]+@[^@]+\\.[^@]+$"          # regex validation
end

type Semver
    pattern "^[0-9]+\\.[0-9]+\\.[0-9]+(-[a-zA-Z0-9.]+)?$"
end

type HexColor
    pattern "^#[0-9a-fA-F]{6}$"
end

type LogLevel
    options "trace" "debug" "info" "warn" "error" "fatal"   # enum
end

type Env
    options "dev" "staging" "prod"
end

type Port
    base int                                  # inheritance: a Port is an int
end

type ServiceName
    pattern "^[a-z][a-z0-9-]{1,30}$"
end
```

Plus the **built-in** kinds Capy already understands without any
declaration: `any`, `ident`, `raw`, `string`, `int`, `float`, `bool`.

## Named captures

Each `arg capture NAME TYPE` binds a typed named variable that the
template can reference by name:

```
function service
    arg literal "service"
    arg capture name ServiceName       # ← name is a ServiceName
    arg literal "version"
    arg capture ver Semver             # ← ver is a Semver
    block_closer end
    template:
        service {{ .name }} {
          version = {{ .ver }}
        ...
end
```

The captures (`name`, `ver`, `stage`, `n`, `who`, `lvl`, `c`, `on`)
are *named* — templates access them as `.name`, `.ver`, etc., not by
position.

## What validation looks like when it fails

`script-invalid.capy` violates every type:

```
service "Bad Name!" version "v2"
    env production
    port 99999
    owner "not-an-email"
    log_level verbose
    brand_color "blue"
    tls maybe
end
```

Running it:

```
$ ./capy run lib.capy script-invalid.capy
error: function "service" arg "name": value "Bad Name!"
       does not match pattern for type "ServiceName"
```

The transpilation stops cold at the first bad value, pointing at the
function, the argument name, the actual value, and the type rule it
violated. Fix it (`api-gateway` matches the slug pattern), re-run,
hit the next error, etc.

Why this matters:

- **Catch typos at the boundary**, not after deployment. `log_level
  verbose` is a no-op-with-a-bug in most config systems; in Capy it's
  a parse error.
- **The library is the schema**. New contributors don't need to read
  source code to learn what fields exist or what values are valid —
  the `type:` blocks ARE the spec.
- **Templates stay simple** — they don't need defensive coding because
  bad data never reaches them.

## Three ways to declare a type

| Declaration                 | Use when…                                         |
|-----------------------------|---------------------------------------------------|
| `pattern "regex"`           | The value has a known textual shape (Email, Semver, hex colors, slugs). |
| `options "a" "b" "c"`       | The value is one of a fixed set (log levels, environments, status). |
| `base BUILTIN`              | The value is a refinement of a built-in (a `Port` is an int with a meaningful name). |

You can combine them — `base int` followed by `pattern "..."` chains
validation: the value must first be a valid int, *then* match the
pattern.
