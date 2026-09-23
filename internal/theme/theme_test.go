package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseOmarchyFixture(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "omarchy", "theme", "colors.toml"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseColors(b)
	if err != nil {
		t.Fatal(err)
	}
	if p.Accent != "#7aa2f7" || p.Bg != "#1a1b26" || !p.Dark {
		t.Errorf("unexpected palette %+v", p)
	}
}

func TestPartialAndGarbage(t *testing.T) {
	p, err := ParseColors([]byte("accent = \"#ff0000\"\n"))
	if err != nil || p.Accent != "#ff0000" || p.Fg != ANSI().Fg {
		t.Errorf("partial: %+v %v", p, err)
	}
	if _, err := ParseColors([]byte("mode = 1 = 2")); err == nil {
		t.Error("garbage toml accepted")
	}
	if _, err := ParseColors([]byte("unrelated = true\n")); err == nil {
		t.Error("empty palette accepted")
	}
}

func TestLoadFallsBackToANSI(t *testing.T) {
	p, err := Load(t.TempDir())
	if err != nil || p.Name != "ansi" {
		t.Errorf("got %+v %v", p, err)
	}
}

func TestLoadFindsStateDir(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".local", "state", "omarchy", "current", "theme")
	os.MkdirAll(dir, 0o755)
	src, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "omarchy", "theme", "colors.toml"))
	os.WriteFile(filepath.Join(dir, "colors.toml"), src, 0o644)
	os.WriteFile(filepath.Join(home, ".local", "state", "omarchy", "current", "theme.name"), []byte("tokyo-night\n"), 0o644)
	p, err := Load(home)
	if err != nil || p.Name != "tokyo-night" || p.Accent != "#7aa2f7" {
		t.Errorf("got %+v %v", p, err)
	}
}
