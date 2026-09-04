/**
 * Explorer color tags for Game / Mod workspace roots.
 *
 * Accent hex comes from Settings (`originHex`) via injected CSS, not
 * static ThemeColor ids.
 */
import * as vscode from "vscode";
import { registerExtension, ExtensionHostKind } from "@codingame/monaco-vscode-api/extensions";
import { ReadFileBase64 } from "@services/fileservice";
import { normFs, type IdeRoot } from "./fsBridge";
import { thumbMime, useWorkspaceStore } from "../stores/workspace";
import iconCk3 from "@assets/Icon_CK3.png?url";
import iconEu5 from "@assets/Icon_EUV.png?url";
import iconVic3 from "@assets/Icon_Vic3.png?url";

const GAME_ICONS: Record<string, string> = {
  ck3: iconCk3,
  eu5: iconEu5,
  vic3: iconVic3,
};

let thumbStyle: HTMLStyleElement | null = null;

/** Fixed Game root accent (muted teal). */
const GAME_COLOR_ID = "pmt.root.game";
/** Soft mod palette — wrap by SortOrder % 10. */
const MOD_COLOR_IDS = [
  "pmt.root.mod0",
  "pmt.root.mod1",
  "pmt.root.mod2",
  "pmt.root.mod3",
  "pmt.root.mod4",
  "pmt.root.mod5",
  "pmt.root.mod6",
  "pmt.root.mod7",
  "pmt.root.mod8",
  "pmt.root.mod9",
] as const;

/** Default hex for contributed colors (work on dark + light themes). */
const COLOR_DEFAULTS: Record<string, string> = {
  [GAME_COLOR_ID]: "#5B9A8B",
  [MOD_COLOR_IDS[0]]: "#7A8FB5",
  [MOD_COLOR_IDS[1]]: "#8F7AA8",
  [MOD_COLOR_IDS[2]]: "#6F9E7A",
  [MOD_COLOR_IDS[3]]: "#A88B6F",
  [MOD_COLOR_IDS[4]]: "#6F8F9E",
  [MOD_COLOR_IDS[5]]: "#9E6F8A",
  [MOD_COLOR_IDS[6]]: "#8A9E6F",
  [MOD_COLOR_IDS[7]]: "#7A7A9E",
  [MOD_COLOR_IDS[8]]: "#B07A6F",
  [MOD_COLOR_IDS[9]]: "#6FA8A0",
};

let roots: IdeRoot[] = [];
let registered = false;
const changeEmitter = new vscode.EventEmitter<vscode.Uri | vscode.Uri[] | undefined>();

/** Hex for a root or origin: custom #RRGGBB, else game default, else palette. */
export function originHex(origin: {
  kind: string;
  path: string;
  origin?: string;
  color?: string;
  wrapIndex?: number;
}): string {
  if (origin.color && /^#[0-9A-Fa-f]{6}$/.test(origin.color)) return origin.color;
  if (origin.kind === "game") return COLOR_DEFAULTS[GAME_COLOR_ID]!;
  const i = wrapIndexFor(origin);
  return COLOR_DEFAULTS[MOD_COLOR_IDS[i]!] ?? COLOR_DEFAULTS[MOD_COLOR_IDS[0]]!;
}

/** Hex for a Go origin id (`vanilla` or mod id). Fallback teal. */
export function originHexByOriginId(id: string): string {
  const key = id || "vanilla";
  const store = useWorkspaceStore();
  if (key === "vanilla") {
    return originHex({
      kind: "game",
      path: "",
      color: store.activeWorkspace?.gameColor,
    });
  }
  const mods = store.workspaceMods;
  const i = mods.findIndex((m) => m.id === key);
  const mod = i >= 0 ? mods[i] : undefined;
  if (mod) {
    return originHex({
      kind: "mod",
      path: mod.path,
      origin: mod.id,
      color: mod.color,
      wrapIndex: i,
    });
  }
  const root = roots.find((r) => r.origin === key);
  if (root) return originHex(root);
  return COLOR_DEFAULTS[GAME_COLOR_ID]!;
}

function wrapIndexFor(root: {
  kind: string;
  path: string;
  wrapIndex?: number;
}): number {
  if (root.kind !== "mod") return 0;
  const n = MOD_COLOR_IDS.length;
  if (root.wrapIndex !== undefined) {
    return ((root.wrapIndex % n) + n) % n;
  }
  const mods = roots.filter((r) => r.kind === "mod");
  const i = mods.findIndex((r) => normFs(r.path) === normFs(root.path));
  return (i < 0 ? 0 : i) % n;
}

/** Tooltip for the decoration color tag. */
function tooltipFor(root: IdeRoot): string {
  switch (root.kind) {
    case "game":
      return "Game root";
    case "mod":
      return `Mod: ${root.label}`;
    default:
      return root.label;
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
    const path = normFs(uri.fsPath || uri.path);
    const root = roots.find((r) => normFs(r.path) === path);
    if (!root) return undefined;
    const deco = new vscode.FileDecoration(undefined, tooltipFor(root));
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

/** CSS selector for an explorer root row by aria-label prefix. */
function explorerRootSel(label: string): string {
  return (
    ".monaco-workbench .explorer-folders-view " +
    `.monaco-list-row[aria-level="1"][aria-label^="${CSS.escape(label)}"]`
  );
}

function iconRule(label: string, url: string): string {
  const sel = `${explorerRootSel(label)} .monaco-icon-label::before`;
  return `${sel} { background-image: url("${url}") !important; }`;
}

function dotRule(label: string, hex: string): string {
  const row = explorerRootSel(label);
  return [
    `${row}::after { background: ${hex}; }`,
    `${row} .monaco-icon-name-container { color: ${hex} !important; }`,
  ].join("\n");
}

async function paintRootThumbs(next: IdeRoot[]): Promise<void> {
  if (!thumbStyle) {
    thumbStyle = document.createElement("style");
    thumbStyle.id = "pmt-root-thumbs";
    document.head.appendChild(thumbStyle);
  }
  const gameUrl = GAME_ICONS[useWorkspaceStore().currentGameId];
  const rules: string[] = [];
  for (const r of next) {
    rules.push(dotRule(r.label, originHex(r)));
    if (r.kind === "game" && gameUrl) {
      rules.push(iconRule(r.label, gameUrl));
      continue;
    }
    if (r.kind !== "mod" || !r.thumbnail) continue;
    try {
      const file = await ReadFileBase64(r.thumbnail);
      if (!file?.exists || !file.b64) continue;
      rules.push(iconRule(r.label, `data:${thumbMime(r.thumbnail)};base64,${file.b64}`));
    } catch {
      /* skip */
    }
  }
  thumbStyle.textContent = rules.join("\n");
}

/** Update roots used for decorations (call when workspace folders change). */
export function setDecoratedRoots(next: IdeRoot[]): void {
  roots = next;
  const uris = next.map((r) => vscode.Uri.file(r.path));
  changeEmitter.fire(uris.length ? uris : undefined);
  void paintRootThumbs(next);
}

/** Re-paint explorer accents without remounting folders. */
export function repaintDecoratedRoots(): void {
  if (roots.length) setDecoratedRoots(roots);
}
