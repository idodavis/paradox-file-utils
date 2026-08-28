/**
 * Register Paradox languages, PMT color themes, and Material+Paradox file icons.
 */
import { registerExtension } from "@codingame/monaco-vscode-api/extensions";
import { ExtensionHostKind } from "@codingame/monaco-vscode-extensions-service-override";
import paradoxGrammar from "../syntaxes/paradox.tmLanguage.json";
import paradoxGuiGrammar from "../syntaxes/paradox-gui.tmLanguage.json";
import paradoxLocGrammar from "../syntaxes/paradox-loc.tmLanguage.json";
import paradoxInfoGrammar from "../syntaxes/paradox-info.tmLanguage.json";
import { registerPmtColorThemes } from "./colorThemes";
import { registerPmtFileIcons } from "./fileIcons";

/** Contribute Paradox grammars, language ids, color themes, and icons. */
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
          {
            id: "paradox-mod",
            aliases: ["Paradox Mod Descriptor"],
            extensions: [".mod"],
            filenames: ["descriptor.mod"],
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
          {
            language: "paradox-mod",
            scopeName: paradoxGrammar.scopeName,
            path: "./syntaxes/paradox.tmLanguage.json",
          },
        ],
      },
    },
    ExtensionHostKind.LocalProcess,
  );

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
  await registerPmtColorThemes();
  await registerPmtFileIcons();
}
