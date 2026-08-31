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
  "pmt", "catppuccin", "one", "monokai", "github",
] as const;
/** Named palette family. */
export type PmtThemeFamily = (typeof PMT_THEME_FAMILIES)[number];
/** Light or dark appearance. */
export type ColorAppearance = "dark" | "light";
/** Workbench color-theme id: `{family}-{appearance}`. */
export type PmtThemeName = `${PmtThemeFamily}-${ColorAppearance}`;
/** All contributed VS Code color themes. */
export const PMT_THEME_NAMES = [
  "pmt-dark", "pmt-light", "catppuccin-dark", "catppuccin-light",
  "one-dark", "one-light", "monokai-dark", "monokai-light",
  "github-dark", "github-light",
] as const satisfies readonly PmtThemeName[];

const FAMILY_LABELS: Record<PmtThemeFamily, string> = {
  pmt: "PMT", catppuccin: "Catppuccin", one: "One",
  monokai: "Monokai", github: "GitHub",
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

/** Workbench id from the live `data-theme` + `.dark` class. */
export function currentWorkbenchTheme(): PmtThemeName {
  const el = document.documentElement;
  const appearance: ColorAppearance = el.classList.contains("dark")
    ? "dark"
    : "light";
  return workbenchThemeId(normalizeThemeFamily(el.dataset.theme), appearance);
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
    info: "#1a7a86", warning: "#9a7b16", success: "#2a8a4a", editorBg: "#f7f3e8",
    chrome: "#f4efe3", chrome2: "#ebe4d4", input: "#fffdf8", button: "#c9a227",
    buttonHover: "#b8911c", badgeFg: "#1b2030", remoteBg: "#1a9e74",
    remoteFg: "#f4efe3", inactive: "#7c4dbd",
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
  "monokai-dark": {
    fg: "#f8f8f2", accent: "#f92672", link: "#66d9ef", error: "#f92672",
    info: "#66d9ef", warning: "#e6db74", success: "#a6e22e", editorBg: "#272822",
    chrome: "#1e1f1c", chrome2: "#141511", input: "#3e3d32", button: "#f92672",
    buttonHover: "#fd971f", badgeFg: "#272822", remoteBg: "#a6e22e",
    remoteFg: "#272822", inactive: "#ae81ff",
  },
  "monokai-light": {
    fg: "#272822", accent: "#d01050", link: "#0088a8", error: "#d01050",
    info: "#0088a8", warning: "#a09020", success: "#5a8a10", editorBg: "#f8f8f2",
    chrome: "#eeeede", chrome2: "#e0e0d0", input: "#ffffff", button: "#d01050",
    buttonHover: "#c56a00", badgeFg: "#f8f8f2", remoteBg: "#5a8a10",
    remoteFg: "#f8f8f2", inactive: "#7b5ea7",
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
    badgeFg, remoteBg, remoteFg, inactive,
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
    "list.activeSelectionBackground": a(accent, "2e"),
    "list.activeSelectionForeground": fg,
    "list.inactiveSelectionBackground": a(inactive, "24"),
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
