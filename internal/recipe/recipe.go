// Package recipe is stage 1 of the pipeline: the pure domain model of a
// multi-lane recipe. It knows about lanes, steps, minute timings and the
// fork/merge relations between lanes — but nothing about pixels, colors as
// rendering, or SVG. It loads and validates that model from YAML.
package recipe

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// MaxLanes is the largest number of parallel lanes a recipe may declare.
const MaxLanes = 5

// Recipe is the root domain model parsed from a YAML config.
type Recipe struct {
	Title    string   `yaml:"title"`
	Subtitle string   `yaml:"subtitle"`
	Time     TimeAxis `yaml:"time"`
	Lanes    []Lane   `yaml:"lanes"`
	Steps    []Step   `yaml:"steps"`
}

// TimeAxis describes the vertical (time) axis. When omitted in the config,
// From/To are derived from the steps and Tick defaults to 1.
type TimeAxis struct {
	From float64 `yaml:"from"`
	To   float64 `yaml:"to"`
	Tick float64 `yaml:"tick"`
	Unit string  `yaml:"unit"`
}

// Lane is one parallel cooking process. Lane order in the config is the
// left-to-right column order in the rendered diagram.
type Lane struct {
	ID    string `yaml:"id"`
	Label string `yaml:"label"`
	Color string `yaml:"color"` // optional "#rrggbb"; filled from palette if empty
}

// Step is a single action on the timeline, anchored to a lane and a minute.
type Step struct {
	Lane string  `yaml:"lane"`
	At   float64 `yaml:"at"`
	Text string  `yaml:"text"`

	// ForkTo lists lanes that branch off from this step (one process splits).
	ForkTo []string `yaml:"fork_to"`
	// MergeInto names a lane this step's lane joins here (processes converge).
	MergeInto string `yaml:"merge_into"`
	// ArrowLabel annotates the fork/merge arrow, e.g. "Nudeln in Pfanne".
	ArrowLabel string `yaml:"arrow_label"`
}

// palette holds the default lane colors, used for lanes that do not set their
// own. It is the ColorHunt set 39b1d1-d6fb61-f6850c-de3e3e, extended with a
// vibrant violet as the fifth lane. Length is MaxLanes.
var palette = [MaxLanes]string{
	"#39b1d1", // sky blue
	"#d6fb61", // lime
	"#f6850c", // orange
	"#de3e3e", // red
	"#8b5cf6", // violet (extends the four-colour ColorHunt set)
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Load reads a YAML recipe file, unmarshals it, applies defaults and validates
// the result.
func Load(path string) (*Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read recipe: %w", err)
	}
	var r Recipe
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse recipe: %w", err)
	}
	r.applyDefaults()
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

// applyDefaults fills derived values that the config may omit: lane colors from
// the palette and the time axis bounds/tick from the steps.
func (r *Recipe) applyDefaults() {
	for i := range r.Lanes {
		if r.Lanes[i].Color == "" && i < len(palette) {
			r.Lanes[i].Color = palette[i]
		}
	}

	if r.Time.Tick <= 0 {
		r.Time.Tick = 1
	}
	if r.Time.Unit == "" {
		r.Time.Unit = "min"
	}
	// Derive From/To from steps when the axis is left at its zero value.
	if r.Time.From == 0 && r.Time.To == 0 && len(r.Steps) > 0 {
		min, max := r.Steps[0].At, r.Steps[0].At
		for _, s := range r.Steps {
			if s.At < min {
				min = s.At
			}
			if s.At > max {
				max = s.At
			}
		}
		r.Time.From, r.Time.To = min, max
	}
}

// Validate checks structural and referential integrity of the recipe.
func (r *Recipe) Validate() error {
	if len(r.Lanes) == 0 {
		return fmt.Errorf("recipe has no lanes")
	}
	if len(r.Lanes) > MaxLanes {
		return fmt.Errorf("recipe has %d lanes, maximum is %d", len(r.Lanes), MaxLanes)
	}

	// Build the set of known lane IDs, checking uniqueness and color format.
	known := make(map[string]bool, len(r.Lanes))
	for _, l := range r.Lanes {
		if l.ID == "" {
			return fmt.Errorf("lane with empty id (label %q)", l.Label)
		}
		if known[l.ID] {
			return fmt.Errorf("duplicate lane id %q", l.ID)
		}
		known[l.ID] = true
		if l.Color != "" && !hexColor.MatchString(l.Color) {
			return fmt.Errorf("lane %q: invalid color %q (want #rrggbb)", l.ID, l.Color)
		}
	}

	if len(r.Steps) == 0 {
		return fmt.Errorf("recipe has no steps")
	}

	// Validate each step and track which lanes actually carry a step.
	used := make(map[string]bool, len(r.Lanes))
	for i, s := range r.Steps {
		if !known[s.Lane] {
			return fmt.Errorf("step %d: unknown lane %q", i, s.Lane)
		}
		used[s.Lane] = true
		if s.At < 0 {
			return fmt.Errorf("step %d (lane %q): negative time %g", i, s.Lane, s.At)
		}
		if s.MergeInto != "" && !known[s.MergeInto] {
			return fmt.Errorf("step %d (lane %q): merge_into unknown lane %q", i, s.Lane, s.MergeInto)
		}
		if s.MergeInto == s.Lane {
			return fmt.Errorf("step %d (lane %q): merge_into its own lane", i, s.Lane)
		}
		for _, f := range s.ForkTo {
			if !known[f] {
				return fmt.Errorf("step %d (lane %q): fork_to unknown lane %q", i, s.Lane, f)
			}
		}
	}

	for _, l := range r.Lanes {
		if !used[l.ID] {
			return fmt.Errorf("lane %q has no steps", l.ID)
		}
	}

	if r.Time.To < r.Time.From {
		return fmt.Errorf("time axis to (%g) is before from (%g)", r.Time.To, r.Time.From)
	}
	return nil
}

// LaneIndex returns the column index of the lane with the given id, or -1.
func (r *Recipe) LaneIndex(id string) int {
	for i, l := range r.Lanes {
		if l.ID == id {
			return i
		}
	}
	return -1
}

// LaneColor returns the resolved color for a lane id (empty string if unknown).
func (r *Recipe) LaneColor(id string) string {
	if i := r.LaneIndex(id); i >= 0 {
		return r.Lanes[i].Color
	}
	return ""
}
