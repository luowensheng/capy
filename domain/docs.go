package domain

import (
	"fmt"
	"sort"
	"strings"
)

// RenderLibraryDocs produces Markdown reference documentation for a
// loaded library. Includes:
//
//   - Library description (top-level)
//   - Output metadata (extension, output_file)
//   - One section per declared TYPE with its constraints
//   - One section per declared FUNCTION with arg table + example usage
//   - Notes about block functions and file outputs
//
// The output is plain Markdown rendered by any Markdown viewer
// (GitHub, MkDocs, the browser playground via marked.js).
func RenderLibraryDocs(lib Library) string {
	var b strings.Builder

	// ─── Header ───────────────────────────────────────────────────
	title := "Library reference"
	if lib.Extension != "" {
		title = fmt.Sprintf("Library reference (→ `.%s`)", lib.Extension)
	}
	fmt.Fprintf(&b, "# %s\n\n", title)

	if lib.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", lib.Description)
	} else {
		fmt.Fprintf(&b, "*This library has no top-level `description`. "+
			"Add one to summarize what it generates and who should use it.*\n\n")
	}

	// Metadata strip
	fmt.Fprintf(&b, "| | |\n|---|---|\n")
	if lib.Extension != "" {
		fmt.Fprintf(&b, "| **Output extension** | `.%s` |\n", lib.Extension)
	}
	if lib.OutputFile != "" {
		fmt.Fprintf(&b, "| **Default output file** | `%s` |\n", lib.OutputFile)
	}
	fmt.Fprintf(&b, "| **Functions** | %d |\n", len(lib.Functions))
	fmt.Fprintf(&b, "| **Types** | %d |\n", len(lib.Types))
	if len(lib.FilesAST) > 0 {
		fmt.Fprintf(&b, "| **Multi-file outputs** | %d (`%s`) |\n",
			len(lib.FilesAST), strings.Join(sortedKeysOfAST(lib.FilesAST), "`, `"))
	}
	b.WriteString("\n")

	// ─── Types ────────────────────────────────────────────────────
	if len(lib.Types) > 0 {
		b.WriteString("## Types\n\n")
		for _, name := range sortedKeys(lib.Types) {
			t := lib.Types[name]
			fmt.Fprintf(&b, "### `%s`\n\n", name)
			if t.Description != "" {
				fmt.Fprintf(&b, "%s\n\n", t.Description)
			}
			rules := []string{}
			if t.Base != "" && t.Base != "any" {
				rules = append(rules, fmt.Sprintf("inherits from `%s`", t.Base))
			}
			if t.Pattern != "" {
				rules = append(rules, fmt.Sprintf("must match regex `%s`", t.Pattern))
			}
			if len(t.Options) > 0 {
				opts := make([]string, len(t.Options))
				for i, o := range t.Options {
					opts[i] = "`" + o + "`"
				}
				rules = append(rules, fmt.Sprintf("must be one of: %s", strings.Join(opts, ", ")))
			}
			if len(rules) == 0 {
				rules = append(rules, "no constraints (accepts any value)")
			}
			for _, r := range rules {
				fmt.Fprintf(&b, "- %s\n", r)
			}
			b.WriteString("\n")
		}
	}

	// ─── Functions ────────────────────────────────────────────────
	if len(lib.Functions) > 0 {
		b.WriteString("## Functions\n\n")
		// Sort functions: priority-tagged ones first, then alphabetical.
		names := make([]string, 0, len(lib.Functions))
		for n := range lib.Functions {
			names = append(names, n)
		}
		sort.SliceStable(names, func(i, j int) bool {
			pi, pj := lib.Functions[names[i]].Priority, lib.Functions[names[j]].Priority
			if pi != pj {
				return pi > pj
			}
			return names[i] < names[j]
		})

		for _, name := range names {
			fn := lib.Functions[name]
			renderFunc(&b, fn)
		}
	}

	return b.String()
}

func renderFunc(b *strings.Builder, fn *FuncDef) {
	// Signature reconstruction: synthesize a source-side call shape from
	// the args. Literals appear verbatim; captures appear as `<name>`.
	parts := []string{}
	for _, a := range fn.Args {
		if a.Kind == "literal" {
			parts = append(parts, a.Value)
		} else {
			parts = append(parts, "<"+a.Name+">")
		}
	}

	fmt.Fprintf(b, "### `%s`\n\n", fn.Name)
	if fn.Description != "" {
		fmt.Fprintf(b, "%s\n\n", fn.Description)
	}

	fmt.Fprintf(b, "```\n%s\n```\n\n", strings.Join(parts, " "))

	// Arg table only if there are captures.
	hasCaptures := false
	for _, a := range fn.Args {
		if a.Kind == "capture" {
			hasCaptures = true
			break
		}
	}
	if hasCaptures {
		fmt.Fprintf(b, "| Argument | Type | Description |\n|---|---|---|\n")
		for _, a := range fn.Args {
			if a.Kind != "capture" {
				continue
			}
			desc := a.Description
			if desc == "" {
				desc = "*(no description)*"
			}
			fmt.Fprintf(b, "| `%s` | `%s` | %s |\n", a.Name, a.Type, desc)
		}
		b.WriteString("\n")
	}

	if fn.Block != nil {
		if fn.Block.Closer != "" {
			fmt.Fprintf(b, "**Opens an indented block** — body runs until `%s`.\n\n", fn.Block.Closer)
		} else if fn.Block.Open != "" {
			fmt.Fprintf(b, "**Opens a delimited block** — `%s` … `%s`.\n\n", fn.Block.Open, fn.Block.Close)
		}
	}
	if fn.Priority > 0 {
		fmt.Fprintf(b, "*Priority: %d — wins over lower-priority functions when patterns overlap.*\n\n", fn.Priority)
	}
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeysOfAST(m map[string]*InnerBlock) []string {
	return sortedKeys(m)
}
