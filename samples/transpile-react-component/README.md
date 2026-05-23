# transpile-react-component

Component DSL → a full TypeScript React function component with hooks.

```sh
../../capy run lib.yaml script.capy > Counter.tsx
```

7 lines of spec produce a typed `Counter.tsx` with imports, prop type,
`useState`/`useEffect` calls, and the JSX body.

## What this teaches

- **Multiple context lists** (props, state, effects) assembled around a
  single rendered body in the component template.
- **TypeScript-typed output** with full hook wiring from declarative source.
- **Raw JSX passthrough** for the render body — the library doesn't try
  to parse JSX itself, it just embeds it.
