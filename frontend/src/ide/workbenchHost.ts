/**
 * Singleton monaco-vscode ViewsService host (init once per app lifetime).
 *
 * IdeWorkbenchLayout.vue owns attachPart DOM refs; WorkspaceIdePage pushes app roots.
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
import { servicesInitialized, waitServicesReady } from "@codingame/monaco-vscode-api/lifecycle";
import getViewsServiceOverride, {
  attachPart,
  isEditorPartVisible,
  isPartVisibile,
  onPartVisibilityChange,
  setPartVisibility,
  Parts,
} from "@codingame/monaco-vscode-views-service-override";
import getQuickAccessServiceOverride from "@codingame/monaco-vscode-quickaccess-service-override";
import getConfigurationServiceOverride, {
  initUserConfiguration,
  reinitializeWorkspace,
} from "@codingame/monaco-vscode-configuration-service-override";
import getKeybindingsServiceOverride from "@codingame/monaco-vscode-keybindings-service-override";
import getModelServiceOverride from "@codingame/monaco-vscode-model-service-override";
import getNotificationServiceOverride from "@codingame/monaco-vscode-notifications-service-override";
import getDialogsServiceOverride from "@codingame/monaco-vscode-dialogs-service-override";
import getTextmateServiceOverride from "@codingame/monaco-vscode-textmate-service-override";
import getThemeServiceOverride from "@codingame/monaco-vscode-theme-service-override";
import getLanguagesServiceOverride from "@codingame/monaco-vscode-languages-service-override";
import getSearchServiceOverride from "@codingame/monaco-vscode-search-service-override";
import getMarkersServiceOverride from "@codingame/monaco-vscode-markers-service-override";
import getStorageServiceOverride from "@codingame/monaco-vscode-storage-service-override";
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
// Material + Paradox icon theme (builtin before initialize).
import "./fileIcons";
import "vscode/localExtensionHost";
import * as monaco from "monaco-editor";
import * as vscode from "vscode";
import { WailsFileSystemProvider, type IdeRoot } from "./fsBridge";
import { workspaceFileJson } from "./workspaceFolders";
import { applyWorkbenchTheme, workbenchSettingsForTheme, lockWorkbenchTheme } from "./themeBridge";
import { registerParadoxLanguages } from "./paradoxLanguages";
import { registerLanguageClient } from "./languageClient";
import { registerRootDecorations, setDecoratedRoots } from "./rootDecorations";
import { useWorkspaceStore } from "../stores/workspace";
import "./explorerLayout.css";

let workbenchReady = false;

/** Whether the workbench has finished initialize(). */
export function isWorkbenchReady(): boolean {
  return workbenchReady;
}

/** attachPart DOM refs owned by IdeWorkbenchLayout.vue. */
export type IdePartRefs = {
  root: HTMLElement;
  activityBar: HTMLElement;
  sidebar: HTMLElement;
  editor: HTMLElement;
  panel: HTMLElement;
};

/** Options applied on first workbench initialize. */
export type WorkbenchInitOpts = {
  theme?: string;
};

const WS_ID = "pmt-app-workspace";
const WS_FILE = monaco.Uri.file("/pmt.code-workspace");

/** Command palette entries owned by Nuxt / the app workspace, not the workbench. */
const HIDDEN_PALETTE_COMMANDS = [
  "workbench.action.files.openFolder",
  "workbench.action.addRootFolder",
  "workbench.action.removeRootFolder",
  "workbench.action.closeFolder",
  "workbench.action.selectTheme",
  "workbench.actions.manageAccounts",
  "workbench.actions.accounts",
] as const;

const ATTACHED_PARTS = [
  Parts.ACTIVITYBAR_PART,
  Parts.SIDEBAR_PART,
  Parts.EDITOR_PART,
  Parts.PANEL_PART,
] as const;

let initPromise: Promise<void> | null = null;
let containerEl: HTMLElement | null = null;
let partRefs: IdePartRefs | null = null;
const readyWaiters: Array<{
  resolve: () => void;
  reject: (err: unknown) => void;
}> = [];
const ready = (): boolean => workbenchReady;
const fsProvider = new WailsFileSystemProvider();
let currentRoots: IdeRoot[] = [];
let langClient: { dispose(): void } | null = null;
let themeLock: { dispose(): void } | null = null;
let folderLock: { dispose(): void } | null = null;
let rootDeco: { dispose(): void } | null = null;
let mutatingFolders = false;
let partDisposables: IDisposable[] = [];
let folderWriteChain: Promise<void> = Promise.resolve();

