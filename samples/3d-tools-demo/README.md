# 3d-tools-demo

**One scene description → scripts for five different 3D / game tools.**

The 8-line `script.capy` describes a tiny scene: a red cube, a blue
sphere, a gray plane, a point light, a camera. Five libraries compile
that one source into runnable scripts for:

| Target tool         | Output script type                       | Library            |
|---------------------|------------------------------------------|--------------------|
| **Blender**         | Python (`bpy`)                           | `lib_blender.capy` |
| **SketchUp**        | Ruby (SketchUp API)                      | `lib_sketchup.capy`|
| **Rhino 3D**        | C# (RhinoCommon)                         | `lib_rhino.capy`   |
| **Unity**           | C# MonoBehaviour                         | `lib_unity.capy`   |
| **Unreal Engine**   | Python (Editor scripting)                | `lib_unreal.capy`  |

## The shared source

```
scene "Studio"

cube   red    0 0 0   2
sphere blue   4 0 0   1
plane  gray   0 0 0   10

light   5 5 5
camera  7 7 3
```

Eight lines. Six primitives. No mention of `bpy`, `Sketchup::`,
`RhinoCommon`, `MonoBehaviour`, or `unreal.EditorLevelLibrary`. The
target-specific machinery lives in each `lib_X.capy`.

## Generate any output

```sh
# Blender Python script (paste into Scripting tab)
../../capy run lib_blender.capy script.capy > scene.py

# SketchUp Ruby script (paste into the Ruby Console)
../../capy run lib_sketchup.capy script.capy > scene.rb

# Rhino C# (drop into a Grasshopper C# component)
../../capy run lib_rhino.capy script.capy > CapyScene.cs

# Unity C# MonoBehaviour (Assets/Scripts/CapyScene.cs)
../../capy run lib_unity.capy script.capy > CapyScene.cs

# Unreal Editor Python (Window → Developer Tools → Python)
../../capy run lib_unreal.capy script.capy > scene.py
```

## Why this matters

3D content pipelines suffer from a tooling fragmentation problem:

- The same procedural building exists as a Python script in Blender,
  a Ruby snippet in SketchUp, a Grasshopper C# block in Rhino, an
  editor command in Unreal, and a runtime script in Unity. They drift.
- LLMs generating these scripts hallucinate API names because each
  tool has a slightly-different vocabulary for the same primitive.
- A small change to the algorithm (different proportions, an extra
  prop) means hand-editing five files.

With Capy you describe the scene **once** in your own domain
vocabulary. Each library encodes the API quirks of one target. Add a
sixth target (Maya MEL? Houdini Python? Three.js? glTF JSON?) by
writing a new ~50-line library — never touch the source.

The libraries also act as a **specification**: a reviewer can glance
at `lib_unity.capy` and see exactly which Unity API calls you'll ever
generate. No surprise scripting.

## Add another target

```
function cube
    arg literal "cube"
    arg capture color ident
    arg capture x any
    arg capture y any
    arg capture z any
    arg capture size any
    template:
        # whatever YOUR_TOOL's API for a cube looks like
end
```

Repeat for sphere/plane/light/camera/scene. Done. The next time the
scene description changes, `your_tool_output` regenerates.

## Caveats

These are demonstration libraries — they cover one primitive each
and use SketchUp/Unreal coordinate conventions naively. A production
library would add: rotations, hierarchies, materials with textures,
instancing, animation, and per-tool unit handling (Unreal uses
centimeters; Blender uses meters; etc.). All of that is "more
patterns in the library," not engine work.
