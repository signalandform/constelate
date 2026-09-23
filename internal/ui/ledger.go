package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/signalandform/constelate/internal/app"
)

// ledger is the XP history and trust-unlock screen.
type ledger struct {
	st     *app.State
	sty    *styles
	pane   int // 0 sessions, 1 unlocks
	selS   int // selected session row (0 = newest)
	selU   int
	scroll int
	modal  *confirm
	status string
	// homeFlat is $HOME with / replaced by -, the prefix Claude Code puts on
	// every project directory name under ~/.claude/projects.
	homeFlat string
}

func newLedger(st *app.State, sty *styles) *ledger {
	home, _ := os.UserHomeDir()
	return &ledger{st: st, sty: sty, homeFlat: strings.ReplaceAll(home, "/", "-")}
}

func (l *ledger) reset(st *app.State) {
	l.st = st
	l.selS = min(l.selS, max(0, len(st.Ledger.Entries)-1))
	l.selU = min(l.selU, max(0, len(st.Unlocks)-1))
}

func (l *ledger) update(msg tea.KeyMsg, apply applyFn) {
	l.status = ""
	if l.modal != nil {
		if l.modal.update(msg, apply) {
			l.modal = nil
		}
		return
	}
	switch msg.String() {
	case "tab", "shift+tab":
		l.pane = 1 - l.pane
	case "j", "down":
		if l.pane == 0 {
			l.selS = min(l.selS+1, max(0, len(l.st.Ledger.Entries)-1))
		} else {
			l.selU = min(l.selU+1, max(0, len(l.st.Unlocks)-1))
		}
	case "k", "up":
		if l.pane == 0 {
			l.selS = max(0, l.selS-1)
		} else {
			l.selU = max(0, l.selU-1)
		}
	case "enter":
		if l.pane == 1 {
			l.propose()
		}
	}
}

// propose opens the allow-rule plan for the selected unlock, if it is ready.
func (l *ledger) propose() {
	if l.selU >= len(l.st.Unlocks) {
		return
	}
	u := l.st.Unlocks[l.selU]
	switch {
	case u.Granted:
		l.status = u.Rule + " is already allowed"
		return
	case !u.Reached:
		l.status = fmt.Sprintf("%s: %d of %d successful %s calls so far", u.Name, u.Count, u.Threshold, u.Prefix)
		return
	}
	plan, err := l.st.PlanAllowRule(u.Rule)
	if err == nil {
		err = l.st.Guard().Check(plan)
	}
	if err != nil {
		l.status = err.Error()
		return
	}
	l.modal = newConfirm(plan, l.st.Opts.DryRun)
}

func (l *ledger) header(inner int) string {
	x := l.st.Ledger
	left := "─ " + l.sty.title.Render("LEDGER") + " " +
		l.sty.tab.Render(fmt.Sprintf("level %d · %d xp total", x.Level, x.Total)) + " "
	return left + l.sty.frame.Render(strings.Repeat("─", max(0, inner-lipWidth(left))))
}

func (l *ledger) view(w, h int) string {
	cv := newCanvas(w, h)
	x := l.st.Ledger
	cfg := l.st.Config.XP

	// weights line, so the numbers are never a mystery
	weights := fmt.Sprintf(" weights: session %d · tool ok %d · tool err %d · prompt ok %d (dormant: not in logs)",
		cfg.SessionCompleted, cfg.ToolCallOK, cfg.ToolCallErr, cfg.PromptApproved)
	cv.text(0, 0, clip(weights, w), &l.sty.legend)

	// sessions pane
	unlockH := len(l.st.Unlocks) + 2
	if unlockH > h/2 {
		unlockH = h / 2
	}
	sessH := h - unlockH - 2 // header row + blank
	title := " SESSIONS "
	st := &l.sty.title
	if l.pane == 0 {
		st = &l.sty.nodeSel
	}
	cv.text(0, 1, title, st)
	hdr := fmt.Sprintf("%-10s %-22s %5s %4s %s %6s", "date", "project", "tools", "err", "done", "xp")
	cv.text(11, 1, clip(hdr, w-12), &l.sty.legend)
	rows := sessH - 1
	entries := x.Entries
	n := len(entries)
	// newest first; keep the selection visible
	if l.selS < l.scroll {
		l.scroll = l.selS
	}
	if l.selS >= l.scroll+rows {
		l.scroll = l.selS - rows + 1
	}
	for i := 0; i < rows; i++ {
		idx := l.scroll + i
		if idx >= n {
			break
		}
		e := entries[n-1-idx]
		s := e.Session
		done := "   "
		if s.Completed {
			done = " ✓ "
		}
		date := "unknown"
		if !s.Started.IsZero() {
			date = s.Started.Format("2006-01-02")
		}
		line := fmt.Sprintf("%-10s %-22s %5d %4d %s %6d", date, padPlain(projectName(s.Project, l.homeFlat), 22),
			s.ToolResults, s.ToolErrors, done, e.XP)
		rs := &l.sty.panel
		if idx == l.selS && l.pane == 0 {
			rs = &l.sty.nodeSel
		}
		cv.text(11, 2+i, clip(line, w-12), rs)
	}
	if n == 0 {
		cv.text(11, 2, "no transcripts found under ~/.claude/projects", &l.sty.help)
	}

	// unlocks pane
	uy := 2 + rows
	title = " TRUST UNLOCKS "
	st = &l.sty.title
	if l.pane == 1 {
		st = &l.sty.nodeSel
	}
	cv.text(0, uy, title, st)
	cv.text(16, uy, clip("after N successful calls, suggest an allow rule; you confirm every time", w-17), &l.sty.legend)
	for i, u := range l.st.Unlocks {
		if i >= unlockH-1 {
			break
		}
		meter := bar(min(u.Count, u.Threshold), u.Threshold, 10)
		state := "locked"
		ss := &l.sty.nodeDim
		switch {
		case u.Granted:
			state, ss = "granted", &l.sty.ok
		case u.Reached:
			state, ss = "READY  enter to suggest", &l.sty.nodeTrust
		}
		line := fmt.Sprintf(" ◆ %-12s %s %3d/%-3d  %-24s %s", clip(u.Name, 12), meter, u.Count, u.Threshold, clip(u.Rule, 24), state)
		rs := ss
		if i == l.selU && l.pane == 1 {
			rs = &l.sty.nodeSel
		}
		cv.text(0, uy+1+i, clip(line, w), rs)
	}
	if len(l.st.Unlocks) == 0 {
		cv.text(1, uy+1, "no unlocks configured in ~/.config/constelate/config.toml", &l.sty.help)
	}

	if l.status != "" {
		cv.text(0, h-1, clip(" "+l.status, w), &l.sty.warn)
	}
	if l.modal != nil {
		l.modal.draw(cv, l.sty, w, h)
	}
	return cv.render()
}

// projectName strips the flattened $HOME prefix Claude Code puts on project
// dir names ("-Users-jack-repos-foo" -> "repos-foo") and keeps the tail,
// which is the specific part, when it must be shortened.
func projectName(dir, homeFlat string) string {
	name := strings.TrimPrefix(dir, homeFlat)
	name = strings.TrimPrefix(name, "-")
	if name == "" {
		return "~"
	}
	if r := []rune(name); len(r) > 22 {
		return "…" + string(r[len(r)-21:])
	}
	return name
}
