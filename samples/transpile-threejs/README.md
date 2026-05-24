# transpile-threejs — three.js scene from a tiny DSL

11 declarative lines → a complete, runnable HTML page that pulls
[three.js](https://threejs.org) from a CDN and animates a 3D scene.

The source language has **no JavaScript**. It declares meshes and
abstract motion verbs (`spin`, `orbit`, `bob`, `pulse`, `none`); the
library is what knows three.js. Swap the library for one that targets
Babylon.js, A-Frame, or raw WebGL and your scene script doesn't change.

## Run

```sh
capy run samples/transpile-threejs/lib.capy samples/transpile-threejs/script.capy > scene.html
open scene.html      # or `python3 -m http.server` and visit it
```

## Source language

```text
scene "title"                              page title + heading
camera <distance>                          camera Z distance
background "#RRGGBB"                       clear color
ambient "#RRGGBB" <intensity>              ambient light (one per scene)
light   "#RRGGBB" <intensity> <x> <y> <z>  directional light (any number)
cube   <name> "#color" <x> <y> <z> <motion>
sphere <name> "#color" <x> <y> <z> <motion>
torus  <name> "#color" <x> <y> <z> <motion>
cone   <name> "#color" <x> <y> <z> <motion>
plane  <name> "#color" <x> <y> <z> <motion>
```

Motion verbs: `spin`, `orbit`, `bob`, `pulse`, `none`.

## What you get

A self-contained HTML page that:

- Uses an `<script type="importmap">` to pull `three@0.160.0` from unpkg.
- Sets up a `PerspectiveCamera`, `WebGLRenderer`, `OrbitControls`
  (drag to rotate, scroll to zoom).
- Adds the declared ambient + directional lights.
- Creates each declared mesh with a `MeshStandardMaterial`.
- Animates motion verbs in a `requestAnimationFrame` loop.

Try it live: in the [playground](https://luowensheng.github.io/capy/playground/),
pick "🌐 Three.js scene" from the dropdown.
