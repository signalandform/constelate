package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/signalandform/constelate/internal/app"
)

type sheetField int

const (
	fieldName sheetField = iota
	fieldModel
	fieldInstr
)

// sheet is the character screen: name, model, and the CLAUDE.md files.
type sheet struct {
	st     *app.State
	sty    *styles
	focus  sheetField
	modal  *confirm
	status string

	// name editing
	nameIn  textinput.Model
	editing bool

	// model picker
	picking  bool
	pickIdx  int
	pickList []string

	// instructions
	instr    textarea.Model
	instrIdx int    // which CLAUDE.md: index into st.Instr
	instrTxt string // text as loaded, to detect changes
	typing   bool   // textarea has focus and swallows keys
}

// editorDoneMsg arrives when the external editor exits.
type editorDoneMsg struct {
	tmp  string
	path string
	err  error
}

func newSheet(st *app.State, sty *styles) *sheet {
	s := &sheet{st: st, sty: sty}
	s.nameIn = textinput.New()
	s.nameIn.CharLimit = 32
	s.nameIn.Prompt = ""
	s.nameIn.Placeholder = "agent name"
	s.nameIn.Width = 24
	s.instr = textarea.New()
	s.instr.ShowLineNumbers = false
	s.instr.Prompt = "│ "
	s.instr.CharLimit = 0
	s.loadInstr(0)
	return s
}

func (s *sheet) agentName() string {
	if n := strings.TrimSpace(s.st.Config.AgentName); n != "" {
		return n
	}
	return "unnamed"
}

func (s *sheet) loadInstr(i int) {
	if i < 0 || i >= len(s.st.Instr) {
		return
	}
	s.instrIdx = i
	b, _ := os.ReadFile(s.st.Instr[i].Path)
	s.instrTxt = string(b)
	s.instr.SetValue(s.instrTxt)
	s.instr.CursorStart()
}

func (s *sheet) dirty() bool { return s.instr.Value() != s.instrTxt }

// capturing reports whether keys must go to a text control, not the app.
func (s *sheet) capturing() bool { return s.editing || s.typing || s.picking || s.modal != nil }

func (s *sheet) reset(st *app.State) {
	s.st = st
	s.loadInstr(min(s.instrIdx, max(0, len(st.Instr)-1)))
}

func (s *sheet) update(msg tea.Msg, apply applyFn) tea.Cmd {
	switch msg := msg.(type) {
	case editorDoneMsg:
		defer os.Remove(msg.tmp)
		if msg.err != nil {
			s.status = "editor: " + msg.err.Error()
			return nil
		}
		b, err := os.ReadFile(msg.tmp)
		if err != nil {
			s.status = err.Error()
			return nil
		}
		s.instr.SetValue(string(b))
		s.propose(msg.path, string(b))
		return nil
	case tea.KeyMsg:
		return s.key(msg, apply)
	}
	if s.typing {
		var cmd tea.Cmd
		s.instr, cmd = s.instr.Update(msg)
		return cmd
	}
	return nil
}

