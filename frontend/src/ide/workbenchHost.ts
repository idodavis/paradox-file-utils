/**
 * Singleton monaco-vscode workbench host (init once per app lifetime).
 *
 * App.vue owns the DOM container; WorkspaceIdePage pushes app workspace roots.
 */
import {
  initialize as initializeMonacoService,
  type IEditorOverrideServices,
  type IWorkbenchConstructionOptions,
  LogLevel,
} from "@codingame/monaco-vscode-api";
import {
  initialize as bindWorkbenchContainer,
} from "@codingame/monaco-vscode-api/workbench";
import {
  servicesInitialized,
  waitServicesReady,
} from "@codingame/monaco-vscode-api/lifecycle";
import getWorkbenchServiceOverride from "@codingame/monaco-vscode-workbench-service-override";
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
import getOutlineServiceOverride from "@codingame/monaco-vscode-outline-service-override";
import getMultiDiffEditorServiceOverride from "@codingame/monaco-vscode-multi-diff-editor-service-override";
import getExplorerServiceOverride from "@codingame/monaco-vscode-explorer-service-override";
import getStatusBarServiceOverride from "@codingame/monaco-vscode-view-status-bar-service-override";
import getPreferencesServiceOverride from "@codingame/monaco-vscode-preferences-service-override";
import {
  createIndexedDBProviders,
  registerFileSystemOverlay,
  RegisteredFileSystemProvider,
  RegisteredMemoryFile,
  initFile,
} from "@codingame/monaco-vscode-files-service-override";
import "@codingame/monaco-vscode-theme-defaults-default-extension";
import "vscode/localExtensionHost";
import * as monaco from "monaco-editor";
import * as vscode from "vscode";
import { WailsFileSystemProvider, type IdeRoot } from "./fsBridge";
import { workspaceFileJson } from "./workspaceFolders";
import {
  applyWorkbenchTheme,
  colorCustomizationsForTheme,
  lockWorkbenchTheme,
  setThemeHost,
} from "./themeBridge";
import { registerParadoxLanguages } from "./paradoxLanguages";
import { registerLanguageClient } from "./languageClient";

/** Options applied on first workbench initialize. */
export type WorkbenchInitOpts = {
  theme?: string;
  editorFontSize?: number;
};

const WS_ID = "pmt-app-workspace";
const WS_FILE = monaco.Uri.file("/pmt.code-workspace");

let ready = false;
let initPromise: Promise<void> | null = null;
let containerEl: HTMLElement | null = null;
const readyWaiters: Array<() => void> = [];
const fsProvider = new WailsFileSystemProvider();
let currentRoots: IdeRoot[] = [];
let langClient: { dispose(): void } | null = null;
let themeLock: { dispose(): void } | null = null;

/** Whether the workbench has finished initialize(). */
export function isWorkbenchReady(): boolean {
  return ready;
}

/** Resolves once the workbench has finished initialize(). */
export function whenWorkbenchReady(): Promise<void> {
  if (ready) return Promise.resolve();
  return new Promise((resolve) => {
    readyWaiters.push(resolve);
  });
}

/** Current IDE roots mounted in the FS bridge. */
export function getIdeRoots(): IdeRoot[] {
  return currentRoots;
}

