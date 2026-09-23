package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/signalandform/constelate/internal/app"
	"github.com/signalandform/constelate/internal/claude"
	"github.com/signalandform/constelate/internal/xp"
)

type nodeKind int

const (
	kindInstalled nodeKind = iota // ★ personal or project skill, editable
	kindManaged                   // ☆ synced or plugin skill, read-only
	kindAvailable                 // ○ parked in disabled/
	kindTrust                     // ◆ trust unlock
)

func (k nodeKind) glyph() string { return [...]string{"★", "☆", "○", "◆"}[k] }

type node struct {
	label     string
	kind      nodeKind
	skill     *claude.Skill
	unlock    *xp.UnlockStatus
	category  string
	suggested bool // category came from the keyword heuristic
	x, y      int  // set by layout, relative to the graph area
}

// constellation is the skill-tree screen: one tab per category.
type constellation struct {
	st     *app.State
	sty    *styles
	cats   []string          // tabs, in order; empty categories are dropped
	nodes  map[string][]node // by category
	tab    int
	sel    map[string]int // selected node index per category
	detail bool
	// last layout, for navigation and drawing
	cols, pitch, rowH int
}

const (
	minPitch = 12 // horizontal pitch bounds; actual pitch follows the longest label
	maxPitch = 22
	maxCols  = 6
	panelW   = 26
	legendH  = 1
	graphTop = 1 // one blank line under the header
)

func newConstellation(st *app.State, sty *styles) *constellation {
	c := &constellation{st: st, sty: sty, nodes: map[string][]node{}, sel: map[string]int{}}
	c.build()
	return c
}

// build groups skills and unlocks into categories. It is re-run after any
// change to State so the view never holds stale pointers.
func (c *constellation) build() {
	c.nodes = map[string][]node{}
	for i := range c.st.Skills {
		s := &c.st.Skills[i]
		if s.Hidden {
			continue
		}
		cat, sug := c.st.Cats.Resolve(string(s.Scope), s.Name, s.Description)
		n := node{label: s.Name, skill: s, category: cat, suggested: sug}
		switch s.Scope {
		case claude.ScopePersonal, claude.ScopeProject:
			n.kind = kindInstalled
		case claude.ScopeDisabled:
			n.kind = kindAvailable
		default:
			n.kind = kindManaged
		}
		c.nodes[cat] = append(c.nodes[cat], n)
	}
	for i := range c.st.Unlocks {
		u := &c.st.Unlocks[i]
		cat := u.Category
		if cat == "" {
			cat = "Ops"
		}
		c.nodes[cat] = append(c.nodes[cat], node{label: u.Name, kind: kindTrust, unlock: u, category: cat})
	}
	c.cats = c.cats[:0]
	for _, cat := range c.st.Cats.All() {
		if len(c.nodes[cat]) > 0 {
			c.cats = append(c.cats, cat)
		}
	}
	if c.tab >= len(c.cats) {
		c.tab = 0
	}
}

func (c *constellation) current() string {
	if len(c.cats) == 0 {
		return ""
	}
	return c.cats[c.tab]
}

func (c *constellation) selected() *node {
	cat := c.current()
	list := c.nodes[cat]
	if len(list) == 0 {
		return nil
	}
	i := c.sel[cat]
	if i >= len(list) {
		i = 0
	}
	return &list[i]
}

func (c *constellation) update(msg tea.KeyMsg) {
	switch msg.String() {
	case "tab":
		if len(c.cats) > 0 {
			c.tab = (c.tab + 1) % len(c.cats)
		}
		return
	case "shift+tab":
		if len(c.cats) > 0 {
			c.tab = (c.tab - 1 + len(c.cats)) % len(c.cats)
		}
		return
	case "enter":
		c.detail = !c.detail
		return
	case "esc":
		c.detail = false
		return
	}
	switch msg.String() {
	case "h", "left":
		c.move(-1, 0)
	case "l", "right":
		c.move(1, 0)
	case "k", "up":
		c.move(0, -1)
	case "j", "down":
		c.move(0, 1)
	}
}

