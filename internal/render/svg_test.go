package render

import (
	"strings"
	"testing"

	"github.com/StephanSchmidt/kochbahn/internal/layout"
)

func TestLogoEmbedParsed(t *testing.T) {
	if logoVBWidth <= 0 {
		t.Error("logo viewBox width not extracted from embedded asset")
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
	// A translate+scale group repositions the mark at the box's coordinates.
	if !strings.Contains(out, `<g transform="translate(300 4) scale(`) {
		t.Errorf("logo not placed at its box coordinates:\n%s", out)
	}
	if !strings.Contains(out, "<path") {
		t.Error("rendered SVG does not embed the logo paths")
	}
}

func TestSVGNoLogoWhenAbsent(t *testing.T) {
	l := &layout.Layout{Width: 400, Height: 200} // Logo nil
	out := string(SVG(l, layout.DefaultStyle()))
	if strings.Contains(out, "translate(") {
		t.Error("logo emitted even though Layout.Logo is nil")
	}
}
