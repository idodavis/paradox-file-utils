/**
 * High-level IDE commands used by Nuxt pages (open file/diff/merge, set roots).
 */
import { nextTick } from "vue";
import { useToast } from "@nuxt/ui/composables/useToast";
import * as monaco from "monaco-editor";
import * as vscode from "vscode";
import { EnsureSession } from "@services/sessionservice";
import { GetIdeRoots } from "@services/workspaceservice";
import {
  ReadFileBase64,
  StatPath,
  WriteFileBase64,
} from "@services/fileservice";
import { currentWorkbenchTheme } from "./colorThemes";
import {
  isWorkbenchReady,
  setMergeChrome,
  setWorkbenchRoots,
  whenLayoutRefs,
  whenWorkbenchReady,
} from "./workbenchHost";
import { normFs, type IdeRoot } from "./fsBridge";
import { useIdeShellStore } from "../stores/ideShell";
import { useWorkspaceStore } from "../stores/workspace";

/** Open a file in the workbench at an optional 1-based line. */
export async function openFile(path: string, line?: number): Promise<void> {
  await whenWorkbenchReady();
  if (!isWorkbenchReady()) return;
  const uri = monaco.Uri.file(path);
  const doc = await vscode.workspace.openTextDocument(uri);
  const editor = await vscode.window.showTextDocument(doc, { preview: false });
  if (line && line > 0) {
    const pos = new vscode.Position(line - 1, 0);
    editor.selection = new vscode.Selection(pos, pos);
    editor.revealRange(
      new vscode.Range(pos, pos),
      vscode.TextEditorRevealType.InCenter,
    );
  }
}

/** Open a side-by-side diff of two paths. */
export async function openDiff(
  leftPath: string,
  rightPath: string,
  title?: string,
): Promise<void> {
  await whenWorkbenchReady();
  if (!isWorkbenchReady()) {
    toastFail("Editor is not ready.");
    return;
  }
  const left = monaco.Uri.file(leftPath);
  const right = monaco.Uri.file(rightPath);
  await vscode.commands.executeCommand(
    "vscode.diff",
    left,
    right,
    title ?? `${leftPath} ↔ ${rightPath}`,
  );
}

function parentDir(path: string): string {
  const i = Math.max(path.lastIndexOf("/"), path.lastIndexOf("\\"));
  return i > 0 ? path.slice(0, i) : path;
}

function covered(path: string, roots: IdeRoot[]): boolean {
  const a = normFs(path);
  return roots.some((r) => {
    const b = normFs(r.path);
    return a === b || a.startsWith(`${b}/`);
  });
}

function extraRoots(files: string[], base: IdeRoot[]): IdeRoot[] {
  const out: IdeRoot[] = [];
  const seen = new Set<string>();
  for (const [i, file] of files.entries()) {
    const dir = parentDir(file);
    const key = normFs(dir);
    if (!dir || seen.has(key) || covered(file, base) || covered(dir, base)) {
      continue;
    }
    seen.add(key);
    out.push({
      label: `Review ${i + 1}`,
      path: dir,
      readOnly: false,
      kind: "mod",
      origin: `review-${i}`,
    });
  }
  return out;
}

let reviewUris: string[] = [];
let reviewBaseRoots: IdeRoot[] = [];
let reviewBack: (() => void) | undefined;

async function closeReviewTabs(paths: string[]): Promise<void> {
  const want = new Set(paths.map((p) => monaco.Uri.file(p).toString()));
  const tabs: vscode.Tab[] = [];
  for (const group of vscode.window.tabGroups.all) {
    for (const tab of group.tabs) {
      const input = tab.input as {
        uri?: vscode.Uri
        original?: vscode.Uri
        modified?: vscode.Uri
        base?: { uri?: vscode.Uri } | vscode.Uri
        input1?: { uri?: vscode.Uri } | vscode.Uri
        input2?: { uri?: vscode.Uri } | vscode.Uri
        result?: vscode.Uri
      } | undefined;
      if (!input) continue;
      const uris = [
        input.uri,
        input.original,
        input.modified,
        input.result,
        input.base && "uri" in input.base ? input.base.uri : input.base,
        input.input1 && "uri" in input.input1 ? input.input1.uri : input.input1,
        input.input2 && "uri" in input.input2 ? input.input2.uri : input.input2,
      ];
      if (uris.some((u) => u && want.has(u.toString()))) tabs.push(tab);
    }
  }
  if (tabs.length) await vscode.window.tabGroups.close(tabs);
}

