package recipe

import (
	"strings"
	"testing"
)

func TestApplyThemeFillsEmptyHonorsExplicit(t *testing.T) {
	r := &Recipe{
		Lanes: []Lane{
			{ID: "a", Color: "#abcdef", explicitColor: true},
			{ID: "b"},
		},
	}
	if err := r.ApplyTheme("warm"); err != nil {
		t.Fatalf("ApplyTheme: %v", err)
	}
	if r.Lanes[0].Color != "#abcdef" {
		t.Errorf("explicit color overwritten: %q", r.Lanes[0].Color)
	}
	if r.Lanes[1].Color != themes["warm"].Palette[1] {
		t.Errorf("empty lane = %q, want warm palette color", r.Lanes[1].Color)
	}
}

func TestApplyThemeMonoForcesOverride(t *testing.T) {
	r := &Recipe{
		Lanes: []Lane{{ID: "a", Color: "#abcdef", explicitColor: true}},
	}
	for _, name := range []string{"mono", "print-bw", "grayscale"} {
		r.Lanes[0].Color = "#abcdef"
		if err := r.ApplyTheme(name); err != nil {
			t.Fatalf("ApplyTheme(%q): %v", name, err)
		}
		if r.Lanes[0].Color != themes["mono"].Palette[0] {
			t.Errorf("%s did not force-override explicit color: got %q", name, r.Lanes[0].Color)
		}
	}
}

func TestApplyThemeUnknown(t *testing.T) {
	r := &Recipe{Lanes: []Lane{{ID: "a"}}}
	err := r.ApplyTheme("neon")
	if err == nil {
		t.Fatal("expected error for unknown theme")
	}
	// The message must list valid names so the user can recover.
	for _, want := range []string{"neon", "warm", "mono"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want substring %q", err.Error(), want)
		}
	}
}
