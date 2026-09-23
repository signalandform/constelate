package claude

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Instructions is one CLAUDE.md: the agent's standing orders.
type Instructions struct {
	Path     string
	Label    string // "user" or "project"
	Exists   bool
	Bytes    int
	Lines    int
	Headline string // first non-empty line, for the summary
	Err      error
}

// CtxEstimate is the rough token cost of the file (chars/4).
func (in Instructions) CtxEstimate() int { return (in.Bytes + 3) / 4 }

// LoadInstructions reads ~/.claude/CLAUDE.md and <project>/CLAUDE.md.
func LoadInstructions(r Roots, project string) []Instructions {
	var out []Instructions
	if r.Home != "" {
		out = append(out, readInstructions(filepath.Join(r.Home, "CLAUDE.md"), "user"))
	}
	if project != "" && r.Project != "" {
		out = append(out, readInstructions(filepath.Join(project, "CLAUDE.md"), "project"))
	}
	return out
}

func readInstructions(path, label string) Instructions {
	in := Instructions{Path: path, Label: label}
	f, err := os.Open(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			in.Err = err
		}
		return in
	}
	defer f.Close()
	in.Exists = true
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		in.Lines++
		in.Bytes += len(line) + 1
		if in.Headline == "" && strings.TrimSpace(line) != "" {
			in.Headline = strings.TrimSpace(strings.TrimLeft(line, "# "))
		}
	}
	if err := sc.Err(); err != nil {
		in.Err = err
	}
	return in
}
