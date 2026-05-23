package orchfeatures

import (
	"fmt"
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
func MakeLibraryLoader(yp infra.YamlParser, tokenize func(string) ([]domain.Token, error)) features.LibraryLoader {
	cp := infra.CapyLibParser{}
	return features.LibraryLoader{
		Load: func(path string) (domain.Library, error) {
			var raw infra.RawLibrary
			var err error
			if strings.HasSuffix(strings.ToLower(path), ".capy") {
				raw, err = cp.ParseFile(path)
			} else {
				raw, err = yp.ParseFile(path)
			}
			if err != nil {
				return domain.Library{}, err
			}
			return mapLibrary(raw, tokenize)
		},
	}
}

func mapLibrary(r infra.RawLibrary, tokenize func(string) ([]domain.Token, error)) (domain.Library, error) {
	lib := domain.Library{
		Extension:    r.Extension,
		OutputFile:   r.OutputFile,
		Functions:    map[string]*domain.FuncDef{},
		Types:        map[string]domain.TypeDef{},
		Context:      r.Context,
		FileTemplate: r.FileTemplate,
	}
	if lib.Context == nil {
		lib.Context = map[string]any{}
	}
	if lib.FileTemplate == "" {
		lib.FileTemplate = "{{ .body }}"
	}

	for name, t := range r.Types {
		lib.Types[name] = domain.TypeDef{
			Name:    name,
			Base:    t.Base,
			Pattern: t.Pattern,
			Options: t.Options,
		}
	}

	for name, f := range r.Functions {
		fd, err := compileFunction(name, f, tokenize)
		if err != nil {
			return lib, err
		}
		lib.Functions[name] = fd
	}

	// Validate cross-references after all functions are loaded.
	for _, fd := range lib.Functions {
		for _, a := range fd.Args {
			if a.Kind == "capture" && !validType(a.Type, lib.Types) {
				return lib, fmt.Errorf("function %q: capture %q has unknown type %q", fd.Name, a.Name, a.Type)
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

	fd := &domain.FuncDef{
		Name:     name,
		Args:     args,
		Elements: elements,
		Template: f.Template,
		Run:      f.Run,
		Priority: f.Priority,
	}
	if f.Block != nil {
		fd.Block = &domain.BlockSpec{Closer: f.Block.Closer, Open: f.Block.Open, Close: f.Block.Close}
	}

	if strings.TrimSpace(f.Run) != "" {
		toks, err := tokenize(f.Run)
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
			out = append(out, domain.ArgEntry{Kind: "literal", Value: r.Value})
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
			out = append(out, domain.ArgEntry{Kind: "capture", Name: r.Name, Type: t})
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

func validType(t string, types map[string]domain.TypeDef) bool {
	switch t {
	case "any", "ident", "raw", "string", "int", "float", "bool":
		return true
	}
	_, ok := types[t]
	return ok
}
