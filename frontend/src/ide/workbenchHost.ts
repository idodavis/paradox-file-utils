/**
 * Singleton monaco-vscode ViewsService host (init once per app lifetime).
 *
 * IdeWorkbenchLayout.vue registers attachPart DOM refs. WorkspaceIdePage
 * calls setWorkbenchRoots, which is the only initialize() entry — the
 * .code-workspace file always contains real folders on first boot.
 *
 * Intentionally excluded VS Code contributions (do not import until a feature
 * needs them): debug, testing, notebooks, terminal, SCM/git, comments, timeline,
 * chat, AI, extension gallery/marketplace, remote, tasks, output panel,
 * welcome/walkthrough, emmet, speech, survey, update.
 * Keep: editor, explorer, problems, search, hover/complete/def/refs, folding,
 * themes, file icons, multi-diff (merge review).
 */
import {
  initialize as initializeMonacoService,
  type IEditorOverrideServices,
  type IWorkbenchConstructionOptions,
  LogLevel,
} from "@codingame/monaco-vscode-api";
import type { IDisposable } from "@codingame/monaco-vscode-api/vscode/vs/base/common/lifecycle";
import * as monacoLifecycle from "@codingame/monaco-vscode-api/lifecycle";
import getViewsServiceOverride, {
  attachPart,
  setPartVisibility,
  Parts,
} from "@codingame/monaco-vscode-views-service-override";
import getQuickAccessServiceOverride from "@codingame/monaco-vscode-quickaccess-service-override";
import getConfigurationServiceOverride, {
  initUserConfiguration,
} from "@codingame/monaco-vscode-configuration-service-override";
import getKeybindingsServiceOverride, {
  initUserKeybindings,
} from "@codingame/monaco-vscode-keybindings-service-override";
import getModelServiceOverride from "@codingame/monaco-vscode-model-service-override";
import getNotificationServiceOverride from "@codingame/monaco-vscode-notifications-service-override";
import getDialogsServiceOverride from "@codingame/monaco-vscode-dialogs-service-override";
import getTextmateServiceOverride from "@codingame/monaco-vscode-textmate-service-override";
import getThemeServiceOverride from "@codingame/monaco-vscode-theme-service-override";
import getLanguagesServiceOverride from "@codingame/monaco-vscode-languages-service-override";
import getSearchServiceOverride from "@codingame/monaco-vscode-search-service-override";
import getMarkersServiceOverride from "@codingame/monaco-vscode-markers-service-override";
import getExtensionServiceOverride from "@codingame/monaco-vscode-extensions-service-override";
import getLifecycleServiceOverride from "@codingame/monaco-vscode-lifecycle-service-override";
import getEnvironmentServiceOverride from "@codingame/monaco-vscode-environment-service-override";
import getLogServiceOverride from "@codingame/monaco-vscode-log-service-override";
import getWorkingCopyServiceOverride from "@codingame/monaco-vscode-working-copy-service-override";
import getMultiDiffEditorServiceOverride from "@codingame/monaco-vscode-multi-diff-editor-service-override";
import getExplorerServiceOverride from "@codingame/monaco-vscode-explorer-service-override";
import {
  createIndexedDBProviders,
  registerFileSystemOverlay,
  RegisteredFileSystemProvider,
  RegisteredMemoryFile,
  initFile,
} from "@codingame/monaco-vscode-files-service-override";
import { registerExtension } from "@codingame/monaco-vscode-api/extensions";
import { ExtensionHostKind } from "@codingame/monaco-vscode-extensions-service-override";
import "@codingame/monaco-vscode-theme-defaults-default-extension";
import "./fileIcons";
import "vscode/localExtensionHost";
import * as monaco from "monaco-editor";
import * as vscode from "vscode";
import {
  WailsFileSystemProvider,
  setModRootDeletedHook,
  type IdeRoot,
} from "./fsBridge";
import { applyWorkbenchTheme, workbenchSettingsForTheme, lockWorkbenchTheme } from "./themeBridge";
import { currentWorkbenchTheme } from "./colorThemes";
import { registerParadoxLanguages } from "./paradoxLanguages";
import { registerLanguageClient } from "./languageClient";
import { registerRootDecorations, setDecoratedRoots } from "./rootDecorations";
import { useWorkspaceStore } from "../stores/workspace";
import { useIdeShellStore } from "../stores/ideShell";
import {
  GetIdeRoots,
  RemoveWorkspaceMod,
  SaveIdeSession,
} from "@services/workspaceservice";
import "./explorerLayout.css";

