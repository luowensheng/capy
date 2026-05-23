package domain

import (
	"fmt"
	"strings"
)

// CapyError is the structured error type produced by the engine. It carries a
// position so the CLI can render a caret-pointed context block.
type CapyError struct {
	Line int
	Col  int
	Msg  string
}

func (e *CapyError) Error() string {
	if e.Line == 0 {
		return e.Msg
	}
	if e.Col == 0 {
		return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
	}
	return fmt.Sprintf("line %d, col %d: %s", e.Line, e.Col, e.Msg)
}

// NewError is the common constructor.
func NewError(line, col int, format string, args ...any) *CapyError {
	return &CapyError{Line: line, Col: col, Msg: fmt.Sprintf(format, args...)}
}

// FormatWithSource renders a CapyError with the offending source line and a
// caret pointing at the column. If err is not a CapyError or has no position,
// it returns the plain error string.
//
// Example:
//   error: no library function matches token "x"
//     3 │     x = 1
//       │     ^
func FormatWithSource(err error, source string) string {
	var ce *CapyError
	switch e := err.(type) {
	case *CapyError:
		ce = e
	default:
		return err.Error()
	}
	if ce.Line == 0 || source == "" {
		return ce.Error()
	}
	lines := strings.Split(source, "\n")
	if ce.Line-1 >= len(lines) {
		return ce.Error()
	}
	line := lines[ce.Line-1]
	var b strings.Builder
	fmt.Fprintf(&b, "error: %s\n", ce.Msg)
	fmt.Fprintf(&b, "  %d │ %s\n", ce.Line, line)
	if ce.Col > 0 {
		pad := strings.Repeat(" ", ce.Col-1)
		fmt.Fprintf(&b, "    │ %s^\n", pad)
	}
	return b.String()
}
