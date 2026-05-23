---
title: Live demos
---

# Live demos

Every demo on this page is a **real Capy library + script + generated
output**. Where the output is something a browser can render (HTML,
CSS, Markdown, Mermaid), the rendered version is **embedded inline
below the code** so you can see exactly what comes out.

For non-renderable targets (Python, SQL, JSON, …) you see the
source-and-generated pair.

All 50 demos live in the [`samples/`](https://github.com/luowensheng/capy/tree/main/samples)
directory if you want to clone and run them yourself.

---

## 🕹️ Playable games — Capy is not a toy

The next two tabs are **fully playable HTML5 games** generated from
3–4 lines of Capy DSL. Click into a canvas and play — keys are
captured locally to the iframe.

=== "Breakout (4 lines of DSL)"

    **Source** (`script.capy`):

    ```
    game     "Breakout"  480 320
    paddle   80 10  7
    ball     6 4 4
    bricks   5 8 56 14 4
    ```

    Four lines. The library generates a **174-line working game**:
    paddle/ball physics, collision with spin off the paddle, 5×8
    brick wall with per-row colors and score values, lives, win/lose
    screens, restart.

    <iframe src="../assets/demos/breakout.html" width="100%" height="400" style="border: 1px solid #444; background: #0a0a14;"></iframe>

    ← / → to move · space to launch · R to restart

    [Library + source → `samples/interactive-breakout/`](https://github.com/luowensheng/capy/tree/main/samples/interactive-breakout)

=== "Snake (3 lines of DSL)"

    **Source**:

    ```
    game  "Snake"  400 400
    grid  20 20
    speed 110
    ```

    Three lines. **131-line working Snake**: arrow + WASD controls,
    anti-reverse, growing snake with per-segment gradient, food
    spawning that avoids the body, best-score saved to
    `localStorage`, pause (space), restart (R).

    <iframe src="../assets/demos/snake.html" width="100%" height="460" style="border: 1px solid #2a3; background: #0a140a;"></iframe>

    [Library + source → `samples/interactive-snake/`](https://github.com/luowensheng/capy/tree/main/samples/interactive-snake)

These two demos prove Capy can produce real interactive artifacts —
not just toy "hello world" code. The DSL is configuration; the
*library* contains the game logic, written once, reused with
different parameters per game.

---

## 🎮 Interactive HTML — see the rendered output

These are full HTML documents. The Capy source is short; the
generated HTML+CSS+JS is dropped into an `<iframe>` so you can
actually use it.

=== "Canvas game"

    **Source** (`script.capy`):

    ```
    game "Block Hopper" 480 320

    sprite player "#4dd" 220 280 40 20
    sprite enemy  "#f64" 100 100 30 30
    sprite goal   "#fd0" 420 20  20 20

    on_key "ArrowLeft"  player -4 0
    on_key "ArrowRight" player  4 0
    on_key "ArrowUp"    player  0 -4
    on_key "ArrowDown"  player  0 4

    tick enemy_bounce "sprites.enemy.x += 1; if (sprites.enemy.x > 450) sprites.enemy.x = 0;"
    ```

    **Generated HTML** (full file, ~67 lines):

    ```html
    <!doctype html>
    <html lang="en">
    <head>
      <title>Block Hopper</title>
      <style>
        body { background: #111; display: grid; place-items: center; }
        canvas { background: #222; border: 1px solid #444; }
      </style>
    </head>
    <body>
      <canvas id="c" width="480" height="320"></canvas>
      <script>
        const canvas = document.getElementById("c");
        const ctx = canvas.getContext("2d");
        const keys = {};
        window.addEventListener("keydown", e => keys[e.key] = true);
        window.addEventListener("keyup",   e => keys[e.key] = false);
        const sprites = {
          player: { x: 220, y: 280, w: 40, h: 20, color: "#4dd" },
          enemy:  { x: 100, y: 100, w: 30, h: 30, color: "#f64" },
          goal:   { x: 420, y: 20,  w: 20, h: 20, color: "#fd0" },
        };
        function update() {
          if (keys["ArrowLeft"])  sprites.player.x += -4;
          if (keys["ArrowRight"]) sprites.player.x +=  4;
          if (keys["ArrowUp"])    sprites.player.y += -4;
          if (keys["ArrowDown"])  sprites.player.y +=  4;
          sprites.enemy.x += 1; if (sprites.enemy.x > 450) sprites.enemy.x = 0;
        }
        function draw() {
          ctx.clearRect(0, 0, canvas.width, canvas.height);
          for (const s of Object.values(sprites)) {
            ctx.fillStyle = s.color;
            ctx.fillRect(s.x, s.y, s.w, s.h);
          }
        }
        function loop() { update(); draw(); requestAnimationFrame(loop); }
        loop();
      </script>
    </body>
    </html>
    ```

    **Rendered** — click the canvas, then use arrow keys:

    <iframe src="../assets/demos/canvas-game.html" width="100%" height="380" style="border:1px solid #ccc;border-radius:6px;"></iframe>

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-canvas-game)

=== "Landing page"

    **Source** (`script.capy`):

    ```
    title "Capy — DSLs in YAML"
    hero  "Define a language. Get a transpiler." "Capy is a tiny engine that turns a YAML file into a working code generator."

    feature "🌱" "Zero default grammar" "Your library is the language."
    feature "⚡" "Fast"                  "Single-binary Go. Boots in milliseconds."
    feature "🧩" "50 sample DSLs"        "From Python to Mermaid to a real x86-64 transpiler."

    cta "Get started" "/docs/getting-started"
    cta "GitHub"      "https://github.com/luowensheng/capy"
    ```

    **Generated** — a complete responsive HTML page with embedded CSS,
    hero section, features grid, and CTAs.

    **Rendered**:

    <iframe src="../assets/demos/landing-page.html" width="100%" height="500" style="border:1px solid #ccc;border-radius:6px;"></iframe>

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-landing-page)

=== "HTML email"

    **Source** (`script.capy`):

    ```
    subject "Welcome to Capy"
    preview "Your account is ready."

    heading "Welcome to Capy!"
    para    "Thanks for signing up. You're all set to start building DSLs."
    para    "Click below to read the getting-started guide."
    button  "Get started" "https://capy.dev/getting-started"
    divider
    footer  "Sent by Capy. Unsubscribe at capy.dev/unsubscribe."
    ```

    **Generated** — an HTML email with all styles inlined (the format
    that survives Gmail, Outlook, etc.).

    **Rendered**:

    <iframe src="../assets/demos/email.html" width="100%" height="500" style="border:1px solid #ccc;border-radius:6px;background:#f4f4f4;"></iframe>

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-email-html)

