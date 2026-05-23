# `.capy` libraries — write the library in Capy too

Capy libraries can be written in two formats. Both produce the exact same
in-memory library and behave identically:

| Format       | When you want…                                                       |
|--------------|----------------------------------------------------------------------|
| **`.yaml`**  | Universal tooling (yq, JSON schema, every editor) and the format LLMs already speak fluently. |
| **`.capy`**  | One syntax to learn, native multi-line templates, no YAML escape gotchas. |

The loader dispatches on file extension:

```sh
capy run lib.yaml script.capy   # YAML library
capy run lib.capy script.capy   # Capy-native library
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
        template:
            int {{ .name }}(int {{ .a }}, int {{ .b }}) {
            {{ .body | indent 4 }}
            }
    end

    function return
        arg literal "return"
        arg capture l any
        arg literal "+"
        arg capture r any
        template_str "return {{ .l }} + {{ .r }};\n"
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
    template_str <STR>                    # single-line inline template
    template:                              # multi-line template; ends at dedent
        {{ .capture }} ...
    run:                                   # inner-DSL block (context mutations)
        append context.imports "json"
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

## Why this exists

A YAML library file requires you to fluently switch between two grammars
on every edit: Capy syntax (the user-facing language) and YAML syntax (how
you describe the user-facing language). For one-off scripts that's fine.
For a real library with a dozen functions and multi-line templates, the
constant context switch — *especially* the YAML block-scalar indent rules
that silently lose whitespace — burns cycles.

`.capy` libraries collapse that to one grammar. The same indentation rules
that govern your *source* files govern your *library* files. The same
string-literal rules. The same comment syntax. One mental model.

The trade-off: you give up YAML's universal tooling. A `.yaml` lib can be
linted by every YAML language server on earth; a `.capy` lib is validated
by `capy check`. Pick the format that fits your workflow — Capy supports
both for the same library.

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

- The Capy-native parser is intentionally line-oriented and small. It
  doesn't yet support YAML's full feature set (anchors, multi-document
  files). Those aren't useful in library files anyway.
- `types:` and `context:` top-level sections are currently YAML-only.
  Add them to your library in YAML form, or open an issue if you need
  them in `.capy` form.
- The two formats are kept feature-parallel for the common case
  (functions + templates + run blocks + file_template). If you find a
  case where the YAML form supports something the `.capy` form doesn't,
  file a bug — it's a parity gap we'd fix.
