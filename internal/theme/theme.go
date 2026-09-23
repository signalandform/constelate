// Package theme maps the active Omarchy theme onto Constelate's palette and
// falls back to plain ANSI colours anywhere else. It is read once at launch.
package theme

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pelletier/go-toml/v2"
)

// Palette is every colour the UI uses. Keep it small so themes map cleanly.
type Palette struct {
	Name   string // theme name or "ansi"
	Source string // file the colours came from, or "" for the fallback
	Dark   bool

	Accent  lipgloss.Color // headings, selected node
	Muted   lipgloss.Color // connectors, legend, borders
	Fg      lipgloss.Color
	Dim     lipgloss.Color // read-only / disabled nodes
	Bg      lipgloss.Color
	Panel   lipgloss.Color // detail panel background
	Red     lipgloss.Color
	Yellow  lipgloss.Color
	Green   lipgloss.Color
	Cyan    lipgloss.Color
	Blue    lipgloss.Color
	Magenta lipgloss.Color
}

// ANSI is the fallback: standard 16-colour terminal palette, no background.
func ANSI() Palette {
	return Palette{
		Name: "ansi", Dark: true,
		Accent: "12", Muted: "8", Fg: "7", Dim: "8", Bg: "", Panel: "",
		Red: "9", Yellow: "11", Green: "10", Cyan: "14", Blue: "12", Magenta: "13",
	}
}

// Candidates lists where an Omarchy install keeps the active theme, newest
// layout first. Each entry is a directory; we look for colors.toml inside.
func Candidates(home string) []string {
	var out []string
	if d := os.Getenv("OMARCHY_THEME_DIR"); d != "" {
		out = append(out, d)
	}
	out = append(out,
		filepath.Join(home, ".local", "state", "omarchy", "current", "theme"),
		filepath.Join(home, ".config", "omarchy", "current", "theme"),
	)
	return out
}

// Load probes the candidates and returns the first theme that parses. When
// none does it returns ANSI() and a nil error; a theme that exists but is
// unreadable returns ANSI() and the error so the UI can mention it.
func Load(home string) (Palette, error) {
	var errs []error
	for _, dir := range Candidates(home) {
		p := filepath.Join(dir, "colors.toml")
		b, err := os.ReadFile(p)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				errs = append(errs, err)
			}
			continue
		}
		pal, err := ParseColors(b)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		pal.Source = p
		if name, err := os.ReadFile(filepath.Join(filepath.Dir(dir), "theme.name")); err == nil {
			pal.Name = strings.TrimSpace(string(name))
		} else if real, err := filepath.EvalSymlinks(dir); err == nil {
			pal.Name = filepath.Base(real)
		} else {
			pal.Name = "omarchy"
		}
		return pal, nil
	}
	return ANSI(), errors.Join(errs...)
}

// colorsFile mirrors Omarchy's colors.toml. Every field is optional; missing
// ones fall back to the ANSI palette so a partial theme still renders.
type colorsFile struct {
	Mode              string `toml:"mode"`
	Accent            string `toml:"accent"`
	Muted             string `toml:"muted"`
	Selection         string `toml:"selection"`
	Background        string `toml:"background"`
	LighterBackground string `toml:"lighter_background"`
	Foreground        string `toml:"foreground"`
	DarkForeground    string `toml:"dark_foreground"`
	Red               string `toml:"red"`
	Yellow            string `toml:"yellow"`
	Green             string `toml:"green"`
	Cyan              string `toml:"cyan"`
	Blue              string `toml:"blue"`
	Magenta           string `toml:"magenta"`
}

// ParseColors builds a Palette from colors.toml bytes.
func ParseColors(b []byte) (Palette, error) {
	var c colorsFile
	if err := toml.Unmarshal(b, &c); err != nil {
		return Palette{}, err
	}
	if c.Background == "" && c.Foreground == "" && c.Accent == "" {
		return Palette{}, errors.New("colors.toml has none of background, foreground, accent")
	}
	base := ANSI()
	pick := func(hex string, fallback lipgloss.Color) lipgloss.Color {
		hex = strings.TrimSpace(hex)
		if len(hex) == 7 && hex[0] == '#' {
			return lipgloss.Color(hex)
		}
		return fallback
	}
	p := Palette{
		Name:    "omarchy",
		Dark:    !strings.EqualFold(c.Mode, "light"),
		Accent:  pick(c.Accent, base.Accent),
		Muted:   pick(c.Muted, base.Muted),
		Fg:      pick(c.Foreground, base.Fg),
		Dim:     pick(c.DarkForeground, base.Dim),
		Bg:      pick(c.Background, base.Bg),
		Panel:   pick(c.LighterBackground, pick(c.Selection, base.Panel)),
		Red:     pick(c.Red, base.Red),
		Yellow:  pick(c.Yellow, base.Yellow),
		Green:   pick(c.Green, base.Green),
		Cyan:    pick(c.Cyan, base.Cyan),
		Blue:    pick(c.Blue, base.Blue),
		Magenta: pick(c.Magenta, base.Magenta),
	}
	return p, nil
}
