package claude

import (
	"path/filepath"
	"testing"
)

func TestParseSkillFile(t *testing.T) {
	cases := []struct {
		dir      string
		wantName string
		wantWarn bool
		invoc    bool
	}{
		{"3d-web-experience", "3d-web-experience", false, false}, // folded multi-line description + extra keys
		{"accessibility", "accessibility", false, false},         // nested metadata map
		{"data-visualization", "data-visualization", false, false},
		{"no-frontmatter", "no-frontmatter", true, false},
	}
	for _, c := range cases {
		p := filepath.Join("..", "..", "testdata", "skills", c.dir, "SKILL.md")
		s := ParseSkillFile(p, ScopePersonal)
		if s.Name != c.wantName {
			t.Errorf("%s: name = %q", c.dir, s.Name)
		}
		if (len(s.Warnings) > 0) != c.wantWarn {
			t.Errorf("%s: warnings = %v", c.dir, s.Warnings)
		}
		if !c.wantWarn && s.Description == "" {
			t.Errorf("%s: empty description", c.dir)
		}
		if !c.wantWarn && s.CtxEstimate() == 0 {
			t.Errorf("%s: zero ctx estimate", c.dir)
		}
	}
}

func TestFoldedDescriptionIsOneLine(t *testing.T) {
	p := filepath.Join("..", "..", "testdata", "skills", "3d-web-experience", "SKILL.md")
	s := ParseSkillFile(p, ScopePersonal)
	for _, r := range s.Description {
		if r == '\n' {
			t.Fatalf("folded description kept a newline: %q", s.Description)
		}
	}
	if s.Extra["risk"] != "critical" {
		t.Errorf("extra keys not kept: %v", s.Extra)
	}
}

func TestDiscoverSkillsMissingRootsAreFine(t *testing.T) {
	skills, err := DiscoverSkills(Roots{Home: "/nonexistent", Project: "/also/none", Disabled: ""})
	if err != nil || len(skills) != 0 {
		t.Fatalf("got %d skills, err %v", len(skills), err)
	}
}
