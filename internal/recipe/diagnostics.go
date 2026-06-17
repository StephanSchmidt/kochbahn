package recipe

import (
	"fmt"
	"sort"
	"strings"
)

// Problem is a single thing wrong with a recipe, optionally tied to a source
// line. Line 0 means "no specific line" (e.g. a whole-recipe rule).
type Problem struct {
	Line int
	Msg  string
}

// Diagnostics is a collection of problems found in one recipe. It implements
// error so it can flow through the normal (*Recipe, error) return convention,
// but unlike a bare error it reports *every* problem at once and renders each
// as a compiler-style, file:line-prefixed line — so a user fixes the whole
// recipe in one pass instead of one error per run.
type Diagnostics struct {
	Path     string // source file, used as the error prefix; may be empty
	Problems []Problem
}

// add appends a problem with no line. Returns the receiver for chaining.
func (d *Diagnostics) add(format string, a ...any) *Diagnostics {
	d.Problems = append(d.Problems, Problem{Msg: fmt.Sprintf(format, a...)})
	return d
}

// addAt appends a problem anchored to a source line.
func (d *Diagnostics) addAt(line int, format string, a ...any) *Diagnostics {
	d.Problems = append(d.Problems, Problem{Line: line, Msg: fmt.Sprintf(format, a...)})
	return d
}

// ok reports whether no problems were recorded.
func (d *Diagnostics) ok() bool { return len(d.Problems) == 0 }

// errorOrNil returns d as an error when it holds problems, else a nil error —
// so callers can write `return diags.errorOrNil()`.
func (d *Diagnostics) errorOrNil() error {
	if d.ok() {
		return nil
	}
	return d
}

// Error renders all problems, one per line, sorted by source line. With more
// than one problem it leads with a count so the scale is obvious at a glance.
func (d *Diagnostics) Error() string {
	ps := make([]Problem, len(d.Problems))
	copy(ps, d.Problems)
	sort.SliceStable(ps, func(i, j int) bool { return ps[i].Line < ps[j].Line })

	// A lone problem reads as a single sentence; several read as an indented,
	// counted list under a summary header.
	if len(ps) == 1 {
		return d.prefix(ps[0].Line) + ps[0].Msg
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d problems in %s:", len(ps), d.location())
	for _, p := range ps {
		b.WriteString("\n  ")
		b.WriteString(d.prefix(p.Line))
		b.WriteString(p.Msg)
	}
	return b.String()
}

// location is the file name for the summary line, falling back to "recipe".
func (d *Diagnostics) location() string {
	if d.Path == "" {
		return "recipe"
	}
	return d.Path
}

// prefix builds the "file:line: " anchor for one problem, omitting whichever of
// path or line is unknown.
func (d *Diagnostics) prefix(line int) string {
	switch {
	case d.Path != "" && line > 0:
		return fmt.Sprintf("%s:%d: ", d.Path, line)
	case d.Path != "":
		return d.Path + ": "
	case line > 0:
		return fmt.Sprintf("line %d: ", line)
	default:
		return ""
	}
}