=== "HTML form"

    **Source** (`script.capy`):

    ```
    form "/contact"
        field name "text" "Your name"
        field email "email" "Email address"
        textarea message "Message"
    end
    ```

    **Generated**:

    ```html
    <form action="/contact" method="post">
      <label for="name">Your name</label>
      <input id="name" name="name" type="text" />
      <label for="email">Email address</label>
      <input id="email" name="email" type="email" />
      <label for="message">Message</label>
      <textarea id="message" name="message"></textarea>
      <button type="submit">Submit</button>
    </form>
    ```

    **Rendered** — try typing in the fields:

    <iframe src="../assets/demos/form.html" width="100%" height="420" style="border:1px solid #ccc;border-radius:6px;"></iframe>

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-form)

=== "Component card"

    **Source** (`script.capy`):

    ```
    component card "Welcome" {
        text "Capy makes transpilers easy."
        text "Try editing this component."
    }
    ```

    **Generated**:

    ```html
    <div id="card" class="card">
      <h3>"Welcome"</h3>
      <p>"Capy makes transpilers easy."</p>
      <p>"Try editing this component."</p>
    </div>
    ```

    **Rendered**:

    <iframe src="../assets/demos/component-card.html" width="100%" height="280" style="border:1px solid #ccc;border-radius:6px;"></iframe>

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/html-component)