async function restoreAfterReview(): Promise<void> {
  setMergeChrome(false);
  if (isWorkbenchReady()) {
    await closeReviewTabs(reviewUris);
    const base = reviewBaseRoots;
    if (base.length) {
      await setWorkbenchRoots(base, currentWorkbenchTheme());
    }
  }
  reviewUris = [];
  reviewBaseRoots = [];
}

function toastFail(title: string, err?: unknown): void {
  const description = err instanceof Error ? err.message : String(err ?? title);
  useToast().add({ title, description, color: "error" });
}

/** Create an empty or copied file so the merge editor has a result path on disk. */
async function ensureDiskFile(path: string, copyFrom?: string): Promise<void> {
  const st = await StatPath(path);
  if (st.exists) return;
  if (copyFrom) {
    const src = await ReadFileBase64(copyFrom);
    if (src.exists && src.b64) {
      await WriteFileBase64(path, src.b64);
      return;
    }
  }
  await WriteFileBase64(path, "");
}

/** Boot workbench if needed, hide chrome, append temp roots, then open. */
export async function startMergeOverlay(opts: {
  files: string[];
  label: string;
  back?: () => void;
  open: () => Promise<void>;
}): Promise<void> {
  const ws = useWorkspaceStore();
  const shell = useIdeShellStore();
  try {
    if (ws.activeWorkspaceId) await EnsureSession(ws.activeWorkspaceId);
    const base = (ws.activeWorkspaceId
      ? ((await GetIdeRoots(ws.activeWorkspaceId))?.roots ?? [])
      : []);
    reviewBaseRoots = base;
    reviewUris = opts.files;
    const extras = extraRoots(opts.files, base);
    const roots = [...base, ...extras];
    if (!roots.length) {
      toastFail("No workspace folders to open for review.");
      return;
    }
    reviewBack = opts.back;
    shell.beginMergeReview(opts.label);
    await nextTick();
    await whenLayoutRefs();
    await setWorkbenchRoots(roots, currentWorkbenchTheme());
    await whenWorkbenchReady();
    setMergeChrome(true);
    await opts.open();
    setMergeChrome(true);
  } catch (err) {
    await endMergeOverlay();
    if (err instanceof Error && err.message.includes("already initialized")) {
      return;
    }
    toastFail("Could not open review", err);
  }
}

/** Hide merge review, restore IDE roots, then run the overlay back callback. */
export async function endMergeOverlay(): Promise<void> {
  const back = reviewBack;
  reviewBack = undefined;
  useIdeShellStore().endMergeReview();
  await restoreAfterReview();
  back?.();
}

/** Open the VS Code merge editor. No ancestor on disk: pass A as base. */
export async function openMergeEditor(opts: {
  input1: string;
  input2: string;
  result: string;
}): Promise<void> {
  await whenWorkbenchReady();
  if (!isWorkbenchReady()) {
    toastFail("Editor is not ready.");
    return;
  }
  await ensureDiskFile(opts.result, opts.input1);
  const a = monaco.Uri.file(opts.input1);
  const b = monaco.Uri.file(opts.input2);
  const out = monaco.Uri.file(opts.result);
  const cmds = await vscode.commands.getCommands(true);
  const names = ["_open.mergeEditor", "vscode.openMergeEditor", "_workbench.openMergeEditor"];
  const cmd = names.find((n) => cmds.includes(n)) ?? "";
  if (!cmd) {
    toastFail("Merge editor is not available.");
    return;
  }
  await vscode.commands.executeCommand(cmd, {
    base: a.toString(),
    input1: { uri: a.toString(), title: "A" },
    input2: { uri: b.toString(), title: "B" },
    output: out.toString(),
  });
}