// move picks the nearest node in the requested direction using the last
// layout's coordinates, so what you see is what the keys do.
func (c *constellation) move(dx, dy int) {
	cat := c.current()
	list := c.nodes[cat]
	if len(list) < 2 {
		return
	}
	cur := list[c.sel[cat]]
	best, bestScore := -1, 1<<30
	for i, n := range list {
		if i == c.sel[cat] {
			continue
		}
		ddx, ddy := n.x-cur.x, n.y-cur.y
		if dx != 0 && (ddx*dx <= 0) {
			continue
		}
		if dy != 0 && (ddy*dy <= 0) {
			continue
		}
		var score int
		if dx != 0 {
			score = abs(ddx) + 4*abs(ddy)
		} else {
			score = abs(ddy)*c.pitch + abs(ddx)
		}
		if score < bestScore {
			best, bestScore = i, score
		}
	}
	if best >= 0 {
		c.sel[cat] = best
	}
}

// header renders the top border: ─ CODING ─── ◂ WRITING · RESEARCH · OPS ▸ ──
func (c *constellation) header(inner int) string {
	if len(c.cats) == 0 {
		s := "─ " + c.sty.title.Render("NO SKILLS FOUND") + " "
		return s + c.sty.frame.Render(strings.Repeat("─", max(0, inner-lipWidth(s))))
	}
	active := strings.ToUpper(c.current())
	var others []string
	for i, cat := range c.cats {
		if i != c.tab {
			others = append(others, strings.ToUpper(cat))
		}
	}
	left := "─ " + c.sty.title.Render(active) + " "
	right := ""
	if len(others) > 0 {
		right = " " + c.sty.tab.Render("◂ "+strings.Join(others, " · ")+" ▸") + " "
	}
	fill := inner - lipWidth(left) - lipWidth(right)
	if fill < 1 {
		// squeeze the tab list first
		right = " " + c.sty.tab.Render(fmt.Sprintf("◂ %d more ▸", len(others))) + " "
		fill = max(1, inner-lipWidth(left)-lipWidth(right))
	}
	return left + c.sty.frame.Render(strings.Repeat("─", fill)) + right
}

// view draws the graph, optional detail panel, and legend into w x h.
func (c *constellation) view(w, h int) string {
	cv := newCanvas(w, h)
	cat := c.current()
	list := c.nodes[cat]

	graphW := w
	if c.detail {
		graphW = w - panelW - 1
	}
	c.layout(list, graphW-2, h-graphTop-legendH)
	c.nodes[cat] = list
	cv.maxX = graphW      // nothing from the graph may reach the panel
	labelW := c.pitch - 3 // glyph, space, label, one cell gap

	// links first so nodes draw over them
	for i := 0; i+1 < len(list); i++ {
		a, b := list[i], list[i+1]
		ax, ay := 2+a.x, graphTop+a.y
		bx, by := 2+b.x, graphTop+b.y
		switch {
		case ay == by:
			lo, hi, left := ax, bx, a
			if lo > hi {
				lo, hi, left = hi, lo, b
			}
			for x := lo + 2 + min(len([]rune(left.label)), labelW); x < hi; x++ {
				cv.put(x, ay, '─', &c.sty.link)
			}
		case c.rowH < 2:
			// dense mode: no link rows, rows read left to right
		case bx > ax:
			cv.put((ax+bx)/2, ay+1, '╲', &c.sty.link)
		case bx < ax:
			cv.put((ax+bx)/2, ay+1, '╱', &c.sty.link)
		default:
			cv.put(ax, ay+1, '│', &c.sty.link)
		}
	}
	selIdx := c.sel[cat]
	for i, n := range list {
		st := c.styleFor(n)
		if i == selIdx {
			st = &c.sty.nodeSel
		}
		cv.text(2+n.x, graphTop+n.y, n.kind.glyph()+" "+clip(n.label, labelW), st)
	}
	cv.maxX = w

	// legend
	legend := " ★ installed   ☆ managed   ○ available   ◆ trust unlock"
	cv.text(0, h-1, clip(legend, graphW), &c.sty.legend)

	if c.detail {
		c.drawPanel(cv, w-panelW, graphTop, panelW, h-graphTop-legendH)
	}
	return cv.render()
}

