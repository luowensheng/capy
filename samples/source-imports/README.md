# source-imports

**`@import "path"` — pull other Capy source files into the current one.**

This sample shows the source-side counterpart to library imports.
A menu file imports shared "drinks" and "desserts" sections from
adjacent files. Multiple menus can reuse the same shared sections
without copy-paste.

## File layout

```
source-imports/
├── lib.capy                  ← menu/section/item/note DSL
├── script.capy               ← the menu — uses @import
└── shared/
    ├── drinks.capy           ← reusable drinks section
    └── desserts.capy         ← reusable desserts section
```

## The main source

```
menu "Capy Cafe — Spring 2026"

    section "Mains"
        item "House pasta"               "$16"
        item "Sheet-pan salmon"          "$22"
        item "Black bean tacos (3)"      "$14"
    end

    @import "shared/drinks.capy"
    @import "shared/desserts.capy"

    note "All dishes are made fresh. Please tell us about allergies."
end
```

The `@import` directives sit inside the `menu ... end` block at the
same indent as the surrounding statements. The preprocessor expands
each one *with the same leading indent* — so the imported content
nests naturally within the parent block.

## What gets imported

`shared/drinks.capy`:

```
section "Drinks"
    item "Espresso"        "$3"
    item "Cappuccino"      "$4.50"
    item "Cold brew"       "$5"
    item "Mint lemonade"   "$5"
end
```

After preprocessing, the source looks like one big file — the
parser never sees the import boundaries.

## Run

```sh
../../capy run lib.capy script.capy > menu.md
```

Output: clean Markdown with all three sections from three files,
plus the note.

## Rules

- **Path resolution** is relative to the file containing the
  `@import` line.
- **`@import` and `@include`** are synonyms.
- **Indentation auto-tracks**: a `    @import "x.capy"` line (4
  spaces of indent) inlines the imported content with each line
  re-indented 4 spaces.
- **Cycles** (A imports B which imports A) are detected by absolute
  path and produce a clean error.
- **Imports are processed before the lexer runs** — the parser
  never knows they happened.

## Use cases

- A menu that imports a shared price list shared with the catering
  arm of the business.
- A blog post that imports a standard footer / author bio block.
- A multi-environment config where `script.capy` selects which
  environment's overrides to `@import` based on… *(actually, you'd
  do this at script-generation time; Capy is single-pass)*.
- A long source file split for readability — the parser sees one
  unified stream after preprocessing.

## Vs. library imports

Both kinds exist for different problems:

| Feature        | Library `import` (`.capy` lib)            | Source `@import` (`.capy` source)       |
|----------------|-------------------------------------------|-----------------------------------------|
| Where it goes  | Top of the library file                   | Inside script.capy at any indent         |
| What it imports| Functions, types, context, file blocks    | Source statements (any DSL keyword)      |
| Resolution     | Merged at library load time               | Expanded at source preprocess time       |
| Conflict rule  | Importer wins                             | n/a (text inclusion)                     |
| Use it for     | Sharing DSL building blocks               | Sharing pieces of authored content       |
