// Package app loads everything Constelate knows into one State. Both the
// summary command and the TUI read from it; neither writes through it.
package app

import (
	"fmt"
	"os"

	"github.com/signalandform/constelate/internal/categories"
	"github.com/signalandform/constelate/internal/claude"
	"github.com/signalandform/constelate/internal/config"
	"github.com/signalandform/constelate/internal/ctxcost"
	"github.com/signalandform/constelate/internal/theme"
	"github.com/signalandform/constelate/internal/xp"
)

// Options are the parsed command-line flags.
type Options struct {
	Project string
	DryRun  bool
	Summary bool
	JSON    bool
}

// State is everything the app knows after one read of the disk.
type State struct {
	Opts     Options
	Roots    claude.Roots
	Config   config.Config
	Cats     categories.Store
	Theme    theme.Palette
	Skills   []claude.Skill
	Settings claude.Settings
	Instr    []claude.Instructions
	Sessions []claude.Session
	Ledger   xp.Ledger
	Unlocks  []xp.UnlockStatus
	Ctx      ctxcost.Report
	Problems []string // non-fatal load errors, shown once
}

// Load reads everything once. Partial failures land in State.Problems.
func Load(o Options) (*State, error) {
	roots, err := claude.DefaultRoots(o.Project)
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	st := &State{Opts: o, Roots: roots}

	if p, err := config.DefaultPath(); err == nil {
		if st.Config, err = config.Load(p); err != nil {
			st.Problems = append(st.Problems, fmt.Sprintf("config: %v (using defaults)", err))
		}
	}
	if p, err := categories.DefaultPath(); err == nil {
		if st.Cats, err = categories.Load(p); err != nil {
			st.Problems = append(st.Problems, fmt.Sprintf("categories: %v (using suggestions)", err))
		}
	}
	if st.Theme, err = theme.Load(home); err != nil {
		st.Problems = append(st.Problems, fmt.Sprintf("theme: %v (using ansi)", err))
	}
	if st.Skills, err = claude.DiscoverSkills(roots); err != nil {
		st.Problems = append(st.Problems, "skills: "+err.Error())
	}
	if st.Settings, err = claude.LoadSettings(roots); err != nil {
		st.Problems = append(st.Problems, "settings: "+err.Error())
	}
	st.Instr = claude.LoadInstructions(roots, o.Project)
	if st.Sessions, err = claude.DiscoverSessions(claude.ProjectsDir(roots)); err != nil {
		st.Problems = append(st.Problems, "transcripts: "+err.Error())
	}
	st.Ledger = xp.Build(st.Sessions, st.Config)
	st.Unlocks = xp.Unlocks(st.Ledger, st.Config, st.Settings.Allow)
	st.Ctx = ctxcost.Estimate(st.Skills, st.Config.CtxBudget)
	return st, nil
}
