/**
 * Compact theme color swatches for the app theme picker.
 */

/** Primary / secondary / accent / base colors for a named theme. */
export type ThemeSwatchColors = readonly [string, string, string, string];

/** Hardcoded swatches from themes.css (reliable without runtime CSS reads). */
export const THEME_SWATCHES: Record<string, ThemeSwatchColors> = {
  PMT: ["oklch(82% 0.136 87.9)", "oklch(77.4% 0.136 167.3)", "oklch(71.294% 0.166 299.844)", "hsl(223 22% 13%)"],
  retro: ["oklch(80% 0.114 19.571)", "oklch(92% 0.084 155.995)", "oklch(68% 0.162 75.834)", "oklch(91.637% 0.034 90.515)"],
  pastel: ["oklch(90% 0.063 306.703)", "oklch(89% 0.058 10.001)", "oklch(90% 0.093 164.15)", "oklch(100% 0 0)"],
  dracula: ["oklch(75.461% 0.183 346.812)", "oklch(74.202% 0.148 301.883)", "oklch(83.392% 0.124 66.558)", "oklch(28.822% 0.022 277.508)"],
  luxury: ["oklch(100% 0 0)", "oklch(27.581% 0.064 261.069)", "oklch(36.674% 0.051 338.825)", "oklch(14.076% 0.004 285.822)"],
  business: ["oklch(41.703% 0.099 251.473)", "oklch(64.092% 0.027 229.389)", "oklch(67.271% 0.167 35.791)", "oklch(24.353% 0 0)"],
};

/** SelectMenu item with theme name and color preview. */
export type ThemeMenuItem = {
  label: string;
  value: string;
  swatches: ThemeSwatchColors;
};

/** Build SelectMenu items for the given theme names. */
export function themeMenuItems(names: readonly string[]): ThemeMenuItem[] {
  return names.map((name) => ({
    label: name,
    value: name,
    swatches: THEME_SWATCHES[name] ?? THEME_SWATCHES.PMT,
  }));
}
