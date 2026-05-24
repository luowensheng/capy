package infra

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// CapyLibParser reads a library written in Capy's native syntax (a `.capy`
// library file) and produces the same RawLibrary DTO as the YAML parser.
//
// Surface grammar (indentation-sensitive, line-based):
//
//	extension <STR>
//	output_file <STR>
//
//	function <NAME>
//	    priority <INT>
//	    arg literal <STR>
//	    arg capture <NAME> <TYPE>
//	    block_closer <NAME>
//	    block_open <STR> close <STR>
//	    template_str <STR>            # single-line inline template
//	    template:                      # multi-line block; ends at dedent
//	        ...
//	    run:                           # multi-line block; ends at dedent
//	        ...
//	end
//
//	file_template:
//	    ...
//
// Strings use double quotes with Go-style escapes (\n \t \" \\). Bare words
// are accepted for `extension`, `output_file`, capture types, and names.
type CapyLibParser struct{}

func (CapyLibParser) ParseFile(path string) (RawLibrary, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return RawLibrary{}, err
	}
	return parseCapyLib(string(b))
}

// ParseBytes parses a Capy-native library directly from in-memory bytes.
// Used by the embedding API (top-level `capy` package).
func (CapyLibParser) ParseBytes(b []byte) (RawLibrary, error) {
	return parseCapyLib(string(b))
}

// ParseString is the ergonomic in-memory entry point.
func (CapyLibParser) ParseString(s string) (RawLibrary, error) {
	return parseCapyLib(s)
}

func parseCapyLib(src string) (RawLibrary, error) {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	p := &capyLibParser{lines: lines, lineNo: 0}
	return p.parseTop()
}

type capyLibParser struct {
	lines  []string
	lineNo int
}

func (p *capyLibParser) errf(format string, args ...any) error {
	return fmt.Errorf("line %d: "+format, append([]any{p.lineNo}, args...)...)
}

// peekLine returns the next non-blank, non-comment line WITHOUT advancing.
// Returns (line, indent, ok). A comment is `#` as the first non-space char.
func (p *capyLibParser) peekLine() (string, int, bool) {
	for i := p.lineNo; i < len(p.lines); i++ {
		raw := p.lines[i]
		stripped := strings.TrimSpace(raw)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		indent := indentOf(raw)
		return raw, indent, true
	}
	return "", 0, false
}

// nextLine returns the next non-blank, non-comment line and advances past it.
func (p *capyLibParser) nextLine() (string, int, bool) {
	for p.lineNo < len(p.lines) {
		raw := p.lines[p.lineNo]
		p.lineNo++
		stripped := strings.TrimSpace(raw)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		return raw, indentOf(raw), true
	}
	return "", 0, false
}

// isIdent reports whether s looks like a bareword identifier — used to
// distinguish a TYPE token from a trailing description string in
// `arg capture NAME TYPE [DESC]`. Our tokenizer already strips quotes
// from string tokens, so a description like "An email address" arrives
// here as a single token with spaces in it (which fails isIdent).
func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if i == 0 {
			if !(c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
				return false
			}
			continue
		}
		if !(c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return true
}

func indentOf(s string) int {
	n := 0
	for _, c := range s {
		if c == ' ' {
			n++
		} else if c == '\t' {
			n += 4
		} else {
			break
		}
	}
	return n
}

