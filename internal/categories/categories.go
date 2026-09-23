// Package categories groups skills into constellations. Assignments live in a
// sidecar file (~/.config/constelate/categories.toml), never in SKILL.md, so
// Claude Code's own files stay untouched. A skill with no assignment gets a
// suggestion from keywords in its description; the user can override it.
package categories

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Built-in constellations, in display order. Users may add their own.
var Builtin = []string{"Coding", "Writing", "Research", "Ops", "Design"}

const Unsorted = "Unsorted"

// File is the on-disk shape of categories.toml.
type File struct {
	// Extra constellations the user defined, in order.
	Custom []string `toml:"custom"`
	// skill name -> category. Names collide across scopes rarely; when they do
	// the user can key by "scope/name".
	Assign map[string]string `toml:"assign"`
}

type Store struct {
	Path string
	File File
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "constelate", "categories.toml"), nil
}

// Load returns an empty store when the file is missing.
func Load(path string) (Store, error) {
	s := Store{Path: path, File: File{Assign: map[string]string{}}}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return s, err
	}
	if err := toml.Unmarshal(b, &s.File); err != nil {
		return s, err
	}
	if s.File.Assign == nil {
		s.File.Assign = map[string]string{}
	}
	return s, nil
}

// All returns every constellation name: built-ins, then custom, then Unsorted.
func (s Store) All() []string {
	out := append([]string{}, Builtin...)
	for _, c := range s.File.Custom {
		if !contains(out, c) {
			out = append(out, c)
		}
	}
	return append(out, Unsorted)
}

// Resolve returns the category for a skill: explicit assignment first (by
// "scope/name" then "name"), then a keyword suggestion, then Unsorted.
// suggested reports whether the answer came from the heuristic.
func (s Store) Resolve(scope, name, description string) (category string, suggested bool) {
	if c, ok := s.File.Assign[scope+"/"+name]; ok && c != "" {
		return c, false
	}
	if c, ok := s.File.Assign[name]; ok && c != "" {
		return c, false
	}
	if c := Suggest(name, description); c != "" {
		return c, true
	}
	return Unsorted, true
}

// keywords is deliberately small and plain. It only has to be right often
// enough that the user fixes fewer than they accept.
var keywords = []struct {
	cat   string
	words []string
}{
	{"Design", []string{"design", "ui", "ux", "css", "tailwind", "layout", "typography", "color", "palette", "animation", "motion", "brand", "logo", "banner", "icon", "visual", "shader", "3d", "three.js", "webgl", "accessib", "wcag"}},
	{"Ops", []string{"deploy", "vercel", "docker", "kubernetes", "ci", "cron", "schedule", "server", "infra", "devops", "monitor", "supabase", "database", "migration", "sql", "backup", "config", "settings", "permission", "shell", "terminal", "browser automation", "automate"}},
	{"Research", []string{"research", "search", "seo", "audit", "analy", "report", "dashboard", "metric", "kpi", "data", "chart", "visualiz", "extract", "scrape", "discover", "find"}},
	{"Writing", []string{"writ", "copy", "content", "blog", "email", "document", "docx", "pdf", "pptx", "slide", "presentation", "memo", "spec", "prd", "proposal", "agreement", "invoice"}},
	{"Coding", []string{"code", "coding", "react", "next.js", "nextjs", "typescript", "javascript", "python", "go ", "golang", "rust", "api", "mcp", "component", "refactor", "test", "debug", "bug", "review", "build", "app router", "framework", "library", "sdk"}},
}

// Suggest picks the category whose keywords appear most in the text. Ties go
// to the earlier entry in the table. Returns "" when nothing matches.
func Suggest(name, description string) string {
	text := strings.ToLower(name + " " + description)
	best, bestN := "", 0
	for _, k := range keywords {
		n := 0
		for _, w := range k.words {
			n += strings.Count(text, w)
		}
		if n > bestN {
			best, bestN = k.cat, n
		}
	}
	return best
}

// Set records an override in memory. Saving is the writer package's job so
// the diff/backup/confirm rules apply uniformly.
func (s *Store) Set(scope, name, category string) {
	s.File.Assign[scope+"/"+name] = category
	if !contains(Builtin, category) && category != Unsorted && !contains(s.File.Custom, category) {
		s.File.Custom = append(s.File.Custom, category)
		sort.Strings(s.File.Custom)
	}
}

// Marshal renders the file for writing.
func (s Store) Marshal() ([]byte, error) { return toml.Marshal(s.File) }

func contains(list []string, x string) bool {
	for _, v := range list {
		if v == x {
			return true
		}
	}
	return false
}
