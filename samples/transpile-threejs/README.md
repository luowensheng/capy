# transpile-threejs — interactive three.js from a tiny DSL

~25 declarative lines → a complete, runnable, **interactive** HTML
page that pulls [three.js](https://threejs.org) from a CDN. The
source language has **no JavaScript** — it declares meshes,
abstract motions, and event bindings; the library is what knows
three.js.

Swap the library for one that targets Babylon.js, A-Frame, or raw
WebGL and your scene script doesn't change.

## Run

```sh
capy run samples/transpile-threejs/lib.capy samples/transpile-threejs/script.capy > scene.html
open scene.html      # or `python3 -m http.server` and visit it
```

## Source language

### Scene + meshes

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

### Events + actions

```text
click  <target> <action>          on raycaster hit
hover  <target> <action>          on pointer hover
key    "K" <target> <action>      on keydown for key "K"
button "Label" <target> <action>  add an HTML HUD button
```

`<target>` is a mesh name, or the keyword `any` to apply to every mesh.

Actions:

| Verb | Effect |
|---|---|
| `randomize_color` | Pick a new random HSL colour. |
| `toggle_motion`   | Pause / resume per-mesh animation. |
| `cycle_motion`    | Cycle spin → orbit → bob → pulse → none. |
| `wireframe`       | Toggle wireframe material. |
| `explode`         | Briefly scale up + fade then return. |
| `reset`           | Return this mesh to its original transform + colour. |
| `reset_all`       | Reset every mesh. |
| `disco`           | Toggle global colour-cycling mode. |

## What you get

A self-contained HTML page that:

- Imports `three@0.160.0` via `<script type="importmap">` from unpkg.
- Sets up a `PerspectiveCamera`, `WebGLRenderer`, `OrbitControls`
  (drag to rotate, scroll to zoom).
- Adds the declared ambient + directional lights.
- Creates each declared mesh with a `MeshStandardMaterial`.
- Wires a raycaster so mouse clicks hit the right object and
  dispatch its declared action.
- Listens for `keydown` and routes each declared key to its action.
- Renders an HTML HUD with one button per declared `button` line.
- Animates motion verbs and `explode` decay in a `requestAnimationFrame`
  loop.

Try it live: in the [playground](https://luowensheng.github.io/capy/playground/),
pick "🌐 Three.js scene (interactive)" from the dropdown.
