/**
 * Seti-based file icon theme with a small Paradox-specific overlay.
 */
import { registerExtension } from "@codingame/monaco-vscode-api/extensions";
import { ExtensionHostKind } from "@codingame/monaco-vscode-extensions-service-override";
import setiTheme from "@codingame/monaco-vscode-theme-seti-default-extension/resources/vs-seti-icon-theme.json";
import setiFontUrl from "@codingame/monaco-vscode-theme-seti-default-extension/resources/seti.woff?url";
import pmtMarkUrl from "../assets/PMT-SquareIcon-Mint.png?url";
import paradoxGuiIcon from "./fileicons/paradox-gui.svg?url";
import paradoxLocIcon from "./fileicons/paradox-loc.svg?url";
import paradoxModIcon from "./fileicons/paradox-mod.svg?url";
import gameRootIcon from "./fileicons/game-root.svg?url";
import stagingRootIcon from "./fileicons/staging-root.svg?url";
import modRootIcon from "./fileicons/mod-root.svg?url";

/** Seti theme JSON shape we extend. */
type IconThemeJson = {
  fonts?: unknown[];
  iconDefinitions: Record<string, Record<string, unknown>>;
  file?: string;
  folder?: string;
  folderExpanded?: string;
  rootFolder?: string;
  rootFolderExpanded?: string;
  fileExtensions?: Record<string, string>;
  fileNames?: Record<string, string>;
  folderNames?: Record<string, string>;
  folderNamesExpanded?: Record<string, string>;
  rootFolderNames?: Record<string, string>;
  rootFolderNamesExpanded?: Record<string, string>;
  languageIds?: Record<string, string>;
  light?: Record<string, unknown>;
  [key: string]: unknown;
};

/** Merge Seti defaults with Paradox/PMT icon overrides. */
function buildPmtIconTheme(): IconThemeJson {
  const base = structuredClone(setiTheme as IconThemeJson);
  const defs = base.iconDefinitions;

  defs["pmt-paradox"] = { iconPath: "./paradox.png" };
  defs["pmt-gui"] = { iconPath: "./paradox-gui.svg" };
  defs["pmt-loc"] = { iconPath: "./paradox-loc.svg" };
  defs["pmt-mod"] = { iconPath: "./paradox-mod.svg" };
  defs["pmt-game-root"] = { iconPath: "./game-root.svg" };
  defs["pmt-staging-root"] = { iconPath: "./staging-root.svg" };
  defs["pmt-mod-root"] = { iconPath: "./mod-root.svg" };

  base.rootFolder = "pmt-mod-root";
  base.rootFolderExpanded = "pmt-mod-root";
  base.folderNames = {
    ...(base.folderNames ?? {}),
    Game: "pmt-game-root",
    Staging: "pmt-staging-root",
  };
  base.folderNamesExpanded = {
    ...(base.folderNamesExpanded ?? {}),
    Game: "pmt-game-root",
    Staging: "pmt-staging-root",
  };
  base.rootFolderNames = {
    Game: "pmt-game-root",
    Staging: "pmt-staging-root",
  };
  base.rootFolderNamesExpanded = {
    Game: "pmt-game-root",
    Staging: "pmt-staging-root",
  };

  base.languageIds = {
    ...(base.languageIds ?? {}),
    paradox: "pmt-paradox",
    "paradox-gui": "pmt-gui",
    "paradox-loc": "pmt-loc",
    "paradox-info": "pmt-mod",
  };

  base.fileExtensions = {
    ...(base.fileExtensions ?? {}),
    txt: "pmt-paradox",
    gui: "pmt-gui",
    yml: "pmt-loc",
    yaml: "pmt-loc",
    mod: "pmt-mod",
    info: "pmt-mod",
  };

  base.fileNames = {
    ...(base.fileNames ?? {}),
    "descriptor.mod": "pmt-mod",
  };

  // Point Seti font at our registered copy next to this theme JSON.
  if (Array.isArray(base.fonts)) {
    for (const font of base.fonts as Array<{ src?: Array<{ path?: string }> }>) {
      if (font.src?.[0]) font.src[0].path = "./seti.woff";
    }
  }

  return base;
}

const ICON_FILES: Record<string, string> = {
  "./fileicons/seti.woff": setiFontUrl,
  "./fileicons/paradox.png": pmtMarkUrl,
  "./fileicons/paradox-gui.svg": paradoxGuiIcon,
  "./fileicons/paradox-loc.svg": paradoxLocIcon,
  "./fileicons/paradox-mod.svg": paradoxModIcon,
  "./fileicons/game-root.svg": gameRootIcon,
  "./fileicons/staging-root.svg": stagingRootIcon,
  "./fileicons/mod-root.svg": modRootIcon,
};

/** Register the PMT icon theme (Seti + Paradox overlay). */
export async function registerPmtFileIcons(): Promise<void> {
  const theme = buildPmtIconTheme();
  const { registerFileUrl, whenReady } = registerExtension(
    {
      name: "pmt-file-icons",
      publisher: "pmt",
      version: "1.0.0",
      engines: { vscode: "*" },
      contributes: {
        iconThemes: [
          {
            id: "pmt-icons",
            label: "PMT Icons (Seti + Paradox)",
            path: "./fileicons/pmt-icon-theme.json",
          },
        ],
      },
    },
    ExtensionHostKind.LocalProcess,
  );

  registerFileUrl(
    "./fileicons/pmt-icon-theme.json",
    `data:application/json,${encodeURIComponent(JSON.stringify(theme))}`,
  );
  for (const [path, url] of Object.entries(ICON_FILES)) {
    registerFileUrl(path, url);
  }

  await whenReady();
}
