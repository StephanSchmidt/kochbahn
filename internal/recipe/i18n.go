package recipe

import "sort"

// Strings are the few labels kochbahn generates itself (everything else is
// recipe content). They are localized via -lang and may be overridden per
// recipe through the YAML `labels:` block, so a recipe can read in any language
// without the tool hard-coding German.
type Strings struct {
	Start string // pill text for the earliest action ("Anfang" / "Start")
	Total string // caption prefix for the derived total time ("Gesamtzeit")
}

// builtinLangs holds the shipped translations. German is the project default
// (the examples are German); English is provided as the second language.
var builtinLangs = map[string]Strings{
	"de": {Start: "Anfang", Total: "Gesamtzeit"},
	"en": {Start: "Start", Total: "Total time"},
}

// DefaultLang is used when none is requested and as the base for overrides.
const DefaultLang = "de"

// knownLangs returns the supported language codes, sorted, for error messages.
func knownLangs() []string {
	out := make([]string, 0, len(builtinLangs))
	for k := range builtinLangs {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Localize resolves the recipe's generated labels for the given language code,
// then layers any per-recipe `labels:` overrides on top. An unknown language is
// a hard error that lists what is available. Calling it is idempotent.
func (r *Recipe) Localize(lang string) error {
	if lang == "" {
		lang = DefaultLang
	}
	base, ok := builtinLangs[lang]
	if !ok {
		return (&Diagnostics{}).add(
			"unknown -lang %q; available: %s", lang, joinQuoted(knownLangs()),
		)
	}
	r.L = base
	return r.applyLabelOverrides()
}

// applyLabelOverrides merges the YAML `labels:` map into the resolved strings,
// rejecting unknown keys so a typo (e.g. "satrt:") fails loudly instead of
// silently doing nothing.
func (r *Recipe) applyLabelOverrides() error {
	if len(r.Labels) == 0 {
		return nil
	}
	d := &Diagnostics{}
	for k, v := range r.Labels {
		switch k {
		case "start":
			r.L.Start = v
		case "total":
			r.L.Total = v
		default:
			d.addAt(r.labelsLine, "labels: unknown key %q (known: \"start\", \"total\")", k)
		}
	}
	return d.errorOrNil()
}

// StartLabel and TotalLabel return the resolved string with a German fallback,
// so a layout built from a hand-constructed Recipe (no Localize call) still has
// sensible text.
func (r *Recipe) StartLabel() string {
	if r.L.Start == "" {
		return builtinLangs[DefaultLang].Start
	}
	return r.L.Start
}

func (r *Recipe) TotalLabel() string {
	if r.L.Total == "" {
		return builtinLangs[DefaultLang].Total
	}
	return r.L.Total
}
