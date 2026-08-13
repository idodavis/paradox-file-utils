<script setup lang="ts">
/**
 * Workspace IDE (Stage 1): Nuxt UI dashboard shell, multi-root UTree,
 * Pinia file/compare tabs, Pierre EditorView, mod/staging file CRUD, explorer search.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useRoute, useRouter } from "vue-router";
import type { CodeViewItem } from "@pierre/diffs";
import { parseDiffFromFile } from "@pierre/diffs";
import type { ContextMenuItem } from "@nuxt/ui";
import { Events } from "@wailsio/runtime";
import EditorView from "../components/EditorView.vue";
import GuiPreviewPanel from "../components/GuiPreviewPanel.vue";
import { EDITOR_THEME_OPTIONS, setEditorTheme, editorTheme } from "../composables/editorTheme";
import { langForPath } from "../composables/langForPath";
import {
  type FileRoot,
  type IdeTreeItem,
  dirEntriesToTreeItems,
  findTreeItem,
  rootsToTreeItems,
  setTreeChildren,
} from "../composables/treeItems";
import { useIdeStore, type IdeTab } from "../stores/ide";
import { useSettingsStore } from "../stores/settings";
import { useWorkspaceStore } from "../stores/workspace";
import {
  GetWorkspace,
  ListWorkspaceMods,
  GetScriptRoot,
  EnsureStagingDir,
} from "@services/workspaceservice";
import { Workspace, WorkspaceMod } from "@services/internal/repos/models";
import {
  ListDirectory,
  ReadFileContent,
  WriteFileContent,
  CreateFile,
  CreateDir,
  RenamePath,
  DeletePath,
  RevealInOs,
  SearchByName,
  SearchInFiles,
} from "@services/fileservice";
import { CopyToClipboard } from "@services/clipboardservice";
import {
  DirEntry,
  LocDiagnostic,
  GuiNode,
  AutoLocResult,
  SemanticsStatus,
  FileSearchHit,
  ContentSearchHit,
} from "@services/models";
import { LintMissingLoc, AutoLocalize } from "@services/locservice";
import {
  RebuildWorkspaceSemantics,
  CancelSemantics,
  GetSemanticsStatus,
} from "@services/semanticsservice";
import { PreviewGui } from "@services/guiservice";
import { ReindexWorkspace, CancelIndex, CountIndexObjects } from "@services/indexerservice";

/** Modal flows for explorer / tab hygiene prompts. */
type PromptKind =
  | "new-file"
  | "new-folder"
  | "rename"
  | "move"
  | "delete"
  | "discard"
  | "close-all";

/** Explorer search mode. */
type SearchMode = "files" | "content";

const route = useRoute();
const router = useRouter();
const ws = useWorkspaceStore();
const ide = useIdeStore();
const settings = useSettingsStore();
const { tabs, activeTabId, activeTab, activeFileTab } = storeToRefs(ide);

const workspace = ref<Workspace | null>(null);
const mods = ref<WorkspaceMod[]>([]);
const loading = ref(false);
const busy = ref("");
const progressLabel = ref("");
const progressPercent = ref(0);
const jobActive = ref(false);
const error = ref("");
const gameScriptRoot = ref("");
const showLocPanel = ref(false);
const locDiags = ref<LocDiagnostic[]>([]);
const guiRoots = ref<GuiNode[]>([]);
const selectedGuiName = ref<string | null>(null);
const semantics = ref<SemanticsStatus | null>(null);
const hadIndex = ref(false);
const autoLocOpen = ref(false);
const autoLocResult = ref<AutoLocResult | null>(null);

const fileRoots = ref<FileRoot[]>([]);
const treeItems = ref<IdeTreeItem[]>([]);
const expandedKeys = ref<string[]>([]);
const selectedTreeItem = ref<IdeTreeItem | undefined>();
const contextTreeItem = ref<IdeTreeItem | null>(null);

const searchQuery = ref("");
const searchMode = ref<SearchMode>("files");
const searchHits = ref<(FileSearchHit | ContentSearchHit)[]>([]);
const searchBusy = ref(false);
let searchTimer: ReturnType<typeof setTimeout> | null = null;

const promptOpen = ref(false);
const promptKind = ref<PromptKind | null>(null);
const promptValue = ref("");
const promptTarget = ref<IdeTreeItem | null>(null);
const pendingCloseId = ref<string | null>(null);

let loadGen = 0;
let offIndex: (() => void) | null = null;
let offSemantics: (() => void) | null = null;

const workspaceId = computed(() => String(route.params.id ?? ""));
const isGuiFile = computed(() =>
  (activeFileTab.value?.path ?? "").toLowerCase().endsWith(".gui"),
);
const indexVerb = computed(() => (hadIndex.value ? "Reindexing" : "Indexing"));
/** Font scale is applied globally; keep store live for consistency. */
const uiFontScale = computed(() => settings.fontScale);

const searchModeItems = [
  { label: "Files", value: "files" },
  { label: "In files", value: "content" },
];

/** True when path is under the game script root. */
function pathUnder(root: string, path: string): boolean {
  const r = root.replace(/\\/g, "/").toLowerCase().replace(/\/$/, "");
  const p = path.replace(/\\/g, "/").toLowerCase();
  return !!r && (p === r || p.startsWith(r + "/"));
}

const isGameRootFile = computed(
  () =>
    !!activeFileTab.value &&
    pathUnder(gameScriptRoot.value, activeFileTab.value.path),
);
/** Save blocked for game-root file tabs; compare tabs are not saved as files. */
const canSave = computed(
  () => !!activeFileTab.value?.dirty && !isGameRootFile.value,
);

