// Package render is stage 3 of the pipeline: it maps a layout.Layout (pure
// geometry) to SVG bytes. It knows about SVG syntax but nothing about recipe
// semantics. A tiny internal writer over bytes.Buffer keeps it dependency-free.
package render

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/StephanSchmidt/kochbahn/internal/layout"
)

// logoSVG is the kochbahn wordmark, embedded so every rendered timeline is a
// standalone file (no external asset reference to break when moved or viewed on
// GitHub). It is the warm variant, matching the README header.
//
//go:embed logo.svg
var logoSVG string

// logoInner is the wordmark's drawable content (everything inside its root
// <svg>) and logoVBWidth is that root's viewBox width, both extracted once at
// startup so the mark can be re-hosted with a plain translate/scale group.
var logoInner, logoVBWidth = parseLogo(logoSVG)

var (
	svgOpenTag  = regexp.MustCompile(`(?s)^.*?<svg[^>]*>`)
	svgCloseTag = regexp.MustCompile(`(?s)</svg>\s*$`)
	viewBoxAttr = regexp.MustCompile(`viewBox="[^"]*?([0-9.]+)\s+[0-9.]+"`)
)

// parseLogo splits an SVG document into its inner content and viewBox width.
func parseLogo(doc string) (inner string, vbWidth float64) {
	if m := viewBoxAttr.FindStringSubmatch(svgOpenTag.FindString(doc)); m != nil {
		vbWidth, _ = strconv.ParseFloat(m[1], 64)
	}
	inner = svgOpenTag.ReplaceAllString(doc, "")
	inner = svgCloseTag.ReplaceAllString(inner, "")
	return strings.TrimSpace(inner), vbWidth
}

