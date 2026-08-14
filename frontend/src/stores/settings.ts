/**
 * Pinia store for persisted app settings (fonts, theme keys, misc).
 */
import { computed, ref, watch } from "vue";
import { defineStore } from "pinia";
import { GetSettings, SaveSettings } from "@services/settingsservice";
import { normalizeSettings } from "../composables/settings";

const FONT_SCALE_MIN = 85;
const FONT_SCALE_MAX = 130;
const EDITOR_FONT_MIN = 11;
const EDITOR_FONT_MAX = 18;

/** Clamp a number into an inclusive range. */
function clamp(n: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, n));
}

/** Apply UI font scale and editor font size to the document. */
export function applyFontCss(fontScale: number, editorFontSize: number): void {
  const root = document.documentElement;
  root.style.setProperty("--pmt-font-scale", String(fontScale / 100));
  root.style.fontSize = `${(16 * fontScale) / 100}px`;
  root.style.setProperty("--editor-font-size", `${editorFontSize}px`);
}

/** App-wide settings including dual font preferences. */
export const useSettingsStore = defineStore("settings", () => {
  const values = ref<Record<string, string>>({});
  const loading = ref(false);
  const saving = ref(false);

  const fontScale = computed(() =>
    clamp(Number(values.value["ui.fontScale"]) || 100, FONT_SCALE_MIN, FONT_SCALE_MAX),
  );
  const editorFontSize = computed(() =>
    clamp(
      Number(values.value["editor.fontSize"]) || 13,
      EDITOR_FONT_MIN,
      EDITOR_FONT_MAX,
    ),
  );

  /** Load settings from the backend and apply fonts. */
  async function load(): Promise<void> {
    loading.value = true;
    try {
      values.value = normalizeSettings(await GetSettings());
      applyFontCss(fontScale.value, editorFontSize.value);
    } finally {
      loading.value = false;
    }
  }

  /** Persist current settings map. */
  async function save(): Promise<void> {
    saving.value = true;
    try {
      await SaveSettings(values.value);
    } finally {
      saving.value = false;
    }
  }

  /** Set one key and optionally persist immediately. */
  async function set(key: string, value: string, persist = true): Promise<void> {
    values.value = { ...values.value, [key]: value };
    if (key === "ui.fontScale" || key === "editor.fontSize") {
      applyFontCss(fontScale.value, editorFontSize.value);
    }
    if (persist) await save();
  }

  /** Update font scale percent and persist. */
  async function setFontScale(percent: number): Promise<void> {
    await set(
      "ui.fontScale",
      String(clamp(percent, FONT_SCALE_MIN, FONT_SCALE_MAX)),
    );
  }

  /** Update Pierre editor font size (px) and persist. */
  async function setEditorFontSize(px: number): Promise<void> {
    await set(
      "editor.fontSize",
      String(clamp(px, EDITOR_FONT_MIN, EDITOR_FONT_MAX)),
    );
  }

  watch([fontScale, editorFontSize], ([scale, size]) => {
    applyFontCss(scale, size);
  });

  return {
    values,
    loading,
    saving,
    fontScale,
    editorFontSize,
    load,
    save,
    set,
    setFontScale,
    setEditorFontSize,
    FONT_SCALE_MIN,
    FONT_SCALE_MAX,
    EDITOR_FONT_MIN,
    EDITOR_FONT_MAX,
  };
});
