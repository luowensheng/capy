# Transpiler Patterns

Capy is, fundamentally, a transpiler: source in, target out. This page
collects the standard ways Capy libraries divide work between
**written output** (text that flows through statement-by-statement
via `write`) and **accumulated context** (state assembled at the
end and consumed by `file_template`).

## Pattern 1: Statement writes directly to body

The simplest pattern. Each statement's body `write`s text that
appears in the output in source order.

```
function say
    arg capture msg any
    write `print(${msg})
`
end
```

```
say "a"      →     print("a")
say "b"      →     print("b")
```

Used in: most "code generators" where source and output have similar
structure.

## Pattern 2: Statement updates context, writes nothing

Use when the source declares something that contributes to a different
output position (often the top, often deduplicated).

```
function import
    arg literal "import"
    arg capture name ident
    # No `write` → nothing flows into body.
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

Used in: import collection, decl ordering, "front matter" assembly.

## Pattern 3: Block opener writes a header + the rendered body

The canonical control-flow shape.

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

`${body}` inside the write literal is the rendered output of every
statement inside the block, indented by 4. The closer here is
silent (empty body).

## Pattern 4: Body is irrelevant, context IS the output

Useful for declarative formats (JSON, TOML, YAML configs).

```
function set_name
    arg capture n any
    set context.name n
end

file_template
    write (toJSONIndent context)
end
```

Used in: `samples/transpile-json/`. No function writes; the entire
output is the marshalled context.

## Pattern 5: Deduplication via context

If the same source can produce duplicate items, use a map-keyed set
instead of a list:

```
context
    seen_imports {}                  # map used as a set
end

function import
    arg literal "import"
    arg capture name ident
    set context.seen_imports[name] true
end

file_template
    for k in (keys context.seen_imports)
        write `import ${k}
`
    end
    write body
end
```

Each `import json` writes to the same key — duplicates collapse.

## Pattern 6: Conditional context updates

`if`/`else` in a function body is library-author logic, not user
logic. Use it to route a single match into different context lists.

```
function import
    arg literal "import"
    arg capture name ident
    if (regex_match name "^test_")
        append context.test_imports name
    else
        append context.app_imports name
    end
end
```

## Pattern 7: Pre/post fragments around a block

When you need to wrap a generated block in target-specific
boilerplate, write the header before `${body}` and the footer after.

```
function class
    arg literal "class"
    arg capture name ident
    block_closer end
    write `class ${name}:
${indent 4 body}
    # end class
`
end

function end
end
```

## Pattern 8: One source produces multiple output sections

When the target has distinct sections (e.g. SQL DDL vs DML),
accumulate each into its own context list and join them in
`file_template`.

```
context
    ddl []
    dml []
end

function create_table
    arg literal "create_table"
    arg capture name string
    append context.ddl "CREATE TABLE ..."
end

function insert
    arg literal "insert"
    arg capture row string
    append context.dml "INSERT INTO ..."
end

file_template
    write `-- DDL
`
    for q in context.ddl
        write `${q}
`
    end
    write `-- DML
`
    for q in context.dml
        write `${q}
`
    end
end
```

## Anti-pattern: trying to execute user logic

Capy does not run the source. `if x ... end` in the source emits an
`if` in the target (or whatever the library defines); it doesn't
decide at transpile time whether to skip rendering.

If you want compile-time conditional emission, do it in the
**library's** function body by reading `context`, not by reading
user-script variables.
