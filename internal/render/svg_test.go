package render

import (
	"strings"
	"testing"

	"github.com/StephanSchmidt/kochbahn/internal/layout"
)

func TestLogoEmbedParsed(t *testing.T) {
	if logoViewBox == "" {
		t.Error("logo viewBox not extracted from embedded asset")
	}
	if !strings.Contains(logoInner, "<path") {
		t.Error("logo inner content missing drawable paths")
	}
	if strings.Contains(logoInner, "<svg") {
		t.Error("logo inner content should not retain the root <svg> wrapper")
	}
}

func TestSVGEmitsLogoBox(t *testing.T) {
	l := &layout.Layout{
		Width: 400, Height: 200,
		Logo: &layout.Logo{X: 300, Y: 4, W: 92, H: 46},
	}
	out := string(SVG(l, layout.DefaultStyle()))
	// A nested <svg> repositions the mark at the box's coordinates.
	if !strings.Contains(out, `viewBox="`+logoViewBox+`"`) {
		t.Error("rendered SVG does not embed the logo viewBox")
	}
	if !strings.Contains(out, `x="300" y="4"`) {
		t.Errorf("logo not placed at its box coordinates:\n%s", out)
	}
}

func TestSVGNoLogoWhenAbsent(t *testing.T) {
	l := &layout.Layout{Width: 400, Height: 200} // Logo nil
	out := string(SVG(l, layout.DefaultStyle()))
	if strings.Contains(out, logoViewBox) {
		t.Error("logo emitted even though Layout.Logo is nil")
	}
}
