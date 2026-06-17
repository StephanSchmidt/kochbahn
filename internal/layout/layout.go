// Package layout is stage 2 of the pipeline: a renderer-agnostic geometric
// model. The engine turns a recipe.Recipe into a Layout — a flat collection of
// positioned visual primitives (rails, nodes, labels, connectors, ticks,
// headers) carrying absolute pixel coordinates and resolved colors. It knows
// nothing about SVG, so any backend (SVG, PNG, canvas) can consume a Layout.
package layout

// ConnKind classifies a connector so the renderer can style it.
type ConnKind int

const (
	// Sequence is the in-lane link between consecutive steps.
	Sequence ConnKind = iota
	// Fork is an arrow from a step to a lane that branches off it.
	Fork
	// Merge is an arrow from a step to the lane its own lane joins.
	Merge
)

// Layout is the complete set of primitives to draw, in document coordinates.
type Layout struct {
	Width, Height   float64
	Title, Subtitle string

	Rails      []Rail
	Nodes      []Node
	Labels     []Label
	Connectors []Connector
	Ticks      []Tick
	Headers    []Header
	Pills      []Pill
}

// Rail is the vertical spine of a lane, spanning its first to last anchor.
type Rail struct {
	X, Y0, Y1 float64
	Color     string
}

// Node is a step marker placed on a lane rail.
type Node struct {
	X, Y  float64
	Color string
}

// Label is a block of one or more text lines anchored at (X, Y). Anchor is the
// SVG-style horizontal anchor ("start", "middle", "end"); the renderer maps it.
type Label struct {
	X, Y     float64
	Lines    []string
	Anchor   string
	FontSize float64
	Color    string
}

// Connector is a poly-line path between primitives, optionally arrow-headed.
type Connector struct {
	Points [][2]float64
	Color  string
	Kind   ConnKind
	Arrow  bool
}

// Tick is one minute marker on the left ruler.
type Tick struct {
	Y    float64
	Text string
}

// Header is a colored lane chip across the top of the diagram.
type Header struct {
	X, Y, W, H float64
	Label      string
	Color      string
	Fg         string // label color, chosen for contrast against Color
}

// Pill is a small rounded time badge placed just left of a step's node, holding
// its relative timing ("+5 min") or "Anfang" for the first action.
type Pill struct {
	X, Y, W, H float64
	Text       string
	Bg, Fg     string
}
