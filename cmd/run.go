// Package cmd wires flags to the read-only model and, later, the TUI.
package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/signalandform/constelate/internal/claude"
	"github.com/signalandform/constelate/internal/config"
	"github.com/signalandform/constelate/internal/ctxcost"
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
	Skills   []claude.Skill
	Settings claude.Settings
	Instr    []claude.Instructions
	Sessions []claude.Session
	Ledger   xp.Ledger
	Unlocks  []xp.UnlockStatus
	Ctx      ctxcost.Report
	Problems []string // non-fatal load errors, shown once
}

func Run(args []string) error {
	fs := flag.NewFlagSet("constelate", flag.ContinueOnError)
	var o Options
	fs.StringVar(&o.Project, "project", "", "project directory (default: current directory)")
	fs.BoolVar(&o.DryRun, "dry-run", false, "show every write without making it")
	fs.BoolVar(&o.Summary, "summary", false, "print a read-only summary and exit")
	fs.BoolVar(&o.JSON, "json", false, "with --summary, print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if o.Project == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		o.Project = wd
	}
	o.Project, _ = filepath.Abs(o.Project)

	st, err := Load(o)
	if err != nil {
		return err
	}
	// Until the TUI lands (build step 2) the summary is the only mode.
	return PrintSummary(os.Stdout, st)
}

// Load reads everything once. Partial failures land in State.Problems.
func Load(o Options) (*State, error) {
	roots, err := claude.DefaultRoots(o.Project)
	if err != nil {
		return nil, err
	}
	st := &State{Opts: o, Roots: roots}

	cfgPath, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	st.Config, err = config.Load(cfgPath)
	if err != nil {
		st.Problems = append(st.Problems, fmt.Sprintf("config: %v (using defaults)", err))
	}

	st.Skills, err = claude.DiscoverSkills(roots)
	if err != nil {
		st.Problems = append(st.Problems, "skills: "+err.Error())
	}
	st.Settings, err = claude.LoadSettings(roots)
	if err != nil {
		st.Problems = append(st.Problems, "settings: "+err.Error())
	}
	st.Instr = claude.LoadInstructions(roots, o.Project)
	st.Sessions, err = claude.DiscoverSessions(claude.ProjectsDir(roots))
	if err != nil {
		st.Problems = append(st.Problems, "transcripts: "+err.Error())
	}
	st.Ledger = xp.Build(st.Sessions, st.Config)
	st.Unlocks = xp.Unlocks(st.Ledger, st.Config, st.Settings.Allow)
	st.Ctx = ctxcost.Estimate(st.Skills, st.Config.CtxBudget)
	return st, nil
}
