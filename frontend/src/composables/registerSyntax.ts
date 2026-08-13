/**
 * Register Paradox TextMate languages with Pierre/Shiki at app startup.
 * Loaders must return `{ default: LanguageRegistration[] }` (Shiki ESM shape).
 */
import { registerCustomLanguage, resolveLanguages } from "@pierre/diffs";
import paradoxGrammar from "../syntaxes/paradox.tmLanguage.json";
import paradoxLocGrammar from "../syntaxes/paradox-loc.tmLanguage.json";
import paradoxInfoGrammar from "../syntaxes/paradox-info.tmLanguage.json";
import paradoxGuiGrammar from "../syntaxes/paradox-gui.tmLanguage.json";

const PARADOX_LANGS = ["paradox", "paradox-loc", "paradox-gui", "paradox-info"] as const;

let registered = false;
let readyPromise: Promise<void> | null = null;

/** Idempotent registration of paradox / paradox-loc / paradox-gui / paradox-info. */
export function registerParadoxLanguages(): Promise<void> {
  if (readyPromise) return readyPromise;
  readyPromise = (async () => {
    if (!registered) {
      registered = true;
      registerCustomLanguage(
        "paradox",
        async () => ({
          default: [{ ...paradoxGrammar, name: "paradox", displayName: "Paradox Script" }],
        }),
        ["txt", "mod"],
      );
      registerCustomLanguage(
        "paradox-info",
        async () => ({
          default: [{ ...paradoxInfoGrammar, name: "paradox-info", displayName: "Paradox Info" }],
        }),
        ["info"],
      );
      registerCustomLanguage(
        "paradox-loc",
        async () => ({
          default: [
            { ...paradoxLocGrammar, name: "paradox-loc", displayName: "Paradox Localization" },
          ],
        }),
        ["yml", "yaml"],
      );
      registerCustomLanguage(
        "paradox-gui",
        async () => ({
          default: [{ ...paradoxGuiGrammar, name: "paradox-gui", displayName: "Paradox GUI" }],
        }),
        ["gui"],
      );
    }
    await resolveLanguages([...PARADOX_LANGS]);
  })();
  return readyPromise;
}

/** True once custom languages are registered and resolved for Shiki. */
export function paradoxLanguagesReady(): Promise<void> {
  return registerParadoxLanguages();
}
