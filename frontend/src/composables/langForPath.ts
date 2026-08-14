/**
 * Guess a language id for Monaco/workbench from a file path.
 */
export function langForPath(path: string): string {
  const lower = path.toLowerCase();
  if (lower.endsWith(".gui")) return "paradox-gui";
  if (lower.endsWith(".yml") || lower.endsWith(".yaml")) return "paradox-loc";
  if (lower.endsWith(".json")) return "json";
  if (lower.endsWith(".md")) return "markdown";
  if (lower.endsWith(".info")) return "paradox-info";
  return "paradox";
}