func (s *sheet) key(msg tea.KeyMsg, apply applyFn) tea.Cmd {
	s.status = ""
	if s.modal != nil {
		if s.modal.update(msg, apply) {
			s.modal = nil
		}
		return nil
	}
	if s.editing {
		switch msg.String() {
		case "enter":
			s.editing = false
			s.nameIn.Blur()
			plan, err := s.st.PlanAgentName(s.nameIn.Value())
			if err != nil {
				s.status = err.Error()
				return nil
			}
			s.modal = newConfirm(plan, s.st.Opts.DryRun)
		case "esc":
			s.editing = false
			s.nameIn.Blur()
		default:
			var cmd tea.Cmd
			s.nameIn, cmd = s.nameIn.Update(msg)
			return cmd
		}
		return nil
	}
	if s.picking {
		switch msg.String() {
		case "j", "down":
			s.pickIdx = (s.pickIdx + 1) % len(s.pickList)
		case "k", "up":
			s.pickIdx = (s.pickIdx - 1 + len(s.pickList)) % len(s.pickList)
		case "enter":
			s.picking = false
			plan, err := s.st.PlanModel(s.pickList[s.pickIdx])
			if err != nil {
				s.status = err.Error()
				return nil
			}
			s.modal = newConfirm(plan, s.st.Opts.DryRun)
		case "esc", "q":
			s.picking = false
		}
		return nil
	}
	if s.typing {
		switch msg.String() {
		case "esc":
			s.typing = false
			s.instr.Blur()
			return nil
		case "ctrl+s":
			s.propose(s.st.Instr[s.instrIdx].Path, s.instr.Value())
			return nil
		}
		var cmd tea.Cmd
		s.instr, cmd = s.instr.Update(msg)
		return cmd
	}

	switch msg.String() {
	case "tab":
		s.focus = (s.focus + 1) % 3
	case "shift+tab":
		s.focus = (s.focus + 2) % 3
	case "enter":
		switch s.focus {
		case fieldName:
			s.editing = true
			s.nameIn.SetValue(strings.TrimSpace(s.st.Config.AgentName))
			s.nameIn.CursorEnd()
			return s.nameIn.Focus()
		case fieldModel:
			s.picking = true
			s.pickList = append([]string{}, app.KnownModels...)
			cur := s.st.Settings.ModelDisplay()
			s.pickIdx = 0
			for i, m := range s.pickList {
				if m == cur {
					s.pickIdx = i
				}
			}
		case fieldInstr:
			if len(s.st.Instr) == 0 {
				s.status = "no CLAUDE.md found"
				return nil
			}
			s.typing = true
			return s.instr.Focus()
		}
	case "u":
		s.switchInstr("user")
	case "p":
		s.switchInstr("project")
	case "ctrl+s":
		if s.focus == fieldInstr && len(s.st.Instr) > 0 {
			s.propose(s.st.Instr[s.instrIdx].Path, s.instr.Value())
		}
	case "e":
		if s.focus == fieldInstr && len(s.st.Instr) > 0 {
			return s.externalEditor()
		}
	}
	return nil
}

func (s *sheet) switchInstr(label string) {
	for i, in := range s.st.Instr {
		if in.Label == label {
			if s.dirty() {
				s.status = "unsaved edits; ctrl+s to save or esc then u/p to discard"
				return
			}
			s.loadInstr(i)
			s.focus = fieldInstr
			return
		}
	}
	s.status = "no " + label + " CLAUDE.md"
}

func (s *sheet) propose(path, text string) {
	plan, err := s.st.PlanInstructions(path, text)
	if err != nil {
		s.status = err.Error()
		return
	}
	s.modal = newConfirm(plan, s.st.Opts.DryRun)
}

// externalEditor opens a scratch copy in $EDITOR. The real file is only
// written through the plan modal afterwards, so the diff/backup rule holds.
func (s *sheet) externalEditor() tea.Cmd {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		s.status = "set $EDITOR (or $VISUAL) to edit externally"
		return nil
	}
	in := s.st.Instr[s.instrIdx]
	tmp, err := os.CreateTemp("", "constelate-*-"+filepath.Base(in.Path))
	if err != nil {
		s.status = err.Error()
		return nil
	}
	tmp.WriteString(s.instr.Value())
	tmp.Close()
	parts := strings.Fields(editor)
	cmd := exec.Command(parts[0], append(parts[1:], tmp.Name())...)
	path := in.Path
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorDoneMsg{tmp: tmp.Name(), path: path, err: err}
	})
}

func (s *sheet) header(inner int) string {
	left := "─ " + s.sty.title.Render("CHARACTER SHEET") + " " + s.sty.tab.Render(s.agentName()) + " "
	return left + s.sty.frame.Render(strings.Repeat("─", max(0, inner-lipWidth(left))))
}