let workbenchReady = false;
let readyResolve: () => void = () => undefined;
const readyPromise = new Promise<void>((resolve) => {
  readyResolve = resolve;
});

/** Whether the workbench has finished initialize(). */
export function isWorkbenchReady(): boolean {
  return workbenchReady;
}

/** Resolves once the workbench has finished initialize(). */
export function whenWorkbenchReady(): Promise<void> {
  return readyPromise;
}

/** attachPart DOM refs owned by IdeWorkbenchLayout.vue. */
export type IdePartRefs = {
  root: HTMLElement;
  activityBar: HTMLElement;
  sidebar: HTMLElement;
  editor: HTMLElement;
  panel: HTMLElement;
};

const WS_ID = "pmt-app-workspace";
const WS_FILE = monaco.Uri.file("/pmt.code-workspace");

/** Command palette / menus / keys owned by PMT, not the workbench. */
const HIDDEN_PALETTE_COMMANDS = [
  "workbench.action.files.openFile",
  "workbench.action.files.openFileFolder",
  "workbench.action.files.openFolder",
  "workbench.action.openWorkspace",
  "workbench.action.openWorkspaceFromFile",
  "workbench.action.openRecent",
  "workbench.action.addRootFolder",
  "workbench.action.removeRootFolder",
  "addRootFolder",
  "removeRootFolder",
  "workbench.action.closeFolder",
  "workbench.action.closeWorkspace",
  "workbench.action.newWindow",
  "workbench.action.newEmptyEditorWindow",
  "workbench.action.files.showOpenedFileInNewWindow",
  "workbench.action.saveWorkspaceAs",
  "workbench.action.duplicateWorkspaceInNewWindow",
  "workbench.action.files.newUntitledFile",
  "workbench.action.selectTheme",
  "workbench.action.selectIconTheme",
  "workbench.action.selectProductIconTheme",
  "workbench.action.openGlobalKeybindings",
  "workbench.action.openGlobalKeybindingsFile",
  "workbench.action.openDefaultKeybindingsFile",
  "workbench.action.openSettings",
  "workbench.action.openGlobalSettings",
  "workbench.action.openWorkspaceSettings",
  "workbench.action.openSettingsJson",
  "workbench.actions.manageAccounts",
  "workbench.actions.accounts",
  "workbench.action.showAbout",
  "workbench.action.openDocumentationUrl",
  "workbench.action.keybindingsReference",
  "workbench.action.openIntroductoryVideosUrl",
  "workbench.action.openTipsAndTricksUrl",
  "workbench.action.openTwitterUrl",
  "workbench.action.openRequestFeatureUrl",
  "workbench.action.openIssueReporter",
  "workbench.action.openLicenseUrl",
  "workbench.action.openPrivacyStatementUrl",
  "update.showCurrentReleaseNotes",
  "workbench.action.openWalkthrough",
  "workbench.action.showInteractivePlayground",
] as const;

const OPEN_WORKSPACE_SETTINGS = "pmt.openWorkspaceSettings";

/** Persist-tabs options for a PMT workspace (not VS Code IndexedDB). */
export type IdeSessionOpts = {
  workspaceId: string;
  persistTabs: boolean;
  openFiles: string[];
  activeFile: string;
};

const ATTACHED_PARTS = [
  Parts.ACTIVITYBAR_PART,
  Parts.SIDEBAR_PART,
  Parts.EDITOR_PART,
  Parts.PANEL_PART,
] as const;

