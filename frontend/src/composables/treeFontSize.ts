/**
 * Persisted file-tree font size preference for the Workspace IDE explorer.
 */
import { ref } from "vue";

const STORAGE_KEY = "ide.treeFontSize";
const MIN = 10;
const MAX = 18;
const DEFAULT = 12;

function clamp(n: number): number {
  return Math.min(MAX, Math.max(MIN, Math.round(n)));
}

/** Active explorer font size in px (persisted to localStorage). */
export const treeFontSize = ref(
  clamp(Number(localStorage.getItem(STORAGE_KEY)) || DEFAULT),
);

/** Persist and apply a tree font size in px. */
export function setTreeFontSize(size: number): void {
  treeFontSize.value = clamp(size);
  localStorage.setItem(STORAGE_KEY, String(treeFontSize.value));
}

/** Increase tree font size by 1px. */
export function bumpTreeFontSize(delta: number): void {
  setTreeFontSize(treeFontSize.value + delta);
}
