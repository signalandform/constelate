package claude

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Session is what one transcript file tells us about outcomes. Every field is
// derived from what the log actually exposes; anything it does not expose is
// zero and explained in Notes. The schema is unstable across Claude Code
// versions, so the parser reads per line and never fails the whole file.
type Session struct {
	Path        string
	Project     string // basename of the project directory under ~/.claude/projects
	SessionID   string
	Version     string // Claude Code version stamped on the lines
	Started     time.Time
	Ended       time.Time
	UserTurns   int
	AgentTurns  int
	ToolCalls   int  // assistant tool_use blocks
	ToolResults int  // user tool_result blocks
	ToolErrors  int  // tool_result blocks with is_error
	Completed   bool // a cost-state line was written, which happens at session end
	LinesAdded  int
	LinesRemove int
	DurationMs  int64
	Rejections  int // heuristic: tool_result text says the user declined
	ToolNames   map[string]int
	CmdPrefixes map[string]int // first word of every Bash command, e.g. "git"
	Unknown     map[string]int // line types we do not model
	BadLines    int            // lines that were not JSON
	Notes       []string
}

// line is the loose shape of one JSONL record. Everything is optional.
type line struct {
	Type            string       `json:"type"`
	Timestamp       string       `json:"timestamp"`
	SessionID       string       `json:"sessionId"`
	Version         string       `json:"version"`
	Message         *lineMessage `json:"message"`
	TotalLinesAdded *int         `json:"totalLinesAdded"`
	TotalLinesRem   *int         `json:"totalLinesRemoved"`
	TotalDuration   *int64       `json:"totalDuration"`
	StartTime       *int64       `json:"startTime"`
}

type lineMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // string or []block
}

type block struct {
	Type    string          `json:"type"`
	Name    string          `json:"name"`
	IsError bool            `json:"is_error"`
	Input   json.RawMessage `json:"input"`
	Content json.RawMessage `json:"content"`
}

// ProjectsDir is where Claude Code keeps transcripts.
func ProjectsDir(r Roots) string {
	if r.Home == "" {
		return ""
	}
	return filepath.Join(r.Home, "projects")
}

// DiscoverSessions parses every *.jsonl under dir. Unreadable files are
// skipped and reported through the returned error; parsed sessions are still
// returned.
func DiscoverSessions(dir string) ([]Session, error) {
	var out []Session
	var errs []error
	if dir == "" {
		return nil, nil
	}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(p, ".jsonl") {
			return nil
		}
		s, err := ParseSession(p)
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		out = append(out, s)
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		errs = append(errs, err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Started.Before(out[j].Started) })
	return out, errors.Join(errs...)
}

// ParseSession reads one transcript. It only errors when the file cannot be
// opened; malformed lines are counted, not fatal.
func ParseSession(path string) (Session, error) {
	s := Session{
		Path:        path,
		Project:     filepath.Base(filepath.Dir(path)),
		ToolNames:   map[string]int{},
		CmdPrefixes: map[string]int{},
		Unknown:     map[string]int{},
	}
	f, err := os.Open(path)
	if err != nil {
		return s, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 64<<20) // tool results can be huge
	for sc.Scan() {
		raw := sc.Bytes()
		if len(raw) == 0 {
			continue
		}
		var l line
		if err := json.Unmarshal(raw, &l); err != nil {
			s.BadLines++
			continue
		}
		if s.SessionID == "" && l.SessionID != "" {
			s.SessionID = l.SessionID
		}
		if l.Version != "" {
			s.Version = l.Version
		}
		if t, err := time.Parse(time.RFC3339Nano, l.Timestamp); err == nil {
			if s.Started.IsZero() || t.Before(s.Started) {
				s.Started = t
			}
			if t.After(s.Ended) {
				s.Ended = t
			}
		}
		switch l.Type {
		case "user":
			s.UserTurns++
			s.countBlocks(l.Message, true)
		case "assistant":
			s.AgentTurns++
			s.countBlocks(l.Message, false)
		case "cost-state":
			s.Completed = true
			if l.TotalLinesAdded != nil {
				s.LinesAdded = *l.TotalLinesAdded
			}
			if l.TotalLinesRem != nil {
				s.LinesRemove = *l.TotalLinesRem
			}
			if l.TotalDuration != nil {
				s.DurationMs = *l.TotalDuration
			}
			if l.StartTime != nil && s.Started.IsZero() {
				s.Started = time.UnixMilli(*l.StartTime)
			}
		case "system", "attachment", "last-prompt", "mode", "permission-mode",
			"bridge-session", "file-history-snapshot", "file-history-delta",
			"ai-title", "custom-title", "queue-operation", "atis-latch", "summary":
			// known bookkeeping, no outcome signal
		default:
			s.Unknown[l.Type]++
		}
	}
	if err := sc.Err(); err != nil {
		s.Notes = append(s.Notes, "stopped early: "+err.Error())
	}
	if s.ToolCalls > 0 && s.ToolResults == 0 {
		s.Notes = append(s.Notes, "tool_use blocks without tool_result blocks; schema may have changed")
	}
	return s, nil
}

func (s *Session) countBlocks(m *lineMessage, isUser bool) {
	if m == nil || len(m.Content) == 0 || m.Content[0] != '[' {
		return // plain string content is a human prompt, not a tool result
	}
	var blocks []block
	if err := json.Unmarshal(m.Content, &blocks); err != nil {
		return
	}
	for _, b := range blocks {
		switch b.Type {
		case "tool_use":
			if isUser {
				continue
			}
			s.ToolCalls++
			s.ToolNames[b.Name]++
			if b.Name == "Bash" {
				var in struct {
					Command string `json:"command"`
				}
				if json.Unmarshal(b.Input, &in) == nil {
					if p := firstWord(in.Command); p != "" {
						s.CmdPrefixes[p]++
					}
				}
			}
		case "tool_result":
			if !isUser {
				continue
			}
			s.ToolResults++
			if b.IsError {
				s.ToolErrors++
			}
			if looksLikeRejection(b.Content) {
				s.Rejections++
			}
		}
	}
}

// looksLikeRejection is a heuristic. Claude Code does not log permission
// decisions as events; the only trace of a declined prompt is the text the
// harness puts in the tool_result. Keep the phrases in one place so a schema
// change is a one-line fix.
func looksLikeRejection(content json.RawMessage) bool {
	if len(content) == 0 {
		return false
	}
	var str string
	if content[0] == '"' {
		if json.Unmarshal(content, &str) != nil {
			return false
		}
	} else {
		str = string(content)
	}
	str = strings.ToLower(str)
	return strings.Contains(str, "user doesn't want to proceed") ||
		strings.Contains(str, "user declined") ||
		strings.Contains(str, "permission denied by user")
}

func firstWord(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	// skip leading env assignments and cd chains: "cd x && git ..." -> git
	if i := strings.Index(cmd, "&&"); i >= 0 && strings.HasPrefix(cmd, "cd ") {
		cmd = strings.TrimSpace(cmd[i+2:])
	}
	for _, w := range strings.Fields(cmd) {
		if strings.Contains(w, "=") && !strings.HasPrefix(w, "=") {
			continue
		}
		return w
	}
	return ""
}
