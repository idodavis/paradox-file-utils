/**
 * App chrome palettes and VS Code color themes generated from SEEDS.
 */
import { registerExtension } from "@codingame/monaco-vscode-api/extensions";
import { ExtensionHostKind } from "@codingame/monaco-vscode-extensions-service-override";
import darkPlus from
  "@codingame/monaco-vscode-theme-defaults-default-extension/resources/dark_plus.json";
import lightPlus from
  "@codingame/monaco-vscode-theme-defaults-default-extension/resources/light_plus.json";
import darkVs from
  "@codingame/monaco-vscode-theme-defaults-default-extension/resources/dark_vs.json";
import lightVs from
  "@codingame/monaco-vscode-theme-defaults-default-extension/resources/light_vs.json";
import Color from "colorjs.io";

/** Palette families for `data-theme`. */
export const PMT_THEME_FAMILIES = [
  "pmt", "ck3", "eu5", "vic3", "catppuccin", "one", "github", "horizon",
] as const;
/** Named palette family. */
export type PmtThemeFamily = (typeof PMT_THEME_FAMILIES)[number];
/** Light or dark appearance. */
export type ColorAppearance = "dark" | "light";
/** Workbench color-theme id: `{family}-{appearance}`. */
export type PmtThemeName = `${PmtThemeFamily}-${ColorAppearance}`;
/** All contributed VS Code color themes. */
export const PMT_THEME_NAMES = [
  "pmt-dark", "pmt-light", "ck3-dark", "ck3-light",
  "eu5-dark", "eu5-light", "vic3-dark", "vic3-light",
  "catppuccin-dark", "catppuccin-light", "one-dark", "one-light",
  "github-dark", "github-light", "horizon-dark", "horizon-light",
] as const satisfies readonly PmtThemeName[];

const FAMILY_LABELS: Record<PmtThemeFamily, string> = {
  pmt: "PMT", ck3: "CK3", eu5: "EU5", vic3: "Vic3",
  catppuccin: "Catppuccin", one: "One", github: "GitHub", horizon: "Horizon",
};

/** Display label for a palette family. */
export function familyLabel(family: PmtThemeFamily): string {
  return FAMILY_LABELS[family];
}

/** Workbench id for a family + appearance. */
export function workbenchThemeId(
  family: PmtThemeFamily,
  appearance: ColorAppearance,
): PmtThemeName {
  return `${family}-${appearance}`;
}

/** Whether a workbench color-theme id is dark. */
export function isDarkTheme(themeName: string): boolean {
  return themeName.endsWith("-dark");
}

/** Map stored/unknown names to a palette family (defaults to pmt). */
export function normalizeThemeFamily(name: string | undefined): PmtThemeFamily {
  if (!name) return "pmt";
  const lower = name.toLowerCase();
  if ((PMT_THEME_FAMILIES as readonly string[]).includes(lower)) {
    return lower as PmtThemeFamily;
  }
  if (lower.endsWith("-dark") || lower.endsWith("-light")) {
    const fam = lower.slice(0, lower.lastIndexOf("-"));
    if ((PMT_THEME_FAMILIES as readonly string[]).includes(fam)) {
      return fam as PmtThemeFamily;
    }
  }
  return "pmt";
}

/** Workbench id from `data-theme` and appearance (or html.dark as fallback). */
export function currentWorkbenchTheme(
  appearance?: ColorAppearance,
): PmtThemeName {
  const el = document.documentElement;
  const mode: ColorAppearance =
    appearance ?? (el.classList.contains("dark") ? "dark" : "light");
  return workbenchThemeId(normalizeThemeFamily(el.dataset.theme), mode);
}

/** ~18 seed colors that expand into the workbench chrome palette. */
type ThemeSeed = {
  fg: string; accent: string; link: string; error: string; info: string;
  warning: string; success: string; editorBg: string; chrome: string;
  chrome2: string; input: string; button: string; buttonHover: string;
  badgeFg: string; remoteBg: string; remoteFg: string; inactive: string;
};

