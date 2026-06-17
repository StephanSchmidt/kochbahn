package recipe

import (
	"strings"
	"testing"
)

func TestLoadValidRecipe(t *testing.T) {
	r, err := Load("../../testdata/salbeibutter.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(r.Lanes) != 2 {
		t.Errorf("lanes = %d, want 2", len(r.Lanes))
	}
	if r.Time.From != 0 || r.Time.To != 9 {
		t.Errorf("time axis = [%g,%g], want [0,9]", r.Time.From, r.Time.To)
	}
	if got := r.LaneColor("nudeln"); got != "#3b82f6" {
		t.Errorf("nudeln color = %q, want #3b82f6", got)
	}
	// The merge step must be parsed.
	var merged bool
	for _, s := range r.Steps {
		if s.Lane == "nudeln" && s.MergeInto == "sosse" {
			merged = true
			if s.ArrowLabel != "Nudeln in Pfanne" {
				t.Errorf("arrow label = %q", s.ArrowLabel)
			}
		}
	}
	if !merged {
		t.Error("merge step not found")
	}
}

func TestPaletteFillsMissingColors(t *testing.T) {
	r, err := Load("../../testdata/fork-merge.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, l := range r.Lanes {
		if l.Color == "" {
			t.Errorf("lane %q has no color after defaults", l.ID)
		}
	}
	if len(r.Lanes) != 5 {
		t.Fatalf("lanes = %d, want 5", len(r.Lanes))
	}
}

func TestValidateRejects(t *testing.T) {
	cases := []struct {
		name string
		r    Recipe
		want string
	}{
		{
			name: "too many lanes",
			r: Recipe{
				Lanes: []Lane{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}, {ID: "e"}, {ID: "f"}},
				Steps: []Step{{Lane: "a"}},
			},
			want: "maximum",
		},
		{
			name: "unknown lane in step",
			r: Recipe{
				Lanes: []Lane{{ID: "a"}},
				Steps: []Step{{Lane: "ghost"}},
			},
			want: "unknown lane",
		},
		{
			name: "unknown merge target",
			r: Recipe{
				Lanes: []Lane{{ID: "a"}},
				Steps: []Step{{Lane: "a", MergeInto: "ghost"}},
			},
			want: "merge_into unknown",
		},
		{
			name: "bad color",
			r: Recipe{
				Lanes: []Lane{{ID: "a", Color: "blue"}},
				Steps: []Step{{Lane: "a"}},
			},
			want: "invalid color",
		},
		{
			name: "negative time",
			r: Recipe{
				Lanes: []Lane{{ID: "a"}},
				Steps: []Step{{Lane: "a", At: -1}},
			},
			want: "negative time",
		},
		{
			name: "lane without steps",
			r: Recipe{
				Lanes: []Lane{{ID: "a"}, {ID: "b"}},
				Steps: []Step{{Lane: "a"}},
			},
			want: "no steps",
		},
		{
			name: "duplicate lane id",
			r: Recipe{
				Lanes: []Lane{{ID: "a"}, {ID: "a"}},
				Steps: []Step{{Lane: "a"}},
			},
			want: "duplicate lane",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.r.Validate()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}
