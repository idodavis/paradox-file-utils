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
  autumn: ["oklch(40.723% 0.161 17.53)", "oklch(61.676% 0.169 23.865)", "oklch(73.425% 0.094 60.729)", "oklch(95.814% 0 0)"],
  business: ["oklch(41.703% 0.099 251.473)", "oklch(64.092% 0.027 229.389)", "oklch(67.271% 0.167 35.791)", "oklch(24.353% 0 0)"],
  coffee: ["oklch(71.996% 0.123 62.756)", "oklch(34.465% 0.029 199.194)", "oklch(42.621% 0.074 224.389)", "oklch(24% 0.023 329.708)"],
  dim: ["oklch(86.133% 0.141 139.549)", "oklch(73.375% 0.165 35.353)", "oklch(74.229% 0.133 311.379)", "oklch(30.857% 0.023 264.149)"],
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
