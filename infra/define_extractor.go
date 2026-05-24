package infra

import (
	"fmt"
	"regexp"
	"strings"
)

var defineNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z_0-9]*$`)

// ExtractDefines scans a Capy source file for `define NAME ... end`
// blocks at the top level (column-0 `define`, matched by column-0
// `end`). Each block has the same body shape as a function declaration
// in a `.capy` library file (`arg literal`, `arg capture`,
// `template:`, `template_str`, `run:`, `block_closer`, `priority`).
//
// This is Capy's META-PROGRAMMING entry point: the source can extend
// the library's grammar with new patterns that subsequent statements
// can then use, without touching the library file at all.
//
// Returns:
//
//   - cleanedSource: the original source with all `define ... end`
//     blocks REMOVED (so the parser only sees calls).
//   - libSrc: a synthetic `.capy` library text containing each block
//     rewritten as a `function NAME ... end`. The caller is expected
//     to load this through the normal library loader and merge the
//     resulting Functions map into the working library (source-defined
//     functions OVERRIDE library functions of the same name). Empty
//     string when the source has no defines.
//   - err: a parse error if any define block is malformed.
func ExtractDefines(source string) (string, string, error) {
	lines := strings.Split(source, "\n")
	var kept []string
	var libBlocks []string // collected define blocks, rewritten as `function ... end`

	i := 0
	for i < len(lines) {
		line := lines[i]
		stripped := strings.TrimLeft(line, " \t")
		// Only top-level `define NAME` (no leading indent) triggers a
		// block. This avoids accidentally swallowing the word "define"
		// inside a multi-line template body or a `run:` snippet.
		indented := line != stripped // had leading whitespace
		if indented || !strings.HasPrefix(stripped, "define ") {
			kept = append(kept, line)
			i++
			continue
		}

		// Find matching `end` at column 0.
		startLine := i
		nameRest := strings.TrimPrefix(stripped, "define ")
		name := strings.TrimSpace(nameRest)
		// Allow a comment after the name, e.g. "define foo  # bar"
		if hash := strings.Index(name, "#"); hash >= 0 {
			name = strings.TrimSpace(name[:hash])
		}
		// The name must be a plain identifier — the lexer can't call a
		// function whose name contains punctuation or whitespace, so a
		// `define "bad-name"` would compile to dead code.
		if !defineNameRE.MatchString(name) {
			return "", "", fmt.Errorf("line %d: `define` name must be a plain identifier (got %q)", i+1, name)
		}

		blockLines := []string{"function " + name}
		closed := false
		for j := i + 1; j < len(lines); j++ {
			l := lines[j]
			trim := strings.TrimSpace(l)
			// `end` at column 0 closes the block.
			if trim == "end" && !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t") {
				blockLines = append(blockLines, "end")
				closed = true
				i = j + 1
				break
			}
			blockLines = append(blockLines, l)
		}
		if !closed {
			return "", "", fmt.Errorf("line %d: `define %s` is missing matching `end`", startLine+1, name)
		}
		libBlocks = append(libBlocks, strings.Join(blockLines, "\n"))
	}

	if len(libBlocks) == 0 {
		// No defines: return the original source unchanged, no allocations.
		return source, "", nil
	}

	// Quick sanity-parse so a malformed define fails here with a clear
	// message instead of in the lexer.
	synthetic := "extension _\n\n" + strings.Join(libBlocks, "\n\n") + "\n"
	if _, err := parseCapyLib(synthetic); err != nil {
		return "", "", fmt.Errorf("define block: %v", err)
	}
	return strings.Join(kept, "\n"), synthetic, nil
}