const editorItems = computed<CodeViewItem[]>(() => {
  const tab = activeTab.value;
  if (!tab) return [];
  if (tab.kind === "compare") {
    const lang = langForPath(tab.leftPath);
    return [
      {
        id: `diff:${tab.id}`,
        type: "diff",
        fileDiff: parseDiffFromFile(
          { name: tab.leftPath, contents: tab.leftContents, lang },
          { name: tab.rightPath, contents: tab.rightContents, lang },
        ),
        version: tab.leftContents.length + tab.rightContents.length,
        // edit enables Pierre find/replace on the compare surface
        edit: true,
      },
    ];
  }
  const lang = langForPath(tab.path);
  return [
    {
      id: `file:${tab.path}`,
      type: "file",
      file: {
        name: tab.path,
        contents: tab.contents,
        lang,
        cacheKey: tab.path,
      },
      version: tab.contents.length + (tab.dirty ? 1 : 0),
      // Always editable in Pierre so Ctrl+F works; game saves remain blocked via canSave.
      edit: true,
    },
  ];
});

/** Per-tab context menu actions. */
function tabMenuItems(tab: IdeTab): ContextMenuItem[] {
  const activeFile = activeFileTab.value;
  return [
    {
      label: "Close",
      icon: "i-lucide-x",
      onSelect: () => requestCloseTab(tab.id),
    },
    {
      label: "Close Others",
      icon: "i-lucide-copy-minus",
      disabled: tabs.value.length < 2,
      onSelect: () => requestCloseOthers(tab.id),
    },
    {
      label: "Close All",
      icon: "i-lucide-files",
      disabled: !tabs.value.length,
      onSelect: () => requestCloseAll(),
    },
    { type: "separator" },
    {
      label: "Compare with active file",
      icon: "i-lucide-columns-2",
      disabled:
        tab.kind !== "file" ||
        !activeFile ||
        tab.path === activeFile.path,
      onSelect: () => {
        if (tab.kind !== "file" || !activeFile) return;
        ide.openCompareTab(
          activeFile.path,
          activeFile.contents,
          tab.path,
          tab.contents,
        );
      },
    },
  ];
}

const explorerMenuItems = computed<ContextMenuItem[][]>(() => {
  const item = contextTreeItem.value ?? selectedTreeItem.value;
  const meta = item?.meta;
  if (!meta) return [];

  const openGroup: ContextMenuItem[] = [];
  if (!meta.isDir) {
    openGroup.push({
      label: "Open",
      icon: "i-lucide-file-text",
      onSelect: () => void openFile(meta.fullPath),
    });
    openGroup.push({
      label: "Compare with open file",
      icon: "i-lucide-columns-2",
      disabled:
        !activeFileTab.value || activeFileTab.value.path === meta.fullPath,
      onSelect: () => void compareWithPath(meta.fullPath),
    });
  }
  openGroup.push({
    label: "Copy path",
    icon: "i-lucide-clipboard",
    onSelect: () => void CopyToClipboard(meta.fullPath),
  });
  openGroup.push({
    label: "Reveal in OS",
    icon: "i-lucide-folder-open",
    onSelect: () => void RevealInOs(meta.fullPath),
  });

  if (meta.rootType === "game") return [openGroup];

  return [
    openGroup,
    [
      {
        label: "New file",
        icon: "i-lucide-file-plus",
        onSelect: () => openPrompt("new-file", item!),
      },
      {
        label: "New folder",
        icon: "i-lucide-folder-plus",
        onSelect: () => openPrompt("new-folder", item!),
      },
      {
        label: "Rename",
        icon: "i-lucide-pencil",
        onSelect: () => openPrompt("rename", item!),
      },
      {
        label: "Move",
        icon: "i-lucide-folder-input",
        onSelect: () => openPrompt("move", item!),
      },
      {
        label: "Delete",
        icon: "i-lucide-trash-2",
        color: "error",
        onSelect: () => openPrompt("delete", item!),
      },
    ],
  ];
});

const promptTitle = computed(() => {
  switch (promptKind.value) {
    case "new-file":
      return "New file";
    case "new-folder":
      return "New folder";
    case "rename":
      return "Rename";
    case "move":
      return "Move to path";
    case "delete":
      return "Delete?";
    case "discard":
      return "Discard changes?";
    case "close-all":
      return "Close all tabs?";
    case null:
      return "";
    default: {
      const _exhaustive: never = promptKind.value;
      return _exhaustive;
    }
  }
});

const promptNeedsInput = computed(() => {
  const k = promptKind.value;
  return k === "new-file" || k === "new-folder" || k === "rename" || k === "move";
});

const dirtyFilesForCloseAll = computed(() => ide.dirtyFileTabs());

/** Soft-cancel messages are not user-facing failures. */
function isCancelled(msg: string): boolean {
  return /cancelled/i.test(msg);
}

/** Join a directory and name using the directory's separator style. */
function joinPath(dir: string, name: string): string {
  const sep = dir.includes("\\") ? "\\" : "/";
  return `${dir.replace(/[/\\]+$/, "")}${sep}${name}`;
}

/** Parent directory of a full path. */
function parentOf(path: string): string {
  return path.replace(/[/\\][^/\\]+$/, "") || path;
}

/** Basename of a full path. */
function baseName(path: string): string {
  return path.split(/[/\\]/).pop() || path;
}

/** Directory used for create ops from a tree item. */
function targetDir(item: IdeTreeItem): string {
  const meta = item.meta!;
  return meta.isDir ? meta.fullPath : parentOf(meta.fullPath);
}

