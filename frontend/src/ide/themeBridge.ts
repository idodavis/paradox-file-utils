/**
 * Bridge Nuxt/PMT CSS tokens onto the monaco-vscode workbench.
 *
 * Uses live `var(--pmt-*)` references (no oklch→hex conversion). A stylesheet
 * with `!important` keeps chrome tints subtle and resistant to theme resets.
 */
import { updateUserConfiguration } from "@codingame/monaco-vscode-configuration-service-override";
import * as vscode from "vscode";

const DARK_THEMES = new Set([
  "PMT",
  "dracula",
  "luxury",
  "business",
  "coffee",
  "dim",
]);

const STYLE_ID = "pmt-workbench-theme-bridge";

/** Whether the named Nuxt theme is dark. */
export function isDarkTheme(themeName: string): boolean {
  return (
    DARK_THEMES.has(themeName) ||
    document.documentElement.classList.contains("dark")
  );
}

/** VS Code color theme id for the active Nuxt theme. */
export function vscodeColorThemeId(themeName: string): string {
  return isDarkTheme(themeName) ? "Default Dark Modern" : "Default Light Modern";
}

/**
 * Inject (or refresh) workbench CSS that maps `--vscode-*` to Nuxt tokens.
 * Token values update automatically when `data-theme` changes on `:root`.
 */
