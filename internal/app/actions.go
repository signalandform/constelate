package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/signalandform/constelate/internal/claude"
	"github.com/signalandform/constelate/internal/writer"
)

// originFile is dropped inside a parked skill folder so "install" knows where
// it came from. It lives with the folder, so it survives a manual move too.
const originFile = ".constelate-origin"

// Guard returns the writer guard for this state: ~/.claude, the project's
// .claude, and Constelate's own config dir.
func (st *State) Guard() writer.Guard {
	g := writer.Guard{}
	if st.Roots.Home != "" {
		g.Allowed = append(g.Allowed, st.Roots.Home)
	}
	if st.Roots.Project != "" {
		g.Allowed = append(g.Allowed, st.Roots.Project)
	}
	if home, err := os.UserHomeDir(); err == nil {
		g.Allowed = append(g.Allowed, filepath.Join(home, ".config", "constelate"))
	}
	return g
}

// BackupRoot is ~/.config/constelate/backups.
func (st *State) BackupRoot() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "constelate", "backups")
}

// PlanDisable moves a personal or project skill folder into disabled/.
func (st *State) PlanDisable(s claude.Skill) (writer.Plan, error) {
	if !s.Scope.Writable() || s.Scope == claude.ScopeDisabled {
		return writer.Plan{}, fmt.Errorf("%s is %s and cannot be disabled here", s.Name, s.Scope)
	}
	dst := filepath.Join(st.Roots.Disabled, filepath.Base(s.Dir))
	if _, err := os.Stat(dst); err == nil {
		dst += "-" + time.Now().Format("20060102-150405") // never overwrite a parked copy
	}
	origin := filepath.Join(s.Dir, originFile)
	old, _ := writer.ReadCurrent(origin)
	return writer.Plan{
		Title: "Disable " + s.Name,
		Ops: []writer.Op{
			{Kind: writer.OpWrite, Path: origin, Old: old, New: []byte(s.Dir + "\n"),
				Why: "remember where it came from"},
			{Kind: writer.OpMoveDir, From: s.Dir, To: dst,
				Why: "park the folder; nothing is deleted"},
		},
	}, nil
}

// PlanInstall restores a parked skill to its origin, or to the personal
// skills dir when the origin is unknown or occupied.
func (st *State) PlanInstall(s claude.Skill) (writer.Plan, error) {
	if s.Scope != claude.ScopeDisabled {
		return writer.Plan{}, fmt.Errorf("%s is already %s", s.Name, s.Scope)
	}
	dst := ""
	if b, err := os.ReadFile(filepath.Join(s.Dir, originFile)); err == nil {
		dst = filepath.Clean(string(trimNL(b)))
	}
	if dst == "" || !st.Guard().Allows(dst) {
		dst = filepath.Join(st.Roots.Home, "skills", filepath.Base(s.Dir))
	}
	if _, err := os.Stat(dst); err == nil {
		return writer.Plan{}, fmt.Errorf("a skill folder already exists at %s; remove or rename it first", dst)
	}
	return writer.Plan{
		Title: "Install " + s.Name,
		Ops: []writer.Op{
			{Kind: writer.OpMoveDir, From: s.Dir, To: dst, Why: "restore the folder"},
		},
	}, nil
}

// PlanCategory records a category override in categories.toml.
func (st *State) PlanCategory(s claude.Skill, category string) (writer.Plan, error) {
	cats := st.Cats
	cats.File.Assign = cloneMap(cats.File.Assign)
	cats.Set(string(s.Scope), s.Name, category)
	nb, err := cats.Marshal()
	if err != nil {
		return writer.Plan{}, err
	}
	old, err := writer.ReadCurrent(cats.Path)
	if err != nil {
		return writer.Plan{}, err
	}
	return writer.Plan{
		Title: "Move " + s.Name + " to " + category,
		Ops:   []writer.Op{{Kind: writer.OpWrite, Path: cats.Path, Old: old, New: nb, Why: "categories are a Constelate sidecar, not SKILL.md"}},
	}, nil
}

// Apply runs a plan under the guard, honouring --dry-run, then reloads state.
func (st *State) Apply(p writer.Plan) (writer.Result, *State, error) {
	res, err := st.Guard().Apply(p, st.Opts.DryRun, st.BackupRoot())
	if err != nil {
		return res, st, err
	}
	if res.DryRun {
		return res, st, nil
	}
	fresh, err := Load(st.Opts)
	if err != nil {
		return res, st, fmt.Errorf("applied, but reload failed: %w", err)
	}
	return res, fresh, nil
}

func trimNL(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}

func cloneMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
