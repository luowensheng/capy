// playground-bundle reads a curated list of samples from samples/ and
// writes a single JSON file the browser playground can fetch.
//
// Usage: go run ./cmd/playground-bundle > docs/assets/playground/samples.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CURATED is the list of samples surfaced in the playground UI. Order
// matters: the first entry is the default shown on page load.
var CURATED = []struct {
	ID          string
	Title       string
	Description string
	Hint        string // shown under the run button
}{
	{"recipe-card", "🍋 Recipe card",
		"Write a recipe in plain words; get a printable HTML recipe card.",
		"Try editing the title, ingredients, or steps."},
	{"event-invite", "🎉 Party invitation",
		"Declare a party; get a pastel HTML invitation card.",
		"Try changing the host, location, or RSVP date."},
	{"weekly-meal-plan", "📅 Weekly meal plan",
		"Seven dinners + notes → printable HTML grid for the fridge.",
		"Swap meals or add notes."},
	{"reading-log", "📚 Reading log",
		"A kid's reading list → bright HTML certificate with progress bar.",
		"Add more `book` lines; the progress bar updates."},
	{"interactive-breakout", "🕹️ Breakout game",
		"Declare entities + key bindings + event handlers → playable HTML5 Breakout.",
		"Try changing `lives 3` to `lives 5`, or rebind keys."},
	{"interactive-snake", "🐍 Snake game",
		"3-line config + key bindings + event handlers → playable HTML5 Snake.",
		"Try changing `tick every 110` to a smaller number for faster gameplay."},
}

type bundle struct {
	Samples []sample `json:"samples"`
}

type sample struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Hint        string `json:"hint"`
	Library     string `json:"library"`
	Script      string `json:"script"`
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	samplesDir := filepath.Join(root, "samples")

	var out bundle
	for _, c := range CURATED {
		libBytes, err := os.ReadFile(filepath.Join(samplesDir, c.ID, "lib.capy"))
		if err != nil {
			// Tolerate YAML libs by trying lib.yaml as a fallback.
			libBytes, err = os.ReadFile(filepath.Join(samplesDir, c.ID, "lib.yaml"))
		}
		if err != nil {
			fail(fmt.Errorf("%s: %v", c.ID, err))
		}
		scriptBytes, err := os.ReadFile(filepath.Join(samplesDir, c.ID, "script.capy"))
		if err != nil {
			fail(fmt.Errorf("%s/script.capy: %v", c.ID, err))
		}
		out.Samples = append(out.Samples, sample{
			ID:          c.ID,
			Title:       c.Title,
			Description: c.Description,
			Hint:        c.Hint,
			Library:     string(libBytes),
			Script:      string(scriptBytes),
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "playground-bundle:", err)
	os.Exit(1)
}
