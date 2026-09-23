import { readFileSync } from "node:fs";
import path from "node:path";

/** A frame is the exact text `constelate --render` printed, kept under content/frames. */
export function loadFrame(name: string): string {
  const file = path.join(process.cwd(), "content", "frames", `${name}.txt`);
  return readFileSync(file, "utf8").replace(/\n$/, "");
}