const SEEDS: Record<PmtThemeName, ThemeSeed> = {
  "pmt-dark": {
    fg: "#9fb9d0", accent: "#e9be4f", link: "#46d1a4", error: "#ff6d5c",
    info: "#89e0eb", warning: "#e6de83", success: "#99e69b", editorBg: "#12151d",
    chrome: "#14171f", chrome2: "#1a1e28", input: "#0d1016", button: "#bf9f4d",
    buttonHover: "#c5c66f", badgeFg: "#160603", remoteBg: "#377a69",
    remoteFg: "#160409", inactive: "#46d1a4",
  },
  "pmt-light": {
    fg: "#3d4555", accent: "#c9a227", link: "#1a9e74", error: "#c23b2e",
    info: "#1a7a86", warning: "#9a7b16", success: "#2a8a4a", editorBg: "#faf8f2",
    chrome: "#f7f5ee", chrome2: "#f0ebe0", input: "#fffdf8", button: "#c9a227",
    buttonHover: "#b8911c", badgeFg: "#1b2030", remoteBg: "#1a9e74",
    remoteFg: "#f4efe3", inactive: "#7c4dbd",
  },
  "ck3-dark": {
    fg: "#ded6bf", accent: "#d19260", link: "#829ca6", error: "#b23131",
    info: "#576071", warning: "#d1c480", success: "#6e7940", editorBg: "#141416",
    chrome: "#1a1a21", chrome2: "#242428", input: "#101012", button: "#d19260",
    buttonHover: "#d1c480", badgeFg: "#1a1a21", remoteBg: "#7f5f45",
    remoteFg: "#ded6bf", inactive: "#ad8f69",
  },
  "ck3-light": {
    fg: "#3a3428", accent: "#1a1a21", link: "#7a1f28", error: "#a32b22",
    info: "#3a4a5c", warning: "#8a7040", success: "#4a5c30", editorBg: "#f6f3eb",
    chrome: "#f2efe7", chrome2: "#e8e4d8", input: "#faf8f3", button: "#1a1a21",
    buttonHover: "#2d2d32", badgeFg: "#f6f3eb", remoteBg: "#7a1f28",
    remoteFg: "#f6f3eb", inactive: "#7a1f28",
  },
  "eu5-dark": {
    fg: "#f0ead9", accent: "#c5a059", link: "#4ebad6", error: "#e36166",
    info: "#4ebad6", warning: "#f0d999", success: "#5db149", editorBg: "#12161f",
    chrome: "#1a222e", chrome2: "#222a38", input: "#10141c", button: "#ad8f69",
    buttonHover: "#d1b37d", badgeFg: "#12161f", remoteBg: "#1a7380",
    remoteFg: "#f0ead9", inactive: "#36595f",
  },
  "eu5-light": {
    fg: "#1a222e", accent: "#8c5e16", link: "#1a3358", error: "#b03a40",
    info: "#1a3358", warning: "#8a7040", success: "#2e7a3a", editorBg: "#f5f3ee",
    chrome: "#f0eee8", chrome2: "#e6e3db", input: "#faf9f6", button: "#8c5e16",
    buttonHover: "#6e4a12", badgeFg: "#f5f3ee", remoteBg: "#1a3358",
    remoteFg: "#f5f3ee", inactive: "#1a2e4a",
  },
  "vic3-dark": {
    fg: "#e4dcd0", accent: "#8c3c4c", link: "#c5a66d", error: "#e05757",
    info: "#69abbd", warning: "#ffac74", success: "#63ab54", editorBg: "#141618",
    chrome: "#2c3233", chrome2: "#262c2d", input: "#111313", button: "#3e6b52",
    buttonHover: "#4a7a5c", badgeFg: "#e4dcd0", remoteBg: "#541b2b",
    remoteFg: "#e4dcd0", inactive: "#3e6b52",
  },
  "vic3-light": {
    fg: "#1e2224", accent: "#6b2434", link: "#2d5a40", error: "#b03a3a",
    info: "#2a6a78", warning: "#a06a30", success: "#3e6b52", editorBg: "#e8e4dc",
    chrome: "#ddd6cc", chrome2: "#cfc8bc", input: "#f2efe8", button: "#3e6b52",
    buttonHover: "#2d5a27", badgeFg: "#e8e4dc", remoteBg: "#541b2b",
    remoteFg: "#e8e4dc", inactive: "#3e6b52",
  },
  "catppuccin-dark": {
    fg: "#cdd6f4", accent: "#cba6f7", link: "#89b4fa", error: "#f38ba8",
    info: "#89dceb", warning: "#f9e2af", success: "#a6e3a1", editorBg: "#1e1e2e",
    chrome: "#181825", chrome2: "#11111b", input: "#313244", button: "#cba6f7",
    buttonHover: "#f5c2e7", badgeFg: "#1e1e2e", remoteBg: "#89b4fa",
    remoteFg: "#1e1e2e", inactive: "#a6e3a1",
  },
  "catppuccin-light": {
    fg: "#4c4f69", accent: "#8839ef", link: "#1e66f5", error: "#d20f39",
    info: "#04a5e5", warning: "#df8e1d", success: "#40a02b", editorBg: "#eff1f5",
    chrome: "#e6e9ef", chrome2: "#dce0e8", input: "#ccd0da", button: "#8839ef",
    buttonHover: "#ea76cb", badgeFg: "#eff1f5", remoteBg: "#1e66f5",
    remoteFg: "#eff1f5", inactive: "#40a02b",
  },
  "one-dark": {
    fg: "#abb2bf", accent: "#61afef", link: "#56b6c2", error: "#e06c75",
    info: "#56b6c2", warning: "#e5c07b", success: "#98c379", editorBg: "#282c34",
    chrome: "#21252b", chrome2: "#181a1f", input: "#1c1f26", button: "#61afef",
    buttonHover: "#528bce", badgeFg: "#282c34", remoteBg: "#98c379",
    remoteFg: "#282c34", inactive: "#c678dd",
  },
  "one-light": {
    fg: "#383a42", accent: "#4078f2", link: "#0184bc", error: "#e45649",
    info: "#0184bc", warning: "#c18401", success: "#50a14f", editorBg: "#fafafa",
    chrome: "#f0f0f0", chrome2: "#e5e5e6", input: "#ffffff", button: "#4078f2",
    buttonHover: "#2d5fd0", badgeFg: "#fafafa", remoteBg: "#50a14f",
    remoteFg: "#fafafa", inactive: "#a626a4",
  },
  "github-dark": {
    fg: "#e6edf3", accent: "#2f81f7", link: "#2f81f7", error: "#f85149",
    info: "#2f81f7", warning: "#d29922", success: "#3fb950", editorBg: "#0d1117",
    chrome: "#010409", chrome2: "#161b22", input: "#21262d", button: "#238636",
    buttonHover: "#2ea043", badgeFg: "#ffffff", remoteBg: "#1f6feb",
    remoteFg: "#ffffff", inactive: "#a371f7",
  },
  "github-light": {
    fg: "#1f2328", accent: "#0969da", link: "#0969da", error: "#cf222e",
    info: "#0969da", warning: "#9a6700", success: "#1a7f37", editorBg: "#ffffff",
    chrome: "#f6f8fa", chrome2: "#d0d7de", input: "#ffffff", button: "#1f883d",
    buttonHover: "#1a7f37", badgeFg: "#ffffff", remoteBg: "#0969da",
    remoteFg: "#ffffff", inactive: "#8250df",
  },
  "horizon-dark": {
    fg: "#e0d6d1", accent: "#e95678", link: "#f09383", error: "#e95678",
    info: "#25b0bc", warning: "#fab795", success: "#29d398", editorBg: "#1c1e26",
    chrome: "#16161c", chrome2: "#232530", input: "#2e303e", button: "#f09383",
    buttonHover: "#e95678", badgeFg: "#1c1e26", remoteBg: "#e95678",
    remoteFg: "#1c1e26", inactive: "#b877db",
  },
  "horizon-light": {
    fg: "#3d3440", accent: "#d72638", link: "#e0583a", error: "#d72638",
    info: "#1d8991", warning: "#c47a2a", success: "#1d8a5b", editorBg: "#fbfaf9",
    chrome: "#f5f3f2", chrome2: "#eceae9", input: "#fdfcfb", button: "#e0583a",
    buttonHover: "#d72638", badgeFg: "#fff6f3", remoteBg: "#d72638",
    remoteFg: "#fff6f3", inactive: "#8a56ac",
  },
};

