package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSessionFixture(t *testing.T) {
	p := filepath.Join("..", "..", "testdata", "transcripts", "session-a.jsonl")
	s, err := ParseSession(p)
	if err != nil {
		t.Fatal(err)
	}
	if s.ToolCalls == 0 || s.ToolResults == 0 {
		t.Fatalf("no tool activity parsed: %+v", s)
	}
	if s.ToolErrors != 1 {
		t.Errorf("ToolErrors = %d, want 1", s.ToolErrors)
	}
	if !s.Completed {
		t.Error("cost-state line not recognised as completion")
	}
	if s.BadLines != 1 {
		t.Errorf("BadLines = %d, want 1 (the deliberate non-JSON line)", s.BadLines)
	}
	if s.Unknown["unknown-future-type"] != 1 {
		t.Errorf("unknown type not counted: %v", s.Unknown)
	}
	if s.Started.IsZero() || s.Ended.Before(s.Started) {
		t.Errorf("bad time range %v..%v", s.Started, s.Ended)
	}
	if s.Version == "" {
		t.Error("version not captured")
	}
}

func TestParseSessionEmptyAndGarbage(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.jsonl")
	os.WriteFile(empty, nil, 0o644)
	if _, err := ParseSession(empty); err != nil {
		t.Fatal(err)
	}
	garbage := filepath.Join(dir, "g.jsonl")
	os.WriteFile(garbage, []byte("not json\n{\"type\":\"user\",\"message\":{\"content\":\"hi\"}}\n"), 0o644)
	s, err := ParseSession(garbage)
	if err != nil || s.BadLines != 1 || s.UserTurns != 1 {
		t.Fatalf("got %+v err %v", s, err)
	}
}

func TestFirstWord(t *testing.T) {
	cases := map[string]string{
		"git status":          "git",
		"cd ~/x && git log":   "git",
		"FOO=1 npm run build": "npm",
		"  gh pr view 3  ":    "gh",
		"":                    "",
	}
	for in, want := range cases {
		if got := firstWord(in); got != want {
			t.Errorf("firstWord(%q) = %q, want %q", in, got, want)
		}
	}
}
