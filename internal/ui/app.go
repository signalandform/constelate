// Package ui is the Bubble Tea front end. It renders app.State and, from
// build step 3 on, routes edits through the writer package. Nothing here
// touches disk directly.
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/signalandform/constelate/internal/app"
	"github.com/signalandform/constelate/internal/writer"
)

type screen int

const (
	screenConstellation screen = iota
	screenSheet
	screenLedger
)

var screenNames = []string{"CONSTELLATION", "CHARACTER", "LEDGER"}

// Model is the root Bubble Tea model.
type Model struct {
	st     *app.State
	sty    *styles
	w, h   int
	screen screen
	cons   *constellation
	help   bool
}

// New builds the root model. Size arrives with the first WindowSizeMsg.
func New(st *app.State) Model {
	sty := newStyles(st.Theme)
	return Model{st: st, sty: sty, cons: newConstellation(st, sty)}
}

// Run starts the program in the alternate screen.
func Run(st *app.State) error {
	_, err := tea.NewProgram(New(st), tea.WithAltScreen()).Run()
	return err
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.screen == screenConstellation && m.cons.modal != nil {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			m.cons.update(msg, m.applier())
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.help = !m.help
			return m, nil
		case "1":
			m.screen = screenConstellation
			return m, nil
		case "2":
			m.screen = screenSheet
			return m, nil
		case "3":
			m.screen = screenLedger
			return m, nil
		}
		if m.screen == screenConstellation {
			m.cons.update(msg, m.applier())
		}
	}
	return m, nil
}

// View lays out the frame:
//
//	┌─ header ────────────────────────┐   1 line
//	│ body                            │   h-4 lines
//	├─────────────────────────────────┤   1
//	│ status                          │   1
//	└─ keys ──────────────────────────┘   1
func (m Model) View() string {
	if m.w == 0 || m.h == 0 {
		return ""
	}
	if m.w < MinWidth || m.h < MinHeight {
		return fmt.Sprintf("Const.Elate needs at least %dx%d (have %dx%d). Resize or press q.",
			MinWidth, MinHeight, m.w, m.h)
	}
	inner := m.w - 2
	bodyH := m.h - 4

	var header, body string
	switch m.screen {
	case screenConstellation:
		header = m.cons.header(inner)
		body = m.cons.view(inner, bodyH)
	case screenSheet:
		header = m.plainHeader("CHARACTER SHEET", inner)
		body = m.placeholder(inner, bodyH, "Character sheet arrives in build step 4.")
	case screenLedger:
		header = m.plainHeader("LEDGER", inner)
		body = m.placeholder(inner, bodyH, "XP ledger and trust unlocks arrive in build step 5.")
	}

	f := m.sty.frame
	var sb strings.Builder
	sb.WriteString(f.Render("┌") + header + f.Render("┐") + "\n")
	for _, line := range strings.Split(body, "\n") {
		sb.WriteString(f.Render("│") + line + f.Render("│") + "\n")
	}
	sb.WriteString(f.Render("├"+strings.Repeat("─", inner)+"┤") + "\n")
	sb.WriteString(f.Render("│") + m.statusLine(inner) + f.Render("│") + "\n")
	sb.WriteString(f.Render("└") + m.keysLine(inner) + f.Render("┘"))
	return sb.String()
}

func (m Model) plainHeader(title string, inner int) string {
	s := "─ " + m.sty.title.Render(title) + " "
	return s + m.sty.frame.Render(strings.Repeat("─", max(0, inner-lipWidth(s))))
}

func (m Model) placeholder(inner, h int, msg string) string {
	lines := make([]string, h)
	for i := range lines {
		lines[i] = strings.Repeat(" ", inner)
	}
	if h > 2 {
		lines[h/2] = padRight("  "+m.sty.help.Render(msg), inner)
	}
	return strings.Join(lines, "\n")
}

// statusLine: LVL 7  XP ████████░░ 820/1000   CTX ██░░░░░░ ~2.1k/4k tok
func (m Model) statusLine(inner int) string {
	l := m.st.Ledger
	c := m.st.Ctx
	xpBar := bar(l.IntoLvl, l.LevelXP, 10)
	ctxBar := bar(c.Loaded, c.Budget, 8)
	ctxStyle := m.sty.ok
	if c.Over {
		ctxStyle = m.sty.warn
	}
	s := fmt.Sprintf(" %s %s  %s %s %d/%d   %s %s ~%s/%s tok",
		m.sty.title.Render("LVL"), m.sty.status.Render(fmt.Sprint(l.Level)),
		m.sty.title.Render("XP"), m.sty.ok.Render(xpBar), l.IntoLvl, l.LevelXP,
		m.sty.title.Render("CTX"), ctxStyle.Render(ctxBar), kfmt(c.Loaded), kfmt(c.Budget))
	if len(m.st.Problems) > 0 {
		s += "  " + m.sty.warn.Render(fmt.Sprintf("%d load issue(s), see --summary", len(m.st.Problems)))
	}
	return padRight(s, inner)
}

func (m Model) keysLine(inner int) string {
	var keys string
	if m.help {
		keys = " h/j/k/l or arrows move · tab/shift+tab constellation · enter detail · esc close · 1/2/3 screens · q quit "
	} else {
		keys = fmt.Sprintf(" %s · %s · ? keys · q quit · theme %s ",
			m.sty.tabActive.Render("1 "+screenNames[0]), m.sty.tab.Render("2 CHARACTER · 3 LEDGER"), m.st.Theme.Name)
		if m.screen != screenConstellation {
			keys = fmt.Sprintf(" 1 %s · 2 %s · 3 %s · ? keys · q quit ",
				screenNames[0], screenNames[1], screenNames[2])
		}
	}
	w := lipWidth(keys)
	if w > inner-1 {
		keys = ansi.Truncate(keys, inner-1, "…")
		w = lipWidth(keys)
	}
	return "─" + keys + m.sty.frame.Render(strings.Repeat("─", max(0, inner-w-1)))
}

func kfmt(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprint(n)
}

// RenderOnce draws a single frame at the given size without starting the
// program. Used by --render for screenshots and headless checks.
func RenderOnce(st *app.State, w, h int, keys []string) string {
	m := New(st)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = mm.(Model)
	for _, k := range keys {
		_ = m.View() // layout must run before movement keys mean anything
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		mm, _ = m.Update(msg)
		m = mm.(Model)
	}
	return m.View()
}

// applier runs a plan through State.Apply and, on success, reloads every
// screen from the fresh state. The pointer swap is what makes the reload
// visible: Model is a value, but m.st and m.cons are shared pointers.
func (m Model) applier() applyFn {
	return func(p writer.Plan) (writer.Result, error) {
		res, fresh, err := m.st.Apply(p)
		if err != nil {
			return res, err
		}
		if fresh != m.st {
			*m.st = *fresh
			m.cons.reset(m.st)
		}
		return res, nil
	}
}
