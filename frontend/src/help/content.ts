/**
 * First-time-modder Help copy keyed by Vue route name.
 * Shown in the app footer Help modal. Not a shipping doc site.
 */

export type HelpCopy = {
  paragraphs: string[];
};

/** Help body for each named route. */
export const HELP_COPY: Record<string, HelpCopy> = {
  library: {
    paragraphs: [
      "This is your workspace library. A workspace is one game install plus the mods you are editing.",
      "Click a card to open that workspace’s default page. Edit opens Workspace Settings. Delete removes the workspace from PMT only — mod folders on disk stay.",
      "New Workspace starts the setup wizard. New Mod creates a skeleton in an existing workspace, or starts the wizard when the library is empty.",
      "Reset all data at the bottom wipes PMT workspaces, install records, caches, and patch history. It does not delete mods or Steam/game installs on disk. Theme, UI scale, and editor font are kept.",
    ],
  },
  wizard: {
    paragraphs: [
      "Walk through game, install, mods, name, and staging. Tags are PMT labels only — they are not written into .mod or metadata.json.",
      "Add existing folders or Create new mod (descriptor, empty common/ and events/, readmes, localization stub with UTF-8 BOM). PMT tags are not written into the descriptor.",
      "Drag mods to set load order (SortOrder). You can change all of this later in Workspace Settings.",
      "After create you land in the IDE so you can open the files you just attached.",
    ],
  },
  "workspace-ide": {
    paragraphs: [
      "Folders come from this workspace: your mods, Staging, then the game files. There is no Open Folder — PMT owns the folder set. Back returns to the Library, not the create-workspace wizard.",
      "Game files (Crusader Kings III, Victoria 3, or Europa Universalis V) are read-only. Edit in a mod folder. Search and Ctrl+P index every root. Deleting a mod’s root folder in the explorer also detaches it from this workspace; files inside a mod delete normally.",
      "Open editor tabs persist across workspace switch and app close unless you turn that off in Workspace Settings. Themes live in the PMT header, not in VS Code settings.",
    ],
  },
  "event-graph": {
    paragraphs: [
      "Pick a root event to see what it fires and what fires it. Origins filter which mods (and game files) feed the graph.",
      "Namespace narrows the picker; leave it empty to search every id. Double-click a node to re-root. Drag is temporary until you change layout or root.",
      "Recenter only pans to fit — it does not undo a layout change.",
    ],
  },
  conflicts: {
    paragraphs: [
      "This lists overlapping definitions (FIOS / LIOS). The winner is who actually loads in game.",
      "Use Conflicts vs game-file overrides to switch scope. Origin names match the IDE explorer.",
      "Open a row to jump to the file in the IDE.",
    ],
  },
  "loc-coverage": {
    paragraphs: [
      "Missing keys are used in script but have no loc. Orphans are loc with no script use. Untranslated compares languages.",
      "Default loc language is a workspace setting. Coverage looks at every language the mods ship.",
    ],
  },
  patcher: {
    paragraphs: [
      "Patch retargets a mod onto a newer (or different) game version. Review diffs in the workbench, then write to staging.",
      "This is not a full merge of two mods — use Ad-hoc Merge for that.",
    ],
  },
  "tools-merge": {
    paragraphs: [
      "Merge two files or folders without a workspace. Useful for one-off compares.",
      "This does not change your workspace mods. Results go where you choose.",
    ],
  },
  "workspace-settings": {
    paragraphs: [
      "Overview: name, tags, default loc language, whether the IDE remembers open files, and which page Library should open.",
      "Game: pick or add an install and pin a version. Changing install rebuilds language intelligence. Mods: attach an existing folder or create a new skeleton, then tag, color, reorder, or detach. Detach does not delete the folder on disk. Staging: where patched output lands.",
      "Remove this workspace deletes the PMT record only. Reset all data lives on the Library page. Appearance (scale, editor font, which tools show) lives in the header Display control, not here.",
    ],
  },
};
