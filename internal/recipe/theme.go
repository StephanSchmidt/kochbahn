package recipe

import "sort"

// Theme is a named lane-color preset. A theme supplies the palette used for
// lanes that do not set their own color; a theme marked Force also overrides
// explicit lane colors (used by the print-oriented grayscale theme, where the
// whole point is to flatten color).
type Theme struct {
	Palette [MaxLanes]string
	Force   bool
}

// themes are the built-in presets selectable with -theme. "default" reproduces
// the original ColorHunt palette so existing output is unchanged.
var themes = map[string]Theme{
	"default": {Palette: palette},
	"warm": {Palette: [MaxLanes]string{
		"#e76f51", // terracotta
		"#f4a261", // sand
		"#e9c46a", // amber
		"#d62828", // chili red
		"#bc6c25", // umber
	}},
	"cool": {Palette: [MaxLanes]string{
		"#2a9d8f", // teal
		"#264653", // slate
		"#4895ef", // azure
		"#6a4c93", // indigo
		"#43aa8b", // sea green
	}},
	"high-contrast": {Palette: [MaxLanes]string{
		"#000000", // black
		"#005bbb", // strong blue
		"#c1121f", // strong red
		"#1a7431", // strong green
		"#6a00f4", // strong violet
	}},
	// Grayscale ramp for cheap black-and-white book printing. Force: it
	// overrides any per-lane color so nothing prints as a muddy gray surprise.
	"mono": {Force: true, Palette: [MaxLanes]string{
		"#222222",
		"#555555",
		"#777777",
		"#999999",
		"#444444",
	}},
}

// themeAliases maps friendlier names onto canonical theme keys.
var themeAliases = map[string]string{
	"print-bw":  "mono",
	"grayscale": "mono",
	"greyscale": "mono",
	"bw":        "mono",
}

// knownThemes returns the selectable theme names (canonical + aliases), sorted.
func knownThemes() []string {
	out := make([]string, 0, len(themes)+len(themeAliases))
	for k := range themes {
		out = append(out, k)
	}
	for k := range themeAliases {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ApplyTheme re-resolves lane colors against the named preset. Non-Force themes
// only fill lanes that left `color:` empty (honoring explicit choices); a Force
// theme overrides every lane. An unknown theme is a hard error listing the
// valid names.
func (r *Recipe) ApplyTheme(name string) error {
	if name == "" {
		name = "default"
	}
	if canon, ok := themeAliases[name]; ok {
		name = canon
	}
	th, ok := themes[name]
	if !ok {
		return (&Diagnostics{}).add(
			"unknown -theme %q; available: %s", name, joinQuoted(knownThemes()),
		)
	}
	for i := range r.Lanes {
		if i >= MaxLanes {
			break
		}
		if th.Force || !r.Lanes[i].explicitColor {
			r.Lanes[i].Color = th.Palette[i]
		}
	}
	return nil
}
