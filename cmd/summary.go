package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/signalandform/constelate/internal/claude"
)

// PrintSummary is build step 1: prove the parsers against a real install.
func PrintSummary(w io.Writer, st *State) error {
	if st.Opts.JSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(summaryJSON(st))
	}

	fmt.Fprintf(w, "CONST.ELATE  read-only summary\n")
	fmt.Fprintf(w, "project: %s\n\n", st.Opts.Project)

	// Skills by scope.
	byScope := map[claude.Scope][]claude.Skill{}
	for _, s := range st.Skills {
		byScope[s.Scope] = append(byScope[s.Scope], s)
	}
	fmt.Fprintf(w, "SKILLS  %d found\n", len(st.Skills))
	for _, sc := range []claude.Scope{claude.ScopePersonal, claude.ScopeProject, claude.ScopeSynced, claude.ScopePlugin, claude.ScopeDisabled} {
		list := byScope[sc]
		if len(list) == 0 {
			continue
		}
		rw := "read-only"
		if sc.Writable() {
			rw = "writable"
		}
		fmt.Fprintf(w, "  %-9s %3d  %s  ~%d tok\n", sc, len(list), rw, st.Ctx.ByScope[sc])
	}
	warn := 0
	for _, s := range st.Skills {
		if len(s.Warnings) > 0 {
			warn++
		}
	}
	if warn > 0 {
		fmt.Fprintf(w, "  %d skill(s) with parse warnings:\n", warn)
		for _, s := range st.Skills {
			for _, wmsg := range s.Warnings {
				fmt.Fprintf(w, "    %s (%s): %s\n", s.Name, s.Scope, wmsg)
			}
		}
	}

	// Context.
	over := ""
	if st.Ctx.Over {
		over = "  OVER BUDGET"
	}
	fmt.Fprintf(w, "\nCONTEXT (estimate, chars/4)\n")
	fmt.Fprintf(w, "  always-loaded skill text  ~%d tok of %d budget%s\n", st.Ctx.Loaded, st.Ctx.Budget, over)
	fmt.Fprintf(w, "  of which you can change   ~%d tok (personal + project)\n", st.Ctx.Managed)
	for _, in := range st.Instr {
		if in.Exists {
			fmt.Fprintf(w, "  CLAUDE.md (%s)  %d lines  ~%d tok  %q\n", in.Label, in.Lines, in.CtxEstimate(), trunc(in.Headline, 50))
		}
	}

	// Character sheet.
	fmt.Fprintf(w, "\nCHARACTER SHEET\n")
	fmt.Fprintf(w, "  model  %s", st.Settings.ModelDisplay())
	if st.Settings.ModelSource != "" {
		fmt.Fprintf(w, "  (from %s)", st.Settings.ModelSource)
	}
	fmt.Fprintln(w)
	for _, f := range st.Settings.Files {
		mark := "absent"
		if f.Exists {
			mark = "ok"
		}
		if f.Err != nil {
			mark = "ERROR " + f.Err.Error()
		}
		fmt.Fprintf(w, "  %-14s %-7s %s\n", f.Label, mark, f.Path)
	}

	// Trust.
	fmt.Fprintf(w, "\nTRUST  %d allow  %d deny  %d ask\n", len(st.Settings.Allow), len(st.Settings.Deny), len(st.Settings.Ask))
	for _, u := range st.Unlocks {
		state := fmt.Sprintf("%d/%d", u.Count, u.Threshold)
		switch {
		case u.Granted:
			state += "  granted"
		case u.Reached:
			state += "  READY to suggest " + u.Rule
		}
		fmt.Fprintf(w, "  %-12s %s\n", u.Name, state)
	}

	// XP.
	l := st.Ledger
	fmt.Fprintf(w, "\nXP  level %d  %d/%d  (total %d)\n", l.Level, l.IntoLvl, l.LevelXP, l.Total)
	fmt.Fprintf(w, "  sessions %d  completed %d  tool ok %d  tool err %d  rejected %d\n",
		l.Sessions, l.Done, l.ToolOK, l.ToolErr, l.Rejected)
	unknown := map[string]int{}
	bad := 0
	versions := map[string]bool{}
	for _, s := range st.Sessions {
		bad += s.BadLines
		if s.Version != "" {
			versions[s.Version] = true
		}
		for k, v := range s.Unknown {
			unknown[k] += v
		}
	}
	vs := make([]string, 0, len(versions))
	for v := range versions {
		vs = append(vs, v)
	}
	sort.Strings(vs)
	fmt.Fprintf(w, "  log versions %s  bad lines %d", strings.Join(vs, ", "), bad)
	if len(unknown) > 0 {
		ks := make([]string, 0, len(unknown))
		for k := range unknown {
			ks = append(ks, fmt.Sprintf("%s×%d", k, unknown[k]))
		}
		sort.Strings(ks)
		fmt.Fprintf(w, "  unmodelled types %s", strings.Join(ks, " "))
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  note: approvals are not in the logs; prompt XP is 0 until they are\n")

	if len(st.Problems) > 0 {
		fmt.Fprintf(w, "\nPROBLEMS\n")
		for _, p := range st.Problems {
			fmt.Fprintf(w, "  %s\n", p)
		}
	}
	return nil
}

func summaryJSON(st *State) map[string]any {
	return map[string]any{
		"project":  st.Opts.Project,
		"skills":   st.Skills,
		"settings": map[string]any{"model": st.Settings.Model, "allow": st.Settings.Allow, "deny": st.Settings.Deny},
		"claudemd": st.Instr,
		"context":  st.Ctx,
		"xp":       map[string]any{"total": st.Ledger.Total, "level": st.Ledger.Level, "sessions": st.Ledger.Sessions, "completed": st.Ledger.Done},
		"unlocks":  st.Unlocks,
		"problems": st.Problems,
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
