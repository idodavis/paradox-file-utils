/**
 * Material Icon Theme base with Paradox file overlays (SVG).
 *
 * Material provides folders/files; PMT SVGs cover script / loc / gui / mod.
 * Vite `?url` paths must be absolutized before registerFileUrl (URI.parse).
 */
import { registerExtension, ExtensionHostKind } from "@codingame/monaco-vscode-api/extensions";
import materialTheme from "material-icon-theme/dist/material-icons.json";
import paradoxScriptIcon from "./fileicons/paradox-script.svg?url";
import paradoxGuiIcon from "./fileicons/paradox-gui.svg?url";
import paradoxLocIcon from "./fileicons/paradox-loc.svg?url";
import paradoxModIcon from "./fileicons/paradox-mod.svg?url";

/** VS Code file icon theme JSON shape. */
type IconThemeJson = {
  iconDefinitions: Record<string, { iconPath?: string; [k: string]: unknown }>;
  file?: string;
  folder?: string;
  folderExpanded?: string;
  rootFolder?: string;
  rootFolderExpanded?: string;
  fileExtensions?: Record<string, string>;
  fileNames?: Record<string, string>;
  languageIds?: Record<string, string>;
  folderNames?: Record<string, string>;
  folderNamesExpanded?: Record<string, string>;
  light?: Record<string, unknown>;
  highContrast?: Record<string, unknown>;
  hidesExplorerArrows?: boolean;
  [key: string]: unknown;
};

/** Eager Material SVG assets (filename → Vite URL). */
const materialSvgUrls = import.meta.glob("../../node_modules/material-icon-theme/icons/*.svg", {
  eager: true,
  query: "?url",
  import: "default",
}) as Record<string, string>;

/** Absolute URL for Vite root-relative asset paths. */
function absoluteAssetUrl(viteUrl: string): string {
  return new URL(viteUrl, import.meta.url).href;
}

/** Basename of a glob key or iconPath. */
function fileName(path: string): string {
  const parts = path.replace(/\\/g, "/").split("/");
  return parts[parts.length - 1] ?? path;
}

/**
 * Clone Material theme, point iconPaths at `./name.svg` next to the theme JSON,
 * and overlay Paradox file types.
 */
function buildPmtIconTheme(): IconThemeJson {
  const base = structuredClone(materialTheme as IconThemeJson);

  for (const def of Object.values(base.iconDefinitions)) {
    if (typeof def.iconPath !== "string") continue;
    // Material ships `./../icons/foo.svg` relative to dist/material-icons.json.
    def.iconPath = `./${fileName(def.iconPath)}`;
  }

  base.iconDefinitions["pmt-paradox"] = { iconPath: "./paradox-script.svg" };
  base.iconDefinitions["pmt-gui"] = { iconPath: "./paradox-gui.svg" };
  base.iconDefinitions["pmt-loc"] = { iconPath: "./paradox-loc.svg" };
  base.iconDefinitions["pmt-mod"] = { iconPath: "./paradox-mod.svg" };

  base.languageIds = {
    ...(base.languageIds ?? {}),
    paradox: "pmt-paradox",
    "paradox-gui": "pmt-gui",
    "paradox-loc": "pmt-loc",
    "paradox-info": "pmt-mod",
    "paradox-mod": "pmt-mod",
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

  return base;
}

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
          label: "PMT Icons (Material + Paradox)",
          path: "./icons/pmt-icon-theme.json",
        },
      ],
    },
  },
  ExtensionHostKind.LocalWebWorker,
  { system: true },
);

registerFileUrl("icons/pmt-icon-theme.json", `data:application/json,${encodeURIComponent(JSON.stringify(theme))}`, {
  mimeType: "application/json",
});

for (const [globPath, url] of Object.entries(materialSvgUrls)) {
  registerFileUrl(`icons/${fileName(globPath)}`, absoluteAssetUrl(url), {
    mimeType: "image/svg+xml",
  });
}

registerFileUrl("icons/paradox-script.svg", absoluteAssetUrl(paradoxScriptIcon), { mimeType: "image/svg+xml" });
registerFileUrl("icons/paradox-gui.svg", absoluteAssetUrl(paradoxGuiIcon), {
  mimeType: "image/svg+xml",
});
registerFileUrl("icons/paradox-loc.svg", absoluteAssetUrl(paradoxLocIcon), {
  mimeType: "image/svg+xml",
});
registerFileUrl("icons/paradox-mod.svg", absoluteAssetUrl(paradoxModIcon), {
  mimeType: "image/svg+xml",
});

/** Await until the PMT icon theme extension is ready. */
export async function registerPmtFileIcons(): Promise<void> {
  await whenReady();
}
