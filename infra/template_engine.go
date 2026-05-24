package infra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

// TemplateEngine renders Go text/templates with a small set of code-generator
// helpers exposed as template functions.
type TemplateEngine struct{}

func (TemplateEngine) Render(tpl string, data map[string]any) (string, error) {
	t, err := template.New("capy").Funcs(funcs).Parse(tpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var funcs = template.FuncMap{
	// indent N: indent every line of s by N spaces. Useful for `body`.
	"indent": func(n int, s string) string {
		pad := strings.Repeat(" ", n)
		lines := strings.Split(s, "\n")
		for i, l := range lines {
			if l == "" {
				continue
			}
			lines[i] = pad + l
		}
		return strings.Join(lines, "\n")
	},
	"lower": strings.ToLower,
	"upper": strings.ToUpper,
	// pascalCase converts a human-readable string ("Habit Tracker", "habit
	// tracker", "habit-tracker", "habit_tracker") into a PascalCase
	// identifier ("HabitTracker"). Useful when target languages require an
	// identifier and the source has a friendly display name.
	"pascalCase": func(s any) string {
		text := toStringAny(s)
		// First strip surrounding quotes from string captures.
		if len(text) >= 2 && (text[0] == '"' || text[0] == '\'' || text[0] == '`') && text[0] == text[len(text)-1] {
			text = text[1 : len(text)-1]
		}
		// Split on space, dash, underscore.
		var b strings.Builder
		nextUpper := true
		for _, r := range text {
			if r == ' ' || r == '_' || r == '-' || r == '.' {
				nextUpper = true
				continue
			}
			if nextUpper {
				b.WriteRune(toUpperRune(r))
				nextUpper = false
			} else {
				b.WriteRune(r)
			}
		}
		return b.String()
	},
	// camelCase is pascalCase with the first char lowered.
	"camelCase": func(s any) string {
		text := toStringAny(s)
		if len(text) >= 2 && (text[0] == '"' || text[0] == '\'' || text[0] == '`') && text[0] == text[len(text)-1] {
			text = text[1 : len(text)-1]
		}
		var b strings.Builder
		nextUpper := false
		first := true
		for _, r := range text {
			if r == ' ' || r == '_' || r == '-' || r == '.' {
				nextUpper = true
				continue
			}
			if first {
				b.WriteRune(toLowerRune(r))
				first = false
				continue
			}
			if nextUpper {
				b.WriteRune(toUpperRune(r))
				nextUpper = false
			} else {
				b.WriteRune(r)
			}
		}
		return b.String()
	},
	// snakeCase converts to lower_snake_case.
	"snakeCase": func(s any) string {
		text := toStringAny(s)
		if len(text) >= 2 && (text[0] == '"' || text[0] == '\'' || text[0] == '`') && text[0] == text[len(text)-1] {
			text = text[1 : len(text)-1]
		}
		var b strings.Builder
		for i, r := range text {
			switch r {
			case ' ', '-', '.':
				b.WriteRune('_')
			default:
				if i > 0 && r >= 'A' && r <= 'Z' {
					b.WriteRune('_')
				}
				b.WriteRune(toLowerRune(r))
			}
		}
		return b.String()
	},
	// dasherize converts snake_case to kebab-case. Useful for CSS property
	// names where the lexer doesn't allow hyphens in identifiers.
	"dasherize": func(s any) string {
		return strings.ReplaceAll(toStringAny(s), "_", "-")
	},
	// unquote strips one layer of surrounding "...", '...', or `...` if
	// present. Useful when the source uses string literals but the target
	// doesn't want the quotes in the output (markdown headings, etc.).
	"unquote": func(s any) string {
		text := toStringAny(s)
		if len(text) >= 2 {
			first, last := text[0], text[len(text)-1]
			if (first == '"' || first == '\'' || first == '`') && first == last {
				return text[1 : len(text)-1]
			}
		}
		return text
	},
	// unescape reverses Go string escaping. Capy preserves the source's
	// backslash sequences verbatim through the lexer and re-quotes via
	// strconv.Quote, so a captured "Hello\n" surfaces in templates as
	// `"Hello\\n"`. Use unescape when the TARGET wants the actual escape
	// sequence (e.g. assembler .ascii / .asciz directives, C string
	// literals, JSON-on-the-wire). Wraps in quotes first if missing.
	"unescape": func(s any) string {
		text := toStringAny(s)
		if len(text) < 2 || text[0] != '"' || text[len(text)-1] != '"' {
			text = "\"" + text + "\""
		}
		v, err := strconv.Unquote(text)
		if err != nil {
			return toStringAny(s)
		}
		return v
	},
	// trimSuffix removes a trailing string if present. Useful for joining
	// generators that emit comma-suffixed lines: drop the dangling comma
	// at the end of a list with `{{ .body | trimSuffix ",\n" }}\n`.
	"trimSuffix": func(suffix string, s any) string {
		text := toStringAny(s)
		return strings.TrimSuffix(text, suffix)
	},
	"trimPrefix": func(prefix string, s any) string {
		text := toStringAny(s)
		return strings.TrimPrefix(text, prefix)
	},
	"join": func(sep string, items []any) string {
		parts := make([]string, 0, len(items))
		for _, x := range items {
			parts = append(parts, toStringAny(x))
		}
		return strings.Join(parts, sep)
	},
	// split breaks a string into a list at each occurrence of SEP.
	// Argument order matches strings.Split (string first, separator
	// second) so it reads naturally inline: `range (split .text "\n")`.
	"split": func(s any, sep string) []string {
		return strings.Split(toStringAny(s), sep)
	},
	// nonEmpty filters a string list down to entries that aren't blank
	// after trimming whitespace. Handy when iterating over `read_file`
	// output without producing trailing empty rows.
	"nonEmpty": func(items []string) []string {
		out := make([]string, 0, len(items))
		for _, s := range items {
			if strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	},
	// toQuoted wraps a string in JSON-style double quotes (good for Python too).
	"toQuoted": func(s any) string {
		b, _ := json.Marshal(toStringAny(s))
		return string(b)
	},
	// toPyLit formats any value as a Python literal.
	"toPyLit": pyLit,
	// toJSON marshals any value to compact JSON (good for config-file output).
	"toJSON": func(v any) string {
		b, _ := json.Marshal(v)
		return string(b)
	},
	// toJSONIndent marshals with indentation.
	"toJSONIndent": func(v any) string {
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	},
	// add returns a + b. Both args coerced to int64. Useful for running
	// totals inside `range` blocks (e.g. summing pages in a reading log).
	"add": func(a, b any) int64 { return toInt(a) + toInt(b) },
	"sub": func(a, b any) int64 { return toInt(a) - toInt(b) },
	"mul": func(a, b any) int64 { return toInt(a) * toInt(b) },
	// percent returns (numerator / denominator) * 100 as an int, clamped
	// to [0, 100]. Handy for progress bars in HTML output.
	"percent": func(n, d any) int64 {
		den := toInt(d)
		if den == 0 {
			return 0
		}
		p := toInt(n) * 100 / den
		if p < 0 {
			return 0
		}
		if p > 100 {
			return 100
		}
		return p
	},
	// stars renders an integer as that many filled-star characters plus
	// the remainder out of five as outlined stars. Useful for rating
	// displays in non-programmer DSLs (reading logs, restaurant lists).
	"stars": func(n any) string {
		k := int(toInt(n))
		if k < 0 {
			k = 0
		}
		if k > 5 {
			k = 5
		}
		return strings.Repeat("★", k) + strings.Repeat("☆", 5-k)
	},
}

// toInt coerces common numeric types to int64. Tolerates strings holding
// digit sequences so `{{ add .x 1 }}` works even when .x came from a
// string-typed capture.
func toInt(v any) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int64:
		return x
	case float64:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		return n
	}
	return 0
}

func pyLit(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case bool:
		if x {
			return "True"
		}
		return "False"
	case string:
		b, _ := json.Marshal(x)
		return string(b)
	case []any:
		parts := make([]string, 0, len(x))
		for _, it := range x {
			parts = append(parts, pyLit(it))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		parts := []string{}
		for k, v := range x {
			parts = append(parts, fmt.Sprintf("%q: %s", k, pyLit(v)))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return fmt.Sprintf("%v", v)
}

func toUpperRune(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}
func toLowerRune(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}

func toStringAny(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	}
	return fmt.Sprintf("%v", v)
}
