package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SettingsFile is one of the four settings files Claude Code consults.
// Later files in the list override earlier ones for scalar keys; permission
// lists are unioned. Claude Code writes approvals to the *.local.json files.
type SettingsFile struct {
	Path   string
	Label  string // "user", "user.local", "project", "project.local"
	Exists bool
	Err    error
	Raw    map[string]any // decoded JSON, untouched
}

// Rule is one permissions.allow or permissions.deny pattern with its source.
type Rule struct {
	Pattern string
	Source  string // SettingsFile.Label
}

// Settings is the merged, read-only view.
type Settings struct {
	Files       []SettingsFile
	Model       string
	ModelSource string
	Allow       []Rule
	Deny        []Rule
	Ask         []Rule
}

// LoadSettings reads user and project settings. A missing file is not an error;
// a file that exists but does not parse is reported in its SettingsFile.Err and
// skipped for merging.
func LoadSettings(r Roots) (Settings, error) {
	var files []SettingsFile
	if r.Home != "" {
		files = append(files,
			SettingsFile{Path: filepath.Join(r.Home, "settings.json"), Label: "user"},
			SettingsFile{Path: filepath.Join(r.Home, "settings.local.json"), Label: "user.local"},
		)
	}
	if r.Project != "" {
		files = append(files,
			SettingsFile{Path: filepath.Join(r.Project, "settings.json"), Label: "project"},
			SettingsFile{Path: filepath.Join(r.Project, "settings.local.json"), Label: "project.local"},
		)
	}

	var s Settings
	var errs []error
	for i := range files {
		f := &files[i]
		b, err := os.ReadFile(f.Path)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				f.Err = err
				errs = append(errs, fmt.Errorf("%s: %w", f.Path, err))
			}
			continue
		}
		f.Exists = true
		if err := json.Unmarshal(b, &f.Raw); err != nil {
			f.Err = err
			errs = append(errs, fmt.Errorf("%s: %w", f.Path, err))
			continue
		}
		if m, ok := f.Raw["model"].(string); ok && m != "" {
			s.Model, s.ModelSource = m, f.Label
		}
		if p, ok := f.Raw["permissions"].(map[string]any); ok {
			s.Allow = append(s.Allow, rules(p["allow"], f.Label)...)
			s.Deny = append(s.Deny, rules(p["deny"], f.Label)...)
			s.Ask = append(s.Ask, rules(p["ask"], f.Label)...)
		}
	}
	s.Files = files
	return s, errors.Join(errs...)
}

func rules(v any, src string) []Rule {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]Rule, 0, len(arr))
	for _, x := range arr {
		if str, ok := x.(string); ok && strings.TrimSpace(str) != "" {
			out = append(out, Rule{Pattern: str, Source: src})
		}
	}
	return out
}

// ModelDisplay strips Claude Code's context-window suffix ("[1m]") for display.
func (s Settings) ModelDisplay() string {
	if s.Model == "" {
		return "(default)"
	}
	if i := strings.Index(s.Model, "["); i > 0 {
		return s.Model[:i]
	}
	return s.Model
}
