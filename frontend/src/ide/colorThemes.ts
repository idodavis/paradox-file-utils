/**
 * Register static VS Code color themes that match Nuxt `data-theme` names.
 */
import { registerExtension } from "@codingame/monaco-vscode-api/extensions";
import { ExtensionHostKind } from "@codingame/monaco-vscode-extensions-service-override";
import darkTokenColors from "@codingame/monaco-vscode-theme-defaults-default-extension/resources/dark_plus.json";
import lightTokenColors from "@codingame/monaco-vscode-theme-defaults-default-extension/resources/light_plus.json";
import { isDarkTheme, PMT_THEME_NAMES, type PmtThemeName } from "./appThemes";
import pmtTheme from "./themes/PMT-color-theme.json";
import retroTheme from "./themes/retro-color-theme.json";
import pastelTheme from "./themes/pastel-color-theme.json";
import draculaTheme from "./themes/dracula-color-theme.json";
import luxuryTheme from "./themes/luxury-color-theme.json";
import businessTheme from "./themes/business-color-theme.json";

export { PMT_THEME_NAMES, type PmtThemeName };

/** Token-color payload merged into each contributed theme. */
type TokenColors = {
  tokenColors: unknown[];
  semanticHighlighting?: boolean;
  semanticTokenColors?: Record<string, string>;
};

/** Color-only theme JSON authored under `ide/themes`. */
type ColorThemeJson = {
  name: string;
  type: "dark" | "light";
  colors: Record<string, string>;
};

/** Baked color theme JSON keyed by app theme name. */
const THEME_JSON: Record<PmtThemeName, ColorThemeJson> = {
  PMT: pmtTheme as ColorThemeJson,
  retro: retroTheme as ColorThemeJson,
  pastel: pastelTheme as ColorThemeJson,
  dracula: draculaTheme as ColorThemeJson,
  luxury: luxuryTheme as ColorThemeJson,
  business: businessTheme as ColorThemeJson,
};

/** Merge baked chrome colors with Default Dark/Light Modern tokenColors. */
function themeDocument(name: PmtThemeName): Record<string, unknown> {
  const base = THEME_JSON[name];
  const tokens = (
    isDarkTheme(name) ? darkTokenColors : lightTokenColors
  ) as unknown as TokenColors;
  return {
    $schema: "vscode://schemas/color-theme",
    name: base.name,
    type: base.type,
    colors: base.colors,
    tokenColors: tokens.tokenColors,
    semanticHighlighting: tokens.semanticHighlighting ?? true,
    semanticTokenColors: tokens.semanticTokenColors ?? {},
  };
}

/** Contribute one color theme per Nuxt theme name. */
export async function registerPmtColorThemes(): Promise<void> {
  const { registerFileUrl, whenReady } = registerExtension(
    {
      name: "pmt-color-themes",
      publisher: "pmt",
      version: "1.0.0",
      engines: { vscode: "*" },
      contributes: {
        themes: PMT_THEME_NAMES.map((name) => ({
          id: name,
          label: name,
          uiTheme: isDarkTheme(name) ? "vs-dark" : "vs",
          path: `./themes/${name}-color-theme.json`,
        })),
      },
    },
    ExtensionHostKind.LocalProcess,
  );

  for (const name of PMT_THEME_NAMES) {
    registerFileUrl(
      `./themes/${name}-color-theme.json`,
      `data:application/json,${encodeURIComponent(
        JSON.stringify(themeDocument(name)),
      )}`,
    );
  }

  await whenReady();
}
