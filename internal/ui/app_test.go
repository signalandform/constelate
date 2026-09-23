package ui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/signalandform/constelate/internal/app"
	"github.com/signalandform/constelate/internal/categories"
	"github.com/signalandform/constelate/internal/claude"
	"github.com/signalandform/constelate/internal/config"
	"github.com/signalandform/constelate/internal/ctxcost"
	"github.com/signalandform/constelate/internal/theme"
	"github.com/signalandform/constelate/internal/xp"
)

// fixtureState builds a State from testdata without touching the real home.
func fixtureState(t *testing.T) *app.State {
	t.Helper()
	root, _ := filepath.Abs(filepath.Join("..", "..", "testdata"))
	var skills []claude.Skill
	for _, d := range []string{"3d-web-experience", "accessibility", "data-visualization", "find-skills", "no-frontmatter"} {
		skills = append(skills, claude.ParseSkillFile(filepath.Join(root, "skills", d, "SKILL.md"), claude.ScopePersonal))
	}
	// a read-only one and an available one
	m := claude.ParseSkillFile(filepath.Join(root, "skills", "accessibility", "SKILL.md"), claude.ScopePlugin)
	m.Name = "plugin-a11y"
	a := claude.ParseSkillFile(filepath.Join(root, "skills", "find-skills", "SKILL.md"), claude.ScopeDisabled)
	a.Name = "parked-skill"
	skills = append(skills, m, a)

	sess, err := claude.ParseSession(filepath.Join(root, "transcripts", "session-a.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	st := &app.State{
		Opts:     app.Options{Project: "/tmp/proj"},
		Roots:    claude.Roots{Home: root, Disabled: filepath.Join(root, "disabled")},
		Config:   cfg,
		Cats:     categories.Store{File: categories.File{Assign: map[string]string{}}},
		Theme:    theme.ANSI(),
		Skills:   skills,
		Sessions: []claude.Session{sess},
	}
	st.Ledger = xp.Build(st.Sessions, cfg)
	st.Unlocks = xp.Unlocks(st.Ledger, cfg, nil)
	st.Ctx = ctxcost.Estimate(skills, cfg.CtxBudget)
	return st
}

func size(m Model, w, h int) Model {
	mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return mm.(Model)
}

func key(m Model, k string) Model {
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
	mm, _ := m.Update(msg)
	return mm.(Model)
}

func checkFrame(t *testing.T, view string, w, h int, label string) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) != h {
		t.Errorf("%s: %d lines, want %d", label, len(lines), h)
	}
	for i, l := range lines {
		if got := lipgloss.Width(l); got != w {
			t.Errorf("%s: line %d width %d, want %d: %q", label, i, got, w, l)
		}
	}
}

func TestRendersExactly80x24(t *testing.T) {
	m := size(New(fixtureState(t)), 80, 24)
	checkFrame(t, m.View(), 80, 24, "constellation")
	m = key(m, "enter")
	checkFrame(t, m.View(), 80, 24, "with detail")
	m = key(m, "tab")
	checkFrame(t, m.View(), 80, 24, "next tab")
	m = key(m, "2")
	checkFrame(t, m.View(), 80, 24, "sheet")
	m = key(m, "3")
	checkFrame(t, m.View(), 80, 24, "ledger")
	m = key(m, "?")
	checkFrame(t, m.View(), 80, 24, "help")
}

func TestRendersWide(t *testing.T) {
	m := size(New(fixtureState(t)), 132, 40)
	m = key(m, "enter")
	checkFrame(t, m.View(), 132, 40, "wide with detail")
}

func TestTooSmallSaysSo(t *testing.T) {
	m := size(New(fixtureState(t)), 60, 20)
	if !strings.Contains(m.View(), "at least") {
		t.Error("no resize hint")
	}
}

func TestNavigationMovesSelection(t *testing.T) {
	m := size(New(fixtureState(t)), 80, 24)
	_ = m.View() // layout happens in view
	before := m.cons.selected().label
	m = key(m, "l")
	_ = m.View()
	if after := m.cons.selected().label; after == before && len(m.cons.nodes[m.cons.current()]) > 1 {
		t.Errorf("l did not move: %s", after)
	}
	m = key(m, "h")
	_ = m.View()
	if got := m.cons.selected().label; got != before {
		t.Errorf("h did not return: %s vs %s", got, before)
	}
}

func TestModalFitsFrame(t *testing.T) {
	st := fixtureState(t)
	st.Opts.DryRun = true
	m := size(New(st), 80, 24)
	_ = m.View()
	for i := 0; i < len(m.cons.cats) && m.cons.selected().kind != kindInstalled; i++ {
		m = key(m, "tab") // find a tab whose first node is a skill we own
		_ = m.View()
	}
	m = key(m, "enter") // detail
	m = key(m, "enter") // plan modal (dry run)
	if m.cons.modal == nil {
		t.Fatalf("no modal; status=%q", m.cons.status)
	}
	checkFrame(t, m.View(), 80, 24, "modal")
	m = key(m, "j")
	checkFrame(t, m.View(), 80, 24, "modal scrolled")
	m = key(m, "n")
	if m.cons.modal != nil {
		t.Error("n did not close the modal")
	}
	checkFrame(t, m.View(), 80, 24, "after cancel")
}

func TestSheetStatesFitFrame(t *testing.T) {
	st := fixtureState(t)
	st.Opts.DryRun = true
	m := size(New(st), 80, 24)
	m = key(m, "2")
	checkFrame(t, m.View(), 80, 24, "sheet")
	m = key(m, "enter") // name input
	checkFrame(t, m.View(), 80, 24, "name editing")
	if !m.sheet.capturing() {
		t.Error("name input not capturing")
	}
	m = key(m, "q") // must type, not quit
	if m.sheet.nameIn.Value() != "q" {
		t.Errorf("q was not typed into the name: %q", m.sheet.nameIn.Value())
	}
	m = key(m, "esc")
	m = key(m, "tab")
	m = key(m, "enter") // model picker
	checkFrame(t, m.View(), 80, 24, "picker")
	m = key(m, "j")
	m = key(m, "enter") // plan modal, dry run
	if m.sheet.modal == nil {
		t.Fatalf("no model plan modal; status=%q", m.sheet.status)
	}
	checkFrame(t, m.View(), 80, 24, "model modal")
	m = key(m, "n")
	m = key(m, "tab")
	m = key(m, "enter") // textarea (no CLAUDE.md in fixture -> status only)
	checkFrame(t, m.View(), 80, 24, "instructions")
	m = size(m, 132, 40)
	checkFrame(t, m.View(), 132, 40, "sheet wide")
}
