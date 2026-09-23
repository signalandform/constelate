package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/signalandform/constelate/internal/claude"
)

func valid(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, s)
	}
	return m
}

func allowList(m map[string]any) []any {
	return m["permissions"].(map[string]any)["allow"].([]any)
}

func TestInsertAllowIntoExistingList(t *testing.T) {
	src := "{\n  \"permissions\": {\n    \"allow\": [\n      \"Bash(gh pr *)\",\n      \"Bash(npm run *)\"\n    ]\n  }\n}\n"
	out, err := insertAllow(src, "Bash(git status *)")
	if err != nil {
		t.Fatal(err)
	}
	m := valid(t, out)
	if l := allowList(m); len(l) != 3 || l[0] != "Bash(git status *)" {
		t.Errorf("list = %v", l)
	}
	if strings.Count(out, "\n") != strings.Count(src, "\n")+1 {
		t.Errorf("formatting changed:\n%s", out)
	}
}

func TestInsertAllowEmptyListAndMissingBlocks(t *testing.T) {
	cases := []string{
		"{\n  \"permissions\": {\n    \"allow\": []\n  }\n}\n",
		"{\n  \"permissions\": {\n    \"deny\": [\"x\"]\n  }\n}\n",
		"{\n  \"theme\": \"auto\"\n}\n",
		"{}\n",
		"",
	}
	for _, src := range cases {
		out, err := insertAllow(src, "Bash(git status *)")
		if err != nil {
			t.Fatalf("%q: %v", src, err)
		}
		m := valid(t, out)
		if l := allowList(m); len(l) != 1 || l[0] != "Bash(git status *)" {
			t.Errorf("%q -> %v", src, l)
		}
		if strings.Contains(src, "theme") && m["theme"] != "auto" {
			t.Errorf("lost sibling key:\n%s", out)
		}
		if strings.Contains(src, "deny") && m["permissions"].(map[string]any)["deny"] == nil {
			t.Errorf("lost deny:\n%s", out)
		}
	}
}

func TestPlanAllowRuleRefusesDuplicate(t *testing.T) {
	home := t.TempDir()
	os.WriteFile(filepath.Join(home, "settings.local.json"), []byte("{\"permissions\":{\"allow\":[\"Bash(git status *)\"]}}"), 0o644)
	st := &State{Roots: claude.Roots{Home: home}, Settings: claude.Settings{Allow: []claude.Rule{{Pattern: "Bash(git status *)", Source: "user.local"}}}}
	if _, err := st.PlanAllowRule("Bash(git status *)"); err == nil {
		t.Error("duplicate accepted")
	}
	plan, err := st.PlanAllowRule("Bash(gh pr *)")
	if err != nil {
		t.Fatal(err)
	}
	valid(t, string(plan.Ops[0].New))
}