func (p *capyLibParser) parseTop() (RawLibrary, error) {
	lib := RawLibrary{
		Functions: map[string]RawFunction{},
		Types:     map[string]RawType{},
		Context:   map[string]any{},
		Files:     map[string]string{},
	}

	for {
		line, indent, ok := p.peekLine()
		if !ok {
			break
		}
		if indent != 0 {
			return lib, p.errf("unexpected indentation at top level: %q", strings.TrimSpace(line))
		}
		tokens, err := tokenizeLibLine(line)
		if err != nil {
			return lib, p.errf("%v", err)
		}
		if len(tokens) == 0 {
			p.nextLine()
			continue
		}
		switch tokens[0] {
		case "extension":
			p.nextLine()
			if len(tokens) < 2 {
				return lib, p.errf("extension requires a value")
			}
			lib.Extension = tokens[1]
		case "output_file":
			p.nextLine()
			if len(tokens) < 2 {
				return lib, p.errf("output_file requires a value")
			}
			lib.OutputFile = tokens[1]
		case "description":
			p.nextLine()
			if len(tokens) < 2 {
				return lib, p.errf("description requires a string")
			}
			lib.Description = tokens[1]
		case "function":
			if len(tokens) < 2 {
				return lib, p.errf("function requires a name")
			}
			name := tokens[1]
			p.nextLine()
			fn, err := p.parseFunction()
			if err != nil {
				return lib, err
			}
			lib.Functions[name] = fn
		case "type":
			if len(tokens) < 2 {
				return lib, p.errf("type requires a name")
			}
			name := tokens[1]
			p.nextLine()
			td, err := p.parseType()
			if err != nil {
				return lib, err
			}
			lib.Types[name] = td
		case "context":
			p.nextLine()
			if err := p.parseContext(lib.Context); err != nil {
				return lib, err
			}
		case "import":
			p.nextLine()
			if len(tokens) != 2 {
				return lib, p.errf("import requires one path string")
			}
			lib.Imports = append(lib.Imports, tokens[1])
		case "file":
			// `file "path":` declares one of many output files.
			// Multiple `file` blocks may appear.
			if len(tokens) < 2 || !strings.HasSuffix(tokens[len(tokens)-1], ":") {
				if len(tokens) == 2 {
					return lib, p.errf("expected `file \"path\":`, got %q (missing colon?)", strings.TrimSpace(line))
				}
			}
			path := tokens[1]
			// Sanity: the last token must be ":" or path-then-":" combined.
			// Authors write `file "x.html":` so tokens are ["file", "x.html:"]
			// or ["file", "x.html", ":"]. Normalize.
			if strings.HasSuffix(path, ":") {
				path = strings.TrimSuffix(path, ":")
			} else if len(tokens) >= 3 && tokens[2] == ":" {
				// ok
			} else {
				return lib, p.errf("expected `file \"path\":` got %v", tokens)
			}
			p.nextLine()
			body, err := p.parseFileBlockBody()
			if err != nil {
				return lib, err
			}
			lib.Files[path] = body
		case "file_template:":
			p.nextLine()
			// file_template is always the last top-level item; capture
			// everything to EOF so authors can put template actions
			// (e.g. `{{ .body | indent 4 }}`) at column 0 — that's the
			// standard Go-template idiom for clean nested indentation.
			body, err := p.parseFileTemplateToEOF()
			if err != nil {
				return lib, err
			}
			lib.FileTemplate = body
		default:
			return lib, p.errf("unknown top-level directive: %q", tokens[0])
		}
	}
	return lib, nil
}

