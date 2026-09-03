/**
 * Active workbench file path. Updated from workbenchHost.
 */
import { readonly, ref } from "vue";

const path = ref("");

/** Record the workbench's active file (absolute path, or empty). */
export function setIdeActiveFile(absPath: string): void {
  path.value = absPath;
}

/** Read-only active file path for the Guide pane. */
export function useIdeActiveFile() {
  return readonly(path);
}
