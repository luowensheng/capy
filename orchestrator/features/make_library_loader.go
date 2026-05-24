package orchfeatures

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/luowensheng/capy/domain"
	"github.com/luowensheng/capy/features"
	"github.com/luowensheng/capy/infra"
)

// MakeLibraryLoader compiles a library file into a domain.Library. It accepts
// either YAML (`.yaml`, `.yml`) or Capy-native (`.capy`) library files; both
// formats produce the same in-memory RawLibrary DTO and are mapped via
// mapLibrary.
//
//   - args list → []ArgEntry → []PatternElement
//   - run: snippet → InnerBlock AST (parsed via the outer lexer + inner parser)
//   - types/context/file_template carried through
// LoadLibraryFromBytes compiles an in-memory library — YAML or Capy-native,
// detected by the `format` argument ("yaml" or "capy") — into a usable
// domain.Library. Public so the top-level `capy` package can embed Capy
// without filesystem round-trips.
func LoadLibraryFromBytes(format string, src []byte, tokenize func(string) ([]domain.Token, error)) (domain.Library, error) {
	var raw infra.RawLibrary
	var err error
	switch strings.ToLower(format) {
	case "capy":
		raw, err = infra.CapyLibParser{}.ParseBytes(src)
	case "yaml", "yml", "":
		raw, err = infra.YamlParser{}.ParseBytes(src)
	default:
		return domain.Library{}, fmt.Errorf("unknown library format %q (want \"yaml\" or \"capy\")", format)
	}
	if err != nil {
		return domain.Library{}, err
	}
	return mapLibrary(raw, tokenize)
}

func MakeLibraryLoader(yp infra.YamlParser, tokenize func(string) ([]domain.Token, error)) features.LibraryLoader {
	return features.LibraryLoader{
		Load: func(path string) (domain.Library, error) {
			raw, err := loadRawWithImports(path, map[string]bool{}, yp, infra.CapyLibParser{})
			if err != nil {
				return domain.Library{}, err
			}
			return mapLibrary(raw, tokenize)
		},
	}
}

// loadRawWithImports parses one library file and recursively pulls in any
// `import` directives, merging them into the result. The IMPORTING file
// wins on conflict (so local overrides shadow imports). Cycles error.
//
// Import paths are resolved relative to the file containing the `import`.
func loadRawWithImports(path string, visited map[string]bool, yp infra.YamlParser, cp infra.CapyLibParser) (infra.RawLibrary, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return infra.RawLibrary{}, err
	}
	if visited[abs] {
		return infra.RawLibrary{}, fmt.Errorf("import cycle detected at %s", path)
	}
	visited[abs] = true

	var raw infra.RawLibrary
	if strings.HasSuffix(strings.ToLower(path), ".capy") {
		raw, err = cp.ParseFile(path)
	} else {
		raw, err = yp.ParseFile(path)
	}
	if err != nil {
		return raw, err
	}

	if len(raw.Imports) == 0 {
		return raw, nil
	}

	// Start from a merged-imports base; the local raw overrides at the end.
	dir := filepath.Dir(abs)
	merged := infra.RawLibrary{
		Functions: map[string]infra.RawFunction{},
		Types:     map[string]infra.RawType{},
		Context:   map[string]any{},
		Files:     map[string]string{},
	}
	for _, imp := range raw.Imports {
		impPath := imp
		if !filepath.IsAbs(impPath) {
			impPath = filepath.Join(dir, imp)
		}
		impRaw, err := loadRawWithImports(impPath, visited, yp, cp)
		if err != nil {
			return raw, fmt.Errorf("import %q: %v", imp, err)
		}
		mergeRaw(&merged, impRaw)
	}
	// Local raw wins on conflict.
	mergeRaw(&merged, raw)
	return merged, nil
}

// mergeRaw copies entries from src into dst. Existing keys in dst are
// OVERRIDDEN — call order determines precedence (last-write-wins).
func mergeRaw(dst *infra.RawLibrary, src infra.RawLibrary) {
	if src.Extension != "" {
		dst.Extension = src.Extension
	}
	if src.OutputFile != "" {
		dst.OutputFile = src.OutputFile
	}
	if src.FileTemplate != "" {
		dst.FileTemplate = src.FileTemplate
	}
	for k, v := range src.Functions {
		dst.Functions[k] = v
	}
	for k, v := range src.Types {
		dst.Types[k] = v
	}
	for k, v := range src.Context {
		dst.Context[k] = v
	}
	for k, v := range src.Files {
		dst.Files[k] = v
	}
	// Preprocess directives are unioned; imports widen what the
	// downstream library can opt into without overriding.
	for _, d := range src.Preprocess {
		seen := false
		for _, e := range dst.Preprocess {
			if e == d {
				seen = true
				break
			}
		}
		if !seen {
			dst.Preprocess = append(dst.Preprocess, d)
		}
	}
}

