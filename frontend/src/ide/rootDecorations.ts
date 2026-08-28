/**
 * Explorer color tags for Game / Mod / Staging workspace roots.
 *
 * Uses FileDecorationProvider badges + contributed theme colors (subtle).
 */
import * as vscode from "vscode";
import { registerExtension, ExtensionHostKind } from "@codingame/monaco-vscode-api/extensions";
import type { IdeRoot } from "./fsBridge";

/** Fixed Game root accent (muted teal). */
const GAME_COLOR_ID = "pmt.root.game";
/** Fixed Staging root accent (muted amber). */
const STAGING_COLOR_ID = "pmt.root.staging";
/** Soft mod palette — pick by stable path hash. */
const MOD_COLOR_IDS = [
  "pmt.root.mod0",
  "pmt.root.mod1",
  "pmt.root.mod2",
  "pmt.root.mod3",
  "pmt.root.mod4",
  "pmt.root.mod5",
  "pmt.root.mod6",
  "pmt.root.mod7",
] as const;

/** Default hex for contributed colors (work on dark + light themes). */
const COLOR_DEFAULTS: Record<string, string> = {
  [GAME_COLOR_ID]: "#5B9A8B",
  [STAGING_COLOR_ID]: "#B08948",
  [MOD_COLOR_IDS[0]]: "#7A8FB5",
  [MOD_COLOR_IDS[1]]: "#8F7AA8",
  [MOD_COLOR_IDS[2]]: "#6F9E7A",
  [MOD_COLOR_IDS[3]]: "#A88B6F",
  [MOD_COLOR_IDS[4]]: "#6F8F9E",
  [MOD_COLOR_IDS[5]]: "#9E6F8A",
  [MOD_COLOR_IDS[6]]: "#8A9E6F",
  [MOD_COLOR_IDS[7]]: "#7A7A9E",
};

let roots: IdeRoot[] = [];
let registered = false;
const changeEmitter = new vscode.EventEmitter<vscode.Uri | vscode.Uri[] | undefined>();

/** Normalize path for root identity comparison. */
function normPath(path: string): string {
  return path.replace(/\\/g, "/").replace(/\/+$/, "").toLowerCase();
}

/** Stable 0..n-1 index from path (not cryptographically random). */
function hashModIndex(path: string, n: number): number {
  let h = 2166136261;
  const s = normPath(path);
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return Math.abs(h) % n;
}

/** Theme color id for a root. */
function colorIdFor(root: IdeRoot): string {
  switch (root.kind) {
    case "game":
      return GAME_COLOR_ID;
    case "staging":
      return STAGING_COLOR_ID;
    case "mod":
      return MOD_COLOR_IDS[hashModIndex(root.path, MOD_COLOR_IDS.length)]!;
    default: {
      const _exhaustive: never = root.kind;
      return _exhaustive;
    }
  }
}

/** One-letter badge: G / S / first letter of mod name. */
function badgeFor(root: IdeRoot): string {
  switch (root.kind) {
    case "game":
      return "G";
    case "staging":
      return "S";
    case "mod": {
      const ch = root.label.trim().charAt(0).toUpperCase();
      return /[A-Z0-9]/.test(ch) ? ch : "M";
    }
    default: {
      const _exhaustive: never = root.kind;
      return _exhaustive;
    }
  }
}

/** Tooltip for the decoration badge. */
function tooltipFor(root: IdeRoot): string {
  switch (root.kind) {
    case "game":
      return "Game root";
    case "staging":
      return "Staging";
    case "mod":
      return `Mod: ${root.label}`;
    default: {
      const _exhaustive: never = root.kind;
      return _exhaustive;
    }
  }
}

/** Contribute pmt.root.* colors once. */
function registerRootColors(): void {
  registerExtension(
    {
      name: "pmt-root-colors",
      publisher: "pmt",
      version: "1.0.0",
      engines: { vscode: "*" },
      contributes: {
        colors: Object.entries(COLOR_DEFAULTS).map(([id, defaults]) => ({
          id,
          description: `PMT explorer root tag (${id})`,
          defaults: { dark: defaults, light: defaults, highContrast: defaults },
        })),
      },
    },
    ExtensionHostKind.LocalProcess,
  );
}

/** FileDecorationProvider for workspace root folders only. */
class RootDecorationProvider implements vscode.FileDecorationProvider {
  readonly onDidChangeFileDecorations = changeEmitter.event;

  provideFileDecoration(
    uri: vscode.Uri,
  ): vscode.ProviderResult<vscode.FileDecoration> {
    const path = normPath(uri.fsPath || uri.path);
    const root = roots.find((r) => normPath(r.path) === path);
    if (!root) return undefined;
    const deco = new vscode.FileDecoration(
      badgeFor(root),
      tooltipFor(root),
      new vscode.ThemeColor(colorIdFor(root)),
    );
    deco.propagate = false;
    return deco;
  }
}

/**
 * Register root color contributions + decoration provider (once).
 * Call after workbench services are ready.
 */
export function registerRootDecorations(): vscode.Disposable {
  if (registered) {
    return { dispose() {} };
  }
  registered = true;
  registerRootColors();
  const provider = new RootDecorationProvider();
  return vscode.window.registerFileDecorationProvider(provider);
}

/** Update roots used for decorations (call when workspace folders change). */
export function setDecoratedRoots(next: IdeRoot[]): void {
  roots = next;
  const uris = next.map((r) => vscode.Uri.file(r.path));
  changeEmitter.fire(uris.length ? uris : undefined);
}
