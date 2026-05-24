package infra

import (
	"os"

	"gopkg.in/yaml.v3"
)

// RawLibrary is the YAML DTO. The orchestrator maps it into a domain.Library.
//
// Top-level YAML keys this DTO accepts:
//
//	extension, output_file, types, context, functions, file_template
//
// Configurable surface syntax (block delimiters, statement terminator, arg
// separator) is deferred to a future version. For now, blocks are
// INDENT/DEDENT or `{...}`, statements end at NEWLINE, args are
// whitespace-separated.
type RawLibrary struct {
	Extension    string                 `yaml:"extension"`
	OutputFile   string                 `yaml:"output_file"`
	Description  string                 `yaml:"description,omitempty"`
	Types        map[string]RawType     `yaml:"types"`
	Context      map[string]interface{} `yaml:"context"`
	Functions    map[string]RawFunction `yaml:"functions"`
	FileTemplate string                 `yaml:"file_template"`

	// Multi-file output. Map of relative-path → Go template body.
	Files map[string]string `yaml:"files,omitempty"`

	// Imports — relative paths to other library files whose functions,
	// types, and context get merged in before this library's own
	// declarations (which take precedence on conflict).
	Imports []string `yaml:"import,omitempty"`

	// Preprocess — names of source-level inclusion directives this
	// library opts into (e.g. ["@import", "@include"]). When empty,
	// no preprocessing runs. Keeps "zero predefined grammar" honest.
	Preprocess []string `yaml:"preprocess,omitempty"`

	// Comments — line-comment markers the library opts into for
	// user scripts. Empty list → user scripts have NO comment
	// syntax. Mirrors Preprocess: the engine ships zero defaults
	// and the library must declare what to recognise.
	Comments []string `yaml:"comments,omitempty"`

	// Manifest fields (set when the library declares them at the
	// top level of its .capy file — `name "X"`, `version "X"`).
	LibName    string `yaml:"name,omitempty"`
	LibVersion string `yaml:"version,omitempty"`

	// Commands declared by the library — `command "X" ... end`
	// blocks at the top level. The body is collected as raw text;
	// the loader parses it via the inner-DSL parser.
	Commands map[string]RawCommand `yaml:"commands,omitempty"`

	// Implementations: `impl "NAME" "FILE" ... end` blocks at the
	// top level. Each entry points at a sibling .capy file that
	// provides the actual functions. The CLI picks one per
	// invocation via --impl / CAPY_IMPL / default. When the map
	// is non-empty, the manifest file itself only carries
	// metadata + commands; the real library lives in the chosen
	// impl file.
	Impls map[string]RawImpl `yaml:"impls,omitempty"`

	// DefaultImpl names the impl chosen when neither --impl nor
	// CAPY_IMPL picks one. If empty + Impls is non-empty + only
	// one impl declared, that one is used; otherwise an error.
	DefaultImpl string `yaml:"default_impl,omitempty"`
}

// RawImpl is one `impl "NAME" "FILE" ... end` declaration.
type RawImpl struct {
	Name        string `yaml:"name"`
	File        string `yaml:"file"`
	Description string `yaml:"description,omitempty"`
	Version     string `yaml:"version,omitempty"`
	IsDefault   bool   `yaml:"default,omitempty"`
}

// RawCommand is one `command "X" ... end` declaration.
type RawCommand struct {
	Description string          `yaml:"description,omitempty"`
	Body        string          `yaml:"body,omitempty"`
	Args        []RawCommandArg `yaml:"args,omitempty"`
	Flags       []RawCommandFlag `yaml:"flags,omitempty"`
}

// RawCommandArg is a positional argument declaration:
//   arg "name" required "description"
//   arg "outpath" optional "Output path. Default: stdout."
type RawCommandArg struct {
	Name        string `yaml:"name"`
	Required    bool   `yaml:"required,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// RawCommandFlag is a flag declaration:
//   flag "--port" "Listen port" default "8080"
//   flag "--verbose" bool "Verbose output"
type RawCommandFlag struct {
	Name        string `yaml:"name"`              // e.g. "--port"
	Description string `yaml:"description,omitempty"`
	Default     string `yaml:"default,omitempty"` // empty = no default
	IsBool      bool   `yaml:"bool,omitempty"`    // presence-only flag
}

type RawFunction struct {
	Description string    `yaml:"description,omitempty"`
	Args        []RawArg  `yaml:"args"`
	Template    string    `yaml:"template"`
	Block       *RawBlock `yaml:"block"`
	Run         string    `yaml:"run"`
	Priority    int       `yaml:"priority"`

	// Body is the new-shape unified function body (inner-DSL
	// statements including `write` calls). When non-empty, the loader
	// translates it into Template + Run before constructing the
	// FuncDef. Mutually exclusive with Template + Run.
	Body string `yaml:"body,omitempty"`
}

type RawBlock struct {
	Closer string `yaml:"closer"`
	Open   string `yaml:"open"`
	Close  string `yaml:"close"`
}

// RawArg is an args-list entry. The Kind discriminator is required and decides
// which other fields are valid:
//
//	kind: literal  → value: "TEXT"
//	kind: capture  → name: NAME, type: TYPE
type RawArg struct {
	Kind        string `yaml:"kind"`
	Value       string `yaml:"value,omitempty"`
	Name        string `yaml:"name,omitempty"`
	Type        string `yaml:"type,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type RawType struct {
	Description string   `yaml:"description,omitempty"`
	Base        string   `yaml:"base"`
	Pattern     string   `yaml:"pattern"`
	Options     []string `yaml:"options"`
}

type YamlParser struct{}

func (YamlParser) ParseFile(path string) (RawLibrary, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return RawLibrary{}, err
	}
	return YamlParser{}.ParseBytes(b)
}

// ParseBytes parses a library directly from in-memory bytes, no filesystem
// round-trip. Used by the embedding API (top-level `capy` package).
func (YamlParser) ParseBytes(b []byte) (RawLibrary, error) {
	var raw RawLibrary
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return raw, err
	}
	return raw, nil
}
