package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/signalandform/constelate/internal/theme"
)

// styles are built once from the palette. Pointers so the canvas can compare
// them cheaply when merging runs.
type styles struct {
	pal       theme.Palette
	frame     lipgloss.Style
	title     lipgloss.Style
	tab       lipgloss.Style
	tabActive lipgloss.Style
	node      lipgloss.Style
	nodeSel   lipgloss.Style
	nodeDim   lipgloss.Style
	nodeAvail lipgloss.Style
	nodeTrust lipgloss.Style
	link      lipgloss.Style
	legend    lipgloss.Style
	panel     lipgloss.Style
	panelHead lipgloss.Style
	warn      lipgloss.Style
	ok        lipgloss.Style
	status    lipgloss.Style
	help      lipgloss.Style
}

func newStyles(p theme.Palette) *styles {
	s := &styles{pal: p}
	s.frame = lipgloss.NewStyle().Foreground(p.Muted)
	s.title = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	s.tab = lipgloss.NewStyle().Foreground(p.Dim)
	s.tabActive = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	s.node = lipgloss.NewStyle().Foreground(p.Fg)
	s.nodeSel = lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Reverse(true)
	s.nodeDim = lipgloss.NewStyle().Foreground(p.Dim)
	s.nodeAvail = lipgloss.NewStyle().Foreground(p.Cyan)
	s.nodeTrust = lipgloss.NewStyle().Foreground(p.Magenta)
	s.link = lipgloss.NewStyle().Foreground(p.Muted)
	s.legend = lipgloss.NewStyle().Foreground(p.Dim)
	s.panel = lipgloss.NewStyle().Foreground(p.Fg)
	s.panelHead = lipgloss.NewStyle().Foreground(p.Yellow).Bold(true)
	s.warn = lipgloss.NewStyle().Foreground(p.Red).Bold(true)
	s.ok = lipgloss.NewStyle().Foreground(p.Green)
	s.status = lipgloss.NewStyle().Foreground(p.Fg)
	s.help = lipgloss.NewStyle().Foreground(p.Dim)
	return s
}
