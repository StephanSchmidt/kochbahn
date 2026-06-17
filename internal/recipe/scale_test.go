package recipe

import (
	"strings"
	"testing"
)

func TestScaleText(t *testing.T) {
	cases := []struct {
		in     string
		factor float64
		want   string
	}{
		{"{200 g} Mehl", 2, "400 g Mehl"},
		{"{200 g} Mehl", 0.5, "100 g Mehl"},
		{"{1/2 TL} Salz", 3, "1.5 TL Salz"},            // pure fraction
		{"{1/2 TL} Salz", 1.5, "0.75 TL Salz"},         // fraction → decimal
		{"{½ TL}", 4, "2 TL"},                          // unicode fraction
		{"{1 ½ EL}", 2, "3 EL"},                        // mixed number
		{"{1,5 l} Wasser", 3, "4,5 l Wasser"},          // comma decimal preserved
		{"{2-3} Eier", 3, "6-9 Eier"},                  // range, both ends scaled
		{"{2–3} Eier", 2, "4–6 Eier"},                  // en-dash range
		{"plain (40 Min)", 2, "plain (40 Min)"},        // no markup, untouched
		{"{200 g} A, {50 g} B", 2, "400 g A, 100 g B"}, // two spans
		{"  {3 Stk} ", 2, "  6 Stk "},                  // leading/trailing text kept
	}
	for _, c := range cases {
		got, err := scaleText(c.in, c.factor)
		if err != nil {
			t.Errorf("scaleText(%q, %g) error: %v", c.in, c.factor, err)
			continue
		}
		if got != c.want {
			t.Errorf("scaleText(%q, %g) = %q, want %q", c.in, c.factor, got, c.want)
		}
	}
}

func TestScaleTextErrors(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"{g} Mehl", "leading number"},
		{"{abc}", "leading number"},
		{"{200 g unbalanced", "unclosed"},
	}
	for _, c := range cases {
		_, err := scaleText(c.in, 2)
		if err == nil {
			t.Errorf("scaleText(%q) expected error", c.in)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("scaleText(%q) error = %q, want substring %q", c.in, err.Error(), c.want)
		}
	}
}

func TestScaleRequiresYield(t *testing.T) {
	r := &Recipe{
		Lanes: []Lane{{ID: "a"}},
		Steps: []Step{{Lane: "a", Text: "{200 g} Mehl"}},
	}
	err := r.Scale(4)
	if err == nil {
		t.Fatal("expected error scaling without yield")
	}
	if !strings.Contains(err.Error(), "no 'yield:'") {
		t.Errorf("error = %q, want mention of missing yield", err.Error())
	}
}

func TestScaleAppliesAndUpdatesYield(t *testing.T) {
	r := &Recipe{
		Yield: 2,
		Lanes: []Lane{{ID: "a"}},
		Steps: []Step{{Lane: "a", Text: "{200 g} Mehl"}},
	}
	if err := r.Scale(6); err != nil {
		t.Fatalf("Scale: %v", err)
	}
	if r.Steps[0].Text != "600 g Mehl" {
		t.Errorf("text = %q, want %q", r.Steps[0].Text, "600 g Mehl")
	}
	if r.Yield != 6 {
		t.Errorf("yield = %d, want 6 after scaling", r.Yield)
	}
}

func TestScaleZeroStripsMarkupVerbatim(t *testing.T) {
	// Without -portions the braces are still markup and must vanish, but the
	// original formatting (½, fractions) is preserved rather than reformatted.
	r := &Recipe{
		Lanes: []Lane{{ID: "a"}},
		Steps: []Step{{Lane: "a", Text: "{200 g} Mehl, {½ TL} Salz"}},
	}
	if err := r.Scale(0); err != nil {
		t.Fatalf("Scale(0): %v", err)
	}
	if want := "200 g Mehl, ½ TL Salz"; r.Steps[0].Text != want {
		t.Errorf("Scale(0) = %q, want %q", r.Steps[0].Text, want)
	}
}
