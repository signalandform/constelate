package app

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/signalandform/constelate/internal/writer"
)

// KnownModels is the picker list for the character sheet. Claude Code also
// accepts the short aliases; the user can type anything else.
var KnownModels = []string{
	"claude-fable-5-1",
	"claude-opus-5",
	"claude-sonnet-5",
	"claude-haiku-4-5-20251001",
	"opus",
	"sonnet",
	"haiku",
}

var modelLine = regexp.MustCompile(`(?m)^(\s*"model"\s*:\s*)"[^"]*"`)

// PlanModel writes the model into the user settings.json, editing the one
// line in place so the rest of the file keeps its formatting. The [1m]
// context suffix is preserved when the current value has it.
func (st *State) PlanModel(model string) (writer.Plan, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return writer.Plan{}, fmt.Errorf("model cannot be empty")
	}
	if i := strings.Index(st.Settings.Model, "["); i > 0 && !strings.Contains(model, "[") {
		model += st.Settings.Model[i:]
	}
	path := filepath.Join(st.Roots.Home, "settings.json")
	old, err := writer.ReadCurrent(path)
	if err != nil {
		return writer.Plan{}, err
	}
	var next string
	switch {
	case old == nil:
		next = fmt.Sprintf("{\n  \"model\": %q\n}\n", model)
	case modelLine.Match(old):
		next = modelLine.ReplaceAllString(string(old), fmt.Sprintf(`${1}%q`, model))
	default:
		// insert as the first key after the opening brace
		s := strings.TrimLeft(string(old), " \t\r\n")
		if !strings.HasPrefix(s, "{") {
			return writer.Plan{}, fmt.Errorf("%s does not look like a JSON object", path)
		}
		body := strings.TrimSpace(s[1:])
		sep := ",\n"
		if strings.HasPrefix(body, "}") {
			sep = "\n"
		}
		next = fmt.Sprintf("{\n  \"model\": %q%s  %s", model, sep, body)
		if !strings.HasSuffix(next, "\n") {
			next += "\n"
		}
	}
	return writer.Plan{
		Title: "Set model to " + model,
		Ops: []writer.Op{{Kind: writer.OpWrite, Path: path, Old: old, New: []byte(next),
			Why: "user settings.json; project settings.json overrides it when present"}},
	}, nil
}

// PlanAgentName records the character's name in Constelate's own config.
// Claude Code has no such setting; the name is how the sheet addresses you.
func (st *State) PlanAgentName(name string) (writer.Plan, error) {
	name = strings.TrimSpace(name)
	old, err := writer.ReadCurrent(st.Config.Path)
	if err != nil {
		return writer.Plan{}, err
	}
	next := setTOMLTopKey(string(old), "agent_name", name)
	return writer.Plan{
		Title: "Name the agent " + name,
		Ops: []writer.Op{{Kind: writer.OpWrite, Path: st.Config.Path, Old: old, New: []byte(next),
			Why: "Constelate's own config; nothing in ~/.claude changes"}},
	}, nil
}

// PlanInstructions replaces a CLAUDE.md wholesale with the edited text.
func (st *State) PlanInstructions(path string, text string) (writer.Plan, error) {
	old, err := writer.ReadCurrent(path)
	if err != nil {
		return writer.Plan{}, err
	}
	if !strings.HasSuffix(text, "\n") && text != "" {
		text += "\n"
	}
	if string(old) == text {
		return writer.Plan{}, fmt.Errorf("no changes to save")
	}
	return writer.Plan{
		Title: "Save " + filepath.Base(path),
		Ops:   []writer.Op{{Kind: writer.OpWrite, Path: path, Old: old, New: []byte(text), Why: path}},
	}, nil
}

var tomlTop = regexp.MustCompile(`(?m)^\s*agent_name\s*=.*$`)

// setTOMLTopKey sets a top-level string key, keeping the file otherwise intact.
// Top-level keys must precede any [table], so a new key goes at the top.
func setTOMLTopKey(src, key, val string) string {
	line := fmt.Sprintf("%s = %q", key, val)
	if key == "agent_name" && tomlTop.MatchString(src) {
		return tomlTop.ReplaceAllString(src, line)
	}
	if strings.TrimSpace(src) == "" {
		return line + "\n"
	}
	return line + "\n" + src
}
