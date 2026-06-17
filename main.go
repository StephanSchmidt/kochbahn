// Command kochbahn renders a multi-lane YAML recipe to an SVG timeline.
//
// Usage:
//
//	kochbahn -in recipe.yaml [-out recipe.svg]
//
// The vertical axis is time in minutes (top to bottom); each lane is a parallel
// cooking process that may fork or merge into others.
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
	flag.Parse()

	if err := run(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, "kochbahn:", err)
		os.Exit(1)
	}
}

func run(in, out string) error {
	if in == "" {
		return fmt.Errorf("missing -in (path to YAML recipe)")
	}

	r, err := recipe.Load(in)
	if err != nil {
		return err
	}

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
