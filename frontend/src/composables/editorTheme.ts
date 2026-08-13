/**
 * Shared Pierre editor theme preference for CodeView / UnresolvedFile.
 */
import { ref } from "vue";

/** Curated Shiki + Pierre theme ids for the IDE toolbar. */
export const EDITOR_THEME_OPTIONS = [
  { label: "Pierre Dark", value: "pierre-dark" },
  { label: "Pierre Light", value: "pierre-light" },
  { label: "GitHub Dark", value: "github-dark" },
  { label: "GitHub Light", value: "github-light" },
  { label: "One Dark Pro", value: "one-dark-pro" },
  { label: "Solarized Dark", value: "solarized-dark" },
  { label: "Solarized Light", value: "solarized-light" },
] as const;

export type EditorThemeId = (typeof EDITOR_THEME_OPTIONS)[number]["value"];

const STORAGE_KEY = "editor.theme";

/** Active editor theme (persisted to localStorage). */
export const editorTheme = ref<EditorThemeId>(
  (localStorage.getItem(STORAGE_KEY) as EditorThemeId) || "pierre-dark",
);

/** Persist and apply an editor theme id. */
export function setEditorTheme(theme: string): void {
  const match = EDITOR_THEME_OPTIONS.find((t) => t.value === theme);
  if (!match) return;
  editorTheme.value = match.value;
  localStorage.setItem(STORAGE_KEY, match.value);
}

/** Pierre theme option object from the active preference. */
export function pierreThemeOption(): { dark: string; light: string } {
  const t = editorTheme.value;
  if (t.includes("light") || t === "solarized-light" || t === "github-light") {
    return { dark: t, light: t };
  }
  return { dark: t, light: t };
}
