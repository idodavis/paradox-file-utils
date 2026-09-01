/**
 * Thin Pinia store for workbench shell visibility (merge review mode).
 */
import { ref } from "vue";
import { defineStore } from "pinia";

/** Controls when the workbench overlays non-IDE Nuxt pages (e.g. patch review). */
export const useIdeShellStore = defineStore("ideShell", () => {
  const mergeReview = ref(false);
  const bannerLabel = ref("Back");
  let onEnd: (() => void) | null = null;

  /** Show workbench over the current flow for diff/merge review. */
  function beginMergeReview(back?: () => void, label = "Back"): void {
    onEnd = back ?? null;
    bannerLabel.value = label;
    mergeReview.value = true;
  }

  /** Hide merge-review overlay and run optional back navigation. */
  function endMergeReview(): void {
    mergeReview.value = false;
    const cb = onEnd;
    onEnd = null;
    cb?.();
  }

  return { mergeReview, bannerLabel, beginMergeReview, endMergeReview };
});
