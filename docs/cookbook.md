# Cookbook

Short, self-contained answers to common needs. Each entry shows just the
delta — copy into your own library and adjust types/names.

## Emit an import block at the top of the output

```
context
    imports []
end

function import
    arg literal "import"
    arg capture name ident
    append context.imports name
end

file_template
    for imp in context.imports
        write `import ${imp}
`
    end
    write body
end
```

## Deduplicate context entries

Use a map as a set:

```
context
    imports {}                       # map, not list
end

function import
    arg literal "import"
    arg capture name ident
    set context.imports[name] true
end

file_template
    for k in (keys context.imports)
        write `import ${k}
`
    end
end
```

## Validate that a string matches multiple patterns

Library-defined types apply `base`, `pattern`, and `options` in that order.
For complex multi-rule validation, encode it in the regex (e.g.
`^(?=.*[A-Z])(?=.*[0-9]).{8,}$`):

```yaml
types:
  StrongPassword:
    pattern: "^(?=.*[A-Z])(?=.*[0-9]).{8,}$"
```

For richer logic (AND across distinct patterns, length checks, etc.), use
multiple captures in different functions or open a feature request for
`validate:` snippets.

## Emit indented bodies for a block construct

```
function if
    arg literal "if"
    arg capture cond any
    block_closer end
    write `if ${cond}:
${indent 4 body}
`
end

function end
end
```

## Render a list of objects from context

```
context
    routes []
end

function route
    arg literal "route"
    arg capture method any
    arg capture path any
    append context.routes {method: method, path: path}
end

file_template
    write `ROUTES = [
`
    for r in context.routes
        write `    { "method": ${toQuoted r.method}, "path": ${toQuoted r.path} },
`
    end
    write `]
`
end
```

## Support both `do…end` and `{…}` block styles

Define two functions:

```
function for_indent
    arg literal "for"
    arg capture v ident
    arg literal "in"
    arg capture i any
    arg literal "do"
    block_closer end
    write `for ${v} in ${i}:
${indent 4 body}`
end

function end
end

function for_brace
    arg literal "for"
    arg capture v ident
    arg literal "in"
    arg capture i any
    block_open "{"
    block_close "}"
    write `for ${v} in ${i}: { ${body} }
`
end
```

The matcher picks whichever shape the source uses; the longer literal
prefix wins on ties.

## Emit a function name only if a context flag is set

In `file_template`:

```
file_template
    write body
    if context.has_main
        write `if __name__ == "__main__":
    main()
`
    end
end
```

Inside a function body:

```
function say
    arg capture msg any
    if context.verbose
        write `# log: ${msg}
`
    end
    write `print(${msg})
`
end
```

## Build a deeply nested context

Use dotted paths in a function body:

```
function configure
    arg literal "configure"
    set context.config.api.url "https://example.com"
    set context.config.api.timeout 30
    set context.config.db.host "localhost"
end
```

You can also use bracket indexing for dynamic keys:

```
function register
    arg capture name ident
    arg capture cmd string
    set context.scripts[name] cmd
end
```

## Stop transpilation with a clear error

```
function declare
    arg capture name ident
    if (regex_match name "^_")
        error "names starting with underscore are reserved"
    end
    append context.declared name
end
```

## Have a function that adds nothing to body, just runs side effects

Just don't call `write`:

```
function register
    arg capture name ident
    append context.registered name
end
```

Body output: empty. Context: gains the new name.

## Capture a multi-line JSON literal as an argument

JSON literals can span lines because the value parser skips newlines inside
`{ }` / `[ ]`:

```
config {
    "host": "localhost",
    "port": 5432
}
```

Just declare the arg as `type: any` and the parser handles the rest.