/** Primary / secondary / success / chrome swatches for the theme picker. */
export type ThemeSwatchColors = readonly [string, string, string, string];

/** SelectMenu item with family name and color preview. */
export type ThemeMenuItem = {
  label: string;
  value: string;
  swatches: ThemeSwatchColors;
};

/** Four-color preview derived from SEEDS for the current appearance. */
export function themeMenuItems(appearance: ColorAppearance): ThemeMenuItem[] {
  return PMT_THEME_FAMILIES.map((family) => {
    const s = SEEDS[workbenchThemeId(family, appearance)];
    return {
      label: familyLabel(family),
      value: family,
      swatches: [s.accent, s.link, s.success, s.chrome],
    };
  });
}

/** Inject family CSS custom properties onto `:root` from SEEDS. */
export function applySeedCss(name: PmtThemeName): void {
  const s = SEEDS[name];
  const root = document.documentElement;
  root.style.colorScheme = isDarkTheme(name) ? "dark" : "light";
  const vars: Record<string, string> = {
    "--pmt-base-100": s.chrome, "--pmt-base-200": s.chrome2,
    "--pmt-base-300": s.input, "--pmt-base-content": s.fg,
    "--pmt-primary": s.accent, "--pmt-primary-content": s.badgeFg,
    "--pmt-secondary": s.link, "--pmt-secondary-content": s.badgeFg,
    "--pmt-accent": s.inactive, "--pmt-accent-content": s.badgeFg,
    "--pmt-neutral": s.input, "--pmt-neutral-content": s.fg,
    "--pmt-info": s.info, "--pmt-info-content": s.badgeFg,
    "--pmt-success": s.success, "--pmt-success-content": s.badgeFg,
    "--pmt-warning": s.warning, "--pmt-warning-content": s.badgeFg,
    "--pmt-error": s.error, "--pmt-error-content": s.badgeFg,
    "--pmt-editor-bg": s.editorBg,
  };
  for (const [k, v] of Object.entries(vars)) root.style.setProperty(k, v);
}

