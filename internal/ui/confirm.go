package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/signalandform/constelate/internal/writer"
)

// confirm is the modal shown before any write: the plan as a diff, where the
// backup will go, and y/n. It owns no state beyond the plan and a scroll.
type confirm struct {
	plan   writer.Plan
	lines  []string
	scroll int
	err    string
	// after apply
	done   bool
	result writer.Result
}

func newConfirm(p writer.Plan, dryRun bool) *confirm {
	c := &confirm{plan: p}
	for i, op := range p.Ops {
		if i > 0 {
			c.lines = append(c.lines, "")
		}
		if op.Why != "" {
			c.lines = append(c.lines, "# "+op.Why)
		}
		c.lines = append(c.lines, writer.Diff(op)...)
	}
	if dryRun {
		c.lines = append([]string{"# DRY RUN: nothing will be written"}, c.lines...)
	}
	return c
}

// update returns true when the modal should close.
func (c *confirm) update(msg tea.KeyMsg, apply func(writer.Plan) (writer.Result, error)) bool {
	if c.done || c.err != "" {
		return true // any key dismisses the result
	}
	switch msg.String() {
	case "y", "Y", "enter":
		res, err := apply(c.plan)
		if err != nil {
			c.err = err.Error()
			return false
		}
		c.result, c.done = res, true
		return false
	case "n", "N", "esc", "q":
		return true
	case "j", "down":
		c.scroll++
	case "k", "up":
		if c.scroll > 0 {
			c.scroll--
		}
	}
	return false
}

// draw paints the modal centred on the canvas.
func (c *confirm) draw(cv *canvas, sty *styles, w, h int) {
	bw := min(w-4, 72)
	bh := min(h-2, 18)
	x0, y0 := (w-bw)/2, (h-bh)/2
	f := &sty.frame
	// clear the area
	for y := y0; y < y0+bh; y++ {
		for x := x0; x < x0+bw; x++ {
			cv.put(x, y, ' ', nil)
		}
	}
	title := " " + c.plan.Title + " "
	cv.text(x0, y0, "┌", f)
	cv.text(x0+1, y0, clip(title, bw-2), &sty.panelHead)
	for x := x0 + 1 + lipWidth(clip(title, bw-2)); x < x0+bw-1; x++ {
		cv.put(x, y0, '─', f)
	}
	cv.put(x0+bw-1, y0, '┐', f)
	for y := y0 + 1; y < y0+bh-1; y++ {
		cv.put(x0, y, '│', f)
		cv.put(x0+bw-1, y, '│', f)
	}
	cv.put(x0, y0+bh-1, '└', f)
	for x := x0 + 1; x < x0+bw-1; x++ {
		cv.put(x, y0+bh-1, '─', f)
	}
	cv.put(x0+bw-1, y0+bh-1, '┘', f)

	inner := bw - 4
	body := c.lines
	var footer string
	switch {
	case c.err != "":
		body = wrap("error: "+c.err, inner)
		footer = " any key to close "
	case c.done:
		body = nil
		if c.result.DryRun {
			body = append(body, "dry run, nothing written:")
		} else {
			body = append(body, "done:")
		}
		for _, d := range c.result.Done {
			body = append(body, wrap("  "+d, inner)...)
		}
		if c.result.BackupDir != "" {
			body = append(body, "", "backup:")
			body = append(body, wrap("  "+c.result.BackupDir, inner)...)
		}
		footer = " any key to close "
	default:
		footer = " [y] apply   [n] cancel   j/k scroll "
	}
	avail := bh - 3
	if c.scroll > max(0, len(body)-avail) {
		c.scroll = max(0, len(body)-avail)
	}
	for i := 0; i < avail && c.scroll+i < len(body); i++ {
		l := body[c.scroll+i]
		st := &sty.panel
		switch {
		case strings.HasPrefix(l, "+"):
			st = &sty.ok
		case strings.HasPrefix(l, "-"):
			st = &sty.warn
		case strings.HasPrefix(l, "#"), strings.HasPrefix(l, "@@"):
			st = &sty.help
		case c.err != "":
			st = &sty.warn
		}
		cv.text(x0+2, y0+1+i, clip(l, inner), st)
	}
	if len(body) > avail {
		footer = fmt.Sprintf(" %d/%d ", c.scroll+1, len(body)-avail+1) + footer
	}
	cv.text(x0+bw-1-lipWidth(footer)-1, y0+bh-1, footer, &sty.help)
}
