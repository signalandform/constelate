// Package xp turns session outcomes into experience points, levels and trust
// unlock progress. Weights come from config so the user can retune them.
package xp

import (
	"sort"

	"github.com/signalandform/constelate/internal/claude"
	"github.com/signalandform/constelate/internal/config"
)

// Entry is one session's contribution, for the ledger.
type Entry struct {
	Session claude.Session
	XP      int
}

type Ledger struct {
	Entries  []Entry
	Total    int
	Level    int
	IntoLvl  int // XP earned within the current level
	LevelXP  int // XP needed per level
	Sessions int
	Done     int
	ToolOK   int
	ToolErr  int
	Rejected int
	Prefixes map[string]int // successful Bash command prefixes across sessions
}

func Build(sessions []claude.Session, cfg config.Config) Ledger {
	l := Ledger{LevelXP: cfg.XP.LevelXP, Prefixes: map[string]int{}}
	for _, s := range sessions {
		ok := s.ToolResults - s.ToolErrors
		if ok < 0 {
			ok = 0
		}
		x := ok*cfg.XP.ToolCallOK + s.ToolErrors*cfg.XP.ToolCallErr + s.Rejections*cfg.XP.PromptRejected
		if s.Completed {
			x += cfg.XP.SessionCompleted
			l.Done++
		}
		l.Entries = append(l.Entries, Entry{Session: s, XP: x})
		l.Total += x
		l.Sessions++
		l.ToolOK += ok
		l.ToolErr += s.ToolErrors
		l.Rejected += s.Rejections
		for p, n := range s.CmdPrefixes {
			l.Prefixes[p] += n
		}
	}
	sort.SliceStable(l.Entries, func(i, j int) bool {
		return l.Entries[i].Session.Started.Before(l.Entries[j].Session.Started)
	})
	l.Level = l.Total/l.LevelXP + 1
	l.IntoLvl = l.Total % l.LevelXP
	return l
}

// UnlockStatus is one trust threshold with the user's progress toward it.
type UnlockStatus struct {
	config.Unlock
	Count   int
	Reached bool
	Granted bool // an allow rule with this exact pattern already exists
}

func Unlocks(l Ledger, cfg config.Config, allow []claude.Rule) []UnlockStatus {
	have := map[string]bool{}
	for _, r := range allow {
		have[r.Pattern] = true
	}
	var out []UnlockStatus
	for _, u := range cfg.Trust.Unlocks {
		n := l.Prefixes[u.Prefix]
		out = append(out, UnlockStatus{
			Unlock:  u,
			Count:   n,
			Reached: n >= u.Threshold,
			Granted: have[u.Rule],
		})
	}
	return out
}
