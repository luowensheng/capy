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
	ID, Category, Title, Description, Hint, ScriptFile string
}{
	// ─── Everyday / no-code-required ────────────────────────────────
	{"recipe-card", "Everyday", "🍋 Recipe card",
		"Write a recipe in plain words; get a printable HTML recipe card.",
		"Try editing the title, ingredients, or steps.", ""},
	{"event-invite", "Everyday", "🎉 Party invitation",
		"Declare a party; get a pastel HTML invitation card.",
		"Try changing the host, location, or RSVP date.", ""},
	{"weekly-meal-plan", "Everyday", "📅 Weekly meal plan",
		"Seven dinners + notes → printable HTML grid for the fridge.",
		"Swap meals or add notes.", ""},
	{"reading-log", "Everyday", "📚 Reading log (for kids)",
		"A kid's reading list → bright HTML certificate with progress bar.",
		"Add more `book` lines; the progress bar updates.", ""},
	{"trip-itinerary", "Everyday", "✈️ Trip itinerary",
		"Day-by-day travel plan → polished HTML itinerary card.",
		"Edit destinations, add activities, change the budget.", ""},
	{"transpile-resume", "Everyday", "📄 Resume",
		"A resume DSL → a clean printable resume.",
		"Edit experience and skills lines.", ""},
	{"transpile-invoice", "Everyday", "🧾 Invoice",
		"Declare an invoice; get a polished HTML invoice you can print.",
		"Edit the line items and rates.", ""},
	{"transpile-markdown-todo", "Everyday", "✅ To-do list",
		"A tiny todo DSL → Markdown checklist.",
		"Add tasks, mark them done.", ""},
	// ─── Interactive games ──────────────────────────────────────────
	{"interactive-breakout", "Games", "🕹️ Breakout (playable)",
		"Event-driven Breakout: entities + key bindings + event handlers → 226-line HTML5 game.",
		"Change `lives 3` to `lives 5`, or rebind keys.", ""},
	{"interactive-snake", "Games", "🐍 Snake (playable)",
		"Event-driven Snake: key bindings + event handlers → 180-line HTML5 game.",
		"Try changing `tick every 110` to a smaller number for faster gameplay.", ""},
	{"transpile-canvas-game", "Games", "🎮 Canvas game",
		"A tiny game DSL → full HTML5 canvas game with sprites + input handlers.",
		"Add new sprites or key bindings.", ""},
	{"scene-dsl", "Games", "🎬 Scene description",
		"High-level scene declaration → rendering-engine setup.",
		"Add new entities.", ""},
	// ─── Web / UI ──────────────────────────────────────────────────
	{"transpile-landing-page", "Web", "🌐 Landing page",
		"Marketing landing page DSL → polished HTML.",
		"Change the headline, hero text, CTAs.", ""},
	{"transpile-react-component", "Web", "⚛️ React component",
		"Component declaration → React TSX with props + state.",
		"Add new props or component fields.", ""},
	{"html-component", "Web", "✨ HTML component",
		"Card / badge / hero primitives → ready-to-paste HTML+CSS.",
		"Try different variants.", ""},
	{"transpile-css-animations", "Web", "🎨 CSS animations",
		"Animation DSL → @keyframes + utility classes.",
		"Tweak duration or easing.", ""},
	{"transpile-email-html", "Web", "📧 Email HTML",
		"Email template DSL → inline-styled HTML for email clients.",
		"Edit headline, body, CTA.", ""},
	{"transpile-form", "Web", "📝 HTML form",
		"Form field declarations → a styled HTML form.",
		"Add or remove fields.", ""},
	{"transpile-blog", "Web", "📰 Blog post",
		"Blog-post DSL → Markdown with frontmatter + structured content.",
		"Add new sections.", ""},
	{"supercharge-markdown", "Web", "📝 Markdown w/ callouts",
		"Markdown extension DSL → real Markdown with HTML callouts + metric cards.",
		"Add more `callout` or `card` blocks.", ""},
	// ─── Diagrams & data ───────────────────────────────────────────
	{"transpile-mermaid", "Diagrams", "🌊 Mermaid diagram",
		"High-level flow DSL → Mermaid syntax.",
		"Add new nodes or edges.", ""},
	{"transpile-csv", "Diagrams", "📊 CSV",
		"Tabular DSL → CSV with header row.",
		"Add columns or rows.", ""},
	{"transpile-json", "Diagrams", "🗂️ JSON",
		"Structured DSL → indented JSON output.",
		"Add nested fields.", ""},
	// ─── DevOps & config ───────────────────────────────────────────
	{"transpile-dockerfile", "DevOps", "🐳 Dockerfile",
		"Multi-stage Docker build DSL → real Dockerfile.",
		"Add layers or environment variables.", ""},
	{"transpile-kubernetes", "DevOps", "☸️ Kubernetes manifest",
		"Deployment + service DSL → full K8s YAML.",
		"Add containers, change replicas.", ""},
	{"transpile-terraform", "DevOps", "🏗️ Terraform module",
		"Infrastructure DSL → Terraform HCL.",
		"Add resources or variables.", ""},
	{"transpile-makefile", "DevOps", "⚙️ Makefile",
		"Build-target DSL → real Makefile.",
		"Add new targets and dependencies.", ""},
	{"transpile-nginx", "DevOps", "🌐 nginx config",
		"Server block DSL → nginx configuration.",
		"Add upstreams or rewrite rules.", ""},
	{"transpile-gh-actions", "DevOps", "⚡ GitHub Actions",
		"Workflow DSL → GitHub Actions YAML.",
		"Add new jobs or steps.", ""},
	{"transpile-cron", "DevOps", "⏰ Cron",
		"Schedule + command DSL → crontab lines.",
		"Add scheduled tasks.", ""},
	{"transpile-prometheus-alerts", "DevOps", "🚨 Prometheus alerts",
		"Alert-rule DSL → Prometheus alerting rules YAML.",
		"Add new alerts.", ""},
	{"transpile-env", "DevOps", "🔐 .env file",
		"Env-var DSL → dotenv config.",
		"Add new variables.", ""},
	{"transpile-systemd", "DevOps", "🐧 systemd unit",
		"Service DSL → systemd .service unit file.",
		"Edit ExecStart and dependencies.", ""},
	{"transpile-chrome-extension", "DevOps", "🌐 Chrome extension manifest",
		"Extension config DSL → manifest.json v3.",
		"Add permissions or content scripts.", ""},
	{"supercharge-sql", "DevOps", "💎 SQL DDL with macros",
		"Macros (pk/fk/timestamps/soft_delete) → idiomatic Postgres DDL.",
		"Add new tables.", ""},
	// ─── Schemas & APIs ────────────────────────────────────────────
	{"transpile-postgres-schema", "Schemas", "🗃️ Postgres schema",
		"Table DSL → CREATE TABLE + indexes + foreign keys.",
		"Add tables and references.", ""},
	{"transpile-prisma-schema", "Schemas", "💠 Prisma schema",
		"Model DSL → Prisma schema.prisma.",
		"Add models or relations.", ""},
	{"transpile-openapi", "Schemas", "🔭 OpenAPI spec",
		"Endpoint DSL → OpenAPI 3.0 YAML.",
		"Add endpoints, parameters, responses.", ""},
	{"transpile-graphql", "Schemas", "📡 GraphQL schema",
		"Type DSL → GraphQL SDL.",
		"Add types and resolvers.", ""},
	{"transpile-zod-schema", "Schemas", "🛡️ Zod schema",
		"Type DSL → Zod TypeScript validation schemas.",
		"Add fields or refinements.", ""},
	{"transpile-typescript", "Schemas", "📑 TypeScript types",
		"Struct DSL → TypeScript interfaces + types.",
		"Add fields.", ""},
	{"transpile-protobuf", "Schemas", "📐 Protobuf",
		"Message DSL → .proto definitions.",
		"Add messages and services.", ""},
	{"transpile-api-docs", "Schemas", "📋 API docs",
		"Endpoint DSL → Markdown API reference.",
		"Add new endpoints.", ""},
	{"typed-config-dsl", "Schemas", "🔒 Typed config (HCL)",
		"Service config DSL with custom types (Email, Semver, LogLevel) → HCL.",
		"Add new services. Bad values give clear type errors.", ""},
	{"metaprogramming", "Schemas", "🧬 Metaprogramming",
		"Source declares its own DSL primitives via `define ... end` — the library doesn't need them.",
		"Add a new `define ...` block at the top and use it below — your source extends the grammar.", ""},
	{"host-capabilities", "Schemas", "🔌 Host capabilities (env / args / read_file)",
		"Generates a Kubernetes deployment from env vars, CLI args, and a sibling secrets file. The CLI's OSHost exposes real values; the playground's sandboxed NoOpHost returns empty strings (read the description in the output for context).",
		"In the CLI: ENV=production capy run lib.capy script.capy v2.3.1 us-west-2. In the playground: see how the same library degrades cleanly when env/args are empty.", ""},

	// ─── Progressive abstraction (3 levels of the SAME library) ────
	{"progressive-abstraction", "Patterns", "🎚️ Abstraction · Level 1 (one-shot)",
		"4-line `landing` declaration. The library decides everything else.",
		"Tweak the tagline or CTA. Want more control? Try Level 2.",
		"script_minimal.capy"},
	{"progressive-abstraction", "Patterns", "🎚️ Abstraction · Level 2 (blocks)",
		"~12-line block style. You pick which sections appear and in what order; library owns visuals.",
		"Add another `feature` line. Want full control? Try Level 3.",
		"script_medium.capy"},
	{"progressive-abstraction", "Patterns", "🎚️ Abstraction · Level 3 (escape hatches)",
		"~30 lines. Same primitives PLUS raw_head / style_override / raw_section / raw_footer.",
		"Anything you'd write in raw HTML/CSS, drop in here. The library never blocks you.",
		"script_full.capy"},

	// ─── Code generation ───────────────────────────────────────────
	{"transpile-py", "Code", "🐍 Python script",
		"Source language DSL → Python.",
		"Try adding more functions.", ""},
	{"transpile-express-server", "Code", "🚂 Express server",
		"Route DSL → Express.js handlers.",
		"Add new endpoints.", ""},
	{"transpile-flask-app", "Code", "🌶️ Flask app",
		"Route DSL → Python Flask handlers.",
		"Add new routes.", ""},
	{"transpile-fastapi-app", "Code", "⚡ FastAPI app",
		"Route DSL → Python FastAPI handlers.",
		"Add new endpoints with type hints.", ""},
	{"transpile-go", "Code", "🐹 Go",
		"Source language DSL → idiomatic Go.",
		"Add functions or types.", ""},
	{"transpile-cli", "Code", "💼 CLI app",
		"Flag DSL → CLI argument parsing scaffold.",
		"Add new flags.", ""},
	{"transpile-slack-blocks", "Code", "💬 Slack message",
		"Block-kit DSL → Slack message JSON.",
		"Try different block types.", ""},
	{"transpile-statemachine", "Code", "🔄 State machine",
		"State + transition DSL → runnable state-machine code.",
		"Add states or transitions.", ""},
	{"transpile-xstate-machine", "Code", "🎛️ XState machine",
		"State-chart DSL → XState v5 machine config.",
		"Add states or guards.", ""},
	{"transpile-bash", "Code", "🐚 Bash script",
		"Command DSL → portable bash script.",
		"Add new commands.", ""},
	{"transpile-changelog", "Code", "📋 Changelog",
		"Release entry DSL → Keep-a-Changelog Markdown.",
		"Add new releases.", ""},
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
		libBytes, err := os.ReadFile(filepath.Join(samplesDir, c.ID, "lib.capy"))
		if err != nil {
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
