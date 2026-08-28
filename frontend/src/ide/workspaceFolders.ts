/**
 * Resolve multi-root IDE folders (game / mods / staging) from workspace state.
 */
import { GetScriptRoot, ListGameInstalls } from "@services/workspaceservice";
import type { Workspace, WorkspaceMod } from "@services/internal/repos/models";
import type { IdeRoot } from "./fsBridge";

/** Build IDE roots for the active workspace. */
export async function buildIdeRoots(
  workspace: Workspace | null,
  mods: WorkspaceMod[],
): Promise<IdeRoot[]> {
  const roots: IdeRoot[] = [];
  if (workspace?.installId) {
    try {
      const scriptRoot = await GetScriptRoot(workspace.installId);
      if (scriptRoot) {
        roots.push({
          label: "Game",
          path: scriptRoot,
          readOnly: true,
          kind: "game",
        });
      } else if (workspace.gameId) {
        const installs = (await ListGameInstalls(workspace.gameId)) ?? [];
        const inst = installs.find((i) => i.id === workspace.installId);
        if (inst?.path) {
          roots.push({
            label: "Game",
            path: inst.path,
            readOnly: true,
            kind: "game",
          });
        }
      }
    } catch {
      /* missing install */
    }
  }
  for (const mod of mods) {
    if (!mod.path || mod.isBroken) continue;
    roots.push({
      label: mod.name || "Mod",
      path: mod.path,
      readOnly: false,
      kind: "mod",
    });
  }
  if (workspace?.stagingDir) {
    roots.push({
      label: "Staging",
      path: workspace.stagingDir,
      readOnly: false,
      kind: "staging",
    });
  }
  return roots;
}

/** Serialize roots into a VS Code multi-root workspace file body. */
export function workspaceFileJson(roots: IdeRoot[]): string {
  return JSON.stringify(
    {
      folders: roots.map((r) => ({
        name: r.label,
        // Absolute OS paths; forward slashes are accepted on Windows by VS Code.
        path: r.path.replace(/\\/g, "/"),
      })),
    },
    null,
    2,
  );
}
