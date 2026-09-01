/** Workspace tool routes for the toolbar, Display popover, and default page. */
export const WORKSPACE_TOOLS = [
  { label: "IDE", icon: "i-lucide-code", name: "workspace-ide" },
  { label: "Event Graph", icon: "i-lucide-git-fork", name: "event-graph" },
  { label: "Conflicts", icon: "i-lucide-layers", name: "conflicts" },
  { label: "Loc Coverage", icon: "i-lucide-languages", name: "loc-coverage" },
  { label: "Mod Patcher", icon: "i-lucide-git-compare", name: "patcher" },
] as const;

/** Route name of a workspace tool page. */
export type WorkspaceToolName = (typeof WORKSPACE_TOOLS)[number]["name"];

/** All five workspace tool route names, in toolbar order. */
export const WORKSPACE_TOOL_NAMES: WorkspaceToolName[] = WORKSPACE_TOOLS.map(
  (t) => t.name,
);

/** Landing route for a workspace, honoring DefaultTool and visible-tool filter. */
export function workspaceHomeRoute(
  ws: { id: string; defaultTool?: string },
  visible: readonly string[],
): { name: string; params: { id: string } } {
  const allowed = new Set<string>(WORKSPACE_TOOL_NAMES);
  const shown = visible.filter((n) => allowed.has(n));
  if (!shown.length) {
    return { name: "workspace-ide", params: { id: ws.id } };
  }
  const want = ws.defaultTool || "workspace-ide";
  const name = shown.includes(want) ? want : shown[0]!;
  return { name, params: { id: ws.id } };
}
