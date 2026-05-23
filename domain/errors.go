package domain

import (
	"fmt"
	"sort"
	"strings"
)

// CapyError is the structured error type produced by the engine. It carries
// a position so the CLI can render a caret-pointed context block, plus an
// optional Hint describing how to fix the problem.
type CapyError struct {
	Line int
	Col  int
	Msg  string

	// Hint is an optional human-readable suggestion shown beneath the
	// error. Use for typo corrections, listing valid options, etc.
	Hint string

	// File is the source file path when known. Errors from imported
	// files set this to the import path; top-level errors leave it
	// empty (the CLI fills in the script path).
	File string
}

func (e *CapyError) Error() string {
	loc := ""
	if e.File != "" {
		loc = e.File
	}
	if e.Line > 0 {
		if loc != "" {
			loc += ":"
		}
		loc += fmt.Sprintf("%d", e.Line)
		if e.Col > 0 {
			loc += fmt.Sprintf(":%d", e.Col)
		}
	}
	if loc == "" {
		return e.Msg
	}
	return fmt.Sprintf("%s: %s", loc, e.Msg)
}

// NewError is the common constructor.
func NewError(line, col int, format string, args ...any) *CapyError {
	return &CapyError{Line: line, Col: col, Msg: fmt.Sprintf(format, args...)}
}

// WithHint attaches a fix suggestion to the error.
func (e *CapyError) WithHint(format string, args ...any) *CapyError {
	e.Hint = fmt.Sprintf(format, args...)
	return e
}

// FormatWithSource renders a CapyError with the offending source line, a
// caret pointing at the column, and an optional hint. If err is not a
// CapyError or has no position, it returns the plain error string.
//
// Example output:
//
//	error: no library function matches token "endpiont"
//	  hint: did you mean "endpoint"?
//	  3 │     endpiont GET "/users"
//	    │     ^
func FormatWithSource(err error, source string) string {
	var ce *CapyError
	switch e := err.(type) {
	case *CapyError:
		ce = e
	default:
		return err.Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "error: %s\n", ce.Msg)
	if ce.Hint != "" {
		fmt.Fprintf(&b, "  hint: %s\n", ce.Hint)
	}
	if ce.Line == 0 || source == "" {
		return strings.TrimRight(b.String(), "\n")
	}
	lines := strings.Split(source, "\n")
	if ce.Line-1 >= len(lines) {
		return strings.TrimRight(b.String(), "\n")
	}
	line := lines[ce.Line-1]
	fmt.Fprintf(&b, "  %d │ %s\n", ce.Line, line)
	if ce.Col > 0 {
		pad := strings.Repeat(" ", ce.Col-1)
		fmt.Fprintf(&b, "    │ %s^\n", pad)
	}
	return strings.TrimRight(b.String(), "\n")
}

// SuggestClosest returns the entry from `candidates` with the smallest edit
// distance to `target`, provided it's within `maxDist`. Returns "" when
// nothing is close enough. Used to power "did you mean X?" hints.
func SuggestClosest(target string, candidates []string, maxDist int) string {
	best := ""
	bestDist := maxDist + 1
	for _, c := range candidates {
		d := editDistance(target, c)
		if d < bestDist {
			bestDist = d
			best = c
		}
	}
	if bestDist > maxDist {
		return ""
	}
	return best
}

// SuggestClosestSorted is like SuggestClosest but takes a map keyset and
// sorts before iterating so the picked candidate is deterministic when
// multiple have equal distance.
func SuggestClosestSorted(target string, m map[string]struct{}, maxDist int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return SuggestClosest(target, keys, maxDist)
}

// editDistance computes Levenshtein distance with a small constant
// optimization. Sufficient for keyword-suggestion use cases (lists of
// 10-100 candidates, words 3-30 chars).
func editDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
