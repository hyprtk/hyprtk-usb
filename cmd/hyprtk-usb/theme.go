package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
)

// Palette is the hyprtk accent scheme, resolved from pywal when available.
// Defaults mirror hyprtk's fallback colours: mauve accent, sky secondary.
type Palette struct {
	Accent  string
	Accent2 string
	Fg      string
	Dim     string
	BG      string
	Err     string
	Warn    string
}

func defaultPalette() Palette {
	return Palette{
		Accent:  "#c084fc",
		Accent2: "#22d3ee",
		Fg:      "#e5e7eb",
		Dim:     "#6b7280",
		BG:      "#1e1e2e",
		Err:     "#f38ba8",
		Warn:    "#f9e2af",
	}
}

// loadPalette overlays ~/.cache/wal/colors.json (pywal16) on the defaults.
func loadPalette() Palette {
	p := defaultPalette()
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	data, err := os.ReadFile(filepath.Join(home, ".cache", "wal", "colors.json"))
	if err != nil {
		return p
	}
	var raw map[string]string
	if json.Unmarshal(data, &raw) != nil {
		return p
	}
	get := func(key, fallback string) string {
		if v := raw[key]; strings.TrimSpace(v) != "" {
			return v
		}
		return fallback
	}
	p.Accent = get("color5", p.Accent)
	p.Accent2 = get("color6", p.Accent2)
	p.Fg = get("color7", p.Fg)
	p.Dim = get("color8", p.Dim)
	p.BG = get("background", p.BG)
	p.Err = get("color1", p.Err)
	p.Warn = get("color3", p.Warn)
	return p
}

// styles holds the lipgloss styles built from a palette.
type styles struct {
	title, dim, accent, accent2, err, warn, ok lipgloss.Style
	selected, box, help                        lipgloss.Style
}

func newStyles(p Palette) styles {
	base := lipgloss.NewStyle().Foreground(lipgloss.Color(p.Fg))
	return styles{
		title:    base.Bold(true).Foreground(lipgloss.Color(p.Accent)),
		dim:      base.Foreground(lipgloss.Color(p.Dim)),
		accent:   base.Foreground(lipgloss.Color(p.Accent)),
		accent2:  base.Foreground(lipgloss.Color(p.Accent2)),
		err:      base.Foreground(lipgloss.Color(p.Err)),
		warn:     base.Foreground(lipgloss.Color(p.Warn)),
		ok:       base.Foreground(lipgloss.Color(p.Accent2)),
		selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(p.BG)).Background(lipgloss.Color(p.Accent)),
		box: lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color(p.Accent)).
			Padding(0, 2),
		help: base.Foreground(lipgloss.Color(p.Dim)),
	}
}