// parseFunction reads everything until a matching `end` at indent 0 (function
// keyword's column). The `function NAME` line has already been consumed.
func (p *capyLibParser) parseFunction() (RawFunction, error) {
	var fn RawFunction
	var block RawBlock
	blockSet := false

	for {
		line, indent, ok := p.peekLine()
		if !ok {
			return fn, p.errf("unexpected EOF inside function")
		}
		if indent == 0 {
			// must be the closing `end`
			tokens, err := tokenizeLibLine(line)
			if err != nil {
				return fn, p.errf("%v", err)
			}
			if len(tokens) == 1 && tokens[0] == "end" {
				p.nextLine()
				if blockSet {
					fn.Block = &block
				}
				return fn, nil
			}
			return fn, p.errf("expected `end` to close function, got %q", strings.TrimSpace(line))
		}

		tokens, err := tokenizeLibLine(line)
		if err != nil {
			return fn, p.errf("%v", err)
		}
		if len(tokens) == 0 {
			p.nextLine()
			continue
		}

		switch tokens[0] {
		case "description":
			p.nextLine()
			if len(tokens) != 2 {
				return fn, p.errf("description requires one string argument")
			}
			fn.Description = tokens[1]
		case "priority":
			p.nextLine()
			if len(tokens) != 2 {
				return fn, p.errf("priority requires one integer argument")
			}
			n, err := strconv.Atoi(tokens[1])
			if err != nil {
				return fn, p.errf("priority: %v", err)
			}
			fn.Priority = n
		case "arg":
			p.nextLine()
			if len(tokens) < 2 {
				return fn, p.errf("arg requires a kind")
			}
			switch tokens[1] {
			case "literal":
				// `arg literal "TEXT"` or `arg literal "TEXT" "DESCRIPTION"`
				if len(tokens) < 3 || len(tokens) > 4 {
					return fn, p.errf("arg literal requires a value (and optional description string)")
				}
				ra := RawArg{Kind: "literal", Value: tokens[2]}
				if len(tokens) == 4 {
					ra.Description = tokens[3]
				}
				fn.Args = append(fn.Args, ra)
			case "capture":
				// `arg capture NAME [TYPE] [DESCRIPTION]`
				if len(tokens) < 3 || len(tokens) > 5 {
					return fn, p.errf("arg capture NAME [TYPE] [DESCRIPTION]")
				}
				a := RawArg{Kind: "capture", Name: tokens[2], Type: "any"}
				if len(tokens) >= 4 {
					// 4th token is TYPE unless it looks like a description
					// (starts with a capital letter or contains spaces). Since
					// our tokenizer already unquotes strings, we differentiate
					// by checking if it looks like an ident.
					t := tokens[3]
					if isIdent(t) {
						a.Type = t
						if len(tokens) == 5 {
							a.Description = tokens[4]
						}
					} else {
						// 4th token is the description; type stays "any"
						a.Description = t
						if len(tokens) == 5 {
							return fn, p.errf("arg capture: TYPE must precede DESCRIPTION")
						}
					}
				}
				fn.Args = append(fn.Args, a)
			default:
				return fn, p.errf("unknown arg kind %q", tokens[1])
			}
		case "block_closer":
			p.nextLine()
			if len(tokens) != 2 {
				return fn, p.errf("block_closer requires a function name")
			}
			block.Closer = tokens[1]
			blockSet = true
		case "block_open":
			p.nextLine()
			if len(tokens) != 4 || tokens[2] != "close" {
				return fn, p.errf("block_open <OPEN> close <CLOSE>")
			}
			block.Open = tokens[1]
			block.Close = tokens[3]
			blockSet = true
		case "template_str":
			p.nextLine()
			if len(tokens) != 2 {
				return fn, p.errf("template_str requires one string argument")
			}
			fn.Template = tokens[1]
		case "template:":
			p.nextLine()
			body, err := p.parseIndentedBlock(indent)
			if err != nil {
				return fn, err
			}
			fn.Template = body
		case "run:":
			p.nextLine()
			body, err := p.parseIndentedBlock(indent)
			if err != nil {
				return fn, err
			}
			fn.Run = body
		default:
			return fn, p.errf("unknown directive inside function: %q", tokens[0])
		}
	}
}

// parseIndentedBlock captures all subsequent lines whose indent is strictly
// greater than `parentIndent`. The deepest common indent is stripped so the
// returned body reads naturally. Trailing newline is preserved.
func (p *capyLibParser) parseIndentedBlock(parentIndent int) (string, error) {
	var raws []string
	var indents []int
	minIndent := -1
	startedAt := p.lineNo

	for p.lineNo < len(p.lines) {
		raw := p.lines[p.lineNo]
		stripped := strings.TrimSpace(raw)
		if stripped == "" {
			// blank line: keep but don't influence minIndent
			raws = append(raws, "")
			indents = append(indents, -1)
			p.lineNo++
			continue
		}
		ind := indentOf(raw)
		if ind <= parentIndent {
			break
		}
		raws = append(raws, raw)
		indents = append(indents, ind)
		if minIndent == -1 || ind < minIndent {
			minIndent = ind
		}
		p.lineNo++
	}

	if minIndent == -1 {
		return "", fmt.Errorf("line %d: indented block expected but none found", startedAt)
	}

	var out strings.Builder
	for i, raw := range raws {
		if indents[i] == -1 {
			out.WriteString("\n")
			continue
		}
		// strip up to minIndent of leading whitespace
		s := stripIndent(raw, minIndent)
		out.WriteString(s)
		out.WriteString("\n")
	}
	// trim trailing blank lines to one
	res := out.String()
	for strings.HasSuffix(res, "\n\n") {
		res = res[:len(res)-1]
	}
	return res, nil
}

