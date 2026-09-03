/**
 * Native Wails v3 file/folder dialogs for browse buttons in forms.
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

/** Open a save-file picker. Cancel yields an empty path. */
export async function pickSave(title: string, filter: string): Promise<string> {
  const dialogs = Dialogs as typeof Dialogs & {
    SaveFile?: (opts: {
      Title: string;
      Filters?: { DisplayName: string; Pattern: string }[];
    }) => Promise<unknown>;
  };
  if (typeof dialogs.SaveFile !== "function") {
    return pickFile(title, filter);
  }
  const path = await dialogs.SaveFile({
    Title: title,
    Filters: filter ? [{ DisplayName: filter, Pattern: filter }] : undefined,
  });
  return typeof path === "string" ? path : "";
}
