/**
 * Opens A/B paths as workbench diffs for manual merge review.
 */
import { openDiff } from "../ide/commands";
import { useIdeShellStore } from "../stores/ideShell";

/** Review a single conflict pair in the workbench. */
export async function reviewDiffInWorkbench(opts: {
  pathA: string;
  pathB: string;
  title?: string;
  onBack?: () => void;
}): Promise<void> {
  const shell = useIdeShellStore();
  shell.beginMergeReview(opts.onBack);
  await openDiff(opts.pathA, opts.pathB, opts.title);
}
