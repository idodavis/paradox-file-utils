/**
 * Force Nuxt theme colors onto the monaco-vscode workbench (CSS + settings).
 */
import { updateUserConfiguration } from "@codingame/monaco-vscode-configuration-service-override";
import * as vscode from "vscode";
import { THEME_SWATCHES } from "../composables/themeSwatches";

const DARK_THEMES = new Set([
  "PMT",
  "dracula",
  "luxury",
  "business",
  "coffee",
  "dim",
]);

/** Resolved Nuxt palette as hex colors. */
export type ThemePalette = {
  primary: string;
  primaryFg: string;
  secondary: string;
  accent: string;
  base100: string;
  base200: string;
  base300: string;
  fg: string;
  editorBg: string;
  dark: boolean;
};

/** Convert CSS color (oklch/hsl/hex/rgb) to #rrggbb. */
function toHexColor(input: string, fallback: string): string {
  const s = input.trim();
  if (!s) return fallback;
  if (/^#[0-9a-fA-F]{6}$/.test(s)) return s.toLowerCase();
  if (/^#[0-9a-fA-F]{8}$/.test(s)) return s.slice(0, 7).toLowerCase();
  if (/^#[0-9a-fA-F]{3}$/.test(s)) {
    return `#${s[1]}${s[1]}${s[2]}${s[2]}${s[3]}${s[3]}`.toLowerCase();
  }
  if (typeof document === "undefined") return fallback;
  const el = document.createElement("span");
  el.style.color = s;
  document.body.appendChild(el);
  const rgb = getComputedStyle(el).color;
  el.remove();
  const m = rgb.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
  if (!m) return fallback;
  const hex = (n: string) => Number(n).toString(16).padStart(2, "0");
  return `#${hex(m[1]!)}${hex(m[2]!)}${hex(m[3]!)}`;
}

function cssRaw(name: string): string {
  if (typeof document === "undefined") return "";
  return getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim();
}

function pickColor(...candidates: string[]): string {
  for (const c of candidates) {
    if (!c) continue;
    const hex = toHexColor(c, "");
    if (hex) return hex;
  }
  return "#888888";
}

function withAlpha(hex: string, alpha: string): string {
  const h = hex.startsWith("#") ? hex.slice(1, 7) : hex.slice(0, 6);
  return `#${h}${alpha}`;
}

/** Resolve the active Nuxt theme into a concrete hex palette. */
export function resolveThemePalette(themeName: string): ThemePalette {
  const dark =
    DARK_THEMES.has(themeName) ||
    document.documentElement.classList.contains("dark");
  const sw = THEME_SWATCHES[themeName] ?? THEME_SWATCHES.PMT;
  const [swPrimary, swSecondary, swAccent, swBase] = sw;

  return {
    dark,
    primary: pickColor(
      cssRaw("--pmt-primary"),
      cssRaw("--color-primary"),
      swPrimary,
      dark ? "#c9b458" : "#2563eb",
    ),
    primaryFg: pickColor(
      cssRaw("--pmt-primary-content"),
      cssRaw("--color-primary-content"),
      dark ? "#111111" : "#ffffff",
    ),
    secondary: pickColor(
      cssRaw("--pmt-secondary"),
      cssRaw("--color-secondary"),
      swSecondary,
      dark ? "#5ec4a8" : "#7c3aed",
    ),
    accent: pickColor(
      cssRaw("--pmt-accent"),
      cssRaw("--color-accent"),
      swAccent,
      dark ? "#a78bfa" : "#db2777",
    ),
    base100: pickColor(
      cssRaw("--pmt-base-100"),
      cssRaw("--color-base-100"),
      swBase,
      dark ? "#1c2030" : "#ffffff",
    ),
    base200: pickColor(
      cssRaw("--pmt-base-200"),
      cssRaw("--color-base-200"),
      dark ? "#161a24" : "#f3f4f6",
    ),
    base300: pickColor(
      cssRaw("--pmt-base-300"),
      cssRaw("--color-base-300"),
      dark ? "#10131a" : "#e5e7eb",
    ),
    fg: pickColor(
      cssRaw("--pmt-base-content"),
      cssRaw("--color-base-content"),
      dark ? "#e8eaed" : "#1f2937",
    ),
    editorBg: pickColor(
      cssRaw("--pmt-editor-bg"),
      cssRaw("--color-dark-input"),
      dark ? "#12151c" : "#ffffff",
    ),
  };
}

/** Map palette → VS Code settings.json fragment. */
export function colorCustomizationsForTheme(
  themeName: string,
  editorFontSize: number,
): Record<string, unknown> {
  const p = resolveThemePalette(themeName);
  const themeId = p.dark ? "Default Dark Modern" : "Default Light Modern";
  const colors: Record<string, string> = {
    foreground: p.fg,
    focusBorder: p.primary,
    "activityBar.background": p.base200,
    "activityBar.foreground": p.fg,
    "activityBar.inactiveForeground": withAlpha(p.fg, "99"),
    "activityBar.border": p.base300,
    "activityBarBadge.background": p.primary,
    "activityBarBadge.foreground": p.primaryFg,
    "sideBar.background": p.base200,
    "sideBar.foreground": p.fg,
    "sideBar.border": p.base300,
    "sideBarTitle.foreground": p.fg,
    "sideBarSectionHeader.background": p.base300,
    "sideBarSectionHeader.foreground": p.fg,
    "editor.background": p.editorBg,
    "editor.foreground": p.fg,
    "editor.selectionBackground": withAlpha(p.primary, "55"),
    "editorGroupHeader.tabsBackground": p.base200,
    "editorGroup.border": p.base300,
    "tab.activeBackground": p.editorBg,
    "tab.activeForeground": p.fg,
    "tab.activeBorderTop": p.primary,
    "tab.inactiveBackground": p.base200,
    "tab.inactiveForeground": withAlpha(p.fg, "99"),
    "tab.border": p.base300,
    "panel.background": p.base200,
    "panel.border": p.base300,
    "panelTitle.activeBorder": p.primary,
    "statusBar.background": p.base100,
    "statusBar.foreground": p.fg,
    "statusBar.border": p.base300,
    "statusBar.noFolderBackground": p.base100,
    "statusBarItem.remoteBackground": p.secondary,
    "titleBar.activeBackground": p.base100,
    "titleBar.activeForeground": p.fg,
    "titleBar.inactiveBackground": p.base200,
    "titleBar.border": p.base300,
    "input.background": p.base300,
    "input.foreground": p.fg,
    "input.border": p.base300,
    "dropdown.background": p.base300,
    "dropdown.foreground": p.fg,
    "button.background": p.primary,
    "button.foreground": p.primaryFg,
    "button.hoverBackground": p.secondary,
    "list.activeSelectionBackground": withAlpha(p.primary, "66"),
    "list.activeSelectionForeground": p.fg,
    "list.focusOutline": p.primary,
    "list.inactiveSelectionBackground": withAlpha(p.secondary, "44"),
    "list.hoverBackground": withAlpha(p.accent, "33"),
    "list.highlightForeground": p.primary,
    "badge.background": p.primary,
    "badge.foreground": p.primaryFg,
    "progressBar.background": p.primary,
    "scrollbarSlider.background": withAlpha(p.fg, "33"),
    "scrollbarSlider.hoverBackground": withAlpha(p.fg, "55"),
  };

  return {
    "workbench.colorTheme": themeId,
    "workbench.iconTheme": "pmt-icons",
    "workbench.tree.indent": 14,
    "workbench.tree.renderIndentGuides": "always",
    "editor.fontSize": editorFontSize,
    "editor.fontFamily":
      "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
    "workbench.colorCustomizations": {
      [`[${themeId}]`]: colors,
      ...colors,
    },
  };
}

/** Paint VS Code CSS variables directly on the workbench host (always wins). */
export function paintWorkbenchCssVars(
  host: HTMLElement | null,
  themeName: string,
): void {
  if (!host) return;
  const p = resolveThemePalette(themeName);
  const set = (name: string, value: string) =>
    host.style.setProperty(name, value);

  set("--vscode-foreground", p.fg);
  set("--vscode-focusBorder", p.primary);
  set("--vscode-activityBar-background", p.base200);
  set("--vscode-activityBar-foreground", p.fg);
  set("--vscode-activityBarBadge-background", p.primary);
  set("--vscode-activityBarBadge-foreground", p.primaryFg);
  set("--vscode-sideBar-background", p.base200);
  set("--vscode-sideBar-foreground", p.fg);
  set("--vscode-sideBar-border", p.base300);
  set("--vscode-sideBarTitle-foreground", p.fg);
  set("--vscode-sideBarSectionHeader-background", p.base300);
  set("--vscode-editor-background", p.editorBg);
  set("--vscode-editor-foreground", p.fg);
  set("--vscode-editor-selectionBackground", withAlpha(p.primary, "55"));
  set("--vscode-editorGroupHeader-tabsBackground", p.base200);
  set("--vscode-tab-activeBackground", p.editorBg);
  set("--vscode-tab-activeForeground", p.fg);
  set("--vscode-tab-activeBorderTop", p.primary);
  set("--vscode-tab-inactiveBackground", p.base200);
  set("--vscode-panel-background", p.base200);
  set("--vscode-statusBar-background", p.base100);
  set("--vscode-statusBar-foreground", p.fg);
  set("--vscode-statusBar-noFolderBackground", p.base100);
  set("--vscode-titleBar-activeBackground", p.base100);
  set("--vscode-titleBar-activeForeground", p.fg);
  set("--vscode-input-background", p.base300);
  set("--vscode-input-foreground", p.fg);
  set("--vscode-button-background", p.primary);
  set("--vscode-button-foreground", p.primaryFg);
  set("--vscode-button-hoverBackground", p.secondary);
  set("--vscode-list-activeSelectionBackground", withAlpha(p.primary, "66"));
  set("--vscode-list-activeSelectionForeground", p.fg);
  set("--vscode-list-inactiveSelectionBackground", withAlpha(p.secondary, "44"));
  set("--vscode-list-hoverBackground", withAlpha(p.accent, "33"));
  set("--vscode-list-focusOutline", p.primary);
  set("--vscode-list-highlightForeground", p.primary);
  set("--vscode-badge-background", p.primary);
  set("--vscode-badge-foreground", p.primaryFg);
  set("--vscode-progressBar-background", p.primary);

  // Comfortable explorer row height (default feels compressed without icons).
  if (!host.dataset.pmtExplorerCss) {
    host.dataset.pmtExplorerCss = "1";
    const style = document.createElement("style");
    style.textContent = `
      .monaco-workbench .monaco-list .monaco-list-row {
        min-height: 24px !important;
      }
      .monaco-workbench .monaco-tl-row {
        min-height: 24px !important;
        align-items: center;
      }
      .monaco-workbench .monaco-icon-label {
        align-items: center;
      }
      .monaco-workbench .explorer-folders-view .monaco-tl-row {
        padding-top: 1px;
        padding-bottom: 1px;
      }
    `;
    host.appendChild(style);
  }
}

let applyingTheme = false;
let themeHost: HTMLElement | null = null;

/** Remember the workbench host element for CSS variable painting. */
export function setThemeHost(el: HTMLElement | null): void {
  themeHost = el;
}

/** Push theme into settings.json and CSS variables on the host. */
export async function applyWorkbenchTheme(
  themeName: string,
  editorFontSize: number,
): Promise<void> {
  if (applyingTheme) return;
  applyingTheme = true;
  try {
    paintWorkbenchCssVars(themeHost, themeName);
    const cfg = colorCustomizationsForTheme(themeName, editorFontSize);
    await updateUserConfiguration(JSON.stringify(cfg, null, 2));
  } finally {
    applyingTheme = false;
  }
}

/**
 * Re-apply the Nuxt theme whenever the workbench theme settings change.
 */
export function lockWorkbenchTheme(
  getOpts: () => { themeName: string; editorFontSize: number },
): vscode.Disposable {
  return vscode.workspace.onDidChangeConfiguration((e) => {
    if (applyingTheme) return;
    if (
      !e.affectsConfiguration("workbench.colorTheme") &&
      !e.affectsConfiguration("workbench.colorCustomizations") &&
      !e.affectsConfiguration("workbench.iconTheme") &&
      !e.affectsConfiguration("editor.fontSize")
    ) {
      return;
    }
    const { themeName, editorFontSize } = getOpts();
    void applyWorkbenchTheme(themeName, editorFontSize);
  });
}
