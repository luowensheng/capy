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
	Types        map[string]RawType     `yaml:"types"`
	Context      map[string]interface{} `yaml:"context"`
	Functions    map[string]RawFunction `yaml:"functions"`
	FileTemplate string                 `yaml:"file_template"`
}

type RawFunction struct {
	Args     []RawArg  `yaml:"args"`
	Template string    `yaml:"template"`
	Block    *RawBlock `yaml:"block"`
	Run      string    `yaml:"run"`
	Priority int       `yaml:"priority"`
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
	Kind  string `yaml:"kind"`
	Value string `yaml:"value,omitempty"`
	Name  string `yaml:"name,omitempty"`
	Type  string `yaml:"type,omitempty"`
}

type RawType struct {
	Base    string   `yaml:"base"`
	Pattern string   `yaml:"pattern"`
	Options []string `yaml:"options"`
}

type YamlParser struct{}

func (YamlParser) ParseFile(path string) (RawLibrary, error) {
	var raw RawLibrary
	b, err := os.ReadFile(path)
	if err != nil {
		return raw, err
	}
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return raw, err
	}
	return raw, nil
}