/** Resolves once the workbench has finished initialize(). */
export function whenWorkbenchReady(): Promise<void> {
  if (ready()) return Promise.resolve();
  return new Promise((resolve, reject) => {
    let settled = false;
    let timer = 0;
    const entry = {
      resolve: () => {
        if (settled) return;
        settled = true;
        clearTimeout(timer);
        resolve();
      },
      reject: (err: unknown) => {
        if (settled) return;
        settled = true;
        clearTimeout(timer);
        const i = readyWaiters.indexOf(entry);
        if (i >= 0) readyWaiters.splice(i, 1);
        reject(err);
      },
    };
    timer = window.setTimeout(() => {
      entry.reject(new Error("Workbench init timed out"));
    }, 30_000);
    readyWaiters.push(entry);
  });
}

function markReady(): void {
  if (workbenchReady) return;
  workbenchReady = true;
  while (readyWaiters.length) readyWaiters.shift()?.resolve();
}

function failReady(err: unknown): void {
  const waiters = readyWaiters.splice(0);
  for (const w of waiters) w.reject(err);
}

function setupWorkers(): void {
  const w = window as Window & {
    MonacoEnvironment?: {
      getWorker: (_: string, label: string) => Worker;
    };
  };
  w.MonacoEnvironment = {
    getWorker(_: string, label: string) {
      if (label === "TextMateWorker") {
        return new Worker(new URL("@codingame/monaco-vscode-textmate-service-override/worker", import.meta.url), {
          type: "module",
        });
      }
      if (label === "LocalFileSearchWorker") {
        return new Worker(new URL("@codingame/monaco-vscode-search-service-override/worker", import.meta.url), {
          type: "module",
        });
      }
      return new Worker(new URL("monaco-editor/esm/vs/editor/editor.worker.js", import.meta.url), { type: "module" });
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
    ...getStorageServiceOverride(),
    ...getLifecycleServiceOverride(),
    ...getEnvironmentServiceOverride(),
    ...getWorkingCopyServiceOverride(),
    ...getMultiDiffEditorServiceOverride(),
    ...getExplorerServiceOverride(),
    ...getViewsServiceOverride(),
    ...getQuickAccessServiceOverride({
      isKeybindingConfigurationVisible: isEditorPartVisible,
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
    // Use view IDs (not viewlet IDs): `workbench.view.explorer` is a container and
    // is ignored by defaultLayout, which left Search as the restored sidebar.
    defaultLayout: {
      views: [{ id: "workbench.explorer.fileView" }, { id: "workbench.panel.markers.view" }],
      force: true,
    },
    configurationDefaults: {
      "window.titleBarStyle": "native",
      "workbench.activityBar.location": "default",
      "workbench.startupEditor": "none",
      "workbench.colorTheme": "PMT",
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
    },
  };
}

const envOpts = { userHome: monaco.Uri.file("/") };

/** Bind ViewsService parts to layout refs and wire visibility listeners. */
export function attachIdeParts(refs: IdePartRefs): void {
  for (const disposable of partDisposables) {
    disposable.dispose();
  }
  partDisposables = [];

  const containers: Record<(typeof ATTACHED_PARTS)[number], HTMLElement> = {
    [Parts.ACTIVITYBAR_PART]: refs.activityBar,
    [Parts.SIDEBAR_PART]: refs.sidebar,
    [Parts.EDITOR_PART]: refs.editor,
    [Parts.PANEL_PART]: refs.panel,
  };

  for (const part of ATTACHED_PARTS) {
    const container = containers[part];
    partDisposables.push(attachPart(part, container));
    if (!isPartVisibile(part)) {
      container.style.display = "none";
    }
    partDisposables.push(
      onPartVisibilityChange(part, (visible) => {
        container.style.display = visible ? "block" : "none";
      }),
    );
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

/** Re-open Explorer and Problems after a folder remount or visibility restore. */
async function restoreIdeViews(): Promise<void> {
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

/** Relayout parts and restore views after the host is shown again. */
export async function revealWorkbench(): Promise<void> {
  if (!ready()) return;
  await folderWriteChain;
  window.dispatchEvent(new Event("resize"));
  await restoreIdeViews();
}

/** Reload multi-root folders from the in-memory .code-workspace file. */
async function reloadWorkspaceFromFile(): Promise<void> {
  await reinitializeWorkspace({
    id: WS_ID,
    configPath: WS_FILE,
  });
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

/** Write app workspace roots into the workbench (suppresses folder-lock). */
async function writeWorkspaceFolders(roots: IdeRoot[]): Promise<void> {
  if (!roots.length) return;
  return enqueueFolderWrite(async () => {
    mutatingFolders = true;
    try {
      await initFile(WS_FILE, workspaceFileJson(roots), { overwrite: true });
      const existing = vscode.workspace.workspaceFolders ?? [];
      if (existing.length === 0) {
        await reloadWorkspaceFromFile();
        if ((vscode.workspace.workspaceFolders?.length ?? 0) === 0) {
          vscode.workspace.updateWorkspaceFolders(
            0,
            0,
            ...roots.map((r) => ({
              uri: vscode.Uri.file(r.path),
              name: r.label,
            })),
          );
        }
      } else {
        vscode.workspace.updateWorkspaceFolders(
          0,
          existing.length,
          ...roots.map((r) => ({
            uri: vscode.Uri.file(r.path),
            name: r.label,
          })),
        );
      }
      await restoreIdeViews();
    } finally {
      mutatingFolders = false;
    }
  });
}

/** Restore Game / mods / Staging if the user mutates workbench folders. */
function lockWorkbenchFolders(): vscode.Disposable {
  return vscode.workspace.onDidChangeWorkspaceFolders(() => {
    if (mutatingFolders || !ready()) return;
    if (foldersMatch(currentRoots)) return;
    void writeWorkspaceFolders(currentRoots);
  });
}

/** Hide Open Folder / Add Root / Select Theme from the command palette. */
function registerWorkbenchLockMenus(): void {
  registerExtension(
    {
      name: "pmt-workbench-lock",
      publisher: "pmt",
      version: "1.0.0",
      engines: { vscode: "*" },
      contributes: {
        menus: {
          commandPalette: HIDDEN_PALETTE_COMMANDS.map((command) => ({
            command,
            when: "false",
          })),
        },
      },
    },
    ExtensionHostKind.LocalProcess,
  );
}

async function runInitialize(theme: string): Promise<void> {
  if (!containerEl || !partRefs) {
    throw new Error("Workbench layout refs not set");
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
        "window.title": "PMT${separator}${activeEditorShort}",
        "files.autoSave": "off",
      },
      null,
      2,
    ),
  );

  const options = constructOptions();
  if (servicesInitialized) {
    attachIdeParts(partRefs);
    await waitServicesReady();
  } else {
    await initializeMonacoService(leanServices(), containerEl, options, envOpts);
    attachIdeParts(partRefs);
  }

  registerWorkbenchLockMenus();
  await registerParadoxLanguages();
  langClient?.dispose();
  langClient = registerLanguageClient(
    () => useWorkspaceStore().activeWorkspaceId ?? "",
  );
  themeLock?.dispose();
  themeLock = lockWorkbenchTheme(() => document.documentElement.dataset.theme || theme);
  rootDeco?.dispose();
  rootDeco = registerRootDecorations();
  setDecoratedRoots(currentRoots);
  await applyWorkbenchTheme(theme);
  if (currentRoots.length) {
    await writeWorkspaceFolders(currentRoots);
  }
  folderLock?.dispose();
  folderLock = lockWorkbenchFolders();
  await restoreIdeViews();
  markReady();
}

/** Whether two attachPart ref records point at the same DOM nodes. */
function samePartRefs(a: IdePartRefs, b: IdePartRefs): boolean {
  return (
    a.root === b.root &&
    a.activityBar === b.activityBar &&
    a.sidebar === b.sidebar &&
    a.editor === b.editor &&
    a.panel === b.panel
  );
}

/**
 * Initialize the ViewsService workbench (once).
 * Must be called with attachPart refs from IdeWorkbenchLayout.vue.
 */
export async function ensureWorkbench(refs?: IdePartRefs | null, opts?: WorkbenchInitOpts): Promise<void> {
  const refsChanged = !!(refs && partRefs && !samePartRefs(partRefs, refs));
  if (refs) {
    containerEl = refs.root;
    partRefs = refs;
  }
  const theme = opts?.theme ?? "PMT";

  if (!containerEl || !partRefs) {
    throw new Error("Workbench layout refs not set");
  }

  if (!initPromise) {
    initPromise = runInitialize(theme).catch((err) => {
      initPromise = null;
      failReady(err);
      throw err;
    });
  }
  await initPromise;

  if (ready() && refsChanged && partRefs) {
    attachIdeParts(partRefs);
  }

  if (ready() && opts?.theme) {
    await applyWorkbenchTheme(theme);
  }
}

/** Remount workspace folders to match the active PMT workspace. */
export async function setWorkbenchRoots(roots: IdeRoot[], themeName: string): Promise<void> {
  currentRoots = roots;
  fsProvider.setRoots(roots);
  setDecoratedRoots(roots);

  if (!ready()) {
    // Roots are picked up during initialize when the host finishes.
    await whenWorkbenchReady();
  }

  if (!foldersMatch(roots)) {
    await writeWorkspaceFolders(roots);
  }
  await applyWorkbenchTheme(themeName);
}