let initPromise: Promise<void> | null = null;
let containerEl: HTMLElement | null = null;
let partRefs: IdePartRefs | null = null;
let refsResolve: () => void = () => undefined;
const refsPromise = new Promise<void>((resolve) => {
  refsResolve = resolve;
});
const fsProvider = new WailsFileSystemProvider();
let currentRoots: IdeRoot[] = [];
let langClient: { dispose(): void } | null = null;
let themeLock: { dispose(): void } | null = null;
let folderLock: { dispose(): void } | null = null;
let rootDeco: { dispose(): void } | null = null;
let mutatingFolders = false;
let partDisposables: IDisposable[] = [];
let folderWriteChain: Promise<void> = Promise.resolve();
let rootsChain: Promise<void> = Promise.resolve();
let mountedWorkspaceId = "";
let persistTabsForMounted = false;
let restoringTabs = false;
let saveTabsTimer: ReturnType<typeof setTimeout> | null = null;
let persistHooked = false;

setModRootDeletedHook(async (originId) => {
  const wsId = mountedWorkspaceId || useWorkspaceStore().activeWorkspaceId;
  if (!wsId) return;
  await RemoveWorkspaceMod(wsId, originId);
  const roots = ((await GetIdeRoots(wsId)) ?? []) as IdeRoot[];
  if (!roots.length) return;
  await setWorkbenchRoots(roots, currentWorkbenchTheme());
});

function setupWorkers(): void {
  const w = window as Window & {
    MonacoEnvironment?: {
      getWorker: (_: string, label: string) => Worker;
    };
  };
  w.MonacoEnvironment = {
    getWorker(_: string, label: string) {
      if (label === "TextMateWorker") {
        return new Worker(
          new URL("@codingame/monaco-vscode-textmate-service-override/worker", import.meta.url),
          { type: "module" },
        );
      }
      if (label === "LocalFileSearchWorker") {
        return new Worker(
          new URL("@codingame/monaco-vscode-search-service-override/worker", import.meta.url),
          { type: "module" },
        );
      }
      return new Worker(
        new URL("monaco-editor/esm/vs/editor/editor.worker.js", import.meta.url),
        { type: "module" },
      );
    },
  };
}

function leanServices(): IEditorOverrideServices {
  return {
    ...getLogServiceOverride(),
    ...getExtensionServiceOverride(),
    ...getModelServiceOverride(),
    ...getNotificationServiceOverride(),
    ...getDialogsServiceOverride(),
    ...getConfigurationServiceOverride(),
    ...getKeybindingsServiceOverride(),
    ...getTextmateServiceOverride(),
    ...getThemeServiceOverride(),
    ...getLanguagesServiceOverride(),
    ...getSearchServiceOverride(),
    ...getMarkersServiceOverride(),
    ...getLifecycleServiceOverride(),
    ...getEnvironmentServiceOverride(),
    ...getWorkingCopyServiceOverride(),
    ...getMultiDiffEditorServiceOverride(),
    ...getExplorerServiceOverride(),
    ...getViewsServiceOverride(),
    ...getQuickAccessServiceOverride({
      isKeybindingConfigurationVisible: () => false,
      shouldUseGlobalPicker: () => true,
    }),
  };
}

function constructOptions(): IWorkbenchConstructionOptions {
  return {
    enableWorkspaceTrust: false,
    windowIndicator: {
      label: "PMT",
      tooltip: "Paradox Modding Tools",
      command: "",
    },
    workspaceProvider: {
      trusted: true,
      async open() {
        return false;
      },
      workspace: { workspaceUri: WS_FILE, id: WS_ID },
    },
    developmentOptions: { logLevel: LogLevel.Warning },
    productConfiguration: {
      nameShort: "PMT",
      nameLong: "Paradox Modding Tools",
      enableTelemetry: false,
    },
    defaultLayout: {
      views: [{ id: "workbench.explorer.fileView" }, { id: "workbench.panel.markers.view" }],
      force: true,
    },
    configurationDefaults: {
      "window.titleBarStyle": "native",
      "workbench.activityBar.location": "default",
      "workbench.startupEditor": "none",
      "workbench.colorTheme": "pmt-dark",
      "workbench.iconTheme": "pmt-icons",
      "workbench.tree.indent": 16,
      "workbench.tree.renderIndentGuides": "always",
      "workbench.enableExperiments": false,
      "workbench.tips.enabled": false,
      "telemetry.telemetryLevel": "off",
      "git.enabled": false,
      "git.autoRepositoryDetection": false,
      "scm.diffDecorations": "none",
      "debug.toolBarLocation": "hidden",
      "debug.showInStatusBar": "never",
      "extensions.autoCheckUpdates": false,
      "extensions.autoUpdate": false,
      "extensions.ignoreRecommendations": true,
      "workbench.activity.showAccounts": false,
      "editor.semanticHighlighting.enabled": false,
      "editor.wordBasedSuggestions": "off",
      "editor.acceptSuggestionOnCommitCharacter": false,
    },
  };
}

