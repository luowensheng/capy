package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Preprocess walks a source string line by line and expands any top-level
// `@import "path"` / `@include "path"` directives into the contents of the
// referenced file (recursively). The result is the merged source that the
// lexer then tokenizes.
//
// Path resolution: relative paths are resolved against `dir`. Absolute
// paths are honored as-is.
//
// Cycle detection: imports are tracked by absolute path. If file A imports
// B which imports A, the loader stops with a clear error.
//
// The directive itself must be at the START of a line (after optional
// leading whitespace, which is preserved on the inlined content for visual
// nesting). Both `@import` and `@include` are accepted as synonyms.
//
// This is library-agnostic — the engine knows about the directive
// regardless of which library is loaded. Libraries that want a different
// surface keyword (e.g. `use "x.capy"`) can define a thin wrapper function
// that emits an `@import` line, or build their own pre-parsing layer.
func Preprocess(source, dir string) (string, error) {
	return preprocess(source, dir, map[string]bool{})
}

func preprocess(source, dir string, visited map[string]bool) (string, error) {
	var out strings.Builder
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		// Find leading indent so imported content can be re-indented to
		// match the @import line's column. This is what authors expect:
		// imports nest inside whatever block they appear in.
		indentLen := 0
		for indentLen < len(line) && (line[indentLen] == ' ' || line[indentLen] == '\t') {
			indentLen++
		}
		trimmed := line[indentLen:]
		if d, path, ok := matchImport(trimmed); ok {
			absPath := resolvePath(path, dir)
			if visited[absPath] {
				return "", fmt.Errorf("line %d: import cycle: %s", i+1, absPath)
			}
			b, err := os.ReadFile(absPath)
			if err != nil {
				return "", fmt.Errorf("line %d: %s %q: %v", i+1, d, path, err)
			}
			visited[absPath] = true
			expanded, err := preprocess(string(b), filepath.Dir(absPath), visited)
			if err != nil {
				return "", err
			}
			delete(visited, absPath)
			indent := line[:indentLen]
			for j, l := range strings.Split(strings.TrimRight(expanded, "\n"), "\n") {
				if j > 0 {
					out.WriteString("\n")
				}
				if strings.TrimSpace(l) == "" {
					continue // keep blank lines blank
				}
				out.WriteString(indent)
				out.WriteString(l)
			}
			if i < len(lines)-1 {
				out.WriteString("\n")
			}
			continue
		}
		out.WriteString(line)
		if i < len(lines)-1 {
			out.WriteString("\n")
		}
	}
	return out.String(), nil
}

// matchImport recognizes `@import "path"` or `@include "path"`. Returns the
// directive name, the path, and ok=true on match.
func matchImport(line string) (string, string, bool) {
	for _, d := range []string{"@import", "@include"} {
		if !strings.HasPrefix(line, d) {
			continue
		}
		rest := strings.TrimSpace(line[len(d):])
		if len(rest) < 2 || rest[0] != '"' {
			continue
		}
		// Find the closing quote.
		end := strings.IndexByte(rest[1:], '"')
		if end < 0 {
			continue
		}
		path := rest[1 : 1+end]
		// Allow trailing comment after the directive.
		after := strings.TrimSpace(rest[2+end:])
		if after != "" && !strings.HasPrefix(after, "#") {
			continue
		}
		return d, path, true
	}
	return "", "", false
}

func resolvePath(path, dir string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(dir, path))
}
