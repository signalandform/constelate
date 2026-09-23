package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/signalandform/constelate/internal/writer"
)

var (
	allowOpen = regexp.MustCompile(`("allow"\s*:\s*\[)(\s*)`)
	permOpen  = regexp.MustCompile(`("permissions"\s*:\s*\{)`)
)

// PlanAllowRule proposes adding one pattern to permissions.allow in the user
// settings.local.json, which is where Claude Code itself records approvals.
// The edit is textual so the file keeps its formatting; the result is parsed
// to prove it is still valid JSON before it is shown.
func (st *State) PlanAllowRule(rule string) (writer.Plan, error) {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return writer.Plan{}, fmt.Errorf("empty rule")
	}
	for _, r := range st.Settings.Allow {
		if r.Pattern == rule {
			return writer.Plan{}, fmt.Errorf("%s is already allowed (%s)", rule, r.Source)
		}
	}
	path := filepath.Join(st.Roots.Home, "settings.local.json")
	old, err := writer.ReadCurrent(path)
	if err != nil {
		return writer.Plan{}, err
	}
	next, err := insertAllow(string(old), rule)
	if err != nil {
		return writer.Plan{}, err
	}
	var check map[string]any
	if err := json.Unmarshal([]byte(next), &check); err != nil {
		return writer.Plan{}, fmt.Errorf("edit would break %s: %v", path, err)
	}
	return writer.Plan{
		Title: "Allow " + rule,
		Ops: []writer.Op{{Kind: writer.OpWrite, Path: path, Old: old, New: []byte(next),
			Why: "permissions.allow in user settings.local.json; Claude Code stops prompting for this"}},
	}, nil
}

func insertAllow(src, rule string) (string, error) {
	q := fmt.Sprintf("%q", rule)
	if strings.TrimSpace(src) == "" {
		return "{\n  \"permissions\": {\n    \"allow\": [\n      " + q + "\n    ]\n  }\n}\n", nil
	}
	if loc := allowOpen.FindStringSubmatchIndex(src); loc != nil {
		after := src[loc[1]:] // text after "[" and following whitespace
		indent := "      "
		if m := regexp.MustCompile(`^\n([ \t]+)`).FindStringSubmatch(src[loc[2*1+1]:]); m != nil {
			indent = m[1]
		}
		if strings.HasPrefix(after, "]") {
			// empty array
			closeIndent := strings.TrimRight(indent, " \t")
			if len(indent) >= 2 {
				closeIndent = indent[:len(indent)-2]
			}
			return src[:loc[3]] + "\n" + indent + q + "\n" + closeIndent + src[loc[1]:], nil
		}
		return src[:loc[3]] + "\n" + indent + q + "," + src[loc[3]:], nil
	}
	if loc := permOpen.FindStringSubmatchIndex(src); loc != nil {
		return src[:loc[1]] + "\n    \"allow\": [\n      " + q + "\n    ]," + src[loc[1]:], nil
	}
	s := strings.TrimLeft(src, " \t\r\n")
	if !strings.HasPrefix(s, "{") {
		return "", fmt.Errorf("settings.local.json is not a JSON object")
	}
	body := strings.TrimSpace(s[1:])
	sep := ",\n"
	if strings.HasPrefix(body, "}") {
		sep = "\n"
	}
	out := "{\n  \"permissions\": {\n    \"allow\": [\n      " + q + "\n    ]\n  }" + sep + "  " + body
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, nil
}
