/**
 * One page catalog: title, description, help, keepAlive, and toolbar pills.
 */
export type PageId =
  | "library"
  | "wizard"
  | "workspace-ide"
  | "event-graph"
  | "conflicts"
  | "loc-coverage"
  | "patcher"
  | "workspace-settings"
  | "tools-merge";

/** Route meta shared by the header, Help modal, toolbar, and KeepAlive. */
export type PageEntry = {
  title: string;
  description: string;
  help: string[];
  keepAlive?: boolean;
  /** beforeEach sets the active workspace from :id. */
  workspace?: boolean;
  /** Toolbar / Display visible-tools pill. */
  tool?: { label: string; icon: string };
};

declare module "vue-router" {
  interface RouteMeta extends PageEntry {}
}

/** Canonical copy and chrome for every named route. */
export const PAGE_CATALOG: Record<PageId, PageEntry> = {
  library: {
    title: "Library",
    description: "Browse and manage your modding workspaces.",
    keepAlive: true,
    help: [
      "This is your workspace library. A workspace is one game install plus the mods you are editing.",
      "Click a card to open that workspace’s default page. Edit opens Workspace Settings. Delete removes the workspace from PMT only — mod folders on disk stay.",
      "New Workspace starts the setup wizard. New Mod creates a skeleton in an existing workspace, or starts the wizard when the library is empty.",
      "Reset all data at the bottom wipes PMT workspaces, install records, caches, and patch history. It does not delete mods or Steam/game installs on disk. Theme, UI scale, and editor font are kept.",
    ],
  },
  wizard: {
    title: "Create Workspace",
    description: "Set up a new modding workspace.",
    help: [
      "Walk through game, install, mods, name, and staging.",
      "Add existing folders or Create new mod (descriptor, empty common/ and events/, readmes, localization stub with UTF-8 BOM).",
      "Drag mods to set load order (SortOrder). You can change all of this later in Workspace Settings.",
      "After create you land in the IDE so you can open the files you just attached.",
    ],
  },
  "workspace-ide": {
    title: "Workspace IDE",
    description: "View and edit files in your workspace.",
    workspace: true,
    tool: { label: "IDE", icon: "i-lucide-code" },
    help: [
      "Folders come from this workspace: your mods, Staging, then the game files. There is no Open Folder — PMT owns the folder set. Back returns to the Library, not the create-workspace wizard.",
      "Game files (Crusader Kings III, Victoria 3, or Europa Universalis V) are read-only. Edit in a mod folder. Search and Ctrl+P index every root. Deleting a mod’s root folder in the explorer also detaches it from this workspace; files inside a mod delete normally.",
      "Open editor tabs persist across workspace switch and app close unless you turn that off in Workspace Settings. Themes live in the PMT header, not in VS Code settings.",
      "The Guide pull-tab on the editor’s right edge opens a resizable wiki pane (search, contents, related pages). It stays closed until you open it. Snooze or turn off the toast under Display.",
    ],
  },
  "event-graph": {
    title: "Event Graph",
    description: "Browse event and on_action links in the workspace.",
    keepAlive: true,
    workspace: true,
    tool: { label: "Event Graph", icon: "i-lucide-git-fork" },
    help: [
      "Pick a root event to see what it fires and what fires it. Origins filter which mods (and game files) feed the graph.",
      "Namespace narrows the picker; leave it empty to search every id. Double-click a node to re-root. Drag is temporary until you change layout or root.",
      "Recenter only pans to fit — it does not undo a layout change.",
    ],
  },
  conflicts: {
    title: "Conflicts",
    description: "FIOS/LIOS overlapping definitions.",
    keepAlive: true,
    workspace: true,
    tool: { label: "Conflicts", icon: "i-lucide-layers" },
    help: [
      "This lists overlapping definitions (FIOS / LIOS). The winner is who actually loads in game. The left rail is workspace load order; last listed wins except first-wins kinds for that game.",
      "Use Conflicts vs game-file overrides to switch scope. Origin names match the IDE explorer.",
      "Open a row to jump to the file in the IDE.",
    ],
  },
  "loc-coverage": {
    title: "Loc Coverage",
    description: "Missing, orphaned, and untranslated localization keys.",
    keepAlive: true,
    workspace: true,
    tool: { label: "Loc Coverage", icon: "i-lucide-languages" },
    help: [
      "Missing keys are used in script but have no loc. Orphans are loc with no script use. Untranslated compares languages.",
      "Default loc language is a workspace setting. Coverage looks at every language the mods ship.",
    ],
  },
  patcher: {
    title: "Patch Center",
    description: "Patch notes, impact check, and file retarget.",
    keepAlive: true,
    workspace: true,
    tool: { label: "Patch Center", icon: "i-lucide-git-compare" },
    help: [
      "Three tabs: Patch Notes (wiki changelog), Impact Check (heuristic overlap with your mod), and Patcher (file retarget).",
      "Patch Notes is a cached wiki reader. Impact Check is incomplete — wiki bullets miss a lot. Patcher previews diffs in the workbench, then writes accepted files.",
      "This is not a full merge of two mods — use Ad-hoc Merge for that.",
    ],
  },
  "workspace-settings": {
    title: "Workspace Settings",
    description: "Configure this workspace.",
    workspace: true,
    help: [
      "Overview: name, default loc language, whether the IDE remembers open files, and which page Library should open.",
      "Game: pick or add an install and pin a version. Changing install rebuilds language intelligence. Mods: attach an existing folder or create a new skeleton, then color, reorder, or detach. Detach does not delete the folder on disk. Staging: where patched output lands.",
      "Remove this workspace deletes the PMT record only. Reset all data lives on the Library page. Appearance (scale, editor font, which tools show) lives in the header Display control, not here.",
    ],
  },
  "tools-merge": {
    title: "Ad-hoc Merge",
    description: "Merge two files or directories without a workspace.",
    keepAlive: true,
    help: [
      "Merge two files or folders without a workspace. Useful for one-off compares.",
      "This does not change your workspace mods. Results go where you choose.",
    ],
  },
};

/** Toolbar pills in catalog order. */
export const WORKSPACE_TOOLS = (
  Object.entries(PAGE_CATALOG) as [PageId, PageEntry][]
)
  .filter((entry): entry is [PageId, PageEntry & { tool: { label: string; icon: string } }] =>
    !!entry[1].tool,
  )
  .map(([name, page]) => ({
    name,
    label: page.tool.label,
    icon: page.tool.icon,
  }));

/** Route name of a workspace tool page. */
export type WorkspaceToolName = (typeof WORKSPACE_TOOLS)[number]["name"];

/** All workspace tool route names, in toolbar order. */
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
