package infra

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func toStringAny(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	}
	return fmt.Sprintf("%v", v)
}
