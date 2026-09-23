// Package config holds Constelate's own settings: XP weights, the context
// budget, trust thresholds. It lives in ~/.config/constelate/config.toml and
// is never required; every field has a default.
package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Path string `toml:"-"`

	// Context budget in estimated tokens for always-loaded skill text.
	CtxBudget int `toml:"ctx_budget"`

	XP    XP    `toml:"xp"`
	Trust Trust `toml:"trust"`
}

// XP weights. Outcomes only; never raw token counts.
type XP struct {
	SessionCompleted int `toml:"session_completed"`
	ToolCallOK       int `toml:"tool_call_ok"`
	ToolCallErr      int `toml:"tool_call_err"`
	PromptApproved   int `toml:"prompt_approved"` // dormant until logs expose approvals
	PromptRejected   int `toml:"prompt_rejected"`
	LevelXP          int `toml:"level_xp"` // XP per level, linear
}

// Trust unlock thresholds: after N successful uses of a command prefix,
// suggest an allow rule. The user confirms every time; nothing auto-grants.
type Trust struct {
	Unlocks []Unlock `toml:"unlocks"`
}

type Unlock struct {
	Name      string `toml:"name"`
	Prefix    string `toml:"prefix"`    // Bash command first word, e.g. "git"
	Threshold int    `toml:"threshold"` // successful calls needed
	Rule      string `toml:"rule"`      // permissions.allow pattern to suggest
	Category  string `toml:"category"`  // constellation the node appears in
}

func Default() Config {
	return Config{
		CtxBudget: 4000,
		XP: XP{
			SessionCompleted: 50,
			ToolCallOK:       2,
			ToolCallErr:      0,
			PromptApproved:   5,
			PromptRejected:   0,
			LevelXP:          1000,
		},
		Trust: Trust{Unlocks: []Unlock{
			{Name: "git basics", Prefix: "git", Threshold: 25, Rule: "Bash(git status *)", Category: "Coding"},
			{Name: "gh cli", Prefix: "gh", Threshold: 25, Rule: "Bash(gh pr *)", Category: "Coding"},
			{Name: "npm scripts", Prefix: "npm", Threshold: 25, Rule: "Bash(npm run *)", Category: "Ops"},
		}},
	}
}

// DefaultPath is ~/.config/constelate/config.toml.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "constelate", "config.toml"), nil
}

// Load reads path over the defaults. A missing file returns defaults and no error.
func Load(path string) (Config, error) {
	c := Default()
	c.Path = path
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c, nil
		}
		return c, err
	}
	if err := toml.Unmarshal(b, &c); err != nil {
		return Default(), err
	}
	if c.XP.LevelXP <= 0 {
		c.XP.LevelXP = Default().XP.LevelXP
	}
	return c, nil
}
