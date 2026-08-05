/**
 * Shared Monaco editor state and lazy loader for the Vue UI.
 */
import { computed, ref } from "vue";
import { init } from "modern-monaco";

export const CODE_THEMES = [
  "one-dark-pro",
  "one-light",
  "ayu-dark",
  "ayu-light",
  "github-dark-default",
  "github-light-default",
  "material-theme-darker",
  "material-theme-lighter",
  "material-theme-palenight",
  "tokyo-night",
  "catppuccin-latte",
] as const;

export const CODE_THEME_LABELS: Record<(typeof CODE_THEMES)[number], string> = {
  "one-dark-pro": "One Dark Pro",
  "one-light": "One Light",
  "ayu-dark": "Ayu Dark",
  "ayu-light": "Ayu Light",
  "github-dark-default": "GitHub Dark",
  "github-light-default": "GitHub Light",
  "material-theme-darker": "Material Darker",
  "material-theme-lighter": "Material Lighter",
  "material-theme-palenight": "Material Palenight",
  "tokyo-night": "Tokyo Night",
  "catppuccin-latte": "Catppuccin Latte",
};

export const CODE_LANGUAGES = [
  "hcl",
  "plaintext",
  "json",
  "yaml",
  "typescript",
  "javascript",
  "html",
  "css",
  "markdown",
  "xml",
  "shellscript",
  "python",
  "cpp",
  "go",
  "rust",
  "java",
] as const;

export type CodeTheme = (typeof CODE_THEMES)[number];
export type CodeLanguage = (typeof CODE_LANGUAGES)[number];
export type MonacoApi = Awaited<ReturnType<typeof init>>;

export const codeTheme = ref<CodeTheme>("one-dark-pro");
export const codeLanguage = ref<CodeLanguage>("hcl");
export const monacoActive = computed(() => ({ theme: codeTheme.value, lang: codeLanguage.value }));

let monacoPromise: Promise<MonacoApi> | null = null;

export function getMonaco(): Promise<MonacoApi> {
  if (!monacoPromise) {
    monacoPromise = init({
      defaultTheme: codeTheme.value,
      themes: [...CODE_THEMES],
      langs: [...CODE_LANGUAGES],
    }).catch(() =>
      init({
        defaultTheme: codeTheme.value,
        themes: [...CODE_THEMES],
        langs: [...CODE_LANGUAGES],
      }),
    );
  }
  return monacoPromise;
}
