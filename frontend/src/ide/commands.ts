/**
 * High-level IDE commands used by Nuxt pages (open file/diff, set roots).
 */
import * as monaco from "monaco-editor";
import * as vscode from "vscode";
import { whenWorkbenchReady, isWorkbenchReady } from "./workbenchHost";

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
  if (!isWorkbenchReady()) return;
  const left = monaco.Uri.file(leftPath);
  const right = monaco.Uri.file(rightPath);
  await vscode.commands.executeCommand(
    "vscode.diff",
    left,
    right,
    title ?? `${leftPath} ↔ ${rightPath}`,
  );
}

/** Reveal a path in the explorer view. */
export async function revealInExplorer(path: string): Promise<void> {
  await whenWorkbenchReady();
  if (!isWorkbenchReady()) return;
  await vscode.commands.executeCommand(
    "revealInExplorer",
    monaco.Uri.file(path),
  );
}

/** Open multiple diffs via multi-diff editor when available. */
export async function openMultiDiff(
  pairs: { left: string; right: string; label?: string }[],
): Promise<void> {
  await whenWorkbenchReady();
  if (!isWorkbenchReady() || !pairs.length) return;
  if (pairs.length === 1) {
    const p = pairs[0]!;
    await openDiff(p.left, p.right, p.label);
    return;
  }
  try {
    await vscode.commands.executeCommand(
      "vscode.changes",
      "PMT conflicts",
      pairs.map((p) => [
        monaco.Uri.file(p.left),
        monaco.Uri.file(p.right),
        p.label ?? `${p.left} ↔ ${p.right}`,
      ]),
    );
  } catch {
    for (const p of pairs) {
      await openDiff(p.left, p.right, p.label);
    }
  }
}
