package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/olivierdevelops/capy/domain"
	orchfeatures "github.com/olivierdevelops/capy/orchestrator/features"
)

// resolveLibWithImpl resolves a library by name (or path) AND
// picks an implementation. Precedence (highest wins):
//
//  1. --impl <name> CLI flag (passed in as implFlag)
//  2. CAPY_IMPL_<UPPER-LIB-NAME> env var
//  3. CAPY_IMPL env var (generic)
//  4. Manifest's `default` directive
//  5. If the library declares exactly ONE impl, use it.
//
// Returns the absolute path to the impl's .capy file (which is
// what the loader actually reads to get FuncDefs / templates),
// plus the manifest path (so the caller can show "selected from"
// in --verbose modes).
//
// For libraries with no impl declarations, returns the manifest
// path as the impl path.
func resolveLibWithImpl(libName, implFlag string) (implPath, manifestPath string, libMeta domain.Library, err error) {
	manifestPath, err = resolveLib(libName)
	if err != nil {
		return "", "", domain.Library{}, err
	}
	// Load the manifest enough to see its declared impls.
	lex := orchfeatures.MakeLexer()
	loader := orchfeatures.MakeLibraryLoader(lex.Tokenize)
	libMeta, err = loader.Load(manifestPath)
	if err != nil {
		return "", "", domain.Library{}, err
	}
	if len(libMeta.Impls) == 0 {
		// No impls declared — the manifest file IS the library.
		return manifestPath, manifestPath, libMeta, nil
	}

	// Pick.
	wanted := implFlag
	if wanted == "" {
		// Env-var per-library.
		if v := os.Getenv("CAPY_IMPL_" + envUpper(libMeta.LibName)); v != "" {
			wanted = v
		}
	}
	if wanted == "" {
		if v := os.Getenv("CAPY_IMPL"); v != "" {
			wanted = v
		}
	}
	if wanted == "" {
		wanted = libMeta.DefaultImpl
	}
	if wanted == "" && len(libMeta.Impls) == 1 {
		// Only one declared — use it.
		for n := range libMeta.Impls {
			wanted = n
		}
	}
	if wanted == "" {
		return "", manifestPath, libMeta, fmt.Errorf(
			"library %q has multiple impls but no default; pass --impl <name> (choices: %s)",
			libName, strings.Join(sortedImplKeys(libMeta.Impls), ", "))
	}
	im, ok := libMeta.Impls[wanted]
	if !ok {
		return "", manifestPath, libMeta, fmt.Errorf(
			"library %q has no impl %q (choices: %s)",
			libName, wanted, strings.Join(sortedImplKeys(libMeta.Impls), ", "))
	}
	// Resolve impl file path (relative to the manifest's dir).
	implPath = im.File
	if !filepath.IsAbs(implPath) {
		implPath = filepath.Join(filepath.Dir(manifestPath), implPath)
	}
	libMeta.SelectedImpl = wanted
	return implPath, manifestPath, libMeta, nil
}

func envUpper(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z':
			out = append(out, c-32)
		case c == '-' || c == ' ':
			out = append(out, '_')
		default:
			out = append(out, c)
		}
	}
	return string(out)
}

func sortedImplKeys(m map[string]*domain.ImplDef) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// Tiny sort.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