const envOpts = { userHome: monaco.Uri.file("/") };

/** Store attachPart DOM refs from IdeWorkbenchLayout. Does not initialize. */
export function registerIdeParts(refs: IdePartRefs): void {
  containerEl = refs.root;
  partRefs = refs;
  refsResolve();
}

/** Bind ViewsService parts to the registered layout refs. */
function attachIdeParts(): void {
  if (!partRefs) return;
  for (const disposable of partDisposables) disposable.dispose();
  partDisposables = [];
  const containers: Record<(typeof ATTACHED_PARTS)[number], HTMLElement> = {
    [Parts.ACTIVITYBAR_PART]: partRefs.activityBar,
    [Parts.SIDEBAR_PART]: partRefs.sidebar,
    [Parts.EDITOR_PART]: partRefs.editor,
    [Parts.PANEL_PART]: partRefs.panel,
  };
  for (const part of ATTACHED_PARTS) {
    partDisposables.push(attachPart(part, containers[part]));
  }
}

/** Normalize a filesystem path for folder-identity comparison. */
function normPath(path: string): string {
  return path.replace(/\\/g, "/").toLowerCase();
}

/** Whether workbench folders already match the app-owned roots. */
function foldersMatch(roots: IdeRoot[]): boolean {
  const folders = vscode.workspace.workspaceFolders ?? [];
  if (folders.length !== roots.length) return false;
  return roots.every((root, i) => {
    const folder = folders[i];
    return !!folder && normPath(folder.uri.fsPath) === normPath(root.path);
  });
}

/** Hide or restore activity bar / sidebar / panel for merge review. */
export function setMergeChrome(hidden: boolean): void {
  setPartVisibility(Parts.ACTIVITYBAR_PART, !hidden);
  setPartVisibility(Parts.SIDEBAR_PART, !hidden);
  setPartVisibility(Parts.PANEL_PART, !hidden);
}

/** Re-open Explorer and Problems after a folder remount. */
async function restoreIdeViews(): Promise<void> {
  if (useIdeShellStore().mergeReview) return;
  setPartVisibility(Parts.ACTIVITYBAR_PART, true);
  setPartVisibility(Parts.SIDEBAR_PART, true);
  setPartVisibility(Parts.EDITOR_PART, true);
  setPartVisibility(Parts.PANEL_PART, true);
  try {
    await vscode.commands.executeCommand("workbench.view.explorer");
  } catch {
    /* command missing in this monaco-vscode build */
  }
  try {
    await vscode.commands.executeCommand("workbench.actions.view.problems");
  } catch {
    try {
      await vscode.commands.executeCommand("workbench.panel.markers");
    } catch {
      /* ignore */
    }
  }
}

/** Serialize folder mutations so init and setWorkbenchRoots cannot interleave. */
function enqueueFolderWrite(fn: () => Promise<void>): Promise<void> {
  const run = folderWriteChain.then(fn, fn);
  folderWriteChain = run.then(
    () => undefined,
    () => undefined,
  );
  return run;
}

/** Serialize setWorkbenchRoots so initialize() cannot run twice. */
function enqueueRoots(fn: () => Promise<void>): Promise<void> {
  const run = rootsChain.then(fn, fn);
  rootsChain = run.then(
    () => undefined,
    () => undefined,
  );
  return run;
}

