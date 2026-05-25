package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// libSearchPath returns the resolved CAPY_LIBS list. When the
// `CAPY_LIBS` env var is set, it overrides everything (treat it as
// the full list). When it's not set, the current working directory
// is the first entry, followed by XDG-style per-platform defaults —
// so a local `interface.capy` next to your script is discoverable
// out of the box without any environment setup.
func libSearchPath() []string {
	if env := os.Getenv("CAPY_LIBS"); env != "" {
		sep := ":"
		if runtime.GOOS == "windows" {
			sep = ";"
		}
		var out []string
		for _, p := range strings.Split(env, sep) {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	// Defaults. CWD goes first so a project's local library always
	// wins over a globally-installed one of the same name.
	out := []string{}
	if cwd, err := os.Getwd(); err == nil {
		out = append(out, cwd)
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		out = append(out, filepath.Join(home, "Library", "Application Support", "Capy", "libs"))
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			appdata = filepath.Join(home, "AppData", "Roaming")
		}
		out = append(out, filepath.Join(appdata, "Capy", "libs"))
	default:
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg == "" {
			xdg = filepath.Join(home, ".config")
		}
		out = append(out,
			filepath.Join(xdg, "capy", "libs"),
			filepath.Join(home, ".capy", "libs"),
		)
	}
	return out
}

// resolveLib tries to find a library by name on the search path.
// Returns the path to the library file (the .capy file the loader
// will read). Search rules:
//
//  1. Direct file: `${name}.capy` in any search dir.
//  2. Directory form: `${name}/${name}.capy` (a library with a
//     manifest is conventionally in a directory whose name matches
//     the library; the entry point is the same name + .capy).
//  3. Directory + lib.capy: `${name}/lib.capy` (convention from
//     before manifests).
//
// Returns empty + nil if not found (caller decides whether that's
// an error).
func resolveLib(name string) (string, error) {
	for _, dir := range libSearchPath() {
		if path, ok := tryResolve(dir, name); ok {
			return path, nil
		}
	}
	// Fallback: current working directory.
	if path, ok := tryResolve(".", name); ok {
		return path, nil
	}
	return "", fmt.Errorf("library %q not found on CAPY_LIBS (%s)", name,
		strings.Join(libSearchPath(), string(os.PathListSeparator)))
}

func tryResolve(dir, name string) (string, bool) {
	candidates := []string{
		filepath.Join(dir, name+".capy"),
		filepath.Join(dir, name, name+".capy"),
		filepath.Join(dir, name, "lib.capy"),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}
	return "", false
}

// listInstalledLibs walks every search path and returns (name → path)
// for each library it finds. Used by `capy lib list`.
func listInstalledLibs() map[string]string {
	out := map[string]string{}
	for _, dir := range libSearchPath() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() {
				inner := filepath.Join(dir, name, name+".capy")
				if _, err := os.Stat(inner); err == nil {
					if _, ok := out[name]; !ok {
						out[name] = inner
					}
					continue
				}
				inner = filepath.Join(dir, name, "lib.capy")
				if _, err := os.Stat(inner); err == nil {
					if _, ok := out[name]; !ok {
						out[name] = inner
					}
				}
				continue
			}
			if strings.HasSuffix(name, ".capy") {
				libname := strings.TrimSuffix(name, ".capy")
				if _, ok := out[libname]; !ok {
					out[libname] = filepath.Join(dir, name)
				}
			}
		}
	}
	return out
}
