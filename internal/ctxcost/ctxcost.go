// Package ctxcost estimates the always-loaded context cost of installed skills
// and compares it with the user's budget. Everything here is an estimate
// (chars/4) and is labelled that way in the UI.
package ctxcost

import "github.com/signalandform/constelate/internal/claude"

type Report struct {
	Loaded  int // estimated tokens for every skill Claude Code will list
	Managed int // the subset Constelate can change (personal + project)
	Budget  int
	Over    bool
	ByScope map[claude.Scope]int
	Count   int
}

// Estimate sums the cost of every skill that is not disabled or hidden.
func Estimate(skills []claude.Skill, budget int) Report {
	r := Report{Budget: budget, ByScope: map[claude.Scope]int{}}
	for _, s := range skills {
		if s.Scope == claude.ScopeDisabled || s.Hidden {
			continue
		}
		c := s.CtxEstimate()
		r.Loaded += c
		r.ByScope[s.Scope] += c
		r.Count++
		if s.Scope == claude.ScopePersonal || s.Scope == claude.ScopeProject {
			r.Managed += c
		}
	}
	r.Over = budget > 0 && r.Loaded > budget
	return r
}
