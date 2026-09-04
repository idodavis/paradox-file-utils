/**
 * Readonly monaco.editor.colorize for Paradox script and loc snippets.
 */
import * as monaco from "monaco-editor";

/** Language ids registered in paradoxLanguages.ts. */
export type ColorizeLang = "paradox" | "paradox-loc" | "paradox-gui";

/** Escape text when the tokenizer is not ready. */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

/** Colorize one string. Falls back to escaped plain text. */
export async function colorizeText(
  text: string,
  language: ColorizeLang,
): Promise<string> {
  if (!text) return "";
  try {
    return await monaco.editor.colorize(text, language, {});
  } catch {
    return escapeHtml(text);
  }
}

/** Colorize each line so a gutter can stay aligned. */
export async function colorizeLines(
  lines: string[],
  language: ColorizeLang,
): Promise<string[]> {
  return Promise.all(lines.map((line) => colorizeText(line || " ", language)));
}
