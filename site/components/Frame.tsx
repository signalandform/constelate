import { loadFrame } from "@/lib/frames";

type Kind = "star" | "hollow" | "unlock" | "bar" | "barEmpty" | "edge" | "ok" | "text";

const STAR = "★";
const HOLLOW = new Set(["☆", "○"]);
const UNLOCK = "◆";
const EDGE = new Set(["╲", "╱"]);

function kindOf(ch: string): Kind {
  if (ch === STAR) return "star";
  if (HOLLOW.has(ch)) return "hollow";
  if (ch === UNLOCK) return "unlock";
  if (ch === "█") return "bar";
  if (ch === "░") return "barEmpty";
  if (EDGE.has(ch)) return "edge";
  if (ch === "✓") return "ok";
  return "text";
}

/**
 * Renders one frame of the TUI as text. Node glyphs and the lines between them
 * get their own spans so the stylesheet can colour them and, on the hero, draw
 * them in once. Everything else stays plain text: this is the tool's own output,
 * not a picture of it.
 */
export function Frame({
  name,
  draw = false,
  label,
}: {
  name: string;
  draw?: boolean;
  label: string;
}) {
  const text = loadFrame(name);
  const lines = text.split("\n");
  let nodeIndex = 0;

  return (
    <pre
      className={draw ? "frame frame--draw" : "frame"}
      aria-label={label}
      role="img"
      tabIndex={0}
    >
      {lines.map((line, li) => {
        const parts: React.ReactNode[] = [];
        let buf = "";
        let inNodeLine = false;
        const flush = () => {
          if (buf) parts.push(buf);
          buf = "";
        };
        for (const ch of Array.from(line)) {
          const k = kindOf(ch);
          if (k === "text") {
            // A run of ─ that sits between two nodes on a node line is an edge.
            if (ch === "─" && inNodeLine) {
              flush();
              parts.push(
                <span key={parts.length} className="f-edge">
                  {ch}
                </span>,
              );
              continue;
            }
            buf += ch;
            continue;
          }
          flush();
          if (k === "star" || k === "hollow" || k === "unlock") {
            inNodeLine = true;
            const i = nodeIndex++;
            parts.push(
              <span
                key={parts.length}
                className={`f-node f-${k}`}
                style={{ ["--i" as string]: i }}
              >
                {ch}
              </span>,
            );
          } else {
            parts.push(
              <span key={parts.length} className={`f-${k}`}>
                {ch}
              </span>,
            );
          }
        }
        flush();
        return (
          <span key={li} className="f-line">
            {parts}
            {"\n"}
          </span>
        );
      })}
    </pre>
  );
}