function a(hex: string, hexAlpha: string): string {
  const c = new Color(hex);
  c.alpha = parseInt(hexAlpha, 16) / 255;
  return c.to("srgb").toString({ format: "hex" });
}

/** Expand a seed into the workbench color keys VS Code consumes. */
function workbenchColors(s: ThemeSeed): Record<string, string> {
  const {
    fg, accent, link, error, info, warning, success,
    editorBg, chrome, chrome2, input, button, buttonHover,
    badgeFg, remoteBg, remoteFg,
  } = s;
  return {
    foreground: fg, focusBorder: a(accent, "80"), "icon.foreground": fg,
    errorForeground: error, descriptionForeground: a(fg, "9e"),
    "editor.background": editorBg, "editor.foreground": fg,
    "editor.selectionBackground": a(accent, "47"),
    "editor.inactiveSelectionBackground": a(accent, "24"),
    "editor.selectionHighlightBackground": a(accent, "29"),
    "editorLineNumber.foreground": a(fg, "66"),
    "editorLineNumber.activeForeground": fg, "editorCursor.foreground": accent,
    "editorWidget.background": chrome, "editorWidget.foreground": fg,
    "editorWidget.border": a(fg, "1f"), "editorSuggestWidget.background": chrome,
    "editorGroup.border": a(fg, "1f"), "editorGroupHeader.tabsBackground": chrome,
    "editorGroupHeader.tabsBorder": a(fg, "1a"),
    "sideBar.background": chrome, "sideBar.foreground": fg,
    "sideBar.border": a(fg, "1f"), "sideBarTitle.foreground": fg,
    "sideBarSectionHeader.background": a(input, "b3"),
    "sideBarSectionHeader.foreground": fg,
    "sideBarSectionHeader.border": a(fg, "1a"),
    "list.activeSelectionBackground": a(accent, "33"),
    "list.activeSelectionForeground": fg,
    "list.inactiveSelectionBackground": a(accent, "1f"),
    "list.inactiveSelectionForeground": fg, "list.hoverBackground": a(fg, "12"),
    "list.focusOutline": a(accent, "73"), "list.highlightForeground": accent,
    "list.focusBackground": a(accent, "24"), "tree.indentGuidesStroke": a(fg, "38"),
    "tab.activeBackground": editorBg, "tab.activeForeground": fg,
    "tab.activeBorderTop": accent, "tab.inactiveBackground": chrome,
    "tab.inactiveForeground": a(fg, "9e"), "tab.border": a(fg, "1a"),
    "tab.hoverBackground": a(fg, "0d"), "activityBar.background": chrome,
    "activityBar.foreground": fg, "activityBar.inactiveForeground": a(fg, "8c"),
    "activityBar.border": a(fg, "1f"), "activityBar.activeBorder": accent,
    "activityBarBadge.background": accent, "activityBarBadge.foreground": badgeFg,
    "statusBar.background": chrome2, "statusBar.foreground": fg,
    "statusBar.border": a(fg, "1f"), "statusBar.noFolderBackground": chrome2,
    "statusBarItem.remoteBackground": remoteBg,
    "statusBarItem.remoteForeground": remoteFg,
    "panel.background": chrome, "panel.border": a(fg, "1f"),
    "panelTitle.activeBorder": accent, "panelTitle.activeForeground": fg,
    "panelTitle.inactiveForeground": a(fg, "9e"),
    "titleBar.activeBackground": chrome2, "titleBar.activeForeground": fg,
    "titleBar.inactiveBackground": chrome, "titleBar.inactiveForeground": a(fg, "9e"),
    "titleBar.border": a(fg, "1f"), "input.background": input,
    "input.foreground": fg, "input.border": a(fg, "29"),
    "input.placeholderForeground": a(fg, "73"), "dropdown.background": input,
    "dropdown.foreground": fg, "dropdown.border": a(fg, "29"),
    "dropdown.listBackground": chrome, "button.background": button,
    "button.foreground": badgeFg, "button.hoverBackground": buttonHover,
    "button.secondaryBackground": input, "button.secondaryForeground": fg,
    "badge.background": button, "badge.foreground": badgeFg,
    "progressBar.background": accent, "scrollbarSlider.background": a(fg, "2e"),
    "inputOption.activeBorder": accent,
    "inputOption.activeBackground": a(accent, "24"),
    "listFilterWidget.background": chrome,
    "listFilterWidget.outline": a(accent, "80"),
    "searchEditor.findMatchBackground": a(accent, "33"),
    "search.resultsInfoForeground": a(fg, "9e"),
    "scrollbarSlider.hoverBackground": a(fg, "47"),
    "scrollbarSlider.activeBackground": a(fg, "5c"),
    "minimap.background": chrome, "peekViewEditor.background": chrome,
    "peekViewResult.background": chrome, "quickInput.background": chrome,
    "quickInput.foreground": fg, "menu.background": chrome,
    "menu.foreground": fg, "menu.selectionBackground": a(accent, "38"),
    "menu.selectionForeground": fg,
    "notificationCenterHeader.background": chrome,
    "notifications.background": chrome, "notifications.foreground": fg,
    "notifications.border": a(fg, "1f"), "widget.border": a(fg, "1f"),
    "editorGutter.addedBackground": success,
    "editorGutter.deletedBackground": error,
    "editorGutter.modifiedBackground": info, "editorInfo.foreground": info,
    "editorWarning.foreground": warning, "editorError.foreground": error,
    "textLink.foreground": link, "textLink.activeForeground": accent,
  };
}

