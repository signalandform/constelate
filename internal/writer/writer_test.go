package writer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setup(t *testing.T) (root string, g Guard) {
	t.Helper()
	root = t.TempDir()
	return root, Guard{Allowed: []string{filepath.Join(root, "claude"), filepath.Join(root, "cfg")}}
}

func TestRefusesOutsideRoots(t *testing.T) {
	root, g := setup(t)
	p := Plan{Ops: []Op{{Kind: OpWrite, Path: filepath.Join(root, "elsewhere", "x"), New: []byte("x")}}}
	if err := g.Check(p); err == nil {
		t.Fatal("outside path accepted")
	}
}

func TestDryRunChangesNothing(t *testing.T) {
	root, g := setup(t)
	src := filepath.Join(root, "claude", "skills", "foo")
	os.MkdirAll(src, 0o755)
	os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("hi"), 0o644)
	dst := filepath.Join(root, "cfg", "disabled", "foo")
	p := Plan{Ops: []Op{{Kind: OpMoveDir, From: src, To: dst}}}
	res, err := g.Apply(p, true, filepath.Join(root, "cfg", "backups"))
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun || len(res.Done) != 1 || !strings.HasPrefix(res.Done[0], "would move") {
		t.Errorf("res = %+v", res)
	}
	if _, err := os.Stat(src); err != nil {
		t.Error("dry run moved the folder")
	}
	if _, err := os.Stat(filepath.Join(root, "cfg", "backups")); err == nil {
		t.Error("dry run wrote a backup")
	}
}

func TestMoveAndWriteWithBackup(t *testing.T) {
	root, g := setup(t)
	src := filepath.Join(root, "claude", "skills", "foo")
	os.MkdirAll(src, 0o755)
	os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("hi"), 0o644)
	dst := filepath.Join(root, "cfg", "disabled", "foo")
	settings := filepath.Join(root, "claude", "settings.json")
	os.WriteFile(settings, []byte("{\"a\":1}\n"), 0o644)

	p := Plan{Title: "t", Ops: []Op{
		{Kind: OpMoveDir, From: src, To: dst},
		{Kind: OpWrite, Path: settings, Old: []byte("{\"a\":1}\n"), New: []byte("{\"a\":2}\n")},
	}}
	res, err := g.Apply(p, false, filepath.Join(root, "cfg", "backups"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "SKILL.md")); err != nil {
		t.Error("folder not moved")
	}
	if _, err := os.Stat(src); err == nil {
		t.Error("source still present after move")
	}
	if b, _ := os.ReadFile(settings); string(b) != "{\"a\":2}\n" {
		t.Errorf("settings = %q", b)
	}
	if res.BackupDir == "" {
		t.Fatal("no backup dir")
	}
	if _, err := os.Stat(filepath.Join(res.BackupDir, "manifest.json")); err != nil {
		t.Error("no manifest")
	}
	entries, _ := os.ReadDir(res.BackupDir)
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "settings.json") {
			b, _ := os.ReadFile(filepath.Join(res.BackupDir, e.Name()))
			found = string(b) == "{\"a\":1}\n"
		}
	}
	if !found {
		t.Error("old settings not backed up")
	}
}

func TestRefusesOverwriteOnMove(t *testing.T) {
	root, g := setup(t)
	src := filepath.Join(root, "claude", "skills", "foo")
	dst := filepath.Join(root, "cfg", "disabled", "foo")
	os.MkdirAll(src, 0o755)
	os.MkdirAll(dst, 0o755)
	if err := g.Check(Plan{Ops: []Op{{Kind: OpMoveDir, From: src, To: dst}}}); err == nil {
		t.Fatal("overwrite accepted")
	}
}

func TestDiff(t *testing.T) {
	op := Op{Kind: OpWrite, Path: "/x", Old: []byte("a\nb\nc\n"), New: []byte("a\nB\nc\nd\n")}
	d := strings.Join(Diff(op), "\n")
	for _, want := range []string{"-b", "+B", "+d", " a", " c"} {
		if !strings.Contains(d, want) {
			t.Errorf("diff missing %q:\n%s", want, d)
		}
	}
	created := Op{Kind: OpWrite, Path: "/new", Old: nil, New: []byte("x\n")}
	if d := strings.Join(Diff(created), "\n"); !strings.Contains(d, "+x") || strings.Contains(d, "-") && strings.Contains(d, "\n-x") {
		t.Errorf("create diff: %s", d)
	}
}

func TestBackupDirsNeverCollide(t *testing.T) {
	root, g := setup(t)
	f := filepath.Join(root, "claude", "settings.json")
	os.WriteFile(f, []byte("1\n"), 0o644)
	var dirs []string
	for i := 0; i < 3; i++ {
		p := Plan{Ops: []Op{{Kind: OpWrite, Path: f, Old: []byte("1\n"), New: []byte("1\n")}}}
		res, err := g.Apply(p, false, filepath.Join(root, "cfg", "backups"))
		if err != nil {
			t.Fatal(err)
		}
		dirs = append(dirs, res.BackupDir)
	}
	if dirs[0] == dirs[1] || dirs[1] == dirs[2] {
		t.Errorf("backup dirs collided: %v", dirs)
	}
}
