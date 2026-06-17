// Package recipe is stage 1 of the pipeline: the pure domain model of a
// multi-lane recipe. It knows about lanes, steps, minute timings and the
// fork/merge relations between lanes — but nothing about pixels, colors as
// rendering, or SVG. It loads and validates that model from YAML.
package recipe

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// MaxLanes is the largest number of parallel lanes a recipe may declare.
const MaxLanes = 5

// Recipe is the root domain model parsed from a YAML config.
type Recipe struct {
	Title    string            `yaml:"title"`
	Subtitle string            `yaml:"subtitle"`
	Yield    int               `yaml:"yield"` // portions the amounts are written for; enables -portions
	Time     TimeAxis          `yaml:"time"`
	Lanes    []Lane            `yaml:"lanes"`
	Steps    []Step            `yaml:"steps"`
	Labels   map[string]string `yaml:"labels"` // overrides for generated strings (see i18n.go)

	L          Strings `yaml:"-"` // resolved generated labels; see Localize
	yieldLine  int     // source line of `yield:`, for diagnostics
	labelsLine int     // source line of `labels:`, for diagnostics
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

	explicitColor bool // true when the config set Color (themes honor it)
	line          int  // source line, for diagnostics
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

	line int // source line, for diagnostics
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

// Load reads a YAML recipe file, unmarshals it, records source lines for
// diagnostics, applies defaults and validates the result. On failure it returns
// a *Diagnostics whose message names the file and lines.
func Load(path string) (*Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read recipe: %w", err)
	}
	var r Recipe
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse recipe (%s): %w", path, err)
	}
	r.recordSourceLines(data)
	r.applyDefaults()
	if err := r.validate(path); err != nil {
		return nil, err
	}
	return &r, nil
}

// recordSourceLines walks the YAML node tree to tag lanes, steps and a couple
// of scalar keys with their source line, so validation can point at them.
func (r *Recipe) recordSourceLines(data []byte) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil || len(root.Content) == 0 {
		return
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(doc.Content); i += 2 {
		key, val := doc.Content[i], doc.Content[i+1]
		switch key.Value {
		case "yield":
			r.yieldLine = key.Line
		case "labels":
			r.labelsLine = key.Line
		case "lanes":
			for idx, item := range val.Content {
				if idx < len(r.Lanes) {
					r.Lanes[idx].line = item.Line
				}
			}
		case "steps":
			for idx, item := range val.Content {
				if idx < len(r.Steps) {
					r.Steps[idx].line = item.Line
				}
			}
		}
	}
}