function markReady(): void {
  if (ready) return;
  ready = true;
  while (readyWaiters.length) readyWaiters.shift()?.();
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
        return new Worker(
          new URL(
            "@codingame/monaco-vscode-textmate-service-override/worker",
            import.meta.url,
          ),
          { type: "module" },
        );
      }
      if (label === "LocalFileSearchWorker") {
        return new Worker(
          new URL(
            "@codingame/monaco-vscode-search-service-override/worker",
            import.meta.url,
          ),
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
    ...getPreferencesServiceOverride(),
    ...getOutlineServiceOverride(),
    ...getStatusBarServiceOverride(),
    ...getSearchServiceOverride(),
    ...getMarkersServiceOverride(),
    ...getStorageServiceOverride(),
    ...getLifecycleServiceOverride(),
    ...getEnvironmentServiceOverride(),
    ...getWorkingCopyServiceOverride(),
    ...getMultiDiffEditorServiceOverride(),
    ...getExplorerServiceOverride(),
    ...getWorkbenchServiceOverride(),
    ...getQuickAccessServiceOverride({
      isKeybindingConfigurationVisible: () => true,
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
    },
    configurationDefaults: {
      "window.titleBarStyle": "native",
      "workbench.activityBar.location": "default",
      "workbench.startupEditor": "none",
      "workbench.colorTheme": "Default Dark Modern",
      "workbench.iconTheme": "pmt-icons",
      "workbench.tree.indent": 14,
      "workbench.tree.renderIndentGuides": "always",
    },
  };
}

const envOpts = { userHome: monaco.Uri.file("/") };

/** Reload multi-root folders from the in-memory .code-workspace file. */
async function reloadWorkspaceFromFile(): Promise<void> {
  await reinitializeWorkspace({
    id: WS_ID,
    configPath: WS_FILE,
  });
}

async function runInitialize(
  theme: string,
  editorFontSize: number,
): Promise<void> {
  if (!containerEl) {
    throw new Error("Workbench container not set");
  }
  setThemeHost(containerEl);
  setupWorkers();
  await createIndexedDBProviders();

  const mem = new RegisteredFileSystemProvider(false);
  mem.registerFile(
    new RegisteredMemoryFile(
      WS_FILE,
      workspaceFileJson(currentRoots),
    ),
  );
  registerFileSystemOverlay(1, mem);
  registerFileSystemOverlay(2, fsProvider);

  await initUserConfiguration(
    JSON.stringify(
      {
        ...colorCustomizationsForTheme(theme, editorFontSize),
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
    // Prior init (HMR / race): re-bind container + workspace id, don't call initialize twice.
    bindWorkbenchContainer(containerEl, options, envOpts);
    await waitServicesReady();
  } else {
    await initializeMonacoService(
      leanServices(),
      containerEl,
      options,
      envOpts,
    );
  }

  await registerParadoxLanguages();
  langClient?.dispose();
  langClient = registerLanguageClient();
  themeLock?.dispose();
  themeLock = lockWorkbenchTheme(() => {
    const name =
      document.documentElement.dataset.theme || theme;
    return { themeName: name, editorFontSize };
  });
  markReady();
  await applyWorkbenchTheme(theme, editorFontSize);
  if (currentRoots.length) {
    await initFile(WS_FILE, workspaceFileJson(currentRoots), {
      overwrite: true,
    });
    await reloadWorkspaceFromFile();
  }
}

/**
 * Initialize the workbench into container (once).
 * Must be called with the host element from App.vue.
 */
export async function ensureWorkbench(
  container?: HTMLElement | null,
  opts?: WorkbenchInitOpts,
): Promise<void> {
  if (container) {
    containerEl = container;
    setThemeHost(container);
  }
  const theme = opts?.theme ?? "PMT";
  const editorFontSize = opts?.editorFontSize ?? 14;

  if (!containerEl) {
    throw new Error("Workbench container not set");
  }

  initPromise ??= runInitialize(theme, editorFontSize);
  await initPromise;

  if (ready && opts?.theme) {
    await applyWorkbenchTheme(theme, editorFontSize);
  }
}

/** Remount workspace folders to match the active PMT workspace. */
export async function setWorkbenchRoots(
  roots: IdeRoot[],
  themeName: string,
  editorFontSize: number,
): Promise<void> {
  currentRoots = roots;
  fsProvider.setRoots(roots);

  if (!ready) {
    // Roots are picked up during initialize when the host finishes.
    await whenWorkbenchReady();
  }

  await initFile(WS_FILE, workspaceFileJson(roots), { overwrite: true });
  await reloadWorkspaceFromFile();

  // If the workspace file reload left folders empty, splice them in directly.
  if ((vscode.workspace.workspaceFolders?.length ?? 0) === 0 && roots.length) {
    vscode.workspace.updateWorkspaceFolders(
      0,
      0,
      ...roots.map((r) => ({
        uri: vscode.Uri.file(r.path),
        name: r.label,
      })),
    );
  }

  await applyWorkbenchTheme(themeName, editorFontSize);
}