=== "CSS animations"

    **Source** (`script.capy`):

    ```
    keyframe pulse
        at 0   transform = "scale(1)"
        at 50  transform = "scale(1.1)"
        at 100 transform = "scale(1)"
    end

    class ".card"
        background = "#fff"
        border_radius = "8px"
        animate slide_in "0.4s" "ease-out"
    end

    class ".badge"
        animate pulse "1.2s" "ease-in-out"
    end
    ```

    **Generated CSS** — `@keyframes` rules and animated classes.

    **Rendered** — the card slides in; the badge pulses:

    <iframe src="../assets/demos/css-animations.html" width="100%" height="280" style="border:1px solid #ccc;border-radius:6px;"></iframe>

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-css-animations)

---

## 📊 Diagrams — generated Mermaid

Capy emits Mermaid; the docs site renders it inline.

=== "Flowchart"

    **Source** (`script.capy`):

    ```
    flowchart LR
        node a "Source"
        node b "Lexer"
        node c "Parser"
        node d "Evaluator"
        node e "Output"
        a -> b
        b -> c
        c -> d : "match + render"
        d -> e
    end
    ```

    **Generated Mermaid** — rendered live:

    ```mermaid
    flowchart LR
      a[Source]
      b[Lexer]
      c[Parser]
      d[Evaluator]
      e[Output]
      a --> b
      b --> c
      c -->|match + render| d
      d --> e
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-mermaid)

=== "State diagram"

    **Source** (`script.capy`):

    ```
    machine Order
        state Pending
        state Paid
        state Shipped
        state Delivered

        Pending -> Paid on "payment"
        Paid -> Shipped on "fulfill"
        Shipped -> Delivered on "arrival"

        final Delivered
    end
    ```

    **Generated state diagram** — rendered live:

    ```mermaid
    stateDiagram-v2
      [*] --> Order
      state Pending
      state Paid
      state Shipped
      state Delivered
      Pending --> Paid : payment
      Paid --> Shipped : fulfill
      Shipped --> Delivered : arrival
      Delivered --> [*]
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-statemachine)

---

## 📝 Rendered Markdown — generated and shown inline

The Capy output IS Markdown, so MkDocs renders it directly on this
page. What you see below is the actual generated text, formatted.

=== "Todo list"

    **Source** (`script.capy`):

    ```
    section "Today"
    todo "Write the launch blog post"
    done  "Tag v0.1.0"
    todo  "Test install script on Linux"

    section "This week"
    todo "Publish VS Code extension"
    done "Move .codestyle to docs"
    ```

    **Generated and rendered**:

    ## Today

    - [ ] Write the launch blog post
    - [x] Tag v0.1.0
    - [ ] Test install script on Linux

    ## This week

    - [ ] Publish VS Code extension
    - [x] Move .codestyle to docs

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-markdown-todo)

=== "Invoice"

    **Source** (`script.capy`):

    ```
    number "INV-2026-001"
    date   "2026-05-23"
    bill_to "Acme Corp"

    item "Consulting hours"     8 "$120.00"
    item "Capy enterprise plan" 1 "$2000.00"
    item "Onboarding workshop"  2 "$500.00"
    ```

    **Generated and rendered**:

    ## Invoice INV-2026-001

    **To:** Acme Corp
    **Date:** 2026-05-23

    | Item                  | Qty | Unit price |
    |-----------------------|----:|-----------:|
    | Consulting hours      |   8 | $120.00    |
    | Capy enterprise plan  |   1 | $2000.00   |
    | Onboarding workshop   |   2 | $500.00    |

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-invoice)

=== "Changelog"

    **Source** (`script.capy`):

    ```
    version "0.2.0" "2026-06-15"
        added   "Configurable surface syntax"
        added   "else arm on inner if"
        fixed   "Indentation tokenisation edge case"
    end

    version "0.1.0" "2026-05-23"
        added   "Initial public release"
        added   "Type system with pattern + options"
        added   "Two block modes"
    end
    ```

    **Generated and rendered**:

    ## [0.2.0] — 2026-06-15

    - Added: Configurable surface syntax
    - Added: else arm on inner if
    - Fixed: Indentation tokenisation edge case

    ## [0.1.0] — 2026-05-23

    - Added: Initial public release
    - Added: Type system with pattern + options
    - Added: Two block modes

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-changelog)

