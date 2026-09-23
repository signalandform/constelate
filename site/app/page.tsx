import Link from "next/link";
import { Frame } from "@/components/Frame";

export default function Home() {
  return (
    <>
      <section className="wrap" style={{ paddingBlock: "4rem 2.5rem" }}>
        <div className="prose-w">
          <h1 className="display">A character sheet for Claude&nbsp;Code.</h1>
          <p className="lede" style={{ marginTop: "1.25rem" }}>
            Your skills drawn as a constellation. Your CLAUDE.md as the sheet. Your sessions as
            experience. Const.Elate is a terminal editor over the plain files Claude Code already
            reads, and nothing more.
          </p>
          <div
            style={{
              marginTop: "1.75rem",
              display: "flex",
              flexWrap: "wrap",
              gap: "0.75rem 1.5rem",
              alignItems: "center",
            }}
          >
            <code className="cmd" style={{ flex: "1 1 22rem" }}>
              git clone https://github.com/signalandform/constelate
              {"\n"}cd constelate &amp;&amp; go build -o constelate .
            </code>
            <Link href="/docs">Read the docs</Link>
          </div>
        </div>
      </section>

      <section className="wrap" aria-label="The constellation screen">
        <Frame
          name="96x30_enter"
          draw
          label="The constellation screen: skills drawn as star nodes joined by lines, with a detail panel open for the go-cli skill"
        />
        <p className="small muted" style={{ marginTop: "0.75rem" }}>
          Real output from <code>constelate --render</code>, not a screenshot. Filled stars are
          installed, hollow ones are managed by a plugin or synced from claude.ai, diamonds are
          trust unlocks.
        </p>
      </section>

      <section className="wrap" style={{ paddingBlock: "5rem 0" }}>
        <div className="prose-w">
          <h2 className="h2">Three screens, one keyboard</h2>
          <p className="lede" style={{ marginTop: "0.75rem" }}>
            Everything is reachable with <kbd>1</kbd>, <kbd>2</kbd> and <kbd>3</kbd>. Vim keys
            or arrows move; <kbd>enter</kbd> opens and confirms; <kbd>q</kbd> leaves.
          </p>
        </div>

        <div className="stack" style={{ marginTop: "2.5rem" }}>
          <div>
            <h3 className="h3">Constellation</h3>
            <p className="muted prose-w" style={{ marginTop: "0.35rem" }}>
              Skills grouped into Coding, Writing, Research, Ops, Design, and any category you
              add. Move between nodes, open one to read what it does and what it costs, then
              install, disable or re-categorise it in place.
            </p>
            <div style={{ marginTop: "1rem" }}>
              <Frame name="96x30_tab" label="The Writing constellation with five skills" />
            </div>
          </div>
          <div>
            <h3 className="h3">Character sheet</h3>
            <p className="muted prose-w" style={{ marginTop: "0.35rem" }}>
              The agent&rsquo;s name, the model from settings, and CLAUDE.md, editable here or in
              your own editor. User and project files sit side by side under <kbd>u</kbd> and{" "}
              <kbd>p</kbd>.
            </p>
            <div style={{ marginTop: "1rem" }}>
              <Frame name="96x30_2" label="The character sheet showing the agent name Wren, the model, and the CLAUDE.md editor" />
            </div>
          </div>
          <div>
            <h3 className="h3">Ledger</h3>
            <p className="muted prose-w" style={{ marginTop: "0.35rem" }}>
              XP per session, read from your transcripts as outcomes rather than tokens. When a
              command has succeeded enough times, the ledger suggests the matching allow rule.
              You confirm every one.
            </p>
            <div style={{ marginTop: "1rem" }}>
              <Frame name="96x30_3" label="The ledger listing sessions with XP and three trust unlocks in progress" />
            </div>
          </div>
        </div>
      </section>

      <section className="wrap" style={{ paddingBlock: "5rem 0" }}>
        <div className="prose-w">
          <h2 className="h2">What it reads</h2>
          <p className="muted" style={{ marginTop: "0.75rem" }}>
            Nothing of its own except a small config folder. Uninstall it and Claude Code keeps
            working exactly as before.
          </p>
        </div>
        <dl className="rows" style={{ marginTop: "1.5rem" }}>
          <div className="row">
            <dt>Skills</dt>
            <dd>
              <code>~/.claude/skills/*/SKILL.md</code> for personal skills, the project&rsquo;s{" "}
              <code>.claude/skills/</code>, and plugin or claude.ai-synced skills shown read-only.
            </dd>
          </div>
          <div className="row">
            <dt>Character</dt>
            <dd>
              <code>CLAUDE.md</code> for the user and the project, and the model named in{" "}
              <code>settings.json</code>.
            </dd>
          </div>
          <div className="row">
            <dt>Trust</dt>
            <dd>
              <code>permissions.allow</code> and <code>permissions.deny</code> merged across the
              four settings files.
            </dd>
          </div>
          <div className="row">
            <dt>Experience</dt>
            <dd>
              Session transcripts under <code>~/.claude/projects/</code>. Outcomes only: sessions
              completed and tool calls that succeeded. Never token counts.
            </dd>
          </div>
          <div className="row">
            <dt>Theme</dt>
            <dd>
              The active Omarchy theme&rsquo;s <code>colors.toml</code> when there is one, plain
              ANSI colours anywhere else.
            </dd>
          </div>
        </dl>
      </section>

      <section className="wrap" style={{ paddingBlock: "5rem 6rem" }}>
        <div className="prose-w">
          <h2 className="h2">Nothing happens without you</h2>
          <ul
            className="muted"
            style={{
              marginTop: "1.25rem",
              display: "grid",
              gap: "0.9rem",
              paddingLeft: "1.2rem",
              listStyle: "disc",
            }}
          >
            <li>
              Every write shows a diff, asks for confirmation, and saves a timestamped backup
              with a manifest under <code>~/.config/constelate/backups/</code>.
            </li>
            <li>
              <code>--dry-run</code> describes every write and makes none.
            </li>
            <li>
              Skill folders are never deleted. Disabling moves them aside with a marker that
              says where they came from, so installing puts them back.
            </li>
            <li>
              Writes outside <code>~/.claude</code>, the project&rsquo;s <code>.claude</code> and{" "}
              <code>~/.config/constelate</code> are refused.
            </li>
          </ul>
          <p style={{ marginTop: "2rem" }}>
            <Link href="/docs/safety">How the guard rails work</Link>
          </p>
        </div>
      </section>
    </>
  );
}