func mapLibrary(r infra.RawLibrary, tokenize func(string) ([]domain.Token, error)) (domain.Library, error) {
	// Detect the new-shape sentinel that the .capy parser stashes on
	// FileTemplate / Files entries and translate the body before the
	// engine sees it. Format: "\x00NEW_SHAPE\x00<inner-dsl source>".
	const sentinel = "\x00NEW_SHAPE\x00"
	translateMaybeNew := func(s, label string) (string, error) {
		if !strings.HasPrefix(s, sentinel) {
			return s, nil
		}
		body := s[len(sentinel):]
		toks, err := tokenize(body)
		if err != nil {
			return "", fmt.Errorf("%s: parsing body: %v", label, err)
		}
		ast, err := ParseInner(toks)
		if err != nil {
			return "", fmt.Errorf("%s: parsing body: %v", label, err)
		}
		tpl, residual, err := translateNewShape(ast)
		if err != nil {
			return "", fmt.Errorf("%s: %v", label, err)
		}
		if len(residual.Stmts) > 0 {
			// file_template / file blocks can't mutate state — there
			// are no per-statement matches happening at this point; the
			// context is already finalised. Reject pure state mutations.
			return "", fmt.Errorf("%s: state-mutation statements (set/append/…) aren't allowed here — file blocks only render", label)
		}
		return tpl, nil
	}
	ft, err := translateMaybeNew(r.FileTemplate, "file_template")
	if err != nil {
		return domain.Library{}, err
	}
	files := r.Files
	if len(files) > 0 {
		newFiles := make(map[string]string, len(files))
		for path, body := range files {
			translated, err := translateMaybeNew(body, "file "+strconv.Quote(path))
			if err != nil {
				return domain.Library{}, err
			}
			newFiles[path] = translated
		}
		files = newFiles
	}
	lib := domain.Library{
		Extension:    r.Extension,
		OutputFile:   r.OutputFile,
		Description:  r.Description,
		Functions:    map[string]*domain.FuncDef{},
		Types:        map[string]domain.TypeDef{},
		Context:      r.Context,
		FileTemplate: ft,
		Files:        files,
		Preprocess:   r.Preprocess,
		Commands:     map[string]*domain.CommandDef{},
		LibName:      r.LibName,
		LibVersion:   r.LibVersion,
	}
	if lib.Files == nil {
		lib.Files = map[string]string{}
	}
	if lib.Context == nil {
		lib.Context = map[string]any{}
	}
	if lib.FileTemplate == "" {
		lib.FileTemplate = "{{ .body }}"
	}

	for name, t := range r.Types {
		lib.Types[name] = domain.TypeDef{
			Name:        name,
			Description: t.Description,
			Base:        t.Base,
			Pattern:     t.Pattern,
			Options:     t.Options,
		}
	}

	for name, f := range r.Functions {
		fd, err := compileFunction(name, f, tokenize)
		if err != nil {
			return lib, err
		}
		lib.Functions[name] = fd
	}

	// Compile library commands (declared via `command "X" ... end`
	// blocks in the .capy manifest). Body is inner-DSL with shell-
	// like primitives the evaluator surfaces.
	for name, c := range r.Commands {
		cd := &domain.CommandDef{
			Name:        name,
			Description: c.Description,
			BodyRaw:     c.Body,
		}
		if strings.TrimSpace(c.Body) != "" {
			toks, err := tokenize(c.Body)
			if err != nil {
				return lib, fmt.Errorf("command %q: parsing body: %v", name, err)
			}
			ast, err := ParseInner(toks)
			if err != nil {
				return lib, fmt.Errorf("command %q: parsing body: %v", name, err)
			}
			cd.Body = ast
		}
		lib.Commands[name] = cd
	}

	// Validate cross-references after all functions are loaded.
	for _, fd := range lib.Functions {
		for _, a := range fd.Args {
			if a.Kind == "capture" && !validType(a.Type, lib.Types) {
				ce := &domain.CapyError{Msg: fmt.Sprintf("function %q: capture %q has unknown type %q", fd.Name, a.Name, a.Type)}
				// Suggest the closest known type (built-ins + library-declared).
				cands := []string{"any", "ident", "raw", "string", "int", "float", "bool"}
				for n := range lib.Types {
					cands = append(cands, n)
				}
				if best := domain.SuggestClosest(a.Type, cands, 2); best != "" {
					ce.Hint = fmt.Sprintf("did you mean %q?", best)
				} else {
					ce.Hint = fmt.Sprintf("built-in types: any, ident, raw, string, int, float, bool; declared types: %v", typeNames(lib.Types))
				}
				return lib, ce
			}
		}
		if fd.Block != nil {
			// Two modes: named-closer OR delimiter pair. Exactly one must be set.
			hasCloser := fd.Block.Closer != ""
			hasDelim := fd.Block.Open != "" && fd.Block.Close != ""
			if hasCloser == hasDelim {
				return lib, fmt.Errorf("function %q: block must set either `closer:` OR both `open:`/`close:`", fd.Name)
			}
			if hasCloser {
				if _, ok := lib.Functions[fd.Block.Closer]; !ok {
					return lib, fmt.Errorf("function %q: block.closer %q not found", fd.Name, fd.Block.Closer)
				}
			}
		}
	}

	return lib, nil
}