---

## 💻 Code generation — source + generated, side by side

These targets are runnable code rather than rendered output. Save
the generated file, run it.

=== "Python"

    **Source**:

    ```
    import json
    import os
    say "hello, world"

    if x
        say "x is set"
    end

    loop n in [1, 2, 3]
        say n
    end
    ```

    **Generated `out.py`**:

    ```python
    import json
    import os
    print("hello, world")
    if x:
        print("x is set")

    for n in [1, 2, 3]:
        print(n)
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-py)

=== "PostgreSQL"

    **Source**:

    ```
    table users
        pk     id
        unique email "varchar(255)"
        col    name  "varchar(255) NOT NULL"
    end

    table posts
        pk     id
        fk     author_id -> users
        col    title "varchar(255) NOT NULL"
    end

    index users email
    index posts author_id
    ```

    **Generated `schema.sql`**:

    ```sql
    CREATE TABLE users (
      id bigserial PRIMARY KEY,
      email varchar(255) UNIQUE NOT NULL,
      name varchar(255) NOT NULL
    );
    CREATE TABLE posts (
      id bigserial PRIMARY KEY,
      author_id bigint NOT NULL REFERENCES users(id),
      title varchar(255) NOT NULL
    );

    CREATE INDEX ix_users_email ON users(email);
    CREATE INDEX ix_posts_author_id ON posts(author_id);
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-postgres-schema)

=== "Express server"

    **Source**:

    ```
    port 8080

    use "morgan('combined')"
    get  "/health" "res.json({ok: true})"
    post "/users"  "const u = req.body; res.status(201).json({id: 42, ...u})"
    ```

    **Generated `server.js`**:

    ```javascript
    const express = require("express");
    const app = express();

    app.use(express.json());
    app.use(morgan('combined'));

    app.get("/health", (req, res) => {
      res.json({ok: true})
    });

    app.post("/users", (req, res) => {
      const u = req.body; res.status(201).json({id: 42, ...u})
    });

    app.listen(8080, () => { console.log("listening on", 8080); });
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-express-server)

=== "Terraform"

    **Source**:

    ```
    provider "aws" "us-east-1"

    resource "aws_instance" web
        ami = "ami-0c55b159cbfafe1f0"
        instance_type = "t3.micro"
        tag "Name" "capy-web"
    end
    ```

    **Generated `main.tf`**:

    ```hcl
    provider "aws" {
      region = "us-east-1"
    }

    resource "aws_instance" "web" {
      ami = "ami-0c55b159cbfafe1f0"
      instance_type = "t3.micro"
      tags = { "Name" = "capy-web" }
    }
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-terraform)

=== "Kubernetes"

    **Source**:

    ```
    deployment capy_api
    image    "ghcr.io/luowensheng/capy:0.1.0"
    replicas 3
    port     8080
    ```

    **Generated `deployment.yaml`**:

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: capy_api
    spec:
      replicas: 3
      template:
        spec:
          containers:
            - name: capy_api
              image: ghcr.io/luowensheng/capy:0.1.0
              ports:
                - containerPort: 8080
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-kubernetes)

