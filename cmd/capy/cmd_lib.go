package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func cmdLib(args []string) error {
	if len(args) == 0 {
		return cmdLibList(nil)
	}
	switch args[0] {
	case "list":
		return cmdLibList(args[1:])
	case "which":
		return cmdLibWhich(args[1:])
	case "new":
		return cmdLibNew(args[1:])
	case "path":
		return cmdLibPath(args[1:])
	case "add":
		return cmdLibAdd(args[1:])
	case "remove", "rm":
		return cmdLibRemove(args[1:])
	case "impl":
		return cmdLibImpl(args[1:])
	case "resolve":
		return cmdLibResolve(args[1:])
	default:
		return fmt.Errorf("unknown lib subcommand %q (try: list / which / new / add / remove / impl / resolve / path)", args[0])
	}
}

// cmdLibImpl prints the impls declared by a library, marking the
// default with a `*`. Used to discover the choices available to
// pass via --impl.
func cmdLibImpl(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: capy lib impl <library>")
	}
	_, manifestPath, lib, err := resolveLibWithImpl(args[0], "")
	// We don't error on "multiple impls + no default" here; we
	// just want to list them.
	if err != nil && !strings.Contains(err.Error(), "impl") {
		return err
	}
	header := lib.LibName
	if lib.LibVersion != "" {
		header = header + " " + lib.LibVersion
	}
	if header != "" {
		fmt.Println(header)
	}
	fmt.Println("manifest:", manifestPath)
	if len(lib.Impls) == 0 {
		fmt.Println("    (no impl declarations — the manifest file is the library)")
		return nil
	}
	fmt.Println("impls:")
	names := make([]string, 0, len(lib.Impls))
	for n := range lib.Impls {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		im := lib.Impls[n]
		mark := "  "
		if n == lib.DefaultImpl {
			mark = "* "
		}
		extra := im.Description
		if im.Version != "" {
			extra = fmt.Sprintf("%s (v%s)", extra, im.Version)
		}
		fmt.Printf("    %s%-12s  %s\n", mark, n, strings.TrimSpace(extra))
	}
	if lib.DefaultImpl != "" {
		fmt.Printf("\nDefault: %s\n", lib.DefaultImpl)
	}
	return nil
}

// cmdLibResolve shows which impl + path would be selected with
// the current environment (--impl flag, env, default).
func cmdLibResolve(args []string) error {
	args = reorderFlagsFirst(args)
	implFlag := ""
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--impl" && i+1 < len(args) {
			implFlag = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(a, "--impl=") {
			implFlag = strings.TrimPrefix(a, "--impl=")
			continue
		}
		pos = append(pos, a)
	}
	if len(pos) != 1 {
		return fmt.Errorf("usage: capy lib resolve <library> [--impl <name>]")
	}
	implPath, manifestPath, lib, err := resolveLibWithImpl(pos[0], implFlag)
	if err != nil {
		return err
	}
	fmt.Printf("library:       %s", lib.LibName)
	if lib.LibVersion != "" {
		fmt.Printf(" v%s", lib.LibVersion)
	}
	fmt.Println()
	fmt.Println("manifest:     ", manifestPath)
	if lib.SelectedImpl != "" {
		fmt.Println("impl:         ", lib.SelectedImpl)
		fmt.Println("impl file:    ", implPath)
	} else {
		fmt.Println("impl:          (no impls declared)")
	}
	return nil
}

func cmdLibList(_ []string) error {
	libs := listInstalledLibs()
	if len(libs) == 0 {
		fmt.Println("no libraries found on CAPY_LIBS")
		fmt.Println("search path:")
		for _, p := range libSearchPath() {
			fmt.Println("  ", p)
		}
		return nil
	}
	names := make([]string, 0, len(libs))
	for n := range libs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Printf("%-20s  %s\n", n, libs[n])
	}
	return nil
}

func cmdLibWhich(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: capy lib which <name>")
	}
	path, err := resolveLib(args[0])
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func cmdLibPath(_ []string) error {
	for _, p := range libSearchPath() {
		fmt.Println(p)
	}
	return nil
}

// cmdLibNew scaffolds a starter library at the first writable
// directory on CAPY_LIBS (creating it if needed).
//
//	capy lib new my-recipe
func cmdLibNew(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: capy lib new <name>")
	}
	name := args[0]
	// Choose target dir: first entry of CAPY_LIBS that exists or
	// can be created.
	var targetDir string
	for _, p := range libSearchPath() {
		if err := os.MkdirAll(p, 0755); err == nil {
			targetDir = p
			break
		}
	}
	if targetDir == "" {
		return fmt.Errorf("no writable directory on CAPY_LIBS")
	}
	libDir := filepath.Join(targetDir, name)
	if _, err := os.Stat(libDir); err == nil {
		return fmt.Errorf("library %q already exists at %s", name, libDir)
	}
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return err
	}

	// Scaffolded files.
	manifest := fmt.Sprintf(`# Library manifest. See https://olivierdevelops.github.io/capy/
name        "%s"
version     "0.1.0"
description "A new Capy library."

extension   "txt"

function greet
    arg literal "greet"
    arg capture who string
    write `+"`"+`Hello from %s, ${unquote who}!
`+"`"+`
end

command "run"
    description "Compile and print to stdout."
    let out = (compile context.arg0)
    print out
end

command "compile"
    description "Compile and write to a .txt file."
    let out    = (compile context.arg0)
    let target = "${context.arg0}.txt"
    write_file target out
    print "wrote ${target}"
end
`, name, name)
	readme := fmt.Sprintf(`# %s

A Capy library generated by `+"`capy lib new %s`"+`.

## Try it

`+"```sh"+`
echo 'greet "world"' > hello.%s
capy %s run hello.%s
`+"```"+`

## Customise

Edit `+"`%s.capy`"+`. Add functions, commands, types. Re-run
`+"`capy lib list`"+` to confirm it's picked up.
`, name, name, name, name, name, name)
	example := `greet "world"
`

	if err := os.WriteFile(filepath.Join(libDir, name+".capy"), []byte(manifest), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(libDir, "README.md"), []byte(readme), 0644); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(libDir, "examples"), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(libDir, "examples", "hello."+name), []byte(example), 0644); err != nil {
		return err
	}

	fmt.Printf("✓ created library %q at %s\n", name, libDir)
	fmt.Printf("  capy %s run %s/examples/hello.%s\n", name, libDir, name)
	return nil
}