export function paintWorkbenchCssVars(): void {
  let style = document.getElementById(STYLE_ID) as HTMLStyleElement | null;
  if (!style) {
    style = document.createElement("style");
    style.id = STYLE_ID;
    document.head.appendChild(style);
  }

  // Subtle accents only — keep Default Dark/Light Modern surfaces intact.
  style.textContent = `
    .monaco-workbench {
      --vscode-focusBorder: color-mix(in oklab, var(--pmt-primary) 50%, transparent) !important;
      --vscode-activityBarBadge-background: var(--pmt-primary) !important;
      --vscode-activityBarBadge-foreground: var(--pmt-primary-content) !important;
      --vscode-sideBar-background: var(--pmt-base-200) !important;
      --vscode-sideBar-foreground: var(--pmt-base-content) !important;
      --vscode-sideBar-border: color-mix(in oklab, var(--pmt-base-content) 12%, transparent) !important;
      --vscode-sideBarTitle-foreground: var(--pmt-base-content) !important;
      --vscode-sideBarSectionHeader-background: color-mix(in oklab, var(--pmt-base-300) 70%, transparent) !important;
      --vscode-sideBarSectionHeader-foreground: var(--pmt-base-content) !important;
      --vscode-editor-background: var(--pmt-editor-bg) !important;
      --vscode-editor-foreground: var(--pmt-base-content) !important;
      --vscode-editor-selectionBackground: color-mix(in oklab, var(--pmt-primary) 28%, transparent) !important;
      --vscode-editorGroupHeader-tabsBackground: var(--pmt-base-200) !important;
      --vscode-editorGroup-border: color-mix(in oklab, var(--pmt-base-content) 12%, transparent) !important;
      --vscode-tab-activeBackground: var(--pmt-editor-bg) !important;
      --vscode-tab-activeForeground: var(--pmt-base-content) !important;
      --vscode-tab-activeBorderTop: var(--pmt-primary) !important;
      --vscode-tab-inactiveBackground: var(--pmt-base-200) !important;
      --vscode-tab-inactiveForeground: color-mix(in oklab, var(--pmt-base-content) 62%, transparent) !important;
      --vscode-tab-border: color-mix(in oklab, var(--pmt-base-content) 10%, transparent) !important;
      --vscode-panel-background: var(--pmt-base-200) !important;
      --vscode-panel-border: color-mix(in oklab, var(--pmt-base-content) 12%, transparent) !important;
      --vscode-panelTitle-activeBorder: var(--pmt-primary) !important;
      --vscode-statusBar-background: var(--pmt-base-100) !important;
      --vscode-statusBar-foreground: var(--pmt-base-content) !important;
      --vscode-statusBar-border: color-mix(in oklab, var(--pmt-base-content) 12%, transparent) !important;
      --vscode-statusBar-noFolderBackground: var(--pmt-base-100) !important;
      --vscode-statusBarItem-remoteBackground: color-mix(in oklab, var(--pmt-secondary) 55%, var(--pmt-base-100)) !important;
      --vscode-activityBar-background: var(--pmt-base-200) !important;
      --vscode-activityBar-foreground: var(--pmt-base-content) !important;
      --vscode-activityBar-inactiveForeground: color-mix(in oklab, var(--pmt-base-content) 55%, transparent) !important;
      --vscode-activityBar-border: color-mix(in oklab, var(--pmt-base-content) 12%, transparent) !important;
      --vscode-titleBar-activeBackground: var(--pmt-base-100) !important;
      --vscode-titleBar-activeForeground: var(--pmt-base-content) !important;
      --vscode-titleBar-inactiveBackground: var(--pmt-base-200) !important;
      --vscode-titleBar-border: color-mix(in oklab, var(--pmt-base-content) 12%, transparent) !important;
      --vscode-input-background: var(--pmt-base-300) !important;
      --vscode-input-foreground: var(--pmt-base-content) !important;
      --vscode-input-border: color-mix(in oklab, var(--pmt-base-content) 16%, transparent) !important;
      --vscode-dropdown-background: var(--pmt-base-300) !important;
      --vscode-dropdown-foreground: var(--pmt-base-content) !important;
      --vscode-button-background: color-mix(in oklab, var(--pmt-primary) 82%, var(--pmt-base-100)) !important;
      --vscode-button-foreground: var(--pmt-primary-content) !important;
      --vscode-button-hoverBackground: color-mix(in oklab, var(--pmt-primary) 70%, var(--pmt-secondary)) !important;
      --vscode-list-activeSelectionBackground: color-mix(in oklab, var(--pmt-primary) 18%, transparent) !important;
      --vscode-list-activeSelectionForeground: var(--pmt-base-content) !important;
      --vscode-list-inactiveSelectionBackground: color-mix(in oklab, var(--pmt-secondary) 14%, transparent) !important;
      --vscode-list-hoverBackground: color-mix(in oklab, var(--pmt-base-content) 7%, transparent) !important;
      --vscode-list-focusOutline: color-mix(in oklab, var(--pmt-primary) 45%, transparent) !important;
      --vscode-list-highlightForeground: var(--pmt-primary) !important;
      --vscode-badge-background: color-mix(in oklab, var(--pmt-primary) 75%, var(--pmt-base-100)) !important;
      --vscode-badge-foreground: var(--pmt-primary-content) !important;
      --vscode-progressBar-background: var(--pmt-primary) !important;
      --vscode-scrollbarSlider-background: color-mix(in oklab, var(--pmt-base-content) 18%, transparent) !important;
      --vscode-scrollbarSlider-hoverBackground: color-mix(in oklab, var(--pmt-base-content) 28%, transparent) !important;
    }

    /* Explorer readability: slightly larger type and roomier rows. */
    .monaco-workbench .explorer-folders-view .monaco-list-row,
    .monaco-workbench .monaco-list .monaco-list-row {
      font-size: 13px !important;
    }
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
}

/** User-settings fragment: base theme + icon theme + tree comfort (no editor font). */
export function workbenchSettingsForTheme(
  themeName: string,
): Record<string, unknown> {
  return {
    "workbench.colorTheme": vscodeColorThemeId(themeName),
    "workbench.iconTheme": "pmt-icons",
    "workbench.tree.indent": 16,
    "workbench.tree.renderIndentGuides": "always",
  };
}

let applyingTheme = false;

/** Push dark/light base theme and ensure CSS bridge is present. */
export async function applyWorkbenchTheme(themeName: string): Promise<void> {
  if (applyingTheme) return;
  applyingTheme = true;
  try {
    paintWorkbenchCssVars();
    await updateUserConfiguration(
      JSON.stringify(
        {
          ...workbenchSettingsForTheme(themeName),
          "workbench.startupEditor": "none",
          "window.title": "PMT${separator}${activeEditorShort}",
          "files.autoSave": "off",
        },
        null,
        2,
      ),
    );
  } finally {
    applyingTheme = false;
  }
}

/**
 * Re-apply the Nuxt↔workbench bridge when the user changes workbench theme settings.
 */
export function lockWorkbenchTheme(
  getThemeName: () => string,
): vscode.Disposable {
  return vscode.workspace.onDidChangeConfiguration((e) => {
    if (applyingTheme) return;
    if (
      !e.affectsConfiguration("workbench.colorTheme") &&
      !e.affectsConfiguration("workbench.iconTheme")
    ) {
      return;
    }
    void applyWorkbenchTheme(getThemeName());
  });
}
