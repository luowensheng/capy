// playground-bundle reads a curated list of samples from samples/ and
// writes a single JSON file the browser playground can fetch.
//
// Usage: go run ./cmd/playground-bundle > docs/assets/playground/samples.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CURATED is the ordered list of samples surfaced in the playground UI.
// The Category field groups them in the dropdown. When ScriptFile is
// empty the bundler reads samples/<ID>/script.capy; set it to use a
// different script filename in the same sample directory (e.g. for
// progressive-abstraction which has script_{minimal,medium,full}.capy
// sharing one lib.capy).
var CURATED = []struct {
	ID, Category, Title, Description, Hint, ScriptFile, LibraryFile string
}{
	// ─── ✨ New features (round-1 & round-2) ────────────────────────
	{"html-xml-parser", "Features", "✨ Parse HTML / XML (one function)",
		"A single generic `element` function parses ANY well-formed `<tag …>…</tag>` — HTML or XML. The tag name is captured, attributes match via a function-typed repetition (`attribute*`), and the block closes on the capture-bound sequence `</NAME>`. Mismatched nesting (`<div><p></div>`) is a hard parse error — real structural parsing of the tag tree.",
		"Add your own nested tags or attributes; try breaking the nesting to see the mismatch error.", "", ""},
	{"bbcode-parser", "Features", "✨ BBCode tags (bracket close-seq)",
		"`block_close_seq` isn't angle-bracket-specific: the same primitive parses BBCode-style `[b]…[/b]` / `[quote]…[/quote]`. Each tag is its own function with a LITERAL close sequence and its own HTML template, plus a `[url=\"…\"]` attribute. Mismatched nesting is a hard parse error.",
		"Swap the tag delimiters or templates; try closing `[b]` with `[/quote]` to see the error.", "", ""},
	{"markdown-from-tags", "Features", "✨ Tags → Markdown (target-agnostic)",
		"The SAME `<h1>…</h1>` / `<p>…</p>` / `<b>…</b>` markup that `html-xml-parser` turns into HTML is rewritten into Markdown here — only the per-tag templates differ. Shows the sequence-closer primitive is target-agnostic: it parses the tag tree; the template decides the output language.",
		"Change a template (e.g. make `<b>` emit `__…__`); add an `<h2>` section.", "", ""},
	{"signature-parser", "Features", "✨ Param lists (sep + join)",
		"A function-signature DSL → typed declaration. The parameter list uses a function-typed repetition with BOTH separators: `param* sep \",\" join \", \"` — `sep` is the INPUT separator consumed while parsing, `join` is the OUTPUT separator inserted between rendered params. Independent, so comma-input renders as comma-space output.",
		"Add a parameter to a `func` line; try a zero-arg `func now() -> int`.", "", ""},
	{"library-keywords", "Features", "✨ Authoring keywords (function/arg/block)",
		"The canonical `if … end` block in ~15 lines: `function` defines a source keyword, `arg literal`/`arg capture` match the line, `block_closer end` opens an indented body, and `write` + `${indent 4 body}` emit it. The building blocks every library is made of — see the Library keyword cookbook in the docs.",
		"Add another statement inside the `if … end`; try nesting a second `if`.", "", ""},
	{"builtin-functions", "Features", "✨ Built-in helper functions",
		"A live tour of Capy's built-in template helpers — the only functions Capy itself defines. Case/identifier (`pascalCase`, `snakeCase`, `dasherize`), string/escaping (`decoded`, `escapeHtml`, `toQuoted`, `asString`), numeric (`add`, `percent`, `stars`), and layout (`indent`). See the Built-in function cookbook in the docs for all 26.",
		"Change the `names`/`text`/`calc` inputs and watch each helper's output update.", "", ""},
	{"template-sugar", "Features", "✨ template … end sugar",
		"`template … end` replaces multi-line backtick `write` literals — same `${…}` interpolation, no backtick bookkeeping. Stacks cards from plain `card`/`p` lines.",
		"Add another `card \"…\" … end` block; edit the paragraphs inside.", "", ""},
	{"optional-args", "Features", "✨ Optional args with defaults",
		"A trailing capture can declare a `default`, so one `button` function serves `button \"Save\"`, `button \"Delete\" \"danger\"`, and `button \"Submit\" \"primary\" \"submit\"`. Replaces whole families of near-duplicate components.",
		"Drop the trailing words from a `button` line — the defaults fill in.", "", ""},
	{"line-mapping", "Features", "✨ Source↔output mapping (${line}/${col})",
		"Every statement exposes its source position via the `${line}` / `${col}` render locals. The library stamps them as `data-capy-line` so an editor can scroll-sync or place inline errors — no source mutation.",
		"Add or remove a `p` line; watch the `data-capy-line` numbers track the source.", "", ""},
	{"string-decoded", "Features", "✨ decoded — escapes & quotes round-trip",
		"`${decoded}` resolves `\\n` / `\\t` and embedded `\\\"` quotes in captured prose, even when the text itself contains quotes. `${escapeHtml}` then neutralises markup.",
		"Add a `p` with your own quotes, newlines (`\\n`) and `<tags>`.", "", ""},
	{"verbatim-pre", "Features", "✨ block_verbatim — raw code blocks",
		"`pre LANG … end` captures its body as raw bytes (no re-parsing), HTML-escapes it, and emits `<pre><code>`. Blank lines and `#`-prefixed lines survive byte-for-byte.",
		"Paste a snippet in any language inside a `pre … end` block.", "", ""},
	{"backtick-codespan", "Features", "✨ Escapable backticks in captures",
		"A backtick-delimited capture can contain a Markdown code span: `\\`code\\`` stays literal instead of closing the capture.",
		"Add a `markdown` line with your own `\\`inline code\\``.", "", ""},
	{"utf8-prose", "Features", "✨ UTF-8 prose (no quoting)",
		"Em-dashes, accented Latin, CJK and emoji tokenise as ordinary idents — bare prose Just Works without wrapping in quotes.",
		"Type a sentence in any language; it becomes a paragraph.", "", ""},
	{"multiline-strings", "Features", "✨ Multi-line backtick captures",
		"Backtick string captures span newlines in user scripts, so a paragraph can wrap across several source lines — blank lines preserved.",
		"Edit a `p \\`…\\`` block to span more lines.", "", ""},
	{"inline-markdown", "Features", "✨ Group types — inline syntax",
		"Group types (`group_open`/`group_close`) let captures consume delimited spans: `link [text](url)`, `bold **text**`, `strike ~~text~~` map cleanly onto HTML.",
		"Add a `link [label](https://…)` or `bold **word**` line.", "", ""},
	{"feature-faq", "Features", "✨ FAQ (template + optional flag)",
		"An FAQ DSL: `q \"…\" open` toggles `<details open>` via an optional arg (default \"closed\"); answers are decoded + escaped.",
		"Drop or add the `open` flag on a `q` line; edit the answers.", "", ""},
	{"feature-pricing", "Features", "✨ Pricing tiers (optional period/CTA)",
		"`tier \"Pro\" \"$29\"` works; add `\"/yr\" \"Contact sales\"` to override the period and button. One function, no duplicate variants.",
		"Add a `feature` line or a fourth `tier`.", "", ""},
	{"feature-callouts", "Features", "✨ Callouts (note/tip/warning)",
		"Admonition blocks where the style is an optional trailing word (default \"note\"). Body markup is escaped so `<script>` is inert.",
		"Change `tip`/`warning`, or omit it for the default note style.", "", ""},
	{"feature-svg-badge", "Features", "✨ SVG badges (verbatim body)",
		"`badge \"label\" … end` captures raw inline SVG byte-for-byte — angle brackets, attributes and line breaks untouched, zero escaping.",
		"Edit the SVG colours or text inside a `badge … end` block.", "", ""},
	{"feature-menu-i18n", "Features", "✨ Multilingual menu (UTF-8 + tail)",
		"A restaurant menu in French, Japanese & Chinese. `dish` keeps the name as a string and slurps the free-text description with a `tail` capture.",
		"Add a `dish \"Name\" \"price\" your description here` line.", "", ""},
	{"feature-changelog", "Features", "✨ Changelog (line + verbatim + optional)",
		"Release notes mixing `${line}` mapping, optional `kind` (added/fixed/removed), and a verbatim `upgrade` code block.",
		"Add a `change \"…\" added` line or a new `release … end`.", "", ""},
	{"feature-social-card", "Features", "✨ Social preview cards",
		"Open-Graph-style cards via `template … end`, with an optional `theme` arg (default \"light\") and safe escaping of arbitrary punctuation.",
		"Add `dark` after a card title, or a new `tag`.", "", ""},
	{"feature-stepper", "Features", "✨ Onboarding stepper (${line})",
		"A how-to stepper; each `step` stamps its source line for scroll-sync, with an optional `done` status flag.",
		"Reorder steps or toggle the `done` flag.", "", ""},
	{"feature-glossary", "Features", "✨ Glossary (multi-line + code spans)",
		"Definition list where each term's body is a multi-line backtick capture containing `\\`code\\`` spans that round-trip.",
		"Add a `term \"Name\" \\`multi-line body\\`` entry.", "", ""},
	{"feature-quiz", "Features", "✨ Quiz (optional correct flag)",
		"Multiple-choice quiz: `choice \"…\" correct` marks the answer via an optional flag (default \"wrong\"); questions stamp their source line.",
		"Add a `choice` or a whole new `question … end`.", "", ""},
	{"mcp-widgets", "Features", "✨ MCP widgets (everything at once)",
		"Weather / chart / product-card widgets that exercise nested blocks, templates, decoded + escapeHtml, and context accumulation together.",
		"Edit the forecast days, bar values, or product details.", "", ""},
	{"math-plots", "Features", "✨ Math plots (template + canvas)",
		"`plot \"EXPR\" … end` → a self-contained `<canvas>` + `drawPlot()` call. Uses `template … end`, `${decoded}`, `${escapeHtml}` and context accumulation.",
		"Change an expression, domain, color, or sample count.", "", ""},

	// ─── Indexed reads (lists & maps by index/key) ──────────────────
	{"word-frequency", "Indexed reads", "🔢 Word frequency (map increment)",
		"`see word` does `set context.counts[w] (add context.counts[w] 1)` — a map element read, incremented, and written back (nil counts as 0). `bar word` then reads `${context.counts[w]}` to draw a star bar. The whole counter is one indexed read/write idiom.",
		"Add more `see <word>` lines; the bars grow.", "", ""},
	{"memo-fib", "Indexed reads", "🔢 Memoised Fibonacci (computed index)",
		"`memo [ 0 1 ]` seeds a list; `fib n` appends and reads back two earlier cells with computed indices `${context.memo[(sub n 1)]}` + `${context.memo[(sub n 2)]}`. Arithmetic inside the `[…]` resolves at read time.",
		"Ask for more `fib` values; each reuses the memo table.", "", ""},
	{"stack-top", "Indexed reads", "🔢 Stack peek (negative index)",
		"`push v` appends; `top` reads `${context.stack[-1]}` and `${context.stack[-2]}` — negative indices count from the end, matching write-side semantics.",
		"Push or pop values and re-peek the top.", "", ""},
	{"enum-lookup", "Indexed reads", "🔢 Enum lookup (list by index)",
		"`days`/`months` are fixed lists; `event d m …` reads `${context.days[d]}` / `${context.months[m]}` to resolve a weekday and month name by numeric index — a classic lookup table.",
		"Change the day/month numbers or add an `event` line.", "", ""},
	{"leaderboard", "Indexed reads", "🔢 Leaderboard (positional reads)",
		"`rank name` builds a list; `podium` reads fixed positions `[0]`/`[1]`/`[2]` for gold/silver/bronze and `[-1]` for last place.",
		"Add or reorder `rank` lines; the podium tracks positions.", "", ""},
	{"color-palette", "Indexed reads", "🔢 Color palette (map by name)",
		"`def name color` records `context.palette[name]`; `swatch name` reads `${context.palette[name]}` by key to emit a colored chip. One `swatch` function serves any color the palette defines.",
		"Add a `def <name> <css-color>` then a matching `swatch`.", "", ""},

	// ─── Everyday / no-code-required ────────────────────────────────
	{"recipe-card", "Everyday", "🍋 Recipe card",
		"Write a recipe in plain words; get a printable HTML recipe card.",
		"Try editing the title, ingredients, or steps.", "", ""},
	{"event-invite", "Everyday", "🎉 Party invitation",
		"Declare a party; get a pastel HTML invitation card.",
		"Try changing the host, location, or RSVP date.", "", ""},
	{"weekly-meal-plan", "Everyday", "📅 Weekly meal plan",
		"Seven dinners + notes → printable HTML grid for the fridge.",
		"Swap meals or add notes.", "", ""},
	{"reading-log", "Everyday", "📚 Reading log (for kids)",
		"A kid's reading list → bright HTML certificate with progress bar.",
		"Add more `book` lines; the progress bar updates.", "", ""},
	{"trip-itinerary", "Everyday", "✈️ Trip itinerary",
		"Day-by-day travel plan → polished HTML itinerary card.",
		"Edit destinations, add activities, change the budget.", "", ""},
	{"transpile-resume", "Everyday", "📄 Resume",
		"A resume DSL → a clean printable resume.",
		"Edit experience and skills lines.", "", ""},
	{"transpile-invoice", "Everyday", "🧾 Invoice",
		"Declare an invoice; get a polished HTML invoice you can print.",
		"Edit the line items and rates.", "", ""},
	{"transpile-markdown-todo", "Everyday", "✅ To-do list",
		"A tiny todo DSL → Markdown checklist.",
		"Add tasks, mark them done.", "", ""},
	// ─── Interactive games ──────────────────────────────────────────
	{"interactive-breakout", "Games", "🕹️ Breakout (playable)",
		"Event-driven Breakout: entities + key bindings + event handlers → 226-line HTML5 game.",
		"Change `lives 3` to `lives 5`, or rebind keys.", "", ""},
	{"interactive-snake", "Games", "🐍 Snake (playable)",
		"Event-driven Snake: key bindings + event handlers → 180-line HTML5 game.",
		"Try changing `tick every 110` to a smaller number for faster gameplay.", "", ""},
	{"transpile-canvas-game", "Games", "🎮 Canvas game",
		"A tiny game DSL → full HTML5 canvas game with sprites + input handlers.",
		"Add new sprites or key bindings.", "", ""},
	{"scene-dsl", "Games", "🎬 Scene description",
		"High-level scene declaration → rendering-engine setup.",
		"Add new entities.", "", ""},
	// ─── 3D ────────────────────────────────────────────────────────
	{"transpile-threejs", "3D", "🌐 Three.js scene (interactive)",
		"~25 declarative lines → a runnable, interactive three.js page. Declare meshes, abstract motions (spin/orbit/bob/pulse), and EVENT bindings — `click <target> <action>`, `hover`, `key`, `button` — wired to named actions (randomize_color, toggle_motion, cycle_motion, wireframe, explode, disco, reset_all). Raycaster picks objects, keys dispatch actions, HUD buttons fire actions. Drag to orbit, scroll to zoom.",
		"Click any object; press Space, R, D, X, W; or hit the HUD buttons. Try adding `click ring explode` or `key \"q\" any disco`.", "", ""},
	// ─── Web / UI ──────────────────────────────────────────────────
	{"transpile-landing-page", "Web", "🌐 Landing page",
		"Marketing landing page DSL → polished HTML.",
		"Change the headline, hero text, CTAs.", "", ""},
	{"transpile-react-component", "Web", "⚛️ React component",
		"Component declaration → React TSX with props + state.",
		"Add new props or component fields.", "", ""},
	{"html-component", "Web", "✨ HTML component",
		"Card / badge / hero primitives → ready-to-paste HTML+CSS.",
		"Try different variants.", "", ""},
	{"transpile-css-animations", "Web", "🎨 CSS animations",
		"Animation DSL → @keyframes + utility classes.",
		"Tweak duration or easing.", "", ""},
	{"transpile-email-html", "Web", "📧 Email HTML",
		"Email template DSL → inline-styled HTML for email clients.",
		"Edit headline, body, CTA.", "", ""},
	{"transpile-form", "Web", "📝 HTML form",
		"Form field declarations → a styled HTML form.",
		"Add or remove fields.", "", ""},
	{"transpile-blog", "Web", "📰 Blog post",
		"Blog-post DSL → Markdown with frontmatter + structured content.",
		"Add new sections.", "", ""},
	{"supercharge-markdown", "Web", "📝 Markdown w/ callouts",
		"Markdown extension DSL → real Markdown with HTML callouts + metric cards.",
		"Add more `callout` or `card` blocks.", "", ""},
	// ─── Diagrams & data ───────────────────────────────────────────
	{"transpile-mermaid", "Diagrams", "🌊 Mermaid diagram",
		"High-level flow DSL → Mermaid syntax.",
		"Add new nodes or edges.", "", ""},
	{"transpile-csv", "Diagrams", "📊 CSV",
		"Tabular DSL → CSV with header row.",
		"Add columns or rows.", "", ""},
	{"transpile-json", "Diagrams", "🗂️ JSON",
		"Structured DSL → indented JSON output.",
		"Add nested fields.", "", ""},
	// ─── DevOps & config ───────────────────────────────────────────
	{"transpile-dockerfile", "DevOps", "🐳 Dockerfile",
		"Multi-stage Docker build DSL → real Dockerfile.",
		"Add layers or environment variables.", "", ""},
	{"transpile-kubernetes", "DevOps", "☸️ Kubernetes manifest",
		"Deployment + service DSL → full K8s YAML.",
		"Add containers, change replicas.", "", ""},
	{"transpile-terraform", "DevOps", "🏗️ Terraform module",
		"Infrastructure DSL → Terraform HCL.",
		"Add resources or variables.", "", ""},
	{"transpile-makefile", "DevOps", "⚙️ Makefile",
		"Build-target DSL → real Makefile.",
		"Add new targets and dependencies.", "", ""},
	{"transpile-nginx", "DevOps", "🌐 nginx config",
		"Server block DSL → nginx configuration.",
		"Add upstreams or rewrite rules.", "", ""},
	{"transpile-gh-actions", "DevOps", "⚡ GitHub Actions",
		"Workflow DSL → GitHub Actions YAML.",
		"Add new jobs or steps.", "", ""},
	{"transpile-cron", "DevOps", "⏰ Cron",
		"Schedule + command DSL → crontab lines.",
		"Add scheduled tasks.", "", ""},
	{"transpile-prometheus-alerts", "DevOps", "🚨 Prometheus alerts",
		"Alert-rule DSL → Prometheus alerting rules YAML.",
		"Add new alerts.", "", ""},
	{"transpile-env", "DevOps", "🔐 .env file",
		"Env-var DSL → dotenv config.",
		"Add new variables.", "", ""},
	{"transpile-systemd", "DevOps", "🐧 systemd unit",
		"Service DSL → systemd .service unit file.",
		"Edit ExecStart and dependencies.", "", ""},
	{"transpile-chrome-extension", "DevOps", "🌐 Chrome extension manifest",
		"Extension config DSL → manifest.json v3.",
		"Add permissions or content scripts.", "", ""},
	{"supercharge-sql", "DevOps", "💎 SQL DDL with macros",
		"Macros (pk/fk/timestamps/soft_delete) → idiomatic Postgres DDL.",
		"Add new tables.", "", ""},
	// ─── Schemas & APIs ────────────────────────────────────────────
	{"transpile-postgres-schema", "Schemas", "🗃️ Postgres schema",
		"Table DSL → CREATE TABLE + indexes + foreign keys.",
		"Add tables and references.", "", ""},
	{"transpile-prisma-schema", "Schemas", "💠 Prisma schema",
		"Model DSL → Prisma schema.prisma.",
		"Add models or relations.", "", ""},
	{"transpile-openapi", "Schemas", "🔭 OpenAPI spec",
		"Endpoint DSL → OpenAPI 3.0 YAML.",
		"Add endpoints, parameters, responses.", "", ""},
	{"transpile-graphql", "Schemas", "📡 GraphQL schema",
		"Type DSL → GraphQL SDL.",
		"Add types and resolvers.", "", ""},
	{"transpile-zod-schema", "Schemas", "🛡️ Zod schema",
		"Type DSL → Zod TypeScript validation schemas.",
		"Add fields or refinements.", "", ""},
	{"transpile-typescript", "Schemas", "📑 TypeScript types",
		"Struct DSL → TypeScript interfaces + types.",
		"Add fields.", "", ""},
	{"transpile-protobuf", "Schemas", "📐 Protobuf",
		"Message DSL → .proto definitions.",
		"Add messages and services.", "", ""},
	{"transpile-api-docs", "Schemas", "📋 API docs",
		"Endpoint DSL → Markdown API reference.",
		"Add new endpoints.", "", ""},
	{"typed-config-dsl", "Schemas", "🔒 Typed config (HCL)",
		"Service config DSL with custom types (Email, Semver, LogLevel) → HCL.",
		"Add new services. Bad values give clear type errors.", "", ""},
	{"metaprogramming", "Schemas", "🧬 Metaprogramming",
		"Source declares its own DSL primitives via `define ... end` — the library doesn't need them.",
		"Add a new `define ...` block at the top and use it below — your source extends the grammar.", "", ""},
	{"host-capabilities", "Schemas", "🔌 Host capabilities (env / args / read_file)",
		"Generates a Kubernetes deployment from env vars, CLI args, and a sibling secrets file. The CLI's OSHost exposes real values; the playground's sandboxed NoOpHost returns empty strings (read the description in the output for context).",
		"In the CLI: ENV=production capy run lib.capy script.capy v2.3.1 us-west-2. In the playground: see how the same library degrades cleanly when env/args are empty.", "", ""},
	{"cross-platform-installer", "DevOps", "📦 Cross-platform installer (sh/ps1/bat)",
		"One DSL declares `install foo`, `mkdir bar`, `setenv X Y`, `service svc`. Multi-file output emits a bash installer, a PowerShell installer, AND a CMD installer — all in lock-step.",
		"Add another `install` line; all three scripts get it.", "", ""},
	{"transpile-websocket-server", "Code", "🔌 WebSocket server (Go)",
		"9 declarative lines → a complete Go WebSocket chat server (~80 lines) with typed JSON envelope, message dispatch, hub, and broadcast helper.",
		"Add another `message` + `on` pair; the dispatch grows automatically.", "", ""},
	{"multi-target-ws-server", "Code", "🌐 WS server · Go",
		"Same 7-line ws DSL → Go (gorilla/websocket). Switch the library to retarget Node or Python.",
		"Compare with the Node and Python variants in the dropdown.", "", "lib-go.capy"},
	{"multi-target-ws-server", "Code", "🌐 WS server · Node",
		"Same 7-line ws DSL → Node.js (ws package).",
		"Compare with the Go and Python variants. The source is identical.", "", "lib-node.capy"},
	{"multi-target-ws-server", "Code", "🌐 WS server · Python",
		"Same 7-line ws DSL → Python (websockets + asyncio).",
		"Compare with the Go and Node variants. The source is identical.", "", "lib-python.capy"},
	{"custom-asm", "Code", "⚙️ Assembly · x86_64",
		"Architecture-neutral assembly DSL (data / func / write / exit) → GNU as syntax for x86_64 Linux (System V). 5-line source → runnable hello-world.",
		"Switch to the arm64 or riscv64 variant; identical source, different syscall ABI.", "", "lib-x86_64-linux.capy"},
	{"custom-asm", "Code", "⚙️ Assembly · arm64",
		"Same source → AArch64 (ARM64) Linux assembly. Syscall number in x8, args in x0-x5, svc #0.",
		"Compare with the x86_64 and riscv64 variants.", "", "lib-arm64-linux.capy"},
	{"custom-asm", "Code", "⚙️ Assembly · riscv64",
		"Same source → RISC-V 64 (RV64I) Linux assembly. Syscall number in a7, args in a0-a5, ecall.",
		"Compare with the x86_64 and arm64 variants. Add new architectures by writing a new library.", "", "lib-riscv64-linux.capy"},

	// ─── Progressive abstraction (3 levels of the SAME library) ────
	{"progressive-abstraction", "Patterns", "🎚️ Abstraction · Level 1 (one-shot)",
		"4-line `landing` declaration. The library decides everything else.",
		"Tweak the tagline or CTA. Want more control? Try Level 2.",
		"script_minimal.capy", ""},
	{"progressive-abstraction", "Patterns", "🎚️ Abstraction · Level 2 (blocks)",
		"~12-line block style. You pick which sections appear and in what order; library owns visuals.",
		"Add another `feature` line. Want full control? Try Level 3.",
		"script_medium.capy", ""},
	{"progressive-abstraction", "Patterns", "🎚️ Abstraction · Level 3 (escape hatches)",
		"~30 lines. Same primitives PLUS raw_head / style_override / raw_section / raw_footer.",
		"Anything you'd write in raw HTML/CSS, drop in here. The library never blocks you.",
		"script_full.capy", ""},

	// ─── Code generation ───────────────────────────────────────────
	{"transpile-py", "Code", "🐍 Python script",
		"Source language DSL → Python.",
		"Try adding more functions.", "", ""},
	{"transpile-express-server", "Code", "🚂 Express server",
		"Route DSL → Express.js handlers.",
		"Add new endpoints.", "", ""},
	{"transpile-flask-app", "Code", "🌶️ Flask app",
		"Route DSL → Python Flask handlers.",
		"Add new routes.", "", ""},
	{"transpile-fastapi-app", "Code", "⚡ FastAPI app",
		"Route DSL → Python FastAPI handlers.",
		"Add new endpoints with type hints.", "", ""},
	{"transpile-go", "Code", "🐹 Go",
		"Source language DSL → idiomatic Go.",
		"Add functions or types.", "", ""},
	{"transpile-cli", "Code", "💼 CLI app",
		"Flag DSL → CLI argument parsing scaffold.",
		"Add new flags.", "", ""},
	{"transpile-slack-blocks", "Code", "💬 Slack message",
		"Block-kit DSL → Slack message JSON.",
		"Try different block types.", "", ""},
	{"transpile-statemachine", "Code", "🔄 State machine",
		"State + transition DSL → runnable state-machine code.",
		"Add states or transitions.", "", ""},
	{"transpile-xstate-machine", "Code", "🎛️ XState machine",
		"State-chart DSL → XState v5 machine config.",
		"Add states or guards.", "", ""},
	{"transpile-bash", "Code", "🐚 Bash script",
		"Command DSL → portable bash script.",
		"Add new commands.", "", ""},
	{"transpile-changelog", "Code", "📋 Changelog",
		"Release entry DSL → Keep-a-Changelog Markdown.",
		"Add new releases.", "", ""},
}