// parseFileBlockBody captures the indented body of a `file "..."` block —
// every subsequent line whose indent is greater than zero, with the first
// non-blank line's indent used as the strip width. Stops at the next
// top-level (column-zero) line.
func (p *capyLibParser) parseFileBlockBody() (string, error) {
	var raws []string
	var indents []int
	stripWidth := -1

	for p.lineNo < len(p.lines) {
		raw := p.lines[p.lineNo]
		stripped := strings.TrimSpace(raw)
		if stripped == "" {
			raws = append(raws, "")
			indents = append(indents, -1)
			p.lineNo++
			continue
		}
		ind := indentOf(raw)
		if ind == 0 {
			break
		}
		p.lineNo++
		raws = append(raws, raw)
		indents = append(indents, ind)
		if stripWidth == -1 {
			stripWidth = ind
		}
	}

	if stripWidth == -1 {
		return "", nil
	}

	var out strings.Builder
	for i, raw := range raws {
		if indents[i] == -1 {
			out.WriteString("\n")
			continue
		}
		out.WriteString(stripIndent(raw, stripWidth))
		out.WriteString("\n")
	}
	res := out.String()
	for strings.HasSuffix(res, "\n\n") {
		res = res[:len(res)-1]
	}
	return res, nil
}

// parseContext reads the body of a `context ... end` block. Each line is
//
//	NAME [ ] | { } | <STRING> | <NUMBER>
//
// where `[]` means initial empty list, `{}` means empty map, a quoted string
// means a string literal default, and a bare number means a numeric default.
// More elaborate shapes belong in YAML.
func (p *capyLibParser) parseContext(ctx map[string]any) error {
	for {
		line, indent, ok := p.peekLine()
		if !ok {
			return p.errf("unexpected EOF inside context")
		}
		if indent == 0 {
			tokens, err := tokenizeLibLine(line)
			if err != nil {
				return p.errf("%v", err)
			}
			if len(tokens) == 1 && tokens[0] == "end" {
				p.nextLine()
				return nil
			}
			return p.errf("expected `end` to close context, got %q", strings.TrimSpace(line))
		}
		tokens, err := tokenizeLibLine(line)
		if err != nil {
			return p.errf("%v", err)
		}
		if len(tokens) == 0 {
			p.nextLine()
			continue
		}
		p.nextLine()
		if len(tokens) < 2 {
			return p.errf("context entry needs `NAME VALUE`, got %v", tokens)
		}
		name := tokens[0]
		rest := tokens[1:]
		val, err := parseContextValue(rest)
		if err != nil {
			return p.errf("context %q: %v", name, err)
		}
		ctx[name] = val
	}
}

func parseContextValue(toks []string) (any, error) {
	if len(toks) == 1 {
		t := toks[0]
		if t == "[]" {
			return []any{}, nil
		}
		if t == "{}" {
			return map[string]any{}, nil
		}
		if t == "true" {
			return true, nil
		}
		if t == "false" {
			return false, nil
		}
		// numeric?
		if n, err := strconv.ParseInt(t, 10, 64); err == nil {
			return n, nil
		}
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return f, nil
		}
		// fallback: string
		return t, nil
	}
	if toks[0] == "[" && toks[len(toks)-1] == "]" {
		// inline list of scalars: [ a b c ]
		out := make([]any, 0, len(toks)-2)
		for _, t := range toks[1 : len(toks)-1] {
			v, err := parseContextValue([]string{t})
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}
	return strings.Join(toks, " "), nil
}

