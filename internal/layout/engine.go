package layout

import (
	"sort"
	"strconv"
	"strings"

	"github.com/StephanSchmidt/kochbahn/internal/recipe"
)

// Style holds the tunable geometry and typography for the layout engine. All
// sizes are in pixels. DefaultStyle returns sensible book-friendly values.
type Style struct {
	PxPerMinute float64 // vertical scale of the time axis
	LaneWidth   float64 // horizontal width of one lane column
	GutterW     float64 // left ruler gutter width
	RightPad    float64
	TopPad      float64 // gap between header chips and the first tick
	BottomPad   float64

	TitleSize    float64
	SubtitleSize float64
	HeaderSize   float64 // lane chip text
	LabelSize    float64 // step text
	TickSize     float64
	ArrowSize    float64 // fork/merge arrow labels
	PillSize     float64 // time pill text

	NodeRadius float64
	LabelGap   float64 // gap between a node and its text
	ChipHeight float64
	FontFamily string
}

// DefaultStyle returns the standard style used when none is supplied.
func DefaultStyle() Style {
	return Style{
		PxPerMinute:  64,
		LaneWidth:    210,
		GutterW:      24,
		RightPad:     24,
		TopPad:       30,
		BottomPad:    36,
		TitleSize:    22,
		SubtitleSize: 13,
		HeaderSize:   13,
		LabelSize:    13,
		TickSize:     11,
		ArrowSize:    11,
		PillSize:     11,
		NodeRadius:   5,
		LabelGap:     12,
		ChipHeight:   26,
		FontFamily:   "Inter, 'Helvetica Neue', Arial, sans-serif",
	}
}

// laneSpan records the first and last node y for a lane, used to size rails and
// to anchor fork/merge arrows.
type laneSpan struct {
	firstY, lastY float64
	seen          bool
}

