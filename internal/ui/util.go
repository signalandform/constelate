package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func lipWidth(s string) int { return lipgloss.Width(s) }

// padRight pads or truncates a possibly styled string to exactly w cells.
func padRight(s string, w int) string {
	n := w - lipWidth(s)
	if n < 0 {
		return ansi.Truncate(s, w, "…")
	}
	return s + strings.Repeat(" ", n)
}