// SVG renders the layout to a standalone SVG document.
func SVG(l *layout.Layout, s layout.Style) []byte {
	w := &writer{}
	w.printf(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	w.printf(`<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s" font-family="%s">`+"\n",
		num(l.Width), num(l.Height), num(l.Width), num(l.Height), esc(s.FontFamily))
	w.printf(`<rect width="%s" height="%s" fill="#ffffff"/>`+"\n", num(l.Width), num(l.Height))

	// Background: faint ruler gridlines (no numbers — time lives on the pills).
	for _, t := range l.Ticks {
		w.printf(`<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="#eceff3" stroke-width="1"/>`+"\n",
			num(s.GutterW), num(t.Y), num(l.Width-s.RightPad), num(t.Y))
	}

	// Rails (lane spines).
	for _, r := range l.Rails {
		w.printf(`<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s" stroke-width="3" stroke-linecap="round" opacity="0.85"/>`+"\n",
			num(r.X), num(r.Y0), num(r.X), num(r.Y1), esc(r.Color))
	}

	// Connectors (fork/merge arrows) with computed arrowheads.
	for _, c := range l.Connectors {
		w.connector(c)
	}

	// Time chips: the step marker is a rounded rectangle on the rail (a white
	// halo lifts it off the rail line) with a centered white label.
	for _, p := range l.Pills {
		w.printf(`<rect x="%s" y="%s" width="%s" height="%s" rx="4" fill="%s"/>`+"\n",
			num(p.X-1.5), num(p.Y-1.5), num(p.W+3), num(p.H+3), "#ffffff")
		w.printf(`<rect x="%s" y="%s" width="%s" height="%s" rx="4" fill="%s"/>`+"\n",
			num(p.X), num(p.Y), num(p.W), num(p.H), esc(p.Bg))
		w.text(p.X+p.W/2, p.Y+p.H/2+s.PillSize*0.34, []string{p.Text}, "middle", s.PillSize, p.Fg)
	}

	// Lane header chips: rounded colored rect + centered white label.
	for _, h := range l.Headers {
		w.printf(`<rect x="%s" y="%s" width="%s" height="%s" rx="6" fill="%s"/>`+"\n",
			num(h.X), num(h.Y), num(h.W), num(h.H), esc(h.Color))
		w.text(h.X+h.W/2, h.Y+h.H/2+s.HeaderSize*0.34, []string{h.Label}, "middle", s.HeaderSize, h.Fg)
	}

	// Labels (title, subtitle, step text, arrow labels, ruler caption).
	for _, lb := range l.Labels {
		w.text(lb.X, lb.Y, lb.Lines, lb.Anchor, lb.FontSize, lb.Color)
	}

	// Corner wordmark, drawn last so it sits above everything else.
	if l.Logo != nil && logoInner != "" {
		w.logo(*l.Logo)
	}

	w.printf("</svg>\n")
	return w.buf.Bytes()
}

// writer is a minimal SVG sink.
type writer struct{ buf bytes.Buffer }

func (w *writer) printf(format string, a ...any) { fmt.Fprintf(&w.buf, format, a...) }

// text emits a (possibly multi-line) text element. Extra lines stack downward
// as <tspan>s at 1.25em leading.
func (w *writer) text(x, y float64, lines []string, anchor string, size float64, color string) {
	if len(lines) == 0 {
		return
	}
	w.printf(`<text x="%s" y="%s" text-anchor="%s" font-size="%s" fill="%s">`,
		num(x), num(y), anchor, num(size), esc(color))
	for i, ln := range lines {
		if i == 0 {
			w.printf(`%s`, esc(ln))
			continue
		}
		w.printf(`<tspan x="%s" dy="%s">%s</tspan>`, num(x), num(size*1.25), esc(ln))
	}
	w.printf("</text>\n")
}

// logo re-hosts the embedded wordmark with a translate+scale group. A plain <g>
// transform renders identically everywhere (GitHub's <img> sanitizer, browsers,
// rasterizers), unlike a nested <svg> with x/y placement.
func (w *writer) logo(b layout.Logo) {
	vbw := logoVBWidth
	if vbw == 0 {
		vbw = 423.33333
	}
	scale := b.W / vbw
	w.printf(`<g transform="translate(%s %s) scale(%s)">%s</g>`+"\n",
		num(b.X), num(b.Y), strconv.FormatFloat(scale, 'f', 5, 64), logoInner)
}

// connector draws a poly-line and, if requested, an arrowhead at its end.
func (w *writer) connector(c layout.Connector) {
	if len(c.Points) < 2 {
		return
	}
	var d strings.Builder
	for i, p := range c.Points {
		if i == 0 {
			fmt.Fprintf(&d, "M %s %s", num(p[0]), num(p[1]))
		} else {
			fmt.Fprintf(&d, " L %s %s", num(p[0]), num(p[1]))
		}
	}
	dash := ""
	if c.Kind == layout.Fork {
		dash = ` stroke-dasharray="5 4"`
	}
	w.printf(`<path d="%s" fill="none" stroke="%s" stroke-width="2"%s/>`+"\n", d.String(), esc(c.Color), dash)

	if c.Arrow {
		tip := c.Points[len(c.Points)-1]
		prev := c.Points[len(c.Points)-2]
		w.arrowhead(prev, tip, c.Color)
	}
}

// arrowhead draws a filled triangle at tip, pointing along prev→tip.
func (w *writer) arrowhead(prev, tip [2]float64, color string) {
	const size, half = 9.0, 4.0
	dx, dy := tip[0]-prev[0], tip[1]-prev[1]
	mag := math.Hypot(dx, dy)
	if mag == 0 {
		return
	}
	ux, uy := dx/mag, dy/mag                 // unit direction
	px, py := -uy, ux                        // unit perpendicular
	bx, by := tip[0]-ux*size, tip[1]-uy*size // base center
	x1, y1 := bx+px*half, by+py*half
	x2, y2 := bx-px*half, by-py*half
	w.printf(`<polygon points="%s,%s %s,%s %s,%s" fill="%s"/>`+"\n",
		num(tip[0]), num(tip[1]), num(x1), num(y1), num(x2), num(y2), esc(color))
}

// num formats a coordinate compactly (up to 2 decimals, no trailing zeros).
func num(f float64) string {
	return strconv.FormatFloat(math.Round(f*100)/100, 'f', -1, 64)
}

// esc escapes the handful of characters that matter inside SVG text/attrs.
func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