// Build is the stage-2 transform: recipe semantics → positioned geometry.
func Build(r *recipe.Recipe, s Style) *Layout {
	l := &Layout{Title: r.Title, Subtitle: r.Subtitle}

	// Vertical bands: title/subtitle, then lane chips, then the timeline.
	titleH := 0.0
	if r.Title != "" {
		titleH += s.TitleSize * 1.4
	}
	if r.Subtitle != "" {
		titleH += s.SubtitleSize * 1.6
	}
	if titleH > 0 {
		titleH += 12 // breathing room below the title block
	}
	headerTop := titleH + 6
	headerH := s.ChipHeight
	timelineTop := headerTop + headerH + s.TopPad

	// Coordinate mappers.
	y := func(at float64) float64 {
		return timelineTop + (at-r.Time.From)*s.PxPerMinute
	}
	laneX := func(i int) float64 {
		return s.GutterW + float64(i)*s.LaneWidth + s.LaneWidth/2
	}

	// Each step's marker is a rounded time chip centered on its rail. Compute the
	// chip texts (chronological, per-lane deltas) and a width helper up front, so
	// the step text can be offset to the right of the chip.
	pillTexts := computePillTexts(r)
	pillW := func(text string) float64 { return estTextWidth([]string{text}, s.PillSize) + 14 }

	// Document size. Grow the width so the rightmost lane's step text — anchored
	// to the right of its chip — is never clipped.
	l.Width = s.GutterW + float64(len(r.Lanes))*s.LaneWidth + s.RightPad
	for idx, st := range r.Steps {
		x0 := laneX(r.LaneIndex(st.Lane)) + pillW(pillTexts[idx])/2 + s.LabelGap
		if right := x0 + estTextWidth(splitLines(st.Text), s.LabelSize) + s.RightPad; right > l.Width {
			l.Width = right
		}
	}
	l.Height = y(r.Time.To) + s.BottomPad

	// Title / subtitle labels, centered over the lane area.
	centerX := s.GutterW + (l.Width-s.GutterW)/2
	if r.Title != "" {
		l.Labels = append(l.Labels, Label{
			X: centerX, Y: s.TitleSize, Lines: []string{r.Title},
			Anchor: "middle", FontSize: s.TitleSize, Color: "#111827",
		})
	}
	if r.Subtitle != "" {
		l.Labels = append(l.Labels, Label{
			X: centerX, Y: titleH - 12, Lines: []string{r.Subtitle},
			Anchor: "middle", FontSize: s.SubtitleSize, Color: "#6b7280",
		})
	}

	// Lane chips and the per-lane span needed for rails and arrows.
	spans := make([]laneSpan, len(r.Lanes))
	for i, lane := range r.Lanes {
		cx := laneX(i)
		chipW := s.LaneWidth - 24
		l.Headers = append(l.Headers, Header{
			X: cx - chipW/2, Y: headerTop, W: chipW, H: headerH,
			Label: lane.Label, Color: lane.Color, Fg: textOn(lane.Color),
		})
	}

	// Faint gridlines at each tick give vertical rhythm and let the eye see when
	// lanes line up. Time itself is shown per-step as relative pills, so the
	// ruler carries no numbers.
	for t := r.Time.From; t <= r.Time.To+1e-9; t += r.Time.Tick {
		l.Ticks = append(l.Ticks, Tick{Y: y(t)})
	}

	// Compute each lane's first/last node y from its steps.
	for _, st := range r.Steps {
		i := r.LaneIndex(st.Lane)
		yp := y(st.At)
		sp := &spans[i]
		if !sp.seen {
			sp.firstY, sp.lastY, sp.seen = yp, yp, true
			continue
		}
		if yp < sp.firstY {
			sp.firstY = yp
		}
		if yp > sp.lastY {
			sp.lastY = yp
		}
	}

	// channelY is the y of the horizontal run of a step's arrow: dropped below
	// the step's own (right-hand) text block so the arrow never strikes through
	// its label. lineH is one line's height.
	lineH := s.LabelSize * 1.25
	channelY := func(st recipe.Step, ny float64) float64 {
		n := len(splitLines(st.Text))
		if n < 1 {
			n = 1
		}
		return ny + float64(n)*lineH + 4
	}

	// Rail extents start from the node spans, then grow to receive any merge
	// arrowhead that lands above a lane's first node (or below its last).
	railLo := make([]float64, len(r.Lanes))
	railHi := make([]float64, len(r.Lanes))
	for i := range r.Lanes {
		railLo[i], railHi[i] = spans[i].firstY, spans[i].lastY
	}
	for _, st := range r.Steps {
		if st.MergeInto == "" {
			continue
		}
		j := r.LaneIndex(st.MergeInto)
		cy := channelY(st, y(st.At))
		if cy < railLo[j] {
			railLo[j] = cy
		}
		if cy > railHi[j] {
			railHi[j] = cy
		}
	}

	// Rails: the colored spine of each lane.
	for i, lane := range r.Lanes {
		if !spans[i].seen {
			continue
		}
		l.Rails = append(l.Rails, Rail{X: laneX(i), Y0: railLo[i], Y1: railHi[i], Color: lane.Color})
	}

	// Markers (time chips), step labels, and fork/merge connectors.
	for idx, st := range r.Steps {
		i := r.LaneIndex(st.Lane)
		nx, ny := laneX(i), y(st.At)
		color := r.Lanes[i].Color
		l.Nodes = append(l.Nodes, Node{X: nx, Y: ny, Color: color})

		// The time chip is the step marker: a rounded rectangle centered on the
		// rail, lane-colored with white text. It replaces the old dot.
		text := pillTexts[idx]
		pw, ph := pillW(text), s.PillSize+9
		l.Pills = append(l.Pills, Pill{
			X: nx - pw/2, Y: ny - ph/2, W: pw, H: ph,
			Text: text, Bg: color, Fg: textOn(color),
		})

		// Step text to the right of the chip.
		l.Labels = append(l.Labels, Label{
			X: nx + pw/2 + s.LabelGap, Y: ny + s.LabelSize*0.34,
			Lines: splitLines(st.Text), Anchor: "start", FontSize: s.LabelSize, Color: "#1f2937",
		})

		cy := channelY(st, ny)

		// Merge: this lane joins another. Drop below the label, then run across
		// to the target rail.
		if st.MergeInto != "" {
			dx := laneX(r.LaneIndex(st.MergeInto))
			l.Connectors = append(l.Connectors, Connector{
				Points: [][2]float64{{nx, ny}, {nx, cy}, {dx, cy}},
				Color:  color, Kind: Merge, Arrow: true,
			})
			l.addArrowLabel(st.ArrowLabel, (nx+dx)/2, cy-5, s.ArrowSize)
		}

		// Fork: branch off into one or more lanes, landing on their first node.
		for _, f := range st.ForkTo {
			j := r.LaneIndex(f)
			dx := laneX(j)
			dy := spans[j].firstY
			if !spans[j].seen {
				dy = cy
			}
			l.Connectors = append(l.Connectors, Connector{
				Points: [][2]float64{{nx, ny}, {nx, cy}, {dx, cy}, {dx, dy}},
				Color:  color, Kind: Fork, Arrow: true,
			})
			l.addArrowLabel(st.ArrowLabel, (nx+dx)/2, cy-5, s.ArrowSize)
		}
	}

	// Keep nodes drawn on top of rails regardless of step order.
	sort.SliceStable(l.Nodes, func(a, b int) bool { return l.Nodes[a].Y < l.Nodes[b].Y })
	return l
}