type bundle struct {
	Samples    []sample `json:"samples"`
	Categories []string `json:"categories"`
}

type sample struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Hint        string `json:"hint"`
	Library     string `json:"library"`
	Script      string `json:"script"`
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	samplesDir := filepath.Join(root, "samples")

	var out bundle
	seenCat := map[string]bool{}
	for _, c := range CURATED {
		libName := c.LibraryFile
		if libName == "" {
			libName = "lib.capy"
		}
		libBytes, err := os.ReadFile(filepath.Join(samplesDir, c.ID, libName))
		if err != nil && c.LibraryFile == "" {
			// Fall back to lib.yaml only when the curated entry didn't
			// explicitly name a library file.
			libBytes, err = os.ReadFile(filepath.Join(samplesDir, c.ID, "lib.yaml"))
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "playground-bundle: SKIP %s (%v)\n", c.ID, err)
			continue
		}
		scriptName := c.ScriptFile
		if scriptName == "" {
			scriptName = "script.capy"
		}
		scriptBytes, err := os.ReadFile(filepath.Join(samplesDir, c.ID, scriptName))
		if err != nil {
			fmt.Fprintf(os.Stderr, "playground-bundle: SKIP %s/%s (%v)\n", c.ID, scriptName, err)
			continue
		}
		out.Samples = append(out.Samples, sample{
			ID:          c.ID,
			Category:    c.Category,
			Title:       c.Title,
			Description: c.Description,
			Hint:        c.Hint,
			Library:     string(libBytes),
			Script:      string(scriptBytes),
		})
		if !seenCat[c.Category] {
			seenCat[c.Category] = true
			out.Categories = append(out.Categories, c.Category)
		}
	}

	fmt.Fprintf(os.Stderr, "playground-bundle: bundled %d samples across %d categories\n",
		len(out.Samples), len(out.Categories))

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "playground-bundle:", err)
	os.Exit(1)
}