=== "Slack Block Kit"

    **Source**:

    ```
    header  "📦 Build complete"
    section "Branch *main* built in *4m 12s* and is ready to deploy."
    divider
    section "Tests: 124/124 passing"
    button  "View build" "https://ci.example.com/build/1234"
    ```

    **Generated JSON** (POST to a Slack webhook):

    ```json
    {
      "blocks": [
        { "type": "header", "text": { "type": "plain_text", "text": "📦 Build complete" } },
        { "type": "section", "text": { "type": "mrkdwn", "text": "Branch *main* built in *4m 12s* and is ready to deploy." } },
        { "type": "divider" },
        { "type": "section", "text": { "type": "mrkdwn", "text": "Tests: 124/124 passing" } },
        { "type": "actions", "elements": [{ "type": "button", "text": { "type": "plain_text", "text": "View build" }, "url": "https://ci.example.com/build/1234" }] }
      ]
    }
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/transpile-slack-blocks)

=== "Assembly (x86-64)"

    **Source**:

    ```
    program "sum-demo"
        var x = 5
        var y = 7
        add x y
        store result
        exit 0
    end
    ```

    **Generated `demo.asm`** (assembles with `nasm -felf64 demo.asm -o demo.o && ld demo.o -o demo`):

    ```asm
    section .data
        x: dq 0
        y: dq 0

    section .text
        global _start

    _start:
        mov rax, 5
        mov [x], rax
        mov rax, 7
        mov [y], rax
        mov rax, [x]
        add rax, [y]
        mov [result], rax
        mov rdi, 0
        mov rax, 60
        syscall
    ```

    [Full sample →](https://github.com/luowensheng/capy/tree/main/samples/assembly)

---

## 🏗️ One scene → five 3D / game tools

The same 8-line scene description compiled to **runnable scripts for
Blender, SketchUp, Rhino, Unity, and Unreal**. No host-API
vocabulary in the source — just primitives.

=== "Source (`script.capy`)"

    ```
    scene "Studio"

    cube   red    0 0 0   2
    sphere blue   4 0 0   1
    plane  gray   0 0 0   10

    light   5 5 5
    camera  7 7 3
    ```

=== "→ Blender (Python `bpy`)"

    ```python
    import bpy

    _COLORS = {
        "red":   (1.0, 0.1, 0.1, 1.0),
        "blue":  (0.1, 0.3, 1.0, 1.0),
        "green": (0.1, 0.8, 0.2, 1.0),
        "gray":  (0.6, 0.6, 0.6, 1.0),
    }

    def _paint(obj, color):
        mat = bpy.data.materials.new(name=color)
        mat.diffuse_color = _COLORS.get(color, (1, 1, 1, 1))
        obj.data.materials.append(mat)

    bpy.context.scene.name = "Studio"
    # cube (red)
    bpy.ops.mesh.primitive_cube_add(location=(0, 0, 0), size=2)
    _paint(bpy.context.active_object, "red")
    # sphere (blue)
    bpy.ops.mesh.primitive_uv_sphere_add(location=(4, 0, 0), radius=1)
    _paint(bpy.context.active_object, "blue")
    # plane (gray)
    bpy.ops.mesh.primitive_plane_add(location=(0, 0, 0), size=10)
    _paint(bpy.context.active_object, "gray")
    bpy.ops.object.light_add(type='POINT', location=(5, 5, 5))
    bpy.ops.object.camera_add(location=(7, 7, 3))
    ```

    Paste into Blender's Scripting tab. Done.

=== "→ SketchUp (Ruby)"

    ```ruby
    # Generated by Capy — paste into the SketchUp Ruby Console.
    model = Sketchup.active_model
    ent = model.active_entities

    # … helpers _add_box / _add_sphere / _add_plane / _paint elided …

    model.name = "Studio"
    # cube (red)
    _add_box(ent, 0, 0, 0, 2, "red")
    # sphere (blue)
    _add_sphere(ent, 4, 0, 0, 1, "blue")
    # plane (gray)
    _add_plane(ent, 0, 0, 0, 10, "gray")
    model.active_view.camera = Sketchup::Camera.new([7, 7, 3], [0, 0, 0], [0, 0, 1])
    ```

=== "→ Rhino (C# / RhinoCommon)"

    ```csharp
    using System.Drawing;
    using Rhino;
    using Rhino.DocObjects;
    using Rhino.Geometry;

    public static class CapyScene
    {
        public static void Build(RhinoDoc doc)
        {
            // cube (red)
            {
                var c = new Point3d(0, 0, 0);
                double s = 2 / 2.0;
                var box = new Box(new Plane(c, Vector3d.ZAxis),
                                  new Interval(-s, s), new Interval(-s, s), new Interval(-s, s));
                doc.Objects.AddBox(box, Attr("red"));
            }
            // sphere (blue)
            {
                var sph = new Sphere(new Point3d(4, 0, 0), 1);
                doc.Objects.AddSphere(sph, Attr("blue"));
            }
            // … plane / light / camera …
        }
    }
    ```

    Drop into a Grasshopper C# scripting component, or compile as a
    Rhino plugin command.

=== "→ Unity (C# MonoBehaviour)"

    ```csharp
    using UnityEngine;
    using System.Collections.Generic;

    public class CapyScene : MonoBehaviour
    {
        // … COLORS table + Spawn helper elided …

        void Start()
        {
            gameObject.name = "Studio";
            // cube (red)
            Spawn(PrimitiveType.Cube, new Vector3(0, 0, 0), Vector3.one * 2, "red");
            // sphere (blue)
            Spawn(PrimitiveType.Sphere, new Vector3(4, 0, 0), Vector3.one * 1 * 2f, "blue");
            // plane (gray)
            Spawn(PrimitiveType.Plane, new Vector3(0, 0, 0), Vector3.one * 10 * 0.1f, "gray");
            // point light
            {
                var go = new GameObject("PointLight");
                go.transform.position = new Vector3(5, 5, 5);
                go.AddComponent<Light>().type = LightType.Point;
            }
            // camera
            {
                var go = new GameObject("Camera");
                go.transform.position = new Vector3(7, 7, 3);
                go.transform.LookAt(Vector3.zero);
                go.AddComponent<Camera>();
            }
        }
    }
    ```

    Drop into `Assets/Scripts/CapyScene.cs` and attach to an empty
    GameObject.

=== "→ Unreal (Python editor scripting)"

    ```python
    import unreal

    # … _PRIMITIVES table + _spawn_primitive helper elided …

    unreal.log("Studio")
    # cube (red)
    _spawn_primitive("Cube", unreal.Vector(0*100, 0*100, 0*100), 2, "red")
    # sphere (blue)
    _spawn_primitive("Sphere", unreal.Vector(4*100, 0*100, 0*100), 1*2, "blue")
    # plane (gray)
    _spawn_primitive("Plane", unreal.Vector(0*100, 0*100, 0*100), 10, "gray")
    unreal.EditorLevelLibrary.spawn_actor_from_class(
        unreal.PointLight, unreal.Vector(5*100, 5*100, 5*100))
    unreal.EditorLevelLibrary.spawn_actor_from_class(
        unreal.CameraActor, unreal.Vector(7*100, 7*100, 3*100))
    ```

    Note the `*100` — Capy's library handles the meters-to-Unreal-cm
    conversion so the source stays in human units.

**Why this matters for 3D / game pipelines.** The same procedural
building gets rewritten in five different host APIs — and they
*drift*. An algorithm change means hand-editing five scripts. LLMs
hallucinate API names because each tool's vocabulary is slightly
different. With Capy you write the scene **once**; each library
encodes one tool's quirks. Add Maya MEL, Houdini Python, Three.js,
glTF — write a 50-line library, never touch the source.

[Full sample → `samples/3d-tools-demo/`](https://github.com/luowensheng/capy/tree/main/samples/3d-tools-demo)

---

## 🌍 One source → five programming languages

The same 10-line `script.capy` compiled to **five different
programming languages** by five different libraries. Each output is a
real, runnable program that prints `12`.

=== "Source (`script.capy`)"

    ```
    fn add(a, b)
        return a + b
    end

    main
        let x = 5
        let y = 7
        let z = add(x, y)
        print z
    end
    ```

    Ten lines. Defines a function, calls it, prints the result. The
    grammar (`fn`, `return`, `main`, `let`, `print`) is defined by the
    libraries — not by Capy itself.

=== "→ Python"

    ```python
    def add(a, b):
        return a + b


    if __name__ == "__main__":
        x = 5
        y = 7
        z = add(x, y)
        print(z)
    ```

=== "→ JavaScript"

    ```javascript
    function add(a, b) {
      return a + b;
    }


    (function main() {
      const x = 5;
      const y = 7;
      const z = add(x, y);
      console.log(z);
    })();
    ```

=== "→ Go"

    ```go
    package main

    import "fmt"


    func add(a int, b int) int {
        return a + b
    }


    func main() {
        x := 5
        y := 7
        z := add(x, y)
        fmt.Println(z)
    }
    ```

=== "→ Rust"

    ```rust
    fn add(a: i32, b: i32) -> i32 {
        return a + b;
    }


    fn main() {
        let x: i32 = 5;
        let y: i32 = 7;
        let z: i32 = add(x, y);
        println!("{}", z);
    }
    ```

=== "→ C"

    ```c
    #include <stdio.h>

    int add(int a, int b) {
        return a + b;
    }


    int main(void) {
        int x = 5;
        int y = 7;
        int z = add(x, y);
        printf("%d\n", z);
        return 0;
    }
    ```

**Why this matters.** Maintaining the "same logic in N languages"
problem is real: client SDKs that drift, an algorithm needed in
Python *and* C++, a validator that runs in the browser *and* on the
server. With Capy you write the logic **once**; adding a sixth target
is a ~50-line library file. The next time you change the algorithm,
all five (or six, or ten) outputs regenerate.

[Full sample → `samples/multi-language-demo/`](https://github.com/luowensheng/capy/tree/main/samples/multi-language-demo)

### Bonus: the library itself, in Capy syntax

Every library in this demo ships in **two forms**. Pick whichever you
prefer — they produce byte-identical output:

=== "`lib_c.yaml` (YAML)"

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
      # ...
    ```

