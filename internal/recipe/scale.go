package recipe

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// unicodeFractions maps the common vulgar-fraction runes to their value, so
// "½ TL" scales like "0.5 TL".
var unicodeFractions = map[rune]float64{
	'½': 0.5, '⅓': 1.0 / 3, '⅔': 2.0 / 3, '¼': 0.25, '¾': 0.75,
	'⅕': 0.2, '⅖': 0.4, '⅗': 0.6, '⅘': 0.8,
	'⅙': 1.0 / 6, '⅚': 5.0 / 6, '⅛': 0.125, '⅜': 0.375, '⅝': 0.625, '⅞': 0.875,
}

// Scale resolves the `{...}` quantity markup in every step. The braces are
// kochbahn's own markup and never appear in output: with portions <= 0 the
// markup is simply unwrapped verbatim (so a recipe with amounts renders cleanly
// at its written yield), and with portions > 0 each amount is multiplied by
// portions / yield first. Only numbers inside braces are touched — e.g.
// `{200 g} Mehl` — so the tool never mistakes a cooking time ("40 Min") for an
// amount.
//
// It is an error to request scaling without a `yield:` to scale from; the
// message says exactly how to fix it.
func (r *Recipe) Scale(portions int) error {
	factor := 1.0
	if portions > 0 {
		if r.Yield <= 0 {
			return (&Diagnostics{}).addAt(r.yieldLine,
				"-portions %d requested, but the recipe has no 'yield:' to scale from; "+
					"add e.g. 'yield: 2' (the portions the amounts are written for)", portions)
		}
		factor = float64(portions) / float64(r.Yield)
	}

	d := &Diagnostics{}
	for i := range r.Steps {
		scaled, err := scaleText(r.Steps[i].Text, factor)
		if err != nil {
			d.addAt(r.Steps[i].line, "step %d (lane %q): %v", i, r.Steps[i].Lane, err)
			continue
		}
		r.Steps[i].Text = scaled
	}
	if !d.ok() {
		return d
	}
	if portions > 0 {
		// Record the new yield so a re-render reports honest amounts.
		r.Yield = portions
	}
	return nil
}

// validateQuantityMarkup reports whether the `{...}` markup in s is well-formed
// (balanced braces, each holding a leading number), without changing the text.
// It runs at validation time so a -check surfaces broken markup even when no
// scaling was requested. A factor other than 1 forces the number parser to run.
func validateQuantityMarkup(s string) error {
	_, err := scaleText(s, 2)
	return err
}

// hasQuantityMarkup reports whether the text uses any `{...}` scalable markup.
func hasQuantityMarkup(s string) bool { return strings.ContainsRune(s, '{') }

// scaleText replaces every `{...}` span with its scaled contents (braces
// removed). Text outside braces is left exactly as written.
func scaleText(s string, factor float64) (string, error) {
	var b strings.Builder
	for {
		open := strings.IndexByte(s, '{')
		if open < 0 {
			b.WriteString(s)
			return b.String(), nil
		}
		close := strings.IndexByte(s[open:], '}')
		if close < 0 {
			return "", fmt.Errorf("unclosed '{' in quantity markup %q", s[open:])
		}
		close += open
		b.WriteString(s[:open])
		inner := s[open+1 : close]
		// Factor 1 just unwraps the markup, preserving the original formatting
		// (so "½ TL" stays "½ TL" rather than becoming "0.5 TL").
		if factor == 1 {
			b.WriteString(inner)
		} else {
			out, err := scaleQuantity(inner, factor)
			if err != nil {
				return "", fmt.Errorf("%q: %w", "{"+inner+"}", err)
			}
			b.WriteString(out)
		}
		s = s[close+1:]
	}
}

