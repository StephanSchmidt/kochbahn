package layout

import (
	"math"
	"testing"

	"github.com/StephanSchmidt/kochbahn/internal/recipe"
)

// sampleRecipe builds a small valid recipe in code (no file I/O).
func sampleRecipe() *recipe.Recipe {
	r := &recipe.Recipe{
		Time:  recipe.TimeAxis{From: 0, To: 9, Tick: 1, Unit: "min"},
		Lanes: []recipe.Lane{{ID: "a", Label: "A", Color: "#111111"}, {ID: "b", Label: "B", Color: "#222222"}},
		Steps: []recipe.Step{
			{Lane: "a", At: 0, Text: "start"},
			{Lane: "a", At: 7, Text: "join", MergeInto: "b", ArrowLabel: "X"},
			{Lane: "b", At: 5, Text: "b1"},
			{Lane: "b", At: 8, Text: "b2"},
		},
	}
	if err := r.Validate(); err != nil {
		panic(err)
	}
	return r
}

func TestBuildCoordinateMapping(t *testing.T) {
	r := sampleRecipe()
	s := DefaultStyle()
	l := Build(r, s)

	// Document width follows the formula.
	wantW := s.GutterW + float64(len(r.Lanes))*s.LaneWidth + s.RightPad
	if l.Width != wantW {
		t.Errorf("width = %g, want %g", l.Width, wantW)
	}

	// Two lanes → two rails and two headers.
	if len(l.Rails) != 2 {
		t.Errorf("rails = %d, want 2", len(l.Rails))
	}
	if len(l.Headers) != 2 {
		t.Errorf("headers = %d, want 2", len(l.Headers))
	}

	// Four steps → four nodes.
	if len(l.Nodes) != 4 {
		t.Errorf("nodes = %d, want 4", len(l.Nodes))
	}

	// Equal time deltas map to equal pixel deltas (proportional axis).
	railA := l.Rails[0]
	gotSpan := railA.Y1 - railA.Y0
	wantSpan := (7.0 - 0.0) * s.PxPerMinute
	if math.Abs(gotSpan-wantSpan) > 1e-6 {
		t.Errorf("lane A rail span = %g, want %g", gotSpan, wantSpan)
	}
}

func TestBuildMergeConnector(t *testing.T) {
	r := sampleRecipe()
	s := DefaultStyle()
	l := Build(r, s)

	var merge *Connector
	for i := range l.Connectors {
		if l.Connectors[i].Kind == Merge {
			merge = &l.Connectors[i]
		}
	}
	if merge == nil {
		t.Fatal("no merge connector produced")
	}
	if !merge.Arrow {
		t.Error("merge connector should be arrow-headed")
	}
	// It must end on lane B's x (column index 1 centerline).
	wantX := s.GutterW + 1*s.LaneWidth + s.LaneWidth/2
	n := len(merge.Points)
	end := merge.Points[n-1]
	if end[0] != wantX {
		t.Errorf("merge end x = %g, want %g (lane B rail)", end[0], wantX)
	}
	// The arrow leaves its source node and ends below it (dropped past the
	// source label), so the final segment runs horizontally into the target.
	if merge.Points[n-2][1] != end[1] {
		t.Errorf("final merge segment should be horizontal: %v -> %v", merge.Points[n-2], end)
	}
	if end[1] <= merge.Points[0][1] {
		t.Errorf("merge should drop below its source node: src y=%g end y=%g", merge.Points[0][1], end[1])
	}
}

func TestBuildFiveLanesWithFork(t *testing.T) {
	r, err := recipe.Load("../../testdata/fork-merge.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	l := Build(r, DefaultStyle())
	if len(l.Rails) != 5 {
		t.Errorf("rails = %d, want 5", len(l.Rails))
	}
	var forks, merges int
	for _, c := range l.Connectors {
		switch c.Kind {
		case Fork:
			forks++
		case Merge:
			merges++
		}
	}
	if forks != 1 {
		t.Errorf("forks = %d, want 1", forks)
	}
	if merges != 3 {
		t.Errorf("merges = %d, want 3", merges)
	}
}

func TestSplitLines(t *testing.T) {
	got := splitLines(`one\ntwo` + "\n" + "three")
	want := []string{"one", "two", "three"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}
