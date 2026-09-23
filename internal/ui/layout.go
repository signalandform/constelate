package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Minimum terminal size the layout is designed for.
const (
	MinWidth  = 80
	MinHeight = 24
)

// canvas is a character grid with a style id per cell. The constellation is
// drawn onto it, then rendered row by row with runs of equal style merged so
// Lip Gloss escape sequences stay short.
type canvas struct {
	w, h  int
	maxX  int // puts at x >= maxX are dropped while > 0; fences the graph off the panel
	cells [][]rune
	style [][]*lipgloss.Style
}

func newCanvas(w, h int) *canvas {
	c := &canvas{w: w, h: h, maxX: w}
	c.cells = make([][]rune, h)
	c.style = make([][]*lipgloss.Style, h)
	for y := range c.cells {
		c.cells[y] = make([]rune, w)
		c.style[y] = make([]*lipgloss.Style, w)
		for x := range c.cells[y] {
			c.cells[y][x] = ' '
		}
	}
	return c
}

func (c *canvas) put(x, y int, r rune, st *lipgloss.Style) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h || x >= c.maxX {
		return
	}
	c.cells[y][x] = r
	c.style[y][x] = st
}

func (c *canvas) text(x, y int, s string, st *lipgloss.Style) {
	for i, r := range []rune(s) {
		c.put(x+i, y, r, st)
	}
}

func (c *canvas) render() string {
	var sb strings.Builder
	for y := 0; y < c.h; y++ {
		x := 0
		for x < c.w {
			st := c.style[y][x]
			start := x
			for x < c.w && c.style[y][x] == st {
				x++
			}
			run := string(c.cells[y][start:x])
			if st != nil {
				run = st.Render(run)
			}
			sb.WriteString(run)
		}
		if y < c.h-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// bar draws a filled/empty meter of the given width.
func bar(filled, total, width int) string {
	if total <= 0 || width <= 0 {
		return strings.Repeat("░", max(width, 0))
	}
	n := filled * width / total
	if n > width {
		n = width
	}
	if n < 0 {
		n = 0
	}
	return strings.Repeat("█", n) + strings.Repeat("░", width-n)
}

// wrap breaks s into lines no wider than w, on spaces.
func wrap(s string, w int) []string {
	if w <= 0 {
		return nil
	}
	var lines []string
	var cur strings.Builder
	for _, word := range strings.Fields(s) {
		if cur.Len() > 0 && cur.Len()+1+len(word) > w {
			lines = append(lines, cur.String())
			cur.Reset()
		}
		if cur.Len() > 0 {
			cur.WriteByte(' ')
		}
		if len(word) > w {
			word = word[:w-1] + "…"
		}
		cur.WriteString(word)
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

func clip(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return string(r[:w])
	}
	return string(r[:w-1]) + "…"
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