=== "`lib_c.capy` (Capy-native)"

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
    ```

Capy supports **both formats** for libraries — the loader dispatches
on file extension. See [`.capy` libraries](capy-libraries.md) for the
full grammar and trade-offs.

---

## 🔀 Same source, three targets

The clearest demonstration of "the library is the grammar". One
input file, three libraries, three completely different artifacts.

=== "Source"

    The same `script.capy` for all three:

    ```
    user alice 30 active
    user bob   25 inactive
    user carol 42 active
    ```

=== "→ SQL"

    Running `capy run lib_sql.yaml script.capy` produces SQL inserts:

    ```sql
    INSERT INTO users (name, age, status) VALUES ('alice', 30, 'active');
    INSERT INTO users (name, age, status) VALUES ('bob', 25, 'inactive');
    INSERT INTO users (name, age, status) VALUES ('carol', 42, 'active');
    ```

=== "→ JSON"

    Running `capy run lib_json.yaml script.capy` produces JSON:

    ```json
    {
      "users": [
        { "name": "alice", "age": 30, "status": "active" },
        { "name": "bob",   "age": 25, "status": "inactive" },
        { "name": "carol", "age": 42, "status": "active" }
      ]
    }
    ```

=== "→ Markdown"

    Running `capy run lib_md.yaml script.capy` produces Markdown:

    | Name  | Age | Status   |
    |-------|----:|----------|
    | alice | 30  | active   |
    | bob   | 25  | inactive |
    | carol | 42  | active   |

The libraries are 8–15 lines each. Add a fourth target (CSV, YAML,
HTML table, …) by writing a fourth library — never touch the source.

[Full sample →](https://github.com/luowensheng/capy/tree/main/samples/multi-target-demo)

---

## What's not shown here

The 50 sample demos in the repo include more code-gen targets that
don't fit on a single doc page:

- **Backend frameworks**: Flask, FastAPI, Cobra CLI
- **Schemas**: Prisma, Zod, XState v5, Protobuf, GraphQL
- **Config**: nginx, systemd, GitHub Actions, cron, .env, Dockerfile,
  Makefile, OpenAPI, Prometheus alerts, Chrome MV3 manifest
- **Other**: CSV, Markdown CV, Markdown blog with YAML front matter,
  Markdown API reference

[Browse all 50 demos →](https://github.com/luowensheng/capy/tree/main/samples)

---

## How to run any of these locally

```sh
go install github.com/luowensheng/capy/cmd/capy@latest
git clone https://github.com/luowensheng/capy
cd capy
capy run samples/transpile-canvas-game/lib.yaml samples/transpile-canvas-game/script.capy > game.html
open game.html
```

Or just look at the lib.yaml — it's the entire grammar in 30–60 lines
of YAML.