func (s *sheet) view(w, h int) string {
	cv := newCanvas(w, h)
	// row 0: NAME and MODEL
	y := 0
	x := 1
	cv.text(x, y, "NAME", s.focusStyle(fieldName))
	x += 6
	if s.editing {
		cv.text(x, y, s.nameIn.View(), &s.sty.panel)
	} else {
		cv.text(x, y, s.agentName(), &s.sty.panel)
	}
	x = w/2 + 1
	cv.text(x, y, "MODEL", s.focusStyle(fieldModel))
	x += 7
	model := s.st.Settings.ModelDisplay()
	if src := s.st.Settings.ModelSource; src != "" {
		model += "  (" + src + ")"
	}
	cv.text(x, y, clip(model, w-x-1), &s.sty.panel)

	// row 1: stats
	l := s.st.Ledger
	info := fmt.Sprintf("level %d · %d sessions · %d completed · ctx budget %s tok · theme %s",
		l.Level, l.Sessions, l.Done, kfmt(s.st.Config.CtxBudget), s.st.Theme.Name)
	cv.text(1, 1, clip(info, w-2), &s.sty.legend)

	// row 2: CLAUDE.md bar
	var tabs []string
	for i, in := range s.st.Instr {
		t := fmt.Sprintf("[%s] %s", string(in.Label[0]), in.Label)
		if in.Exists {
			t += fmt.Sprintf(" %d lines", in.Lines)
		} else {
			t += " (none)"
		}
		if i == s.instrIdx {
			t = "▸ " + t
		}
		tabs = append(tabs, t)
	}
	bar := " CLAUDE.md  " + strings.Join(tabs, "   ")
	if s.dirty() {
		bar += "   *unsaved*"
	}
	hint := " enter edit · ctrl+s save · e $EDITOR "
	if s.typing {
		hint = " typing · esc done · ctrl+s save "
	}
	cv.text(0, 2, clip(bar, w-lipWidth(hint)-1), s.focusStyle(fieldInstr))
	cv.text(w-lipWidth(hint), 2, hint, &s.sty.help)

	// rows 3..: textarea
	taH := h - 4
	s.instr.SetWidth(w - 2)
	s.instr.SetHeight(taH)
	if len(s.st.Instr) == 0 {
		cv.text(2, 4, "No CLAUDE.md in ~/.claude or this project.", &s.sty.help)
	} else {
		for i, line := range strings.Split(s.instr.View(), "\n") {
			if i >= taH {
				break
			}
			cv.raw(3+i, " "+line)
		}
	}

	// bottom: status
	if s.status != "" {
		cv.text(0, h-1, clip(" "+s.status, w), &s.sty.warn)
	}

	if s.picking {
		s.drawPicker(cv, w, h)
	}
	if s.modal != nil {
		s.modal.draw(cv, s.sty, w, h)
	}
	return cv.render()
}

func (s *sheet) focusStyle(f sheetField) *lipgloss.Style {
	if s.focus == f {
		return &s.sty.nodeSel
	}
	return &s.sty.title
}

func (s *sheet) drawPicker(cv *canvas, w, h int) {
	bw := 34
	bh := len(s.pickList) + 2
	x0, y0 := w/2-2, 1
	f := &s.sty.frame
	for y := y0; y < y0+bh; y++ {
		for x := x0; x < x0+bw; x++ {
			cv.put(x, y, ' ', nil)
		}
	}
	cv.text(x0, y0, "┌ model "+strings.Repeat("─", bw-9)+"┐", f)
	for i, m := range s.pickList {
		cv.put(x0, y0+1+i, '│', f)
		st := &s.sty.panel
		mark := "  "
		if i == s.pickIdx {
			st = &s.sty.nodeSel
			mark = "▸ "
		}
		cv.text(x0+1, y0+1+i, padPlain(mark+m, bw-2), st)
		cv.put(x0+bw-1, y0+1+i, '│', f)
	}
	cv.text(x0, y0+bh-1, "└"+strings.Repeat("─", bw-2)+"┘", f)
	foot := " j/k · enter · esc "
	cv.text(x0+bw-1-lipWidth(foot), y0+bh-1, foot, &s.sty.help)
}

func padPlain(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w])
	}
	return s + strings.Repeat(" ", w-len(r))
}