// layout assigns x,y to nodes in a zig-zag grid so consecutive nodes sit on
// diagonals and the links read as a constellation, not a table.
func (c *constellation) layout(list []node, w, h int) {
	longest := 0
	for _, n := range list {
		longest = max(longest, len([]rune(n.label)))
	}
	pitch := min(max(longest+4, minPitch), maxPitch)
	cols := min(max(1, (w-3)/pitch), maxCols)
	rows := (len(list) + cols - 1) / cols
	rowH := 2
	if rows*rowH > h {
		rowH = 1 // dense: drop the link rows rather than lose nodes
	}
	c.cols, c.pitch, c.rowH = cols, pitch, rowH
	for i := range list {
		row, col := i/cols, i%cols
		if rowH == 2 && row%2 == 1 {
			col = cols - 1 - col // snake back so neighbours stay adjacent
		}
		x := col*pitch + (row%2)*3
		y := row * rowH
		if y >= h {
			y = h - 1 // still too many: stack on the last line; scrolling comes later
		}
		list[i].x, list[i].y = x, y
	}
}

func (c *constellation) styleFor(n node) *lipgloss.Style {
	switch n.kind {
	case kindManaged:
		return &c.sty.nodeDim
	case kindAvailable:
		return &c.sty.nodeAvail
	case kindTrust:
		return &c.sty.nodeTrust
	}
	return &c.sty.node
}

// drawPanel renders the detail box for the selected node.
func (c *constellation) drawPanel(cv *canvas, x, y, w, h int) {
	n := c.selected()
	if n == nil || h < 4 {
		return
	}
	f := &c.sty.frame
	title := clip(" "+n.label+" ", w-4)
	cv.text(x, y, "┌ ", f)
	cv.text(x+2, y, title, &c.sty.panelHead)
	for i := x + 2 + len([]rune(title)); i < x+w-1; i++ {
		cv.put(i, y, '─', f)
	}
	cv.put(x+w-1, y, '┐', f)
	for row := y + 1; row < y+h-1; row++ {
		cv.put(x, row, '│', f)
		cv.put(x+w-1, row, '│', f)
	}
	cv.put(x, y+h-1, '└', f)
	for i := x + 1; i < x+w-1; i++ {
		cv.put(i, y+h-1, '─', f)
	}
	cv.put(x+w-1, y+h-1, '┘', f)

	var lines []string
	var lineStyle []*lipgloss.Style
	add := func(s string, st *lipgloss.Style) {
		lines = append(lines, s)
		lineStyle = append(lineStyle, st)
	}
	inner := w - 4
	switch n.kind {
	case kindTrust:
		u := n.unlock
		add(fmt.Sprintf("trust unlock  %d/%d", u.Count, u.Threshold), &c.sty.panel)
		switch {
		case u.Granted:
			add("granted", &c.sty.ok)
		case u.Reached:
			add("READY to suggest", &c.sty.ok)
		default:
			add("locked", &c.sty.nodeDim)
		}
		add("", nil)
		for _, l := range wrap("rule: "+u.Rule, inner) {
			add(l, &c.sty.panel)
		}
		add("", nil)
		add("[enter] step 5", &c.sty.help)
	default:
		s := n.skill
		state := map[nodeKind]string{kindInstalled: "installed", kindManaged: "managed by Claude Code", kindAvailable: "not installed"}[n.kind]
		add(state, &c.sty.panel)
		add(fmt.Sprintf("scope: %s", s.Scope), &c.sty.panel)
		add(fmt.Sprintf("ctx: ~%d tok (est.)", s.CtxEstimate()), &c.sty.panel)
		cat := n.category
		if n.suggested {
			cat += " (suggested)"
		}
		add(clip("in: "+cat, inner), &c.sty.panel)
		if s.Invocable {
			add("user-invocable", &c.sty.panel)
		}
		add("", nil)
		desc := wrap(s.Description, inner)
		for i, l := range desc {
			if len(lines) >= h-4 {
				add("…", &c.sty.nodeDim)
				break
			}
			_ = i
			add(l, &c.sty.panel)
		}
		if len(s.Warnings) > 0 {
			add("", nil)
			add(clip("! "+s.Warnings[0], inner), &c.sty.warn)
		}
		add("", nil)
		switch n.kind {
		case kindInstalled:
			add("[enter] disable (step 3)", &c.sty.help)
		case kindAvailable:
			add("[enter] install (step 3)", &c.sty.help)
		default:
			add("read-only", &c.sty.help)
		}
	}
	for i, l := range lines {
		if i >= h-2 {
			break
		}
		cv.text(x+2, y+1+i, clip(l, inner), lineStyle[i])
	}
}