// parseType reads `type NAME` body lines until matching `end` at column 0.
// Body lines accept `base TYPE`, `pattern STRING`, `options STR STR ...`.
func (p *capyLibParser) parseType() (RawType, error) {
	var td RawType
	for {
		line, indent, ok := p.peekLine()
		if !ok {
			return td, p.errf("unexpected EOF inside type")
		}
		if indent == 0 {
			tokens, err := tokenizeLibLine(line)
			if err != nil {
				return td, p.errf("%v", err)
			}
			if len(tokens) == 1 && tokens[0] == "end" {
				p.nextLine()
				return td, nil
			}
			return td, p.errf("expected `end` to close type, got %q", strings.TrimSpace(line))
		}
		tokens, err := tokenizeLibLine(line)
		if err != nil {
			return td, p.errf("%v", err)
		}
		if len(tokens) == 0 {
			p.nextLine()
			continue
		}
		switch tokens[0] {
		case "description":
			p.nextLine()
			if len(tokens) != 2 {
				return td, p.errf("description requires one string argument")
			}
			td.Description = tokens[1]
		case "base":
			p.nextLine()
			if len(tokens) != 2 {
				return td, p.errf("base requires a type name")
			}
			td.Base = tokens[1]
		case "pattern":
			p.nextLine()
			if len(tokens) != 2 {
				return td, p.errf("pattern requires one regex string")
			}
			td.Pattern = tokens[1]
		case "options":
			p.nextLine()
			if len(tokens) < 2 {
				return td, p.errf("options requires one or more values")
			}
			td.Options = append(td.Options, tokens[1:]...)
		default:
			return td, p.errf("unknown directive inside type: %q", tokens[0])
		}
	}
}

// parseFileTemplateToEOF captures every remaining line of the file as the
// file_template body. The first non-blank line's indent is used as the
// strip width — lines that are dedented below that (e.g. a `{{ .body }}`
// action at column 0 for clean nested indentation) keep whatever leading
// whitespace they have, since stripIndent is bounded by actual whitespace.
func (p *capyLibParser) parseFileTemplateToEOF() (string, error) {
	var raws []string
	var indents []int
	stripWidth := -1

	for p.lineNo < len(p.lines) {
		raw := p.lines[p.lineNo]
		stripped := strings.TrimSpace(raw)
		p.lineNo++
		if stripped == "" {
			raws = append(raws, "")
			indents = append(indents, -1)
			continue
		}
		ind := indentOf(raw)
		raws = append(raws, raw)
		indents = append(indents, ind)
		if stripWidth == -1 {
			stripWidth = ind
		}
	}

	if stripWidth == -1 {
		return "", nil
	}
	minIndent := stripWidth

	var out strings.Builder
	for i, raw := range raws {
		if indents[i] == -1 {
			out.WriteString("\n")
			continue
		}
		out.WriteString(stripIndent(raw, minIndent))
		out.WriteString("\n")
	}
	res := out.String()
	for strings.HasSuffix(res, "\n\n") {
		res = res[:len(res)-1]
	}
	return res, nil
}

func stripIndent(s string, n int) string {
	i := 0
	stripped := 0
	for i < len(s) && stripped < n {
		c := s[i]
		if c == ' ' {
			stripped++
			i++
		} else if c == '\t' {
			stripped += 4
			i++
		} else {
			break
		}
	}
	return s[i:]
}

// tokenizeLibLine splits a line into tokens, respecting double-quoted strings
// with Go-style escapes. The result is the list of token *contents* (quotes
// removed, escapes decoded). Comments after `#` (outside strings) are dropped.
func tokenizeLibLine(line string) ([]string, error) {
	var toks []string
	s := strings.TrimRight(line, " \t")
	i := 0
	// skip leading indent
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == '#':
			return toks, nil
		case c == '"':
			j := i + 1
			var b strings.Builder
			for j < len(s) && s[j] != '"' {
				if s[j] == '\\' && j+1 < len(s) {
					switch s[j+1] {
					case 'n':
						b.WriteByte('\n')
					case 't':
						b.WriteByte('\t')
					case 'r':
						b.WriteByte('\r')
					case '"':
						b.WriteByte('"')
					case '\\':
						b.WriteByte('\\')
					default:
						b.WriteByte(s[j+1])
					}
					j += 2
					continue
				}
				b.WriteByte(s[j])
				j++
			}
			if j >= len(s) {
				return nil, fmt.Errorf("unterminated string literal")
			}
			toks = append(toks, b.String())
			i = j + 1
		default:
			j := i
			for j < len(s) && s[j] != ' ' && s[j] != '\t' && s[j] != '#' {
				j++
			}
			toks = append(toks, s[i:j])
			i = j
		}
	}
	return toks, nil
}
