// Package claude is a read-only model of a Claude Code installation:
// skills, settings, CLAUDE.md and session transcripts. Nothing in this
// package writes to disk.
package claude

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Scope says where a skill was found and therefore who owns it.
type Scope string

const (
	ScopePersonal Scope = "personal" // ~/.claude/skills/<name>
	ScopeProject  Scope = "project"  // <project>/.claude/skills/<name>
	ScopeSynced   Scope = "synced"   // ~/.claude/skills/synced/<id>/<name>, managed by claude.ai
	ScopePlugin   Scope = "plugin"   // ~/.claude/plugins/marketplaces/.../skills/<name>
	ScopeDisabled Scope = "disabled" // ~/.config/constelate/disabled/<name>, parked by Constelate
)

// Writable reports whether Constelate may move or edit skills in this scope.
// Synced and plugin skills are owned by Claude Code and are shown read-only.
func (s Scope) Writable() bool {
	return s == ScopePersonal || s == ScopeProject || s == ScopeDisabled
}

// Skill is one SKILL.md on disk plus what we could parse from it.
type Skill struct {
	Name        string
	Description string
	Scope       Scope
	Dir         string // directory containing SKILL.md
	Path        string // full path to SKILL.md
	Invocable   bool   // user-invocable: true in frontmatter
	Hidden      bool
	Extra       map[string]any // every other frontmatter key, kept verbatim
	Warnings    []string       // parse problems; the skill is still listed
}

// CtxEstimate is the rough always-loaded cost of this skill in tokens.
// Claude Code puts each skill's name and description in the system prompt,
// so that text is what every session pays for. chars/4 is a coarse estimate
// and is labelled as such wherever it is shown.
func (s Skill) CtxEstimate() int {
	n := len(s.Name) + len(s.Description)
	return (n + 3) / 4
}

// Roots names the directories a discovery run looks in. Empty strings are skipped.
type Roots struct {
	Home     string // ~/.claude
	Project  string // <project>/.claude
	Disabled string // ~/.config/constelate/disabled
}

// DefaultRoots resolves the standard locations for the given project directory.
func DefaultRoots(project string) (Roots, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Roots{}, err
	}
	r := Roots{
		Home:     filepath.Join(home, ".claude"),
		Disabled: filepath.Join(home, ".config", "constelate", "disabled"),
	}
	if project != "" {
		p := filepath.Join(project, ".claude")
		// Running from $HOME would make the project scope a mirror of the
		// personal one and double every count.
		if p != r.Home {
			r.Project = p
		}
	}
	return r, nil
}

// DiscoverSkills finds every SKILL.md under the roots. Missing directories are
// not errors. A SKILL.md that fails to parse is still returned, with Warnings.
func DiscoverSkills(r Roots) ([]Skill, error) {
	var out []Skill
	var errs []error

	add := func(scope Scope, pattern string) {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", pattern, err))
			return
		}
		for _, p := range matches {
			// The synced tree lives under personal; do not list it twice.
			if scope == ScopePersonal && strings.Contains(p, string(filepath.Separator)+"synced"+string(filepath.Separator)) {
				continue
			}
			out = append(out, ParseSkillFile(p, scope))
		}
	}

	if r.Home != "" {
		add(ScopePersonal, filepath.Join(r.Home, "skills", "*", "SKILL.md"))
		add(ScopeSynced, filepath.Join(r.Home, "skills", "synced", "*", "*", "SKILL.md"))
		add(ScopePlugin, filepath.Join(r.Home, "plugins", "marketplaces", "*", "plugins", "*", "skills", "*", "SKILL.md"))
	}
	if r.Project != "" {
		add(ScopeProject, filepath.Join(r.Project, "skills", "*", "SKILL.md"))
	}
	if r.Disabled != "" {
		add(ScopeDisabled, filepath.Join(r.Disabled, "*", "SKILL.md"))
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Scope != out[j].Scope {
			return out[i].Scope < out[j].Scope
		}
		return out[i].Name < out[j].Name
	})
	return out, errors.Join(errs...)
}

// ParseSkillFile reads one SKILL.md. It never returns an error: problems go in
// Warnings so a single odd file cannot hide the rest of the constellation.
func ParseSkillFile(path string, scope Scope) Skill {
	s := Skill{
		Scope: scope,
		Dir:   filepath.Dir(path),
		Path:  path,
		Extra: map[string]any{},
	}
	s.Name = filepath.Base(s.Dir) // fallback; frontmatter name wins

	raw, err := os.ReadFile(path)
	if err != nil {
		s.Warnings = append(s.Warnings, "read: "+err.Error())
		return s
	}
	fm, ok := splitFrontmatter(raw)
	if !ok {
		s.Warnings = append(s.Warnings, "no YAML frontmatter; using folder name")
		return s
	}
	var m map[string]any
	if err := yaml.Unmarshal(fm, &m); err != nil {
		s.Warnings = append(s.Warnings, "frontmatter: "+err.Error())
		return s
	}
	for k, v := range m {
		switch k {
		case "name":
			if str, ok := v.(string); ok && strings.TrimSpace(str) != "" {
				s.Name = strings.TrimSpace(str)
			} else {
				s.Warnings = append(s.Warnings, "name is not a string; using folder name")
			}
		case "description":
			if str, ok := v.(string); ok {
				s.Description = strings.TrimSpace(str)
			} else {
				s.Warnings = append(s.Warnings, "description is not a string")
			}
		case "user-invocable":
			s.Invocable = asBool(v)
		case "hidden":
			s.Hidden = asBool(v)
		default:
			s.Extra[k] = v
		}
	}
	if s.Description == "" {
		s.Warnings = append(s.Warnings, "no description")
	}
	return s
}

// splitFrontmatter returns the YAML between the first two "---" lines.
func splitFrontmatter(b []byte) ([]byte, bool) {
	b = bytes.TrimPrefix(b, []byte("\xEF\xBB\xBF")) // UTF-8 BOM
	if !bytes.HasPrefix(b, []byte("---")) {
		return nil, false
	}
	rest := b[3:]
	// tolerate "---\r\n"
	nl := bytes.IndexByte(rest, '\n')
	if nl < 0 {
		return nil, false
	}
	rest = rest[nl+1:]
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		if bytes.HasPrefix(rest, []byte("---")) {
			return []byte{}, true
		}
		return nil, false
	}
	return rest[:end], true
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(strings.TrimSpace(t), "true")
	}
	return false
}
