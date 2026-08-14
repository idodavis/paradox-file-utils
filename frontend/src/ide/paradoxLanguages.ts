/**
 * Register Paradox languages + PMT file/folder icon theme with the workbench.
 */
import { registerExtension } from "@codingame/monaco-vscode-api/extensions";
import { ExtensionHostKind } from "@codingame/monaco-vscode-extensions-service-override";
import paradoxGrammar from "../syntaxes/paradox.tmLanguage.json";
import paradoxGuiGrammar from "../syntaxes/paradox-gui.tmLanguage.json";
import paradoxLocGrammar from "../syntaxes/paradox-loc.tmLanguage.json";
import paradoxInfoGrammar from "../syntaxes/paradox-info.tmLanguage.json";

import iconThemeUrl from "./fileicons/pmt-icon-theme.json?url";
import folderIcon from "./fileicons/folder.svg?url";
import folderOpenIcon from "./fileicons/folder-open.svg?url";
import gameRootIcon from "./fileicons/game-root.svg?url";
import modRootIcon from "./fileicons/mod-root.svg?url";
import stagingRootIcon from "./fileicons/staging-root.svg?url";
import fileIcon from "./fileicons/file.svg?url";
import paradoxIcon from "./fileicons/paradox.svg?url";
import paradoxGuiIcon from "./fileicons/paradox-gui.svg?url";
import paradoxLocIcon from "./fileicons/paradox-loc.svg?url";
import paradoxModIcon from "./fileicons/paradox-mod.svg?url";
import markdownIcon from "./fileicons/markdown.svg?url";
import imageIcon from "./fileicons/image.svg?url";
import jsonIcon from "./fileicons/json.svg?url";

const ICON_FILES: Record<string, string> = {
  "./fileicons/pmt-icon-theme.json": iconThemeUrl,
  "./fileicons/folder.svg": folderIcon,
  "./fileicons/folder-open.svg": folderOpenIcon,
  "./fileicons/game-root.svg": gameRootIcon,
  "./fileicons/mod-root.svg": modRootIcon,
  "./fileicons/staging-root.svg": stagingRootIcon,
  "./fileicons/file.svg": fileIcon,
  "./fileicons/paradox.svg": paradoxIcon,
  "./fileicons/paradox-gui.svg": paradoxGuiIcon,
  "./fileicons/paradox-loc.svg": paradoxLocIcon,
  "./fileicons/paradox-mod.svg": paradoxModIcon,
  "./fileicons/markdown.svg": markdownIcon,
  "./fileicons/image.svg": imageIcon,
  "./fileicons/json.svg": jsonIcon,
};

/** Contribute Paradox grammars, language ids, and the PMT icon theme. */
export async function registerParadoxLanguages(): Promise<void> {
  const { registerFileUrl, whenReady } = registerExtension(
    {
      name: "pmt-paradox-languages",
      publisher: "pmt",
      version: "1.0.0",
      engines: { vscode: "*" },
      contributes: {
        languages: [
          {
            id: "paradox",
            aliases: ["Paradox Script"],
            extensions: [".txt"],
          },
          {
            id: "paradox-gui",
            aliases: ["Paradox GUI"],
            extensions: [".gui"],
          },
          {
            id: "paradox-loc",
            aliases: ["Paradox Localization"],
            extensions: [".yml", ".yaml"],
          },
          {
            id: "paradox-info",
            aliases: ["Paradox Info"],
            filenames: ["*.info"],
          },
        ],
        grammars: [
          {
            language: "paradox",
            scopeName: paradoxGrammar.scopeName,
            path: "./syntaxes/paradox.tmLanguage.json",
          },
          {
            language: "paradox-gui",
            scopeName: paradoxGuiGrammar.scopeName,
            path: "./syntaxes/paradox-gui.tmLanguage.json",
          },
          {
            language: "paradox-loc",
            scopeName: paradoxLocGrammar.scopeName,
            path: "./syntaxes/paradox-loc.tmLanguage.json",
          },
          {
            language: "paradox-info",
            scopeName: paradoxInfoGrammar.scopeName,
            path: "./syntaxes/paradox-info.tmLanguage.json",
          },
        ],
        iconThemes: [
          {
            id: "pmt-icons",
            label: "PMT Icons",
            path: "./fileicons/pmt-icon-theme.json",
          },
        ],
      },
    },
    ExtensionHostKind.LocalProcess,
  );

  for (const [path, url] of Object.entries(ICON_FILES)) {
    registerFileUrl(path, url);
  }
  registerFileUrl(
    "./syntaxes/paradox.tmLanguage.json",
    `data:application/json,${encodeURIComponent(JSON.stringify(paradoxGrammar))}`,
  );
  registerFileUrl(
    "./syntaxes/paradox-gui.tmLanguage.json",
    `data:application/json,${encodeURIComponent(JSON.stringify(paradoxGuiGrammar))}`,
  );
  registerFileUrl(
    "./syntaxes/paradox-loc.tmLanguage.json",
    `data:application/json,${encodeURIComponent(JSON.stringify(paradoxLocGrammar))}`,
  );
  registerFileUrl(
    "./syntaxes/paradox-info.tmLanguage.json",
    `data:application/json,${encodeURIComponent(JSON.stringify(paradoxInfoGrammar))}`,
  );

  await whenReady();
}
