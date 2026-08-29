/**
 * Register static VS Code color themes that match Nuxt `data-theme` names.
 */
import { registerExtension } from "@codingame/monaco-vscode-api/extensions";
import { ExtensionHostKind } from "@codingame/monaco-vscode-extensions-service-override";
import darkPlus from "@codingame/monaco-vscode-theme-defaults-default-extension/resources/dark_plus.json";
import lightPlus from "@codingame/monaco-vscode-theme-defaults-default-extension/resources/light_plus.json";
import darkVs from "@codingame/monaco-vscode-theme-defaults-default-extension/resources/dark_vs.json";
import lightVs from "@codingame/monaco-vscode-theme-defaults-default-extension/resources/light_vs.json";
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
  tokenColors?: unknown[];
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

/** Parse #rgb / #rrggbb into components. */
function parseHex(hex: string): [number, number, number] | null {
  const h = hex.replace("#", "");
  if (h.length === 3) {
    return [
      parseInt(h[0]! + h[0], 16),
      parseInt(h[1]! + h[1], 16),
      parseInt(h[2]! + h[2], 16),
    ];
  }
  if (h.length >= 6) {
    return [
      parseInt(h.slice(0, 2), 16),
      parseInt(h.slice(2, 4), 16),
      parseInt(h.slice(4, 6), 16),
    ];
  }
  return null;
}

function hexRgb(r: number, g: number, b: number): string {
  const c = (n: number) =>
    Math.max(0, Math.min(255, Math.round(n))).toString(16).padStart(2, "0");
  return `#${c(r)}${c(g)}${c(b)}`;
}

/** Mix two hex colors; t=0 is a, t=1 is b. */
function mix(a: string, b: string, t: number): string {
  const A = parseHex(a);
  const B = parseHex(b);
  if (!A || !B) return b;
  return hexRgb(
    A[0] + (B[0] - A[0]) * t,
    A[1] + (B[1] - A[1]) * t,
    A[2] + (B[2] - A[2]) * t,
  );
}

/**
 * Per-theme comment grey from editor.foreground (not VS Code's green).
 * Dark themes lean toward black; light themes toward a mid grey.
 */
function commentGrey(fg: string, dark: boolean): string {
  const rgb = parseHex(fg);
  if (!rgb) return dark ? "#6b7280" : "#9ca3af";
  const grey = (rgb[0] + rgb[1] + rgb[2]) / 3;
  const v = dark ? grey * 0.42 + 28 : grey * 0.35 + 96;
  return hexRgb(v, v, v);
}

/** Merge baked chrome colors with vs + plus tokenColors (include resolved). */
function themeDocument(name: PmtThemeName): Record<string, unknown> {
  const base = THEME_JSON[name];
  const dark = isDarkTheme(name);
  const vs = (dark ? darkVs : lightVs) as unknown as TokenColors;
  const plus = (dark ? darkPlus : lightPlus) as unknown as TokenColors;
  const accent = base.colors["editorCursor.foreground"] ?? "#e9be4f";
  const link = base.colors["textLink.foreground"] ?? accent;
  const info = base.colors["editorInfo.foreground"] ?? link;
  const fg = base.colors["editor.foreground"] ?? "#9fb9d0";
  const comment = commentGrey(fg, dark);
  const property = mix(fg, info, 0.55);
  return {
    $schema: "vscode://schemas/color-theme",
    name: base.name,
    type: base.type,
    colors: base.colors,
    tokenColors: [
      ...(vs.tokenColors ?? []),
      ...(plus.tokenColors ?? []),
      {
        scope: ["comment", "comment.line.number-sign.paradox"],
        settings: { foreground: comment },
      },
      {
        scope: [
          "keyword.control.paradox",
          "keyword.operator.iterator.paradox",
        ],
        settings: { foreground: accent },
      },
      {
        scope: "entity.name.type.paradox",
        settings: { foreground: info },
      },
      {
        scope: "variable.other.property.paradox",
        settings: { foreground: property },
      },
      {
        scope: "entity.name.function.event-id.paradox",
        settings: { foreground: link },
      },
    ],
    semanticHighlighting: false,
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