func compileFunction(name string, f infra.RawFunction, tokenize func(string) ([]domain.Token, error)) (*domain.FuncDef, error) {
	args, err := compileArgs(f.Args, name)
	if err != nil {
		return nil, err
	}
	// Auto-name-prepend rule.
	hasLiteral := false
	for _, a := range args {
		if a.Kind == "literal" {
			hasLiteral = true
			break
		}
	}
	if !hasLiteral {
		args = append([]domain.ArgEntry{{Kind: "literal", Value: name}}, args...)
	}

	elements := compileElements(args)

	// New-shape `body:` (unified write/state block) → translate into
	// the equivalent Template + Run before constructing the FuncDef.
	template := f.Template
	run := f.Run
	if strings.TrimSpace(f.Body) != "" {
		toks, err := tokenize(f.Body)
		if err != nil {
			return nil, fmt.Errorf("function %q: parsing body: %v", name, err)
		}
		ast, err := ParseInner(toks)
		if err != nil {
			return nil, fmt.Errorf("function %q: parsing body: %v", name, err)
		}
		tpl, runAST, err := translateNewShape(ast)
		if err != nil {
			return nil, fmt.Errorf("function %q: %v", name, err)
		}
		template = tpl
		// Re-serialise the run AST as inner-DSL source text — the
		// later "if RunAST is non-empty, tokenize+parse" path below
		// re-parses it. (Slightly wasteful but keeps one code path.)
		run = renderInnerBlock(runAST)
	}

	fd := &domain.FuncDef{
		Name:        name,
		Description: f.Description,
		Args:        args,
		Elements:    elements,
		Template:    template,
		Run:         run,
		Priority:    f.Priority,
	}
	if f.Block != nil {
		fd.Block = &domain.BlockSpec{Closer: f.Block.Closer, Open: f.Block.Open, Close: f.Block.Close}
	}

	if strings.TrimSpace(run) != "" {
		toks, err := tokenize(run)
		if err != nil {
			return nil, fmt.Errorf("function %q: parsing run: %v", name, err)
		}
		ast, err := ParseInner(toks)
		if err != nil {
			return nil, fmt.Errorf("function %q: parsing run: %v", name, err)
		}
		fd.RunAST = &ast
	}

	return fd, nil
}

func compileArgs(raws []infra.RawArg, fname string) ([]domain.ArgEntry, error) {
	var out []domain.ArgEntry
	for i, r := range raws {
		switch r.Kind {
		case "literal":
			if r.Value == "" {
				return nil, fmt.Errorf("function %q arg %d: kind=literal requires value", fname, i)
			}
			if r.Name != "" || r.Type != "" {
				return nil, fmt.Errorf("function %q arg %d: kind=literal cannot have name/type", fname, i)
			}
			out = append(out, domain.ArgEntry{Kind: "literal", Value: r.Value, Description: r.Description})
		case "capture":
			if r.Name == "" {
				return nil, fmt.Errorf("function %q arg %d: kind=capture requires name", fname, i)
			}
			if r.Value != "" {
				return nil, fmt.Errorf("function %q arg %d: kind=capture cannot have value", fname, i)
			}
			t := r.Type
			if t == "" {
				t = "any"
			}
			out = append(out, domain.ArgEntry{Kind: "capture", Name: r.Name, Type: t, Description: r.Description})
		default:
			return nil, fmt.Errorf("function %q arg %d: unknown or missing kind %q (must be \"literal\" or \"capture\")", fname, i, r.Kind)
		}
	}
	return out, nil
}

func compileElements(args []domain.ArgEntry) []domain.PatternElement {
	var out []domain.PatternElement
	for _, a := range args {
		if a.Kind == "literal" {
			// Split on `.` so dotted names (scene.create_sphere) match how the
			// outer lexer tokenises them.
			out = append(out, splitLiteral(a.Value)...)
		} else {
			out = append(out, domain.PatternElement{IsCapture: true, Name: a.Name, CapType: a.Type})
		}
	}
	return out
}

// splitLiteral converts "scene.create_sphere" into 3 PatternElements; non-dotted
// literals (including multi-char operators like `:=`, `->`) pass through whole.
func splitLiteral(lit string) []domain.PatternElement {
	parts := []string{}
	cur := strings.Builder{}
	flush := func() {
		if cur.Len() > 0 {
			parts = append(parts, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(lit); i++ {
		c := lit[i]
		if c == '.' {
			flush()
			parts = append(parts, ".")
			continue
		}
		cur.WriteByte(c)
	}
	flush()
	out := make([]domain.PatternElement, 0, len(parts))
	for _, p := range parts {
		out = append(out, domain.PatternElement{Literal: p})
	}
	return out
}

// typeNames returns the sorted list of declared type names; used in error
// hints when a capture references an unknown type.
func typeNames(types map[string]domain.TypeDef) []string {
	out := make([]string, 0, len(types))
	for n := range types {
		out = append(out, n)
	}
	return out
}

func validType(t string, types map[string]domain.TypeDef) bool {
	switch t {
	case "any", "ident", "raw", "string", "int", "float", "bool":
		return true
	}
	_, ok := types[t]
	return ok
}
