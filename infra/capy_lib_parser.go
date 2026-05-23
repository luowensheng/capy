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
		case "file_template:":
			p.nextLine()
			body, err := p.parseIndentedBlock(0)
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
				if len(tokens) != 3 {
					return fn, p.errf("arg literal requires a value")
				}
				fn.Args = append(fn.Args, RawArg{Kind: "literal", Value: tokens[2]})
			case "capture":
				if len(tokens) < 3 || len(tokens) > 4 {
					return fn, p.errf("arg capture NAME [TYPE]")
				}
				a := RawArg{Kind: "capture", Name: tokens[2], Type: "any"}
				if len(tokens) == 4 {
					a.Type = tokens[3]
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
