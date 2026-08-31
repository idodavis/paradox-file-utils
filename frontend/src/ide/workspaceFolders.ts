/**
 * Resolve multi-root IDE folders (game / mods / staging) from WorkspaceService.
 */
import { GetIdeRoots } from "@services/workspaceservice";
import type { IdeRoot } from "./fsBridge";

/** Build IDE roots for the active workspace. */
export async function buildIdeRoots(workspaceId: string): Promise<IdeRoot[]> {
  if (!workspaceId) return [];
  return ((await GetIdeRoots(workspaceId)) ?? []) as IdeRoot[];
}

/** Serialize roots into a VS Code multi-root workspace file body. */
export function workspaceFileJson(roots: IdeRoot[]): string {
  return JSON.stringify(
    {
      folders: roots.map((r) => ({
        name: r.label,
        path: r.path.replace(/\\/g, "/"),
      })),
    },
    null,
    2,
  );
}
