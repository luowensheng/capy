# `.capy` libraries — the primary format

Capy libraries are written in `.capy`, Capy's native syntax. It's
terser than YAML, multi-line `template:` and `run:` blocks read
natively, and you don't fight string-escape rules.

YAML is also accepted as a secondary format for teams that need
existing YAML tooling (yq, JSON Schema). Both formats produce the
exact same in-memory library and behave identically:

| Format       | When                                                                                          |
|--------------|-----------------------------------------------------------------------------------------------|
| **`.capy`**  | **The default.** One syntax to learn, native multi-line templates, no YAML escape gotchas.    |
| **`.yaml`**  | When you need yq / JSON-schema tooling, or your config layer is already YAML.                 |

The loader dispatches on file extension:

```sh
capy run lib.capy script.capy   # primary
capy run lib.yaml script.capy   # secondary
```

Same engine, same output.

## Side by side

=== "YAML"

    ```yaml
    extension: c

    functions:
      fn:
        args:
          - { kind: literal, value: "fn" }
          - { kind: capture, name: name, type: ident }
          - { kind: literal, value: "(" }
          - { kind: capture, name: a, type: ident }
          - { kind: literal, value: "," }
          - { kind: capture, name: b, type: ident }
          - { kind: literal, value: ")" }
        block: { closer: end }
        template: |
          int {{ .name }}(int {{ .a }}, int {{ .b }}) {
          {{ .body | indent 4 }}
          }

      return:
        args:
          - { kind: literal, value: "return" }
          - { kind: capture, name: l, type: any }
          - { kind: literal, value: "+" }
          - { kind: capture, name: r, type: any }
        template: "return {{ .l }} + {{ .r }};\n"

      end: {}

    file_template: |
      #include <stdio.h>

      {{ .body }}
    ```

=== "Capy-native"

    ```
    extension c

    function fn
        arg literal "fn"
        arg capture name ident
        arg literal "("
        arg capture a ident
        arg literal ","
        arg capture b ident
        arg literal ")"
        block_closer end
        write `int ${name}(int ${a}, int ${b}) {
${indent 4 body}
}
`
    end

    function return
        arg literal "return"
        arg capture l any
        arg literal "+"
        arg capture r any
        write `return ${l} + ${r};
`
    end

    function end
    end

    file_template:
        #include <stdio.h>

        {{ .body }}
    ```

Both files, run against the same `script.capy`, emit byte-identical C source.

## The full surface

```
extension <STR>                           # output file extension
output_file <STR>                         # optional: write here instead of stdout

function <NAME>
    priority <INT>                        # optional: higher wins ambiguous matches
    arg literal <STR>                     # match this token literally
    arg capture <NAME> <TYPE>             # capture a token: any | ident | int | string | ...
    block_closer <NAME>                   # block opener: body runs until <NAME> appears
    block_open <STR> close <STR>          # OR: explicit delimiters

    # Function body — inner-DSL statements:
    write `text ${capture} ...`          # emit literal text + ${EXPR} interpolation
    set context.field value              # mutate state
    append context.list value            # …or push to a list
    if cond                              # conditional / control flow
        write `…`
    end

    # Legacy shape (still accepted):
    # template_str <STR>
    # template:
    #     ...
    # run:
    #     ...
end

file_template:
    {{ .body }}                            # whole-file wrapper
```

- **Strings** use double quotes with Go-style escapes (`\n`, `\t`, `\"`, `\\`).
- **Bare words** are accepted for `extension`, type names, and capture names.
- **Indentation** delimits `template:` / `run:` / `file_template:` blocks — the
  block ends when a line returns to the parent indent. The deepest common
  indent is stripped, so your template reads naturally.
- **Comments** start with `#` and run to end of line.

## Why `.capy` is the default

A YAML library file forces you to switch between two grammars on
every edit: Capy syntax (the user-facing language) and YAML syntax
(how you describe the user-facing language). For one-off scripts
that's fine. For a real library with a dozen functions and multi-line
templates, the constant context switch — *especially* YAML's
block-scalar indent rules that silently lose whitespace — burns
cycles.

`.capy` collapses that to one grammar. The same indentation rules
that govern your *source* files govern your *library* files. The
same string-literal rules. The same comment syntax. One mental
model. `capy check lib.capy` validates it the same way.

YAML stays in the box for the cases that genuinely need it: dropping
into an existing YAML-driven pipeline, or when you want to point a
JSON-Schema-aware editor at the library. Both formats produce
byte-identical output.

## See it in action

The [multi-language demo](showcase.md#one-source-five-programming-languages)
ships **both** forms of each of its five libraries:

```
samples/multi-language-demo/
├── lib_python.yaml      lib_python.capy
├── lib_javascript.yaml  lib_javascript.capy
├── lib_go.yaml          lib_go.capy
├── lib_rust.yaml        lib_rust.capy
├── lib_c.yaml           lib_c.capy
└── script.capy           ← the one shared source file
```

Diff any pair — same shape, no engine difference, byte-identical output.

```sh
diff \
  <(capy run lib_c.yaml script.capy) \
  <(capy run lib_c.capy script.capy)
# (empty — they match exactly)
```

## Caveats

- The `.capy` parser is intentionally line-oriented and small. It
  doesn't support YAML's full feature set (anchors, multi-document
  files). Those aren't useful in library files anyway.
- The two formats are kept feature-parallel: everything you can
  write in `.capy` you can write in YAML (and vice versa). If you find a
  case where the YAML form supports something the `.capy` form doesn't,
  file a bug — it's a parity gap we'd fix.
