/**
 * Sync the monaco-vscode workbench color/icon theme with the Nuxt theme name.
 *
 * Colors come from contributed theme JSON (hex). This module only writes
 * `workbench.colorTheme` and related settings — no CSS variable painting.
 */
import { updateUserConfiguration } from "@codingame/monaco-vscode-configuration-service-override";
import * as vscode from "vscode";
import { isDarkTheme } from "./appThemes";

export { isDarkTheme };

/** User-settings fragment: named theme + icon theme + tree indent. */
export function workbenchSettingsForTheme(
  themeName: string,
): Record<string, unknown> {
  return {
    "workbench.colorTheme": themeName,
    "workbench.iconTheme": "pmt-icons",
    "workbench.tree.indent": 16,
    "workbench.tree.renderIndentGuides": "always",
  };
}

let applyingTheme = false;

/** Push workbench.colorTheme / iconTheme / tree indent from the Nuxt name. */
export async function applyWorkbenchTheme(themeName: string): Promise<void> {
  if (applyingTheme) return;
  applyingTheme = true;
  try {
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
