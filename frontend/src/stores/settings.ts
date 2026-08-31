/**
 * Pinia store for persisted app settings (UI scale, theme keys, misc).
 */
import { computed, ref, watch } from "vue";
import { defineStore } from "pinia";
import { clamp } from "es-toolkit";
import { GetSettings, SaveSettings } from "@services/settingsservice";

/** Drop undefined values from a settings map. */
function normalizeSettings(
  input: Record<string, string | undefined> | null | undefined,
): Record<string, string> {
  return Object.fromEntries(
    Object.entries(input ?? {}).filter(
      (entry): entry is [string, string] => entry[1] !== undefined,
    ),
  );
}

const FONT_SCALE_MIN = 85;
const FONT_SCALE_MAX = 130;

/** Apply UI font scale to the document (workbench owns its own editor font). */
export function applyFontCss(fontScale: number): void {
  const root = document.documentElement;
  root.style.setProperty("--pmt-font-scale", String(fontScale / 100));
  root.style.fontSize = `${(16 * fontScale) / 100}px`;
}

/** App-wide settings including UI scale. */
export const useSettingsStore = defineStore("settings", () => {
  const values = ref<Record<string, string>>({});
  const loading = ref(false);
  const saving = ref(false);

  const fontScale = computed(() =>
    clamp(Number(values.value["ui.fontScale"]) || 100, FONT_SCALE_MIN, FONT_SCALE_MAX),
  );

  /** Load settings from the backend and apply fonts. */
  async function load(): Promise<void> {
    loading.value = true;
    try {
      values.value = normalizeSettings(await GetSettings());
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
    if (persist) await save();
  }

  /** Update font scale percent and persist. */
  async function setFontScale(percent: number): Promise<void> {
    await set(
      "ui.fontScale",
      String(clamp(percent, FONT_SCALE_MIN, FONT_SCALE_MAX)),
    );
  }

  watch(fontScale, (scale) => applyFontCss(scale), { immediate: true });

  return {
    values,
    loading,
    saving,
    fontScale,
    load,
    save,
    set,
    setFontScale,
    FONT_SCALE_MIN,
    FONT_SCALE_MAX,
  };
});
