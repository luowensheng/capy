# transpile-landing-page

Section DSL → a complete responsive HTML landing page with embedded
CSS. No external assets; drop into any static host.

```sh
../../capy run lib.yaml script.capy > index.html
open index.html
```

10 lines of source produce a ~70-line standalone HTML document with
grid-based features section, two CTAs, and a clean type ramp.

## What this teaches

- **Context-driven page assembly.** Every function only updates
  context; no body emission. The file template owns the entire page
  structure, looping context lists for features and CTAs.
- **Embedded CSS** rendered through the template engine — pre-computed,
  no runtime needed.
- **Single-file output**, ready to deploy.
