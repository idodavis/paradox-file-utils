/** Shared app theme names for Nuxt chrome and monaco-vscode workbench sync. */

/** Theme ids matching `data-theme` and contributed VS Code color themes. */
export const PMT_THEME_NAMES = [
  "PMT",
  "retro",
  "pastel",
  "dracula",
  "luxury",
  "business",
] as const;

/** Named PMT theme identifier. */
export type PmtThemeName = (typeof PMT_THEME_NAMES)[number];

const DARK_THEMES = new Set<string>(["PMT", "dracula", "luxury", "business"]);

/** Whether the named theme uses dark chrome and vs-dark syntax base. */
export function isDarkTheme(themeName: string): boolean {
  return DARK_THEMES.has(themeName);
}

/** Map stored/unknown theme names to a supported theme (defaults to PMT). */
export function normalizeThemeName(name: string | undefined): PmtThemeName {
  if (name && (PMT_THEME_NAMES as readonly string[]).includes(name)) {
    return name as PmtThemeName;
  }
  return "PMT";
}