/** Skip monaco.editor until initialize() — calling it early breaks boot. */
function syncMountedEditors(): void {
  if (!workbenchReady) return;
  for (const ed of monaco.editor.getEditors()) syncEditorReadOnly(ed);
}

/** monaco-vscode initialize() throws this if StandaloneServices already exist. */
function isAlreadyInitialized(err: unknown): boolean {
  return err instanceof Error && err.message === "Services are already initialized";
}

/** Serialize roots into a VS Code multi-root workspace file body. */
function workspaceFileJson(roots: IdeRoot[]): string {
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

/** Write app workspace roots into the workbench (suppresses folder-lock). */
async function writeWorkspaceFolders(roots: IdeRoot[]): Promise<void> {
  if (!roots.length) return;
  return enqueueFolderWrite(async () => {
    mutatingFolders = true;
    try {
      await initFile(WS_FILE, workspaceFileJson(roots), { overwrite: true });
      const existing = vscode.workspace.workspaceFolders ?? [];
      vscode.workspace.updateWorkspaceFolders(
        0,
        existing.length,
        ...roots.map((r) => ({
          uri: vscode.Uri.file(r.path),
          name: r.label,
        })),
      );
      await restoreIdeViews();
    } finally {
      mutatingFolders = false;
    }
  });
}

/** Restore Game / mods / Staging if the user mutates workbench folders. */
function lockWorkbenchFolders(): vscode.Disposable {
  return vscode.workspace.onDidChangeWorkspaceFolders(() => {
    if (mutatingFolders || !workbenchReady) return;
    if (foldersMatch(currentRoots)) return;
    void writeWorkspaceFolders(currentRoots);
  });
}

function hideEntries(commands: readonly string[]): { command: string; when: string }[] {
  return commands.map((command) => ({ command, when: "false" }));
}

/** Hide File/Help/theme/settings commands; Manage gear opens Workspace Settings. */
function registerWorkbenchLockMenus(): void {
  const hidden = hideEntries(HIDDEN_PALETTE_COMMANDS);
  registerExtension(
    {
      name: "pmt-workbench-lock",
      publisher: "pmt",
      version: "1.0.0",
      engines: { vscode: "*" },
      contributes: {
        commands: [
          { command: OPEN_WORKSPACE_SETTINGS, title: "Workspace Settings" },
        ],
        menus: {
          commandPalette: hidden,
          MenubarFileMenu: hidden,
          MenubarHelpMenu: hidden,
          "explorer/context": hidden,
        },
      },
    },
    ExtensionHostKind.LocalProcess,
  );
  vscode.commands.registerCommand(OPEN_WORKSPACE_SETTINGS, () => {
    const id = useWorkspaceStore().activeWorkspaceId;
    if (id) window.location.hash = `#/workspace/${id}/settings`;
  });
}

function tabUri(tab: vscode.Tab): vscode.Uri | undefined {
  const input = tab.input as { uri?: vscode.Uri } | undefined;
  const uri = input?.uri;
  if (!uri || uri.scheme !== "file") return undefined;
  return uri;
}

function collectOpenFiles(): { files: string[]; active: string } {
  const files: string[] = [];
  let active = "";
  for (const group of vscode.window.tabGroups.all) {
    for (const tab of group.tabs) {
      const uri = tabUri(tab);
      if (!uri) continue;
      const path = uri.fsPath;
      if (!files.includes(path)) files.push(path);
      if (tab.isActive) active = path;
    }
  }
  return { files, active };
}

function pathUnderRoots(path: string): boolean {
  return currentRoots.some((r) => {
    const a = normPath(path);
    const b = normPath(r.path);
    return a === b || a.startsWith(`${b}/`);
  });
}

async function flushIdeSession(): Promise<void> {
  if (useIdeShellStore().mergeReview) return;
  if (!persistTabsForMounted || !mountedWorkspaceId || restoringTabs) return;
  const { files, active } = collectOpenFiles();
  try {
    await SaveIdeSession(mountedWorkspaceId, files, active);
  } catch {
    /* persist is best-effort */
  }
}

function scheduleIdeSessionSave(): void {
  if (!persistTabsForMounted || restoringTabs) return;
  if (saveTabsTimer) clearTimeout(saveTabsTimer);
  saveTabsTimer = setTimeout(() => {
    saveTabsTimer = null;
    void flushIdeSession();
  }, 1000);
}

async function closeAllEditors(): Promise<void> {
  try {
    await vscode.commands.executeCommand("workbench.action.closeAllEditors");
  } catch {
    /* command missing */
  }
}

async function restoreIdeTabs(session: IdeSessionOpts): Promise<void> {
  if (!session.persistTabs) return;
  restoringTabs = true;
  try {
    const files = session.openFiles.filter((p) => pathUnderRoots(p));
    for (const path of files) {
      const uri = vscode.Uri.file(path);
      try {
        await vscode.workspace.fs.stat(uri);
        await vscode.window.showTextDocument(uri, {
          preview: false,
          preserveFocus: true,
        });
      } catch {
        /* missing file */
      }
    }
    if (session.activeFile && pathUnderRoots(session.activeFile)) {
      try {
        await vscode.window.showTextDocument(
          vscode.Uri.file(session.activeFile),
          { preview: false },
        );
      } catch {
        /* missing active file */
      }
    }
  } finally {
    restoringTabs = false;
  }
}

function syncEditorReadOnly(ed: monaco.editor.ICodeEditor): void {
  const model = ed.getModel();
  if (!model) return;
  const path = model.uri.fsPath || model.uri.path;
  ed.updateOptions({ readOnly: fsProvider.isReadOnly(path) });
}

function hookEditorReadOnly(): void {
  monaco.editor.onDidCreateEditor((ed) => {
    syncEditorReadOnly(ed);
    ed.onDidChangeModel(() => syncEditorReadOnly(ed));
  });
}

function hookIdeSessionPersist(): void {
  if (persistHooked) return;
  persistHooked = true;
  vscode.window.tabGroups.onDidChangeTabs(() => scheduleIdeSessionSave());
  const onHide = () => {
    if (document.visibilityState === "hidden") void flushIdeSession();
  };
  document.addEventListener("visibilitychange", onHide);
  document.addEventListener("pagehide", () => void flushIdeSession());
}

function lockKeybindingsJson(): string {
  const unbind = (key: string, command: string) => ({
    key,
    command: `-${command}`,
  });
  return JSON.stringify([
    { key: "space", command: "-list.toggleExpand" },
    { key: "space", command: "-list.stickyScrolltoggleExpand" },
    unbind("ctrl+o", "workbench.action.files.openFile"),
    unbind("cmd+o", "workbench.action.files.openFile"),
    unbind("ctrl+o", "workbench.action.files.openFileFolder"),
    unbind("cmd+o", "workbench.action.files.openFileFolder"),
    unbind("ctrl+k ctrl+o", "workbench.action.files.openFolder"),
    unbind("cmd+k cmd+o", "workbench.action.files.openFolder"),
    unbind("ctrl+r", "workbench.action.openRecent"),
    unbind("cmd+r", "workbench.action.openRecent"),
    unbind("ctrl+n", "workbench.action.files.newUntitledFile"),
    unbind("cmd+n", "workbench.action.files.newUntitledFile"),
    unbind("ctrl+,", "workbench.action.openSettings"),
    unbind("cmd+,", "workbench.action.openSettings"),
    unbind("ctrl+k ctrl+s", "workbench.action.openGlobalKeybindings"),
    unbind("cmd+k cmd+s", "workbench.action.openGlobalKeybindings"),
    unbind("ctrl+k ctrl+t", "workbench.action.selectTheme"),
    unbind("cmd+k cmd+t", "workbench.action.selectTheme"),
  ]);
}

async function runInitialize(theme: string): Promise<void> {
  if (!containerEl || !partRefs) {
    throw new Error("Workbench layout refs not set");
  }
  if (!currentRoots.length) {
    throw new Error("Workbench init requires at least one workspace folder");
  }
  setupWorkers();
  await createIndexedDBProviders();

  const mem = new RegisteredFileSystemProvider(false);
  mem.registerFile(new RegisteredMemoryFile(WS_FILE, workspaceFileJson(currentRoots)));
  registerFileSystemOverlay(1, mem);
  registerFileSystemOverlay(2, fsProvider);

  await initUserConfiguration(
    JSON.stringify(
      {
        ...workbenchSettingsForTheme(theme),
        "workbench.startupEditor": "none",
        "workbench.activity.showAccounts": false,
        "window.title": "PMT${separator}${activeEditorShort}",
        "files.autoSave": "off",
        "editor.acceptSuggestionOnCommitCharacter": false,
      },
      null,
      2,
    ),
  );
  if (!monacoLifecycle.servicesInitialized) {
    await initUserKeybindings(lockKeybindingsJson());
  }

  const options = constructOptions();
  if (!monacoLifecycle.serviceInitializedBarrier.isOpen()) {
    try {
      await initializeMonacoService(
        leanServices(), containerEl, options, envOpts,
      );
    } catch (err) {
      if (!isAlreadyInitialized(err)) throw err;
    }
  }
  attachIdeParts();
  await monacoLifecycle.waitServicesReady();

  registerWorkbenchLockMenus();
  hookEditorReadOnly();
  hookIdeSessionPersist();
  await registerParadoxLanguages();
  langClient?.dispose();
  langClient = registerLanguageClient(
    () => useWorkspaceStore().activeWorkspaceId ?? "",
  );
  themeLock?.dispose();
  themeLock = lockWorkbenchTheme(() => currentWorkbenchTheme());
  rootDeco?.dispose();
  rootDeco = registerRootDecorations();
  setDecoratedRoots(currentRoots);
  await applyWorkbenchTheme(theme);
  if (!foldersMatch(currentRoots)) {
    await writeWorkspaceFolders(currentRoots);
  }
  folderLock?.dispose();
  folderLock = lockWorkbenchFolders();
  await restoreIdeViews();
  workbenchReady = true;
  readyResolve();
}

/** Refresh explorer colors without remounting folders. */
export function refreshIdeRootDecorations(roots: IdeRoot[]): void {
  currentRoots = roots;
  fsProvider.setRoots(roots);
  setDecoratedRoots(roots);
  syncMountedEditors();
}

/**
 * Push app workspace folders into the workbench. First call initializes
 * monaco with those folders already in the .code-workspace file.
 */
export async function setWorkbenchRoots(
  roots: IdeRoot[],
  themeName: string,
  session?: IdeSessionOpts,
): Promise<void> {
  return enqueueRoots(() => applyWorkbenchRoots(roots, themeName, session));
}

async function applyWorkbenchRoots(
  roots: IdeRoot[],
  themeName: string,
  session?: IdeSessionOpts,
): Promise<void> {
  if (!roots.length) {
    throw new Error("No workspace folders to mount");
  }

  const first = !!session && !mountedWorkspaceId;
  const switching =
    !!session &&
    !!mountedWorkspaceId &&
    session.workspaceId !== mountedWorkspaceId;
  if (switching) {
    await flushIdeSession();
    await closeAllEditors();
  }

  currentRoots = roots;
  fsProvider.setRoots(roots);
  setDecoratedRoots(roots);
  syncMountedEditors();

  await refsPromise;
  if (!containerEl || !partRefs) {
    throw new Error("Workbench layout refs not set");
  }

  if (!initPromise) {
    initPromise = runInitialize(themeName).catch((err) => {
      initPromise = null;
      throw err;
    });
  }
  await initPromise;

  if (!foldersMatch(roots)) {
    await writeWorkspaceFolders(roots);
  }
  await applyWorkbenchTheme(themeName);

  if (session) {
    mountedWorkspaceId = session.workspaceId;
    persistTabsForMounted = session.persistTabs;
    if ((first || switching) && session.persistTabs) {
      await restoreIdeTabs(session);
    }
  }
}