/** True if any dirty file tab is this path or under it. */
function hasDirtyTabsFor(path: string): boolean {
  return tabs.value.some(
    (t) =>
      t.kind === "file" &&
      t.dirty &&
      (t.path === path ||
        t.path.startsWith(path + "/") ||
        t.path.startsWith(path + "\\")),
  );
}

/** UTree key for an item. */
function treeKey(item: IdeTreeItem): string {
  return String(item.value ?? item.meta?.fullPath ?? item.label ?? "");
}

/** True when the tree row is a top-level game/mod/staging root. */
function isExplorerRoot(item: IdeTreeItem): boolean {
  const meta = item.meta;
  return !!meta && meta.isDir && meta.fullPath === meta.rootPath;
}

/** Text color for top-level root labels. */
function rootLabelClass(item: IdeTreeItem): string {
  if (!isExplorerRoot(item) || !item.meta) return "";
  switch (item.meta.rootType) {
    case "game":
      return "text-primary font-medium";
    case "mod":
      return "text-secondary font-medium";
    case "staging":
      return "text-warning font-medium";
    default: {
      const _exhaustive: never = item.meta.rootType;
      return _exhaustive;
    }
  }
}

/** Badge color for top-level root type chips. */
function rootBadgeColor(
  rootType: NonNullable<IdeTreeItem["meta"]>["rootType"],
): "primary" | "secondary" | "warning" {
  switch (rootType) {
    case "game":
      return "primary";
    case "mod":
      return "secondary";
    case "staging":
      return "warning";
    default: {
      const _exhaustive: never = rootType;
      return _exhaustive;
    }
  }
}

/** Remember the tree row under the context menu. */
function onTreeItemContext(item: IdeTreeItem): void {
  contextTreeItem.value = item;
  selectedTreeItem.value = item;
}

/** True when a search hit is a content (in-file) match. */
function isContentHit(
  hit: FileSearchHit | ContentSearchHit,
): hit is ContentSearchHit {
  return "line" in hit && typeof (hit as ContentSearchHit).line === "number";
}

/** Open a search result (skip directories). */
function openSearchHit(hit: FileSearchHit | ContentSearchHit): void {
  if (!isContentHit(hit) && hit.isDir) return;
  void openFile(hit.fullPath);
}

