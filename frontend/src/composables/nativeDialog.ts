/**
 * Native Wails v3 file/folder dialogs used by the wizard and settings.
 */
import { Dialogs } from "@wailsio/runtime";

/** Open a folder picker. Cancel yields an empty path. */
export async function pickDirectory(title: string): Promise<string> {
  const path = await Dialogs.OpenFile({
    Title: title,
    CanChooseDirectories: true,
    CanChooseFiles: false,
  });
  return typeof path === "string" ? path : "";
}

/** Open a file picker. Cancel yields an empty path. */
export async function pickFile(title: string, filter: string): Promise<string> {
  const path = await Dialogs.OpenFile({
    Title: title,
    CanChooseDirectories: false,
    CanChooseFiles: true,
    Filters: filter ? [{ DisplayName: filter, Pattern: filter }] : undefined,
  });
  return typeof path === "string" ? path : "";
}