// applyDefaults fills derived values that the config may omit: which colors were
// set explicitly, lane colors from the palette, the default generated strings,
// and the time axis bounds/tick from the steps.
func (r *Recipe) applyDefaults() {
	for i := range r.Lanes {
		r.Lanes[i].explicitColor = r.Lanes[i].Color != ""
		if r.Lanes[i].Color == "" && i < len(palette) {
			r.Lanes[i].Color = palette[i]
		}
	}

	// Default generated labels (German); -lang/labels: refine these via Localize.
	r.L = builtinLangs[DefaultLang]

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

// validate runs Validate and stamps the source file onto the diagnostics so the
// rendered error reads as "file:line: ...".
func (r *Recipe) validate(path string) error {
	err := r.Validate()
	if d, ok := err.(*Diagnostics); ok {
		d.Path = path
	}
	return err
}

// Validate checks structural and referential integrity of the recipe and
// reports *all* problems it finds at once (not just the first), as a
// *Diagnostics. It returns nil when the recipe is sound.
func (r *Recipe) Validate() error {
	d := &Diagnostics{}

	if len(r.Lanes) == 0 {
		d.add("recipe has no lanes (declare at least one under 'lanes:')")
	}
	if len(r.Lanes) > MaxLanes {
		d.addAt(r.lanesLine(MaxLanes), "recipe has %d lanes, the maximum is %d", len(r.Lanes), MaxLanes)
	}

	// Build the set of known lane IDs, checking uniqueness and color format.
	known := make(map[string]bool, len(r.Lanes))
	for _, l := range r.Lanes {
		if l.ID == "" {
			d.addAt(l.line, "lane with empty id (label %q): every lane needs a unique 'id:'", l.Label)
			continue
		}
		if known[l.ID] {
			d.addAt(l.line, "duplicate lane id %q: lane ids must be unique", l.ID)
		}
		known[l.ID] = true
		if l.Color != "" && !hexColor.MatchString(l.Color) {
			d.addAt(l.line, "lane %q: invalid color %q (want a 6-digit hex like #39b1d1)", l.ID, l.Color)
		}
	}

	if len(r.Steps) == 0 {
		d.add("recipe has no steps (add actions under 'steps:')")
	}

	// Validate each step and track which lanes actually carry a step.
	used := make(map[string]bool, len(r.Lanes))
	for i, s := range r.Steps {
		if !known[s.Lane] {
			d.addAt(s.line, "step %d: unknown lane %q (known lanes: %s)", i, s.Lane, joinQuoted(laneIDs(r.Lanes)))
		}
		used[s.Lane] = true
		if s.At < 0 {
			d.addAt(s.line, "step %d (lane %q): negative time %s", i, s.Lane, trimNum(s.At))
		}
		if s.MergeInto != "" && !known[s.MergeInto] {
			d.addAt(s.line, "step %d (lane %q): merge_into unknown lane %q", i, s.Lane, s.MergeInto)
		}
		if s.MergeInto == s.Lane && s.Lane != "" {
			d.addAt(s.line, "step %d (lane %q): merge_into its own lane (a lane cannot merge into itself)", i, s.Lane)
		}
		for _, f := range s.ForkTo {
			if !known[f] {
				d.addAt(s.line, "step %d (lane %q): fork_to unknown lane %q", i, s.Lane, f)
			}
		}
		// Catch broken quantity markup early, even without -portions, so a -check
		// run surfaces it.
		if hasQuantityMarkup(s.Text) {
			if err := validateQuantityMarkup(s.Text); err != nil {
				d.addAt(s.line, "step %d (lane %q): %v", i, s.Lane, err)
			}
		}
	}

	for _, l := range r.Lanes {
		if l.ID != "" && !used[l.ID] {
			d.addAt(l.line, "lane %q has no steps (every lane needs at least one step)", l.ID)
		}
	}

	if r.Yield < 0 {
		d.addAt(r.yieldLine, "yield %d is negative (use the number of portions the amounts are written for)", r.Yield)
	}

	// Validate label override keys here too, so typos fail at -check time.
	for k := range r.Labels {
		if k != "start" && k != "total" {
			d.addAt(r.labelsLine, "labels: unknown key %q (known keys: \"start\", \"total\")", k)
		}
	}

	if r.Time.To < r.Time.From {
		d.add("time axis 'to' (%s) is before 'from' (%s)", trimNum(r.Time.To), trimNum(r.Time.From))
	}
	return d.errorOrNil()
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

// TotalTime is the recipe's elapsed span on the time axis, used for the derived
// total-time caption in the header.
func (r *Recipe) TotalTime() float64 { return r.Time.To - r.Time.From }

// lanesLine returns the source line of the nth lane (for the "too many lanes"
// message), or 0 if unavailable.
func (r *Recipe) lanesLine(n int) int {
	if n < len(r.Lanes) {
		return r.Lanes[n].line
	}
	return 0
}

// laneIDs returns the distinct declared lane ids in order, for hint messages.
func laneIDs(lanes []Lane) []string {
	seen := make(map[string]bool, len(lanes))
	out := make([]string, 0, len(lanes))
	for _, l := range lanes {
		if l.ID != "" && !seen[l.ID] {
			seen[l.ID] = true
			out = append(out, l.ID)
		}
	}
	return out
}

// joinQuoted renders a list as `"a", "b", "c"` for error messages.
func joinQuoted(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(q, ", ")
}

// trimNum formats a float without a trailing ".0", for messages.
func trimNum(f float64) string {
	return strings.TrimSuffix(fmt.Sprintf("%g", f), ".0")
}
