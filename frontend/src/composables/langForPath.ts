/**
 * Map file paths to Pierre/Shiki language ids (Paradox script, loc, JSON).
 */
import type { BundledLanguage } from "@pierre/diffs";

/** Resolve a Pierre language id from a filesystem path. */
export function langForPath(path: string): BundledLanguage | string {
  const base = path.replace(/\\/g, "/").split("/").pop()?.toLowerCase() ?? "";
  if (base.endsWith(".json")) return "json";
  if (base.endsWith(".yml") || base.endsWith(".yaml")) return "paradox-loc";
  if (base.endsWith(".gui")) return "paradox-gui";
  if (base.endsWith(".info")) return "paradox-info";
  if (base.endsWith(".txt") || base.endsWith(".mod")) return "paradox";
  return "paradox";
}