// scaleQuantity scales the leading number (or "a-b" range) of a markup body and
// re-attaches the trailing unit/text unchanged.
func scaleQuantity(inner string, factor float64) (string, error) {
	trimmed := strings.TrimLeft(inner, " ")
	lead := len(inner) - len(trimmed)

	v1, n1, comma, ok := parseNumber(trimmed)
	if !ok {
		return "", fmt.Errorf("expected a leading number, e.g. {200 g} or {1/2 TL}")
	}
	rest := trimmed[n1:]

	// Optional range: "2-3", "2–3" (en dash) or "2—3" (em dash).
	if sep, after, isRange := splitRange(rest); isRange {
		v2, n2, comma2, ok := parseNumber(strings.TrimLeft(after, " "))
		if ok {
			padded := after[:len(after)-len(strings.TrimLeft(after, " "))]
			tail := strings.TrimLeft(after, " ")[n2:]
			return inner[:lead] +
				formatNumber(v1*factor, comma) + sep + padded +
				formatNumber(v2*factor, comma2) + tail, nil
		}
	}

	return inner[:lead] + formatNumber(v1*factor, comma) + rest, nil
}

// splitRange detects a leading range separator ("-", "–", "—") in rest and
// returns the separator plus the remainder after it.
func splitRange(rest string) (sep, after string, ok bool) {
	for _, r := range []string{"-", "–", "—"} {
		if strings.HasPrefix(rest, r) {
			return r, rest[len(r):], true
		}
	}
	return "", "", false
}

// parseNumber reads one quantity value from the front of s: an integer or
// decimal (with "." or "," as the decimal mark), a vulgar fraction (½ or 1/2),
// or a mixed number (1 ½ / 1 1/2). It returns the value, the bytes consumed,
// whether a comma decimal mark was seen (so output can match), and ok.
func parseNumber(s string) (val float64, n int, comma bool, ok bool) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	hasInt := i > 0
	intEnd := i

	// Pure fraction "a/b": the leading integer run is the numerator.
	if hasInt && i < len(s) && s[i] == '/' {
		j := i + 1
		denStart := j
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		if j > denStart {
			num, _ := strconv.ParseFloat(s[:intEnd], 64)
			den, _ := strconv.ParseFloat(s[denStart:j], 64)
			if den != 0 {
				return num / den, j, false, true
			}
		}
		// "5/" with no denominator falls through to plain integer.
	}

	// Decimal "a.b" or "a,b".
	if i < len(s) && (s[i] == '.' || s[i] == ',') && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
		comma = s[i] == ','
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		f, err := strconv.ParseFloat(strings.Replace(s[:i], ",", ".", 1), 64)
		if err != nil {
			return 0, 0, false, false
		}
		return f, i, comma, true
	}

	whole := 0.0
	if hasInt {
		whole, _ = strconv.ParseFloat(s[:intEnd], 64)
	}

	// Mixed number ("1 ½", "1 1/2") or a bare leading fraction ("½", "3/4"):
	// an optional fraction after the integer, possibly separated by a space.
	j := intEnd
	for j < len(s) && s[j] == ' ' {
		j++
	}
	if frac, fn, fok := parseFraction(s[j:]); fok {
		// A bare fraction (no integer) must start at the very front, not after a
		// space that belonged to a following word.
		if hasInt || j == 0 {
			return whole + frac, j + fn, comma, true
		}
	}
	if hasInt {
		return whole, intEnd, comma, true
	}
	return 0, 0, false, false
}

// parseFraction reads a leading fraction: a single vulgar-fraction rune or an
// ASCII "a/b". Returns the value and bytes consumed.
func parseFraction(s string) (val float64, n int, ok bool) {
	if s == "" {
		return 0, 0, false
	}
	// Unicode vulgar fraction.
	for _, r := range s {
		if v, isFrac := unicodeFractions[r]; isFrac {
			return v, len(string(r)), true
		}
		break
	}
	// ASCII a/b.
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(s) || s[i] != '/' {
		return 0, 0, false
	}
	num, _ := strconv.ParseFloat(s[:i], 64)
	i++
	denStart := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == denStart {
		return 0, 0, false
	}
	den, _ := strconv.ParseFloat(s[denStart:i], 64)
	if den == 0 {
		return 0, 0, false
	}
	return num / den, i, true
}

// formatNumber renders a scaled value: integers stay integers, others round to
// two decimals with trailing zeros trimmed. comma echoes the input's decimal
// mark so German amounts read naturally.
func formatNumber(v float64, comma bool) string {
	rounded := math.Round(v*100) / 100
	s := strconv.FormatFloat(rounded, 'f', -1, 64)
	if comma {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s
}
