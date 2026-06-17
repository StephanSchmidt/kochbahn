package recipe

import (
	"strings"
	"testing"
)

// TestLoadCollectsAllProblems verifies the "check" phase reports every problem
// at once, each anchored to its source line, rather than failing on the first.
func TestLoadCollectsAllProblems(t *testing.T) {
	_, err := Load("../../testdata/broken.yaml")
	if err == nil {
		t.Fatal("expected broken.yaml to fail validation")
	}
	d, ok := err.(*Diagnostics)
	if !ok {
		t.Fatalf("error is %T, want *Diagnostics", err)
	}
	if len(d.Problems) < 7 {
		t.Errorf("collected %d problems, want many at once", len(d.Problems))
	}

	msg := err.Error()
	// Every distinct issue should surface in one pass.
	for _, want := range []string{
		"negative", "invalid color", "duplicate lane id", "has no steps",
		"leading number", "unknown lane", "merge_into unknown", "labels: unknown key",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q:\n%s", want, msg)
		}
	}
	// The file name must prefix the report so the user knows where to look.
	if !strings.Contains(msg, "broken.yaml") {
		t.Errorf("message lacks file path:\n%s", msg)
	}
	// Line anchoring: the duplicate lane id sits on line 7 of the fixture.
	if !strings.Contains(msg, "broken.yaml:7:") {
		t.Errorf("message lacks expected line anchor broken.yaml:7:\n%s", msg)
	}
}

func TestSingleProblemReadsAsSentence(t *testing.T) {
	d := &Diagnostics{Path: "r.yaml", Problems: []Problem{{Line: 3, Msg: "boom"}}}
	if got := d.Error(); got != "r.yaml:3: boom" {
		t.Errorf("single problem = %q, want %q", got, "r.yaml:3: boom")
	}
}
