package orchestrator

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/olivierdevelops/capy/domain"
)

// ParseCommandArgs walks `args` once, consuming flags by name and
// then positional arguments by declared order. Returns:
//   - pos: a map keyed by declared positional name → value
//   - flags: a map keyed by trimmed flag name → value (string for
//     normal flags, bool for `IsBool` flags)
//   - extra: positional args supplied beyond what the command
//     declared (surfaced as context.extra so authors can opt in to
//     variadic shapes)
//
// Errors when a required positional is missing or an unknown flag
// appears.
func ParseCommandArgs(cmd *domain.CommandDef, args []string) (map[string]any, map[string]any, []string, error) {
	pos := map[string]any{}
	flags := map[string]any{}
	var extra []string

	// Seed flag defaults.
	for _, f := range cmd.Flags {
		key := trimFlagName(f.Name)
		if f.IsBool {
			flags[key] = false
		} else {
			flags[key] = f.Default
		}
	}

	posIdx := 0
	i := 0
	for i < len(args) {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			// Flag.
			name, value, hasEq := splitFlag(a)
			fd, found := findFlag(cmd, name)
			if !found {
				return nil, nil, nil, fmt.Errorf("unknown flag %q (command %q)", a, cmd.Name)
			}
			if fd.IsBool {
				flags[trimFlagName(fd.Name)] = true
				i++
				continue
			}
			if hasEq {
				flags[trimFlagName(fd.Name)] = value
				i++
				continue
			}
			// `--flag VALUE` form — consume next arg.
			if i+1 >= len(args) {
				return nil, nil, nil, fmt.Errorf("flag %q expects a value", fd.Name)
			}
			flags[trimFlagName(fd.Name)] = args[i+1]
			i += 2
			continue
		}
		// Positional.
		if posIdx < len(cmd.Args) {
			pos[cmd.Args[posIdx].Name] = a
			posIdx++
		} else {
			extra = append(extra, a)
		}
		i++
	}

	// Missing required positional?
	for j := posIdx; j < len(cmd.Args); j++ {
		if cmd.Args[j].Required {
			return nil, nil, nil, fmt.Errorf("missing required argument %q for command %q",
				cmd.Args[j].Name, cmd.Name)
		}
		// Optional with no value — bind empty so context lookups
		// don't error.
		pos[cmd.Args[j].Name] = ""
	}

	return pos, flags, extra, nil
}

// PrintCommandHelp writes a generated help screen for the command
// to stdout based on its declared args / flags / description.
func PrintCommandHelp(lib domain.Library, cmd *domain.CommandDef) {
	out := os.Stdout
	libName := lib.LibName
	if libName == "" {
		libName = "<lib>"
	}
	fmt.Fprintf(out, "%s — %s\n\n", cmd.Name, fallback(cmd.Description, "(no description)"))
	// Usage line.
	fmt.Fprintf(out, "USAGE\n    capy %s %s", libName, cmd.Name)
	for _, f := range cmd.Flags {
		if f.IsBool {
			fmt.Fprintf(out, " [%s]", f.Name)
		} else {
			fmt.Fprintf(out, " [%s VALUE]", f.Name)
		}
	}
	for _, a := range cmd.Args {
		if a.Required {
			fmt.Fprintf(out, " <%s>", a.Name)
		} else {
			fmt.Fprintf(out, " [%s]", a.Name)
		}
	}
	fmt.Fprintln(out, "")
	if len(cmd.Args) > 0 {
		fmt.Fprintln(out, "\nARGUMENTS")
		for _, a := range cmd.Args {
			req := "(optional)"
			if a.Required {
				req = "(required)"
			}
			fmt.Fprintf(out, "    %-12s  %s %s\n", a.Name, a.Description, req)
		}
	}
	if len(cmd.Flags) > 0 {
		fmt.Fprintln(out, "\nFLAGS")
		// Sort for stable output.
		flagsSorted := make([]domain.CommandFlag, len(cmd.Flags))
		copy(flagsSorted, cmd.Flags)
		sort.Slice(flagsSorted, func(i, j int) bool {
			return flagsSorted[i].Name < flagsSorted[j].Name
		})
		for _, f := range flagsSorted {
			extra := f.Description
			if f.Default != "" {
				extra = fmt.Sprintf("%s (default: %s)", extra, f.Default)
			}
			fmt.Fprintf(out, "    %-12s  %s\n", f.Name, strings.TrimSpace(extra))
		}
	}
}

func findFlag(cmd *domain.CommandDef, name string) (*domain.CommandFlag, bool) {
	for i := range cmd.Flags {
		if cmd.Flags[i].Name == name {
			return &cmd.Flags[i], true
		}
	}
	return nil, false
}

func trimFlagName(name string) string {
	return strings.TrimLeft(name, "-")
}

// splitFlag returns (name, value, hasEq). Handles `--foo=bar` →
// ("--foo", "bar", true). For `--foo` returns ("--foo", "", false).
func splitFlag(s string) (string, string, bool) {
	if i := strings.IndexByte(s, '='); i >= 0 {
		return s[:i], s[i+1:], true
	}
	return s, "", false
}

func fallback(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
