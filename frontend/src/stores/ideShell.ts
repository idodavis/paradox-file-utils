/**
 * Thin Pinia store for workbench shell visibility (merge review mode).
 */
import { ref } from "vue";
import { defineStore } from "pinia";

/** Controls when the workbench overlays non-IDE Nuxt pages (e.g. patch review). */
export const useIdeShellStore = defineStore("ideShell", () => {
  const mergeReview = ref(false);
  const label = ref("Back");

  /** Show workbench over the current flow for diff/merge review. */
  function beginMergeReview(text = "Back"): void {
    label.value = text;
    mergeReview.value = true;
  }

  /** Hide merge-review overlay. Restore is owned by commands.endMergeOverlay. */
  function endMergeReview(): void {
    mergeReview.value = false;
  }

  return { mergeReview, label, beginMergeReview, endMergeReview };
});