/** Debounced explorer search against all file roots. */
async function runSearch(): Promise<void> {
  const q = searchQuery.value.trim();
  if (!q) {
    searchHits.value = [];
    searchBusy.value = false;
    return;
  }
  const roots = fileRoots.value.map((r) => r.path);
  if (!roots.length) {
    searchHits.value = [];
    return;
  }
  searchBusy.value = true;
  error.value = "";
  try {
    if (searchMode.value === "files") {
      searchHits.value = (await SearchByName(roots, q, 100)) ?? [];
    } else {
      searchHits.value = (await SearchInFiles(roots, q, 100)) ?? [];
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
    searchHits.value = [];
  } finally {
    searchBusy.value = false;
  }
}

/** Schedule a debounced search run. */
function scheduleSearch(): void {
  if (searchTimer) clearTimeout(searchTimer);
  searchTimer = setTimeout(() => void runSearch(), 300);
}

/** Rebuild top-level tree roots from workspace state. */
function buildRoots(): void {
  const roots: FileRoot[] = [];
  if (gameScriptRoot.value) {
    roots.push({ label: "Game", path: gameScriptRoot.value, type: "game" });
  }
  for (const mod of mods.value) {
    roots.push({ label: mod.name, path: mod.path, type: "mod" });
  }
  const staging = workspace.value?.stagingDir;
  if (staging) {
    roots.push({ label: "Staging", path: staging, type: "staging" });
  }
  fileRoots.value = roots;
  treeItems.value = rootsToTreeItems(roots, {}, (path) => void openFile(path));
}

/** Load directory children into the tree for a dir item. */
async function loadDirChildren(item: IdeTreeItem): Promise<void> {
  const meta = item.meta;
  if (!meta?.isDir) return;
  try {
    const entries = ((await ListDirectory(meta.fullPath)) ?? []) as DirEntry[];
    const kids = dirEntriesToTreeItems(
      entries,
      meta.rootType,
      meta.rootPath,
      (path) => void openFile(path),
    );
    treeItems.value = setTreeChildren(treeItems.value, meta.fullPath, kids);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

/** Refresh a directory node after a mutation. */
async function refreshDir(dirPath: string): Promise<void> {
  const item = findTreeItem(treeItems.value, dirPath);
  if (item?.meta?.isDir) {
    await loadDirChildren(item);
    return;
  }
  const root = fileRoots.value.find((r) => r.path === dirPath);
  if (!root) return;
  try {
    const entries = ((await ListDirectory(dirPath)) ?? []) as DirEntry[];
    treeItems.value = setTreeChildren(
      treeItems.value,
      dirPath,
      dirEntriesToTreeItems(entries, root.type, root.path, (path) =>
        void openFile(path),
      ),
    );
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

/** Hydrate previously expanded folders after load. */
async function hydrateExpanded(paths: string[]): Promise<void> {
  const sorted = [...paths].sort((a, b) => a.length - b.length);
  for (const p of sorted) {
    const item = findTreeItem(treeItems.value, p);
    if (item?.meta?.isDir) await loadDirChildren(item);
  }
}

/** Load workspace metadata, roots, and restored explorer expansion. */
async function loadWorkspace(): Promise<void> {
  const id = workspaceId.value;
  if (!id) return;
  const gen = ++loadGen;
  loading.value = true;
  error.value = "";
  searchQuery.value = "";
  searchHits.value = [];
  ide.resetSession();
  try {
    ws.setActiveWorkspace(id);
    const wsData = await GetWorkspace(id);
    if (gen !== loadGen) return;
    workspace.value = wsData;
    mods.value = (await ListWorkspaceMods(id)) ?? [];
    if (gen !== loadGen) return;
    if (wsData?.installId) {
      gameScriptRoot.value = await GetScriptRoot(wsData.installId);
    } else {
      gameScriptRoot.value = "";
    }
    if (gen !== loadGen) return;
    try {
      const staging = await EnsureStagingDir(id);
      if (wsData) wsData.stagingDir = staging;
    } catch {
      /* staging optional */
    }
    const [sem, count] = await Promise.all([
      GetSemanticsStatus(id).catch(() => null),
      CountIndexObjects(id).catch(() => 0),
    ]);
    if (gen !== loadGen) return;
    semantics.value = sem;
    hadIndex.value = count > 0;
    buildRoots();
    const expanded = [...ide.loadExpanded(id)];
    expandedKeys.value = expanded;
    await hydrateExpanded(expanded);
  } catch (e) {
    if (gen !== loadGen) return;
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    if (gen === loadGen) loading.value = false;
  }
}

/** Lazy-load children when a directory is expanded. */
async function onTreeToggle(_e: Event, item: IdeTreeItem): Promise<void> {
  if (!item.meta?.isDir) return;
  if ((item.children?.length ?? 0) === 0) {
    await loadDirChildren(item);
  }
  if (workspaceId.value) {
    ide.saveExpanded(workspaceId.value, new Set(expandedKeys.value));
  }
}

/** Persist expansion when the controlled model changes. */
function onExpandedUpdate(keys: string[]): void {
  expandedKeys.value = keys;
  if (workspaceId.value) {
    ide.saveExpanded(workspaceId.value, new Set(keys));
  }
}

/** Open a file tab and refresh GUI preview when needed. */
async function openFile(path: string): Promise<void> {
  error.value = "";
  try {
    const contents = await ReadFileContent(path);
    ide.openFileTab(path, contents);
    selectedGuiName.value = null;
    if (path.toLowerCase().endsWith(".gui")) {
      const preview = await PreviewGui(path);
      guiRoots.value = preview?.roots ?? [];
    } else {
      guiRoots.value = [];
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

/** Open a dedicated compare tab against the active file tab. */
async function compareWithPath(otherPath: string): Promise<void> {
  const activeFile = activeFileTab.value;
  if (!activeFile || otherPath === activeFile.path) {
    error.value = "Pick a different file to compare with the open tab.";
    return;
  }
  error.value = "";
  try {
    const otherContents = await ReadFileContent(otherPath);
    ide.openCompareTab(
      activeFile.path,
      activeFile.contents,
      otherPath,
      otherContents,
    );
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

/** Apply live Pierre edits onto the active buffer. */
function onItemEdit(payload: { id: string; contents: string }): void {
  const path = payload.id.replace(/^file:/, "");
  ide.patchTabContents(path, payload.contents);
}

/** Save the active writable dirty file tab. */
async function saveActive(): Promise<void> {
  const tab = activeFileTab.value;
  if (!tab?.dirty || isGameRootFile.value) return;
  busy.value = "Saving…";
  error.value = "";
  try {
    await WriteFileContent(tab.path, tab.contents);
    ide.markSaved(tab.path);
    busy.value = "";
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
    busy.value = "";
  }
}

/** Request close; confirm discard when dirty file tab. */
function requestCloseTab(id: string): void {
  const tab = tabs.value.find((t) => t.id === id);
  if (tab?.kind === "file" && tab.dirty) {
    pendingCloseId.value = id;
    promptTarget.value = null;
    promptKind.value = "discard";
    promptValue.value = "";
    promptOpen.value = true;
    return;
  }
  ide.closeTab(id);
}

/** Close others; confirm if any dirty file tabs would be discarded. */
function requestCloseOthers(id: string): void {
  const dirtyOthers = tabs.value.some(
    (t) => t.id !== id && t.kind === "file" && t.dirty,
  );
  if (dirtyOthers) {
    pendingCloseId.value = id;
    promptTarget.value = null;
    promptKind.value = "discard";
    promptValue.value = "others";
    promptOpen.value = true;
    return;
  }
  ide.closeOthers(id);
}

/** Close all tabs; prompt when any file tabs are dirty. */
function requestCloseAll(): void {
  if (ide.dirtyFileTabs().length) {
    pendingCloseId.value = null;
    promptTarget.value = null;
    promptKind.value = "close-all";
    promptValue.value = "";
    promptOpen.value = true;
    return;
  }
  ide.closeAll();
}

/** Discard all dirty buffers and close every tab. */
function discardAllAndClose(): void {
  promptOpen.value = false;
  promptKind.value = null;
  ide.closeAll();
}

/** Save every dirty file tab, then close all. */
async function saveAllAndClose(): Promise<void> {
  const dirty = ide.dirtyFileTabs();
  busy.value = "Saving…";
  error.value = "";
  try {
    for (const tab of dirty) {
      if (pathUnder(gameScriptRoot.value, tab.path)) continue;
      await WriteFileContent(tab.path, tab.contents);
      ide.markSaved(tab.path);
    }
    promptOpen.value = false;
    promptKind.value = null;
    ide.closeAll();
    busy.value = "";
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
    busy.value = "";
  }
}

/** Open a name/path/delete prompt for the given tree item. */
function openPrompt(kind: PromptKind, item: IdeTreeItem): void {
  if (item.meta?.rootType === "game" && kind !== "discard" && kind !== "close-all") {
    error.value = "Game root is read-only.";
    return;
  }
  promptKind.value = kind;
  promptTarget.value = item;
  pendingCloseId.value = null;
  switch (kind) {
    case "new-file":
    case "new-folder":
      promptValue.value = "";
      break;
    case "rename":
      promptValue.value = baseName(item.meta!.fullPath);
      break;
    case "move":
      promptValue.value = item.meta!.fullPath;
      break;
    case "delete":
    case "discard":
    case "close-all":
      promptValue.value = "";
      break;
    default: {
      const _exhaustive: never = kind;
      return _exhaustive;
    }
  }
  promptOpen.value = true;
}

/** Confirm the active prompt modal. */
async function confirmPrompt(): Promise<void> {
  const kind = promptKind.value;
  if (!kind) return;

  if (kind === "close-all") {
    // Footer handles save/discard actions explicitly.
    return;
  }

  if (kind === "discard") {
    const id = pendingCloseId.value;
    const mode = promptValue.value;
    promptOpen.value = false;
    promptKind.value = null;
    if (!id) return;
    if (mode === "others") ide.closeOthers(id);
    else ide.closeTab(id);
    pendingCloseId.value = null;
    return;
  }

  const item = promptTarget.value;
  const meta = item?.meta;
  if (!meta || meta.rootType === "game") {
    error.value = "Game root is read-only.";
    promptOpen.value = false;
    return;
  }

  if (
    (kind === "rename" || kind === "move" || kind === "delete") &&
    hasDirtyTabsFor(meta.fullPath)
  ) {
    error.value = "Save or discard dirty tabs for this path first.";
    promptOpen.value = false;
    return;
  }

  const name = promptValue.value.trim();
  promptOpen.value = false;
  promptKind.value = null;

  try {
    switch (kind) {
      case "new-file": {
        if (!name) return;
        const full = joinPath(targetDir(item!), name);
        await CreateFile(full);
        await refreshDir(parentOf(full));
        await openFile(full);
        break;
      }
      case "new-folder": {
        if (!name) return;
        const full = joinPath(targetDir(item!), name);
        await CreateDir(full);
        await refreshDir(parentOf(full));
        break;
      }
      case "rename": {
        if (!name || name === baseName(meta.fullPath)) return;
        const dest = joinPath(parentOf(meta.fullPath), name);
        await RenamePath(meta.fullPath, dest);
        ide.retargetPath(meta.fullPath, dest);
        await refreshDir(parentOf(meta.fullPath));
        break;
      }
      case "move": {
        if (!name || name === meta.fullPath) return;
        await RenamePath(meta.fullPath, name);
        ide.retargetPath(meta.fullPath, name);
        await refreshDir(parentOf(meta.fullPath));
        await refreshDir(parentOf(name));
        break;
      }
      case "delete": {
        await DeletePath(meta.fullPath);
        ide.dropDeleted(meta.fullPath);
        await refreshDir(parentOf(meta.fullPath));
        break;
      }
      default: {
        const _exhaustive: never = kind;
        return _exhaustive;
      }
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

/** Run localization lint for the workspace. */
async function runLocLint(): Promise<void> {
  if (!workspaceId.value) return;
  busy.value = "Linting localization…";
  error.value = "";
  try {
    locDiags.value = (await LintMissingLoc(workspaceId.value, "l_english")) ?? [];
    showLocPanel.value = true;
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    busy.value = "";
  }
}

/** Stub missing loc keys then re-lint. */
async function runAutoLoc(): Promise<void> {
  if (!workspaceId.value) return;
  busy.value = "Auto-localizing…";
  error.value = "";
  try {
    const result = await AutoLocalize(workspaceId.value, "l_english", "l_english", "");
    autoLocResult.value = result;
    autoLocOpen.value = true;
    await runLocLint();
    busy.value = "";
    if (result?.path) await openFile(result.path);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
    busy.value = "";
  }
}

/** Rebuild semantics cache then reindex. */
async function rebuildSemantics(): Promise<void> {
  if (!workspaceId.value) return;
  jobActive.value = true;
  busy.value = "Rebuilding semantics…";
  progressLabel.value = "";
  progressPercent.value = 0;
  error.value = "";
  try {
    const cache = await RebuildWorkspaceSemantics(workspaceId.value);
    if (!jobActive.value) return;
    if (!cache) {
      busy.value = "";
      progressLabel.value = "";
      progressPercent.value = 0;
      return;
    }
    busy.value = `${indexVerb.value}…`;
    await ReindexWorkspace(workspaceId.value);
    if (!jobActive.value) return;
    semantics.value = await GetSemanticsStatus(workspaceId.value);
    hadIndex.value = true;
    busy.value = "Semantics rebuilt";
    progressLabel.value = "";
    progressPercent.value = 0;
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    if (!isCancelled(msg)) error.value = msg;
    busy.value = "";
    progressLabel.value = "";
    progressPercent.value = 0;
  } finally {
    jobActive.value = false;
  }
}

/** Cancel in-flight semantics/index jobs. */
function cancelJobs(): void {
  void CancelSemantics();
  void CancelIndex();
  jobActive.value = false;
  busy.value = "";
  progressLabel.value = "";
  progressPercent.value = 0;
}

/** Open the file for a loc diagnostic. */
function jumpToDiag(d: LocDiagnostic): void {
  void openFile(d.filePath);
}

/** Apply index/semantics progress events to the overlay. */
function applyProgress(raw: unknown, fallback: string): void {
  const wrapped = raw as { data?: Record<string, unknown> } & Record<string, unknown>;
  const d = wrapped.data ?? wrapped;
  if (!d) return;
  if (d.phase === "cancelled") {
    jobActive.value = false;
    busy.value = "";
    progressLabel.value = "";
    progressPercent.value = 0;
    return;
  }
  const pct = typeof d.percent === "number" ? d.percent : undefined;
  if (pct != null) progressPercent.value = pct;
  const pctText = pct != null ? ` ${Math.round(pct)}%` : "";
  progressLabel.value = `${String(d.message ?? d.phase ?? fallback)}${pctText}`;
}

/** Global save shortcut. */
function onGlobalKey(ev: KeyboardEvent): void {
  if ((ev.ctrlKey || ev.metaKey) && ev.key.toLowerCase() === "s") {
    ev.preventDefault();
    void saveActive();
  }
}

watch(workspaceId, loadWorkspace, { immediate: true });

watch([searchQuery, searchMode], scheduleSearch);

watch(
  () => activeFileTab.value?.path,
  async (path) => {
    if (!path) {
      guiRoots.value = [];
      return;
    }
    if (!path.toLowerCase().endsWith(".gui")) {
      guiRoots.value = [];
      return;
    }
    try {
      const preview = await PreviewGui(path);
      guiRoots.value = preview?.roots ?? [];
    } catch {
      guiRoots.value = [];
    }
  },
);

onMounted(() => {
  window.addEventListener("keydown", onGlobalKey);
  offIndex = Events.On("index:progress", (ev) => applyProgress(ev, "Indexing"));
  offSemantics = Events.On("semantics:progress", (ev) => applyProgress(ev, "Semantics"));
});

onBeforeUnmount(() => {
  loadGen++;
  if (searchTimer) clearTimeout(searchTimer);
  window.removeEventListener("keydown", onGlobalKey);
  offIndex?.();
  offSemantics?.();
  cancelJobs();
});
</script>

<template>
  <div class="relative flex h-full min-h-0 flex-col overflow-hidden">
    <div
      class="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-default bg-muted/50 px-3 py-2"
    >
      <div class="flex min-w-0 items-center gap-2">
        <UTooltip text="Back to library">
          <UButton
            icon="i-lucide-arrow-left"
            variant="ghost"
            size="sm"
            @click="router.push({ name: 'library' })"
          />
        </UTooltip>
        <span class="truncate font-semibold">{{ workspace?.name ?? "Workspace" }}</span>
        <UBadge v-if="workspace" color="neutral" variant="outline" size="xs">
          {{ workspace.gameId.toUpperCase() }}
        </UBadge>
        <UBadge
          v-if="semantics"
          :color="semantics.present ? 'success' : 'warning'"
          variant="subtle"
          size="xs"
        >
          {{
            semantics.present
              ? `Semantics ${semantics.scannedAt?.slice(0, 10) ?? "ready"}`
              : "No semantics"
          }}
        </UBadge>
        <span class="sr-only">UI scale {{ uiFontScale }}%</span>
        <UTooltip v-if="jobActive || busy" text="Cancel rebuild / index">
          <UButton label="Cancel" size="xs" color="neutral" variant="ghost" @click="cancelJobs" />
        </UTooltip>
      </div>
      <div class="flex flex-wrap items-center gap-1.5">
        <USelect
          :model-value="editorTheme"
          :items="[...EDITOR_THEME_OPTIONS]"
          value-key="value"
          class="w-36"
          size="sm"
          @update:model-value="(v: string) => setEditorTheme(v)"
        />
        <UTooltip text="Save active file (Ctrl+S)">
          <UButton
            label="Save"
            icon="i-lucide-save"
            variant="outline"
            size="sm"
            :disabled="!canSave"
            @click="saveActive"
          />
        </UTooltip>
        <UTooltip text="Find missing localization keys">
          <UButton
            label="Loc lint"
            icon="i-lucide-languages"
            variant="outline"
            size="sm"
            @click="runLocLint"
          />
        </UTooltip>
        <UTooltip text="Stub missing loc keys into mod file">
          <UButton
            label="Auto-loc"
            icon="i-lucide-wand-sparkles"
            variant="outline"
            size="sm"
            @click="runAutoLoc"
          />
        </UTooltip>
        <UTooltip text="Rebuild install semantics then index">
          <UButton
            label="Rebuild semantics"
            icon="i-lucide-refresh-cw"
            variant="outline"
            size="sm"
            :disabled="jobActive"
            @click="rebuildSemantics"
          />
        </UTooltip>
        <UTooltip text="Open Patch Center">
          <UButton
            label="Patch Center"
            icon="i-lucide-external-link"
            variant="soft"
            size="sm"
            @click="router.push({ name: 'patch-center', params: { id: workspaceId } })"
          />
        </UTooltip>
        <UTooltip text="Open Mod Patcher">
          <UButton
            label="Patcher"
            icon="i-lucide-external-link"
            variant="soft"
            size="sm"
            @click="router.push({ name: 'patcher', params: { id: workspaceId } })"
          />
        </UTooltip>
        <UTooltip text="Open Event Graph">
          <UButton
            label="Graph"
            icon="i-lucide-external-link"
            variant="soft"
            size="sm"
            @click="router.push({ name: 'event-graph', params: { id: workspaceId } })"
          />
        </UTooltip>
      </div>
    </div>

    <UAlert v-if="error" color="error" variant="subtle" :description="error" class="m-2 shrink-0" />
    <UAlert
      v-else-if="semantics && !semantics.present"
      color="warning"
      variant="subtle"
      class="m-2 shrink-0"
      title="Semantics cache missing"
      description="Rebuild semantics before relying on type-aware tools."
    >
      <template #actions>
        <UButton size="xs" label="Rebuild now" @click="rebuildSemantics" />
      </template>
    </UAlert>

    <div
      v-if="jobActive"
      class="absolute inset-0 z-20 flex items-center justify-center bg-default/70 backdrop-blur-[1px]"
    >
      <div class="w-full max-w-md rounded-lg border border-default bg-default p-5 shadow-lg">
        <div class="mb-1 text-sm font-semibold">{{ busy || indexVerb + "…" }}</div>
        <div class="mb-3 text-xs text-muted">{{ progressLabel || "Working…" }}</div>
        <UProgress
          :model-value="progressPercent > 0 ? progressPercent : null"
          :max="100"
          size="md"
        />
        <div class="mt-3 flex justify-end">
          <UButton label="Cancel" size="xs" color="neutral" variant="outline" @click="cancelJobs" />
        </div>
      </div>
    </div>

    <UDashboardGroup
      class="relative min-h-0 flex-1 overflow-hidden"
      :ui="{ base: 'relative flex min-h-0 flex-1 overflow-hidden' }"
      storage="local"
      storage-key="ide-dashboard"
      unit="px"
    >
      <UDashboardSidebar
        id="ide-explorer"
        collapsible
        resizable
        :default-size="260"
        :min-size="160"
        :max-size="480"
        :ui="{
          root: 'relative flex min-h-0 h-auto flex-col shrink-0 border-e border-default',
          body: 'flex min-h-0 flex-1 flex-col gap-0 overflow-hidden px-0 py-0',
          header: 'shrink-0 flex items-center gap-1.5 px-2 py-1 border-b border-default',
        }"
      >
        <template #header>
          <div class="text-xs font-semibold text-muted">Explorer</div>
        </template>

        <div class="shrink-0 space-y-1.5 border-b border-default px-2 py-1.5">
          <UInput
            v-model="searchQuery"
            icon="i-lucide-search"
            placeholder="Search…"
            size="sm"
            :loading="searchBusy"
          />
          <UTabs
            :model-value="searchMode"
            :items="searchModeItems"
            :content="false"
            size="xs"
            class="w-full"
            :ui="{ list: 'w-full' }"
            @update:model-value="(v) => (searchMode = String(v) as SearchMode)"
          />
        </div>

        <UContextMenu :items="explorerMenuItems">
          <div class="min-h-0 flex-1 overflow-auto px-1 py-1">
            <div v-if="searchQuery.trim()" class="flex flex-col gap-0.5">
              <div
                v-if="!searchBusy && !searchHits.length"
                class="px-2 py-3 text-xs text-muted"
              >
                No results
              </div>
              <UButton
                v-for="(hit, i) in searchHits"
                :key="isContentHit(hit) ? `${hit.fullPath}:${hit.line}:${i}` : `${hit.fullPath}:${i}`"
                color="neutral"
                variant="ghost"
                size="xs"
                class="h-auto w-full justify-start px-2 py-1.5 text-left"
                @click="openSearchHit(hit)"
              >
                <span class="flex min-w-0 flex-col gap-0.5">
                  <span class="truncate font-medium">
                    {{ isContentHit(hit) ? baseName(hit.fullPath) : hit.name }}
                  </span>
                  <span class="truncate text-[10px] text-muted">
                    <template v-if="isContentHit(hit)">
                      {{ hit.fullPath }}:{{ hit.line }} — {{ hit.text.trim() }}
                    </template>
                    <template v-else>{{ hit.fullPath }}</template>
                  </span>
                </span>
              </UButton>
            </div>

            <template v-else>
              <UTree
                v-model="selectedTreeItem"
                :expanded="expandedKeys"
                :items="treeItems"
                :get-key="treeKey"
                size="sm"
                :ui="{ link: 'text-xs' }"
                @update:expanded="onExpandedUpdate"
                @toggle="onTreeToggle"
              >
                <template #item-label="{ item }">
                  <span
                    class="flex min-w-0 flex-1 items-center gap-1.5"
                    @contextmenu="onTreeItemContext(item as IdeTreeItem)"
                  >
                    <span
                      class="truncate"
                      :class="rootLabelClass(item as IdeTreeItem)"
                    >
                      {{ item.label }}
                    </span>
                    <UBadge
                      v-if="isExplorerRoot(item as IdeTreeItem)"
                      :color="rootBadgeColor((item as IdeTreeItem).meta!.rootType)"
                      variant="subtle"
                      size="xs"
                      class="shrink-0"
                    >
                      {{ (item as IdeTreeItem).meta!.rootType }}
                    </UBadge>
                  </span>
                </template>
              </UTree>
              <div v-if="!treeItems.length && !loading" class="px-2 py-3 text-xs text-muted">
                No roots yet
              </div>
            </template>
          </div>
        </UContextMenu>
      </UDashboardSidebar>

      <UDashboardPanel
        id="ide-editor"
        class="min-w-0"
        :ui="{
          root: 'relative flex min-h-0 min-w-0 flex-1 flex-col',
          body: 'flex min-h-0 flex-1 flex-col gap-0 overflow-hidden p-0',
        }"
      >
        <template #body>
          <div class="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
            <div
              v-if="tabs.length"
              class="flex shrink-0 items-center border-b border-default bg-muted/30"
            >
              <div class="flex min-w-0 flex-1 gap-0.5 overflow-x-auto px-1">
                <UContextMenu
                  v-for="tab in tabs"
                  :key="tab.id"
                  :items="tabMenuItems(tab)"
                >
                  <button
                    type="button"
                    class="group flex max-w-52 shrink-0 items-center gap-1 rounded-t px-2 py-1.5 text-xs"
                    :class="
                      tab.id === activeTabId
                        ? 'bg-default text-highlighted border-b-2 border-primary'
                        : 'text-muted hover:bg-muted/60'
                    "
                    @click="activeTabId = tab.id"
                    @auxclick.middle="requestCloseTab(tab.id)"
                  >
                    <span class="truncate">
                      {{ tab.kind === "compare" ? "⇄ " : ""
                      }}{{ tab.kind === "file" && tab.dirty ? "● " : ""
                      }}{{ tab.name }}
                    </span>
                    <UButton
                      icon="i-lucide-x"
                      size="xs"
                      color="neutral"
                      variant="ghost"
                      class="size-5 shrink-0 opacity-60 group-hover:opacity-100"
                      @click.stop="requestCloseTab(tab.id)"
                    />
                  </button>
                </UContextMenu>
              </div>
            </div>

            <div class="relative flex min-h-0 flex-1 flex-col overflow-hidden">
              <EditorView
                class="absolute inset-0"
                :items="editorItems"
                placeholder="Select a file to edit"
                @item-edit="onItemEdit"
              />
              <div
                v-if="showLocPanel"
                class="absolute inset-x-0 bottom-0 z-10 max-h-48 overflow-auto border-t border-default bg-default"
              >
                <div class="flex items-center justify-between px-2 py-1 text-xs font-semibold">
                  <span>Missing loc ({{ locDiags.length }})</span>
                  <UButton
                    size="xs"
                    variant="ghost"
                    icon="i-lucide-x"
                    @click="showLocPanel = false"
                  />
                </div>
                <button
                  v-for="(d, i) in locDiags"
                  :key="`${d.key}-${i}`"
                  class="flex w-full flex-col px-2 py-1 text-left text-xs hover:bg-muted/50"
                  @click="jumpToDiag(d)"
                >
                  <span class="font-medium">{{ d.key }}</span>
                  <span class="truncate text-muted">{{ d.filePath }}:{{ d.line }}</span>
                </button>
              </div>
            </div>
          </div>
        </template>
      </UDashboardPanel>

      <UDashboardPanel
        v-if="isGuiFile"
        id="ide-gui"
        resizable
        :default-size="320"
        :min-size="200"
        :max-size="520"
        :ui="{
          root: 'relative flex min-h-0 flex-col shrink-0 border-s border-default',
          body: 'flex min-h-0 flex-1 flex-col gap-0 overflow-hidden p-0',
        }"
      >
        <template #body>
          <GuiPreviewPanel
            class="h-full min-h-0"
            :roots="guiRoots"
            :selected-name="selectedGuiName"
            @select="(n) => (selectedGuiName = n)"
          />
        </template>
      </UDashboardPanel>
    </UDashboardGroup>

    <UModal v-model:open="autoLocOpen" title="Auto-loc result">
      <template #body>
        <div v-if="autoLocResult" class="space-y-2 text-sm">
          <p>
            Wrote <strong>{{ autoLocResult.count }}</strong> key(s) to:
          </p>
          <code class="block break-all rounded bg-muted px-2 py-1 text-xs">
            {{ autoLocResult.path }}
          </code>
          <pre class="max-h-64 overflow-auto rounded border border-default p-2 text-xs">{{
            autoLocResult.preview || "(empty)"
          }}</pre>
        </div>
      </template>
      <template #footer>
        <UButton label="Close" color="neutral" variant="outline" @click="autoLocOpen = false" />
        <UButton
          v-if="autoLocResult?.path"
          label="Open file"
          @click="
            () => {
              autoLocOpen = false;
              void openFile(autoLocResult!.path);
            }
          "
        />
      </template>
    </UModal>

    <UModal v-model:open="promptOpen" :title="promptTitle">
      <template #body>
        <p v-if="promptKind === 'delete'" class="text-sm">
          Delete
          <code class="rounded bg-muted px-1 text-xs">{{ promptTarget?.meta?.fullPath }}</code>
          permanently?
        </p>
        <p v-else-if="promptKind === 'discard'" class="text-sm">
          {{
            promptValue === "others"
              ? "Discard unsaved changes in other tabs?"
              : "Discard unsaved changes in this tab?"
          }}
        </p>
        <div v-else-if="promptKind === 'close-all'" class="space-y-2 text-sm">
          <p>Unsaved changes in:</p>
          <ul class="max-h-40 space-y-1 overflow-auto text-xs">
            <li v-for="t in dirtyFilesForCloseAll" :key="t.id">
              <code class="rounded bg-muted px-1">{{ t.path }}</code>
            </li>
          </ul>
        </div>
        <UFormField
          v-else-if="promptNeedsInput"
          :label="promptKind === 'move' ? 'Full path' : 'Name'"
        >
          <UInput v-model="promptValue" autofocus class="w-full" />
        </UFormField>
      </template>
      <template #footer="{ close }">
        <template v-if="promptKind === 'close-all'">
          <UButton label="Cancel" color="neutral" variant="outline" @click="close()" />
          <UButton
            label="Discard all & close"
            color="error"
            variant="outline"
            @click="discardAllAndClose"
          />
          <UButton label="Save all & close" color="primary" @click="saveAllAndClose" />
        </template>
        <template v-else>
          <UButton label="Cancel" color="neutral" variant="outline" @click="close()" />
          <UButton
            :label="promptKind === 'delete' || promptKind === 'discard' ? 'Confirm' : 'OK'"
            :color="promptKind === 'delete' || promptKind === 'discard' ? 'error' : 'primary'"
            @click="confirmPrompt"
          />
        </template>
      </template>
    </UModal>
  </div>
</template>