// cmdLibAdd clones a library from a git URL into the first
// writable directory on CAPY_LIBS.
//
//	capy lib add github.com/user/repo
//	capy lib add github.com/user/repo --as my-name
//	capy lib add https://github.com/user/repo
//	capy lib add ./local/path             (also works — copies)
func cmdLibAdd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: capy lib add <git-url-or-path> [--as <name>]")
	}
	source := args[0]
	asName := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--as" && i+1 < len(args) {
			asName = args[i+1]
			i++
		}
	}

	// Choose target dir.
	var targetParent string
	for _, p := range libSearchPath() {
		if err := os.MkdirAll(p, 0755); err == nil {
			targetParent = p
			break
		}
	}
	if targetParent == "" {
		return fmt.Errorf("no writable directory on CAPY_LIBS")
	}

	// Determine target name.
	libName := asName
	if libName == "" {
		libName = inferLibName(source)
	}
	if libName == "" {
		return fmt.Errorf("could not infer library name from %q; use --as <name>", source)
	}
	target := filepath.Join(targetParent, libName)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("library %q already exists at %s (remove first or pass --as <other-name>)", libName, target)
	}

	// Local path? Just copy / symlink.
	if _, err := os.Stat(source); err == nil {
		if err := copyTree(source, target); err != nil {
			return fmt.Errorf("copy %s: %v", source, err)
		}
		fmt.Printf("✓ added local library %q at %s\n", libName, target)
		return nil
	}

	// Otherwise: git clone. Normalise common shorthand.
	url := source
	if !strings.Contains(url, "://") && !strings.HasPrefix(url, "git@") {
		url = "https://" + url
	}
	fmt.Printf("Cloning %s → %s\n", url, target)
	cmd := exec.Command("git", "clone", "--depth=1", "--quiet", url, target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %v", err)
	}
	// Sanity check: ensure there's a parseable library file inside.
	if _, ok := tryResolve(targetParent, libName); !ok {
		// Try lib.capy at the cloned root.
		if _, err := os.Stat(filepath.Join(target, "lib.capy")); err != nil {
			fmt.Fprintf(os.Stderr,
				"warning: cloned %q but found no <name>.capy / lib.capy entry point\n",
				libName)
		}
	}
	fmt.Printf("✓ added library %q\n", libName)
	return nil
}

// cmdLibRemove deletes an installed library from CAPY_LIBS. Used
// for cleanup or upgrades.
//
//	capy lib remove recipe
func cmdLibRemove(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: capy lib remove <name>")
	}
	name := args[0]
	// Find first dir that contains it.
	for _, dir := range libSearchPath() {
		candidate := filepath.Join(dir, name)
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			if err := os.RemoveAll(candidate); err != nil {
				return fmt.Errorf("remove %s: %v", candidate, err)
			}
			fmt.Printf("✓ removed %s\n", candidate)
			return nil
		}
		// Bare-file form.
		file := filepath.Join(dir, name+".capy")
		if _, err := os.Stat(file); err == nil {
			if err := os.Remove(file); err != nil {
				return fmt.Errorf("remove %s: %v", file, err)
			}
			fmt.Printf("✓ removed %s\n", file)
			return nil
		}
	}
	return fmt.Errorf("library %q not found on CAPY_LIBS", name)
}

// inferLibName tries to guess a library name from a URL or path.
//   github.com/user/repo       → repo
//   github.com/user/repo-capy  → repo  (strip -capy suffix)
//   ./libs/my-recipe           → my-recipe
//   https://gh.com/u/r.git     → r
func inferLibName(source string) string {
	base := source
	// Strip protocol.
	if i := strings.Index(base, "://"); i >= 0 {
		base = base[i+3:]
	}
	// Strip trailing .git.
	base = strings.TrimSuffix(base, ".git")
	// Trailing path component.
	base = filepath.Base(strings.TrimRight(base, "/\\"))
	// Strip common Capy-suffix conventions.
	base = strings.TrimSuffix(base, "-capy")
	base = strings.TrimSuffix(base, "_capy")
	return base
}

// copyTree recursively copies src → dst. Files only; preserves
// the relative tree shape but not perms / ownership.
func copyTree(src, dst string) error {
	st, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		// Single file copy.
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0644)
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// libraryNameLooksValid returns true when s is plausibly a library
// name (no path separators, no leading dots). Used in dispatch to
// disambiguate `capy <subcommand>` from `capy <lib>`.
func libraryNameLooksValid(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, ".") || strings.HasPrefix(s, "-") {
		return false
	}
	for _, c := range s {
		if c == '/' || c == '\\' || c == ' ' || c == ':' {
			return false
		}
	}
	return true
}
