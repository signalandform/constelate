package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/signalandform/constelate/internal/claude"
	"github.com/signalandform/constelate/internal/config"
)

func TestPlanModelEditsOneLine(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, "settings.json")
	os.WriteFile(p, []byte("{\n  \"theme\": \"auto\",\n  \"model\": \"claude-opus-5[1m]\",\n  \"hooks\": {}\n}\n"), 0o644)
	st := &State{Roots: claude.Roots{Home: home}, Settings: claude.Settings{Model: "claude-opus-5[1m]"}}
	plan, err := st.PlanModel("claude-sonnet-5")
	if err != nil {
		t.Fatal(err)
	}
	got := string(plan.Ops[0].New)
	if !strings.Contains(got, `"model": "claude-sonnet-5[1m]"`) || !strings.Contains(got, `"theme": "auto"`) {
		t.Errorf("bad edit:\n%s", got)
	}
	if strings.Count(got, "\n") != 5 {
		t.Errorf("formatting changed:\n%s", got)
	}
}

func TestPlanModelInsertsWhenMissing(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, "settings.json")
	os.WriteFile(p, []byte("{\n  \"theme\": \"auto\"\n}\n"), 0o644)
	st := &State{Roots: claude.Roots{Home: home}}
	plan, err := st.PlanModel("haiku")
	if err != nil {
		t.Fatal(err)
	}
	got := string(plan.Ops[0].New)
	if !strings.HasPrefix(got, "{\n  \"model\": \"haiku\",\n") || !strings.Contains(got, `"theme"`) {
		t.Errorf("bad insert:\n%s", got)
	}
	st2 := &State{Roots: claude.Roots{Home: t.TempDir()}}
	plan, _ = st2.PlanModel("opus")
	if plan.Ops[0].Old != nil || !strings.Contains(string(plan.Ops[0].New), `"model": "opus"`) {
		t.Errorf("create case: %+v", plan.Ops[0])
	}
}

func TestPlanAgentName(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.toml")
	os.WriteFile(cfg, []byte("ctx_budget = 4000\n\n[xp]\nlevel_xp = 1000\n"), 0o644)
	st := &State{Config: config.Config{Path: cfg}}
	plan, err := st.PlanAgentName("Rowan")
	if err != nil {
		t.Fatal(err)
	}
	got := string(plan.Ops[0].New)
	if !strings.HasPrefix(got, "agent_name = \"Rowan\"\nctx_budget") {
		t.Errorf("key not at top:\n%s", got)
	}
	os.WriteFile(cfg, []byte(got), 0o644)
	plan, _ = st.PlanAgentName("Ash")
	if got := string(plan.Ops[0].New); strings.Count(got, "agent_name") != 1 || !strings.Contains(got, `"Ash"`) {
		t.Errorf("rename duplicated:\n%s", got)
	}
}

func TestPlanInstructionsNoop(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "CLAUDE.md")
	os.WriteFile(p, []byte("hi\n"), 0o644)
	st := &State{}
	if _, err := st.PlanInstructions(p, "hi"); err == nil {
		t.Error("no-op save accepted")
	}
	plan, err := st.PlanInstructions(p, "hello")
	if err != nil || string(plan.Ops[0].New) != "hello\n" {
		t.Errorf("got %+v %v", plan, err)
	}
}
