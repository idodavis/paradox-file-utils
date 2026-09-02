/**
 * Sync the monaco-vscode workbench color/icon theme with `{family}-{dark|light}`.
 *
 * Colors come from contributed theme JSON (hex). This module only writes
 * `workbench.colorTheme` and related settings — no CSS variable painting.
 */
import { updateUserConfiguration } from "@codingame/monaco-vscode-configuration-service-override";
import * as vscode from "vscode";

/** Bracket match / auto-close defaults (also in workbench construction). */
export const EDITOR_BRACKET_DEFAULTS: Record<string, unknown> = {
  "editor.matchBrackets": "always",
  "editor.bracketPairColorization.enabled": true,
  "editor.guides.bracketPairs": "active",
  "editor.autoClosingBrackets": "languageDefined",
};

/** User-settings fragment: named theme + icon theme + tree indent. */
export function workbenchSettingsForTheme(
  themeName: string,
): Record<string, unknown> {
  return {
    "workbench.colorTheme": themeName,
    "workbench.iconTheme": "pmt-icons",
    "workbench.tree.indent": 16,
    "workbench.tree.renderIndentGuides": "always",
    "editor.semanticHighlighting.enabled": false,
    ...EDITOR_BRACKET_DEFAULTS,
  };
}

let applyingTheme = false;
let lastThemeName = "";
let queuedTheme: string | null = null;
let editorFontSize = 14;

/** Merge editor.fontSize into the next workbench settings write. */
export function applyEditorFontSize(px: number): void {
  editorFontSize = Math.min(24, Math.max(10, Math.round(px)));
  if (lastThemeName) void applyWorkbenchTheme(lastThemeName);
}

/** Push workbench.colorTheme / iconTheme / tree indent from the Nuxt name. */
export async function applyWorkbenchTheme(themeName: string): Promise<void> {
  if (applyingTheme) {
    queuedTheme = themeName;
    return;
  }
  applyingTheme = true;
  lastThemeName = themeName;
  try {
    do {
      const name = queuedTheme ?? lastThemeName;
      queuedTheme = null;
      lastThemeName = name;
      await updateUserConfiguration(
        JSON.stringify(
          {
            ...workbenchSettingsForTheme(name),
            "editor.fontSize": editorFontSize,
            "workbench.startupEditor": "none",
            "window.title": "PMT${separator}${activeEditorShort}",
            "files.autoSave": "off",
          },
          null,
          2,
        ),
      );
    } while (queuedTheme);
  } finally {
    applyingTheme = false;
  }
}

/**
 * Re-apply the Nuxt-owned color/icon theme if the workbench picker drifts.
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
