// Command kochbahn renders a multi-lane YAML recipe to an SVG timeline.
//
// Usage:
//
//	kochbahn -in recipe.yaml [-out recipe.svg] [-theme NAME] [-lang de|en]
//	         [-portions N] [-check]
//
// The vertical axis is time in minutes (top to bottom); each lane is a parallel
// cooking process that may fork or merge into others. Every run validates the
// recipe in full before rendering: a bad recipe yields a list of file:line
// problems and no half-drawn SVG.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/StephanSchmidt/kochbahn/internal/layout"
	"github.com/StephanSchmidt/kochbahn/internal/recipe"
	"github.com/StephanSchmidt/kochbahn/internal/render"
)

func main() {
	in := flag.String("in", "", "path to the YAML recipe (required)")
	out := flag.String("out", "", "output SVG path (default: input with .svg; \"-\" for stdout)")
	theme := flag.String("theme", "default", "lane color preset: default, warm, cool, high-contrast, mono (alias print-bw)")
	lang := flag.String("lang", recipe.DefaultLang, "language for generated labels: de, en")
	portions := flag.Int("portions", 0, "rescale {amount} markup to this many portions (needs 'yield:' in the recipe)")
	check := flag.Bool("check", false, "validate the recipe and report problems without rendering")
	flag.Parse()

	if err := run(*in, *out, *theme, *lang, *portions, *check); err != nil {
		fmt.Fprintln(os.Stderr, "kochbahn:", err)
		os.Exit(1)
	}
}

// run is the whole pipeline: load + validate (the "check"), then — unless we
// were asked only to check — apply theme/language/portions and render. Each
// stage that can fail returns a message precise enough to fix the recipe from.
func run(in, out, theme, lang string, portions int, check bool) error {
	if in == "" {
		return fmt.Errorf("missing -in (path to YAML recipe)")
	}

	// Stage 1: parse, default and fully validate. recipe.Load reports every
	// structural problem at once, prefixed with file:line.
	r, err := recipe.Load(in)
	if err != nil {
		return err
	}

	if check {
		fmt.Fprintf(os.Stderr, "ok: %s — %d lanes, %d steps, no problems\n", in, len(r.Lanes), len(r.Steps))
		return nil
	}

	// Stage 1b: apply the presentation/scaling options. These run only on a
	// validated recipe, and each fails loudly with a fix-it message.
	if err := r.Localize(lang); err != nil {
		return err
	}
	if err := r.ApplyTheme(theme); err != nil {
		return err
	}
	if err := r.Scale(portions); err != nil {
		return err
	}

	// Stages 2 & 3: geometry, then SVG.
	style := layout.DefaultStyle()
	svg := render.SVG(layout.Build(r, style), style)

	if out == "" {
		base := strings.TrimSuffix(strings.TrimSuffix(in, ".yaml"), ".yml")
		out = base + ".svg"
	}
	if out == "-" {
		_, err := os.Stdout.Write(svg)
		return err
	}
	if err := os.WriteFile(out, svg, 0o644); err != nil {
		return fmt.Errorf("write svg: %w", err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d lanes, %d steps)\n", out, len(r.Lanes), len(r.Steps))
	return nil
}
