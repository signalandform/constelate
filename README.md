# Const.Elate

A terminal character sheet and skill tree for [Claude Code](https://claude.com/claude-code),
in the style of an RPG constellation tree. It is a visual editor over the plain files Claude
Code already reads. It never runs an agent of its own, and if you uninstall it your Claude Code
setup keeps working unchanged.

```
┌─ CODING ─────────────────── ◂ WRITING · RESEARCH · OPS · DESIGN · UNSORTED ▸ ┐
│                                                                              │
│  ★ mcp-builder─────────★ nextjs-app-router-…       ┌  mcp-builder ──────────┐│
│                         ╲                          │ installed              ││
│     ★ r3f-shaders─────────★ r3f-fundamentals       │ scope: personal        ││
│   ╱                                                │ ctx: ~72 tok (est.)    ││
│  ☆ build-mcp-app───────☆ build-mcp-server          │ in: Coding (suggested) ││
│                         ╲                          │                        ││
│     ☆ cardputer-buddy─────☆ build-mcpb             │ Guide for creating     ││
│   ╱                                                │ high-quality MCP       ││
│  ☆ claude-automation-…─☆ example-skill             │ (Model Context         ││
│                         ╲                          │ Protocol) servers that ││
│     ☆ mcp-integration─────☆ hook-development       │ enable LLMs to         ││
│   ╱                                                │ interact with external ││
│  ☆ plugin-structure────☆ receipts                  │ services through       ││
│                         ╲                          │ well-designed tools.   ││
│     ◆ gh cli──────────────◆ git basics             │ Use when building MCP  ││
│                                                    │ …                      ││
│                                                    │                        ││
│                                                    └────────────────────────┘│
│ ★ installed   ☆ managed   ○ available   ◆ trust u…                           │
├──────────────────────────────────────────────────────────────────────────────┤
│ LVL 6  XP ░░░░░░░░░░ 26/1000   CTX ████████ ~7.7k/4.0k tok                   │
└─ 1 CONSTELLATION · 2 CHARACTER · 3 LEDGER · ? keys · q quit · theme ansi ────┘
```

## What it reads

| | |
|---|---|
| Skills | `~/.claude/skills/*/SKILL.md` (personal), `<project>/.claude/skills/` (project), plus claude.ai-synced and plugin skills shown read-only |
| Character | `CLAUDE.md` (user and project), the model in `settings.json` |
| Trust | `permissions.allow` / `deny` across the four settings files |
| XP | session transcripts in `~/.claude/projects/**/*.jsonl`; outcomes only, never token counts |
| Theme | the active [Omarchy](https://omarchy.org) theme's `colors.toml`, falling back to ANSI colours anywhere else |

## Screens

1. **Constellation** (`1`): skills grouped into Coding, Writing, Research, Ops, Design and your
   own categories. `hjkl` or arrows move between nodes, `tab` switches constellations,
   `enter` opens the detail panel, `enter` again installs or disables, `c` re-categorises.
2. **Character sheet** (`2`): agent name, model picker, CLAUDE.md editor (in-app or `e` for
   `$EDITOR`).
3. **Ledger** (`3`): XP per session and trust unlocks. When you have used a command enough times,
   it suggests the matching allow rule. You confirm every time; nothing is auto-granted.

## Mechanics

- **Perk points are context cost.** Each skill's always-loaded cost is estimated from its
  frontmatter (chars/4, labelled as an estimate). The CTX bar compares the total against a
  budget you set. Over budget warns; it never blocks.
- **XP comes from outcomes**: completed sessions, tool calls that succeeded, and approved
  prompts if the logs ever expose them (they do not today). Weights live in config.
- **Categories** are a sidecar, `~/.config/constelate/categories.toml`, never SKILL.md. A
  keyword heuristic suggests one; you can override it.

## Safety

- Every write to `~/.claude` or `./.claude` shows a diff, asks for confirmation, and saves a
  timestamped backup with a manifest under `~/.config/constelate/backups/`.
- `--dry-run` shows every write and makes none.
- Skill folders are never deleted. Disable moves them to `~/.config/constelate/disabled/`
  with a marker recording where they came from, so install puts them back.
- Writes are refused outside `~/.claude`, the project's `.claude`, and `~/.config/constelate`.

## Install

```sh
git clone https://github.com/signalandform/constelate
cd constelate && go build -o constelate .
./constelate                 # TUI, current directory as the project
./constelate --summary       # read-only text report
./constelate --dry-run       # TUI, but every write is only described
./constelate --render 80x24+tab+enter   # print one frame and exit
```

Needs Go 1.24 or newer. Renders cleanly at 80x24; wider is nicer.

## Config

`~/.config/constelate/config.toml` is optional. Defaults:

```toml
agent_name = ""
ctx_budget = 4000

[xp]
session_completed = 50
tool_call_ok = 2
tool_call_err = 0
prompt_approved = 5   # dormant until Claude Code logs approvals
level_xp = 1000

[[trust.unlocks]]
name = "git basics"
prefix = "git"
threshold = 25
rule = "Bash(git status *)"
category = "Coding"
```