// computePillTexts returns the time-chip text for each step, indexed by the
// step's position in r.Steps. Reading steps in chronological order, each lane's
// first action shows "Anfang" (if it is the earliest of all) or "+N" from the
// global start (when to fire up that lane); later actions show "+N" since the
// previous step in the same lane (how long until the next move there).
func computePillTexts(r *recipe.Recipe) []string {
	out := make([]string, len(r.Steps))
	if len(r.Steps) == 0 {
		return out
	}

	// Walk steps in chronological order; ties keep input order (stable).
	order := make([]int, len(r.Steps))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return r.Steps[order[a]].At < r.Steps[order[b]].At
	})
	globalStart := r.Steps[order[0]].At

	lastAt := make([]float64, len(r.Lanes))
	seen := make([]bool, len(r.Lanes))
	for _, idx := range order {
		st := r.Steps[idx]
		i := r.LaneIndex(st.Lane)
		switch {
		case !seen[i] && st.At <= globalStart:
			out[idx] = "Anfang"
		case !seen[i]:
			out[idx] = "+" + fmtNum(st.At-globalStart) + " " + r.Time.Unit
		default:
			out[idx] = "+" + fmtNum(st.At-lastAt[i]) + " " + r.Time.Unit
		}
		seen[i], lastAt[i] = true, st.At
	}
	return out
}

// addArrowLabel appends a centered annotation for a fork/merge arrow, if any.
func (l *Layout) addArrowLabel(text string, x, y, size float64) {
	if text == "" {
		return
	}
	l.Labels = append(l.Labels, Label{
		X: x, Y: y, Lines: []string{text},
		Anchor: "middle", FontSize: size, Color: "#6b7280",
	})
}

// splitLines splits step text on line breaks, accepting both real newlines and
// the literal two-character sequence "\n" (e.g. from single-quoted YAML).
// Splitting on substrings keeps multi-byte UTF-8 runes intact.
func splitLines(s string) []string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	return strings.Split(s, "\n")
}

// estTextWidth approximates the rendered width of the widest line, in pixels.
// There is no font engine, so we use an average glyph-width factor over the
// rune count (counting runes, not bytes, keeps umlauts honest).
func estTextWidth(lines []string, fontSize float64) float64 {
	max := 0
	for _, ln := range lines {
		if n := len([]rune(ln)); n > max {
			max = n
		}
	}
	return float64(max) * fontSize * 0.6
}

// fmtNum formats a float without a trailing ".0".
func fmtNum(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// textOn returns a legible label color (dark or white) for text drawn on top of
// the given "#rrggbb" background, based on its perceived brightness.
func textOn(hex string) string {
	if len(hex) != 7 || hex[0] != '#' {
		return "#ffffff"
	}
	v, err := strconv.ParseUint(hex[1:], 16, 32)
	if err != nil {
		return "#ffffff"
	}
	r := float64((v >> 16) & 0xff)
	g := float64((v >> 8) & 0xff)
	b := float64(v & 0xff)
	if (0.299*r+0.587*g+0.114*b)/255 > 0.62 {
		return "#1f2937"
	}
	return "#ffffff"
}