/** Token-color payload merged into each contributed theme. */
type TokenColors = {
  tokenColors?: unknown[];
};

/** Mix two hex colors; t=0 is a, t=1 is b. */
function mix(aHex: string, bHex: string, t: number): string {
  return Color.mix(aHex, bHex, t, { space: "srgb", outputSpace: "srgb" })
    .toString({ format: "hex" });
}

/**
 * Per-theme comment grey from editor.foreground (not VS Code's green).
 * Dark themes lean toward black; light themes toward a mid grey.
 */
function commentGrey(fg: string, dark: boolean): string {
  const rgb = new Color(fg).to("srgb").coords ?? [0, 0, 0];
  const grey = ((rgb[0] ?? 0) + (rgb[1] ?? 0) + (rgb[2] ?? 0)) / 3;
  const v = dark ? grey * 0.42 + 28 / 255 : grey * 0.35 + 96 / 255;
  return new Color("srgb", [v, v, v]).toString({ format: "hex" });
}

/** Merge generated chrome colors with vs + plus tokenColors. */
function themeDocument(name: PmtThemeName): Record<string, unknown> {
  const seed = SEEDS[name];
  const colors = workbenchColors(seed);
  const dark = isDarkTheme(name);
  const vs = (dark ? darkVs : lightVs) as unknown as TokenColors;
  const plus = (dark ? darkPlus : lightPlus) as unknown as TokenColors;
  const accent = colors["editorCursor.foreground"] ?? "#e9be4f";
  const link = colors["textLink.foreground"] ?? accent;
  const info = colors["editorInfo.foreground"] ?? link;
  const fg = colors["editor.foreground"] ?? "#9fb9d0";
  const comment = commentGrey(fg, dark);
  const property = mix(fg, info, 0.55);
  return {
    $schema: "vscode://schemas/color-theme",
    name,
    type: dark ? "dark" : "light",
    colors,
    tokenColors: [
      ...(vs.tokenColors ?? []),
      ...(plus.tokenColors ?? []),
      { scope: ["comment", "comment.line.number-sign.paradox"],
        settings: { foreground: comment } },
      { scope: ["keyword.control.paradox", "keyword.operator.iterator.paradox"],
        settings: { foreground: accent } },
      { scope: "entity.name.type.paradox", settings: { foreground: info } },
      { scope: "variable.other.property.paradox", settings: { foreground: property } },
      { scope: "entity.name.function.event-id.paradox", settings: { foreground: link } },
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
