/**
 * Frontend plain-text merge that builds git-style conflict markers from A/B.
 * Used by manual merge (no Go GetMergeConflicts / Paradox key semantics).
 */
import { parseDiffFromFile } from "@pierre/diffs";

/** Result of building a conflict-marked document from two file texts. */
export type TextMergeResult = {
  content: string;
  conflictCount: number;
  identical: boolean;
};

/**
 * Build a conflict-marker document from two file contents using a line-based
 * 2-way diff. Identical inputs return the original text with zero conflicts.
 */
export function buildConflictMarkedFile(
  textA: string,
  textB: string,
  opts: { fileName: string; labelA: string; labelB: string },
): TextMergeResult {
  if (textA === textB) {
    return { content: textA, conflictCount: 0, identical: true };
  }

  const fileDiff = parseDiffFromFile(
    { name: opts.fileName, contents: textA },
    { name: opts.fileName, contents: textB },
  );

  const del = fileDiff.deletionLines ?? [];
  const add = fileDiff.additionLines ?? [];
  const out: string[] = [];
  let conflictCount = 0;
  let aCursor = 0;
  let bCursor = 0;

  for (const hunk of fileDiff.hunks) {
    for (let i = 0; i < hunk.collapsedBefore; i += 1) {
      out.push(del[aCursor] ?? "");
      aCursor += 1;
      bCursor += 1;
    }

    for (const block of hunk.hunkContent) {
      if (block.type === "context") {
        for (let i = 0; i < block.lines; i += 1) {
          out.push(del[aCursor] ?? "");
          aCursor += 1;
          bCursor += 1;
        }
        continue;
      }

      const aLines: string[] = [];
      const bLines: string[] = [];
      for (let i = 0; i < block.deletions; i += 1) {
        aLines.push(del[aCursor] ?? "");
        aCursor += 1;
      }
      for (let i = 0; i < block.additions; i += 1) {
        bLines.push(add[bCursor] ?? "");
        bCursor += 1;
      }

      out.push(`<<<<<<< ${opts.labelA}\n`);
      out.push(...aLines);
      out.push("=======\n");
      out.push(...bLines);
      out.push(`>>>>>>> ${opts.labelB}\n`);
      conflictCount += 1;
    }
  }

  while (aCursor < del.length) {
    out.push(del[aCursor] ?? "");
    aCursor += 1;
  }

  return { content: out.join(""), conflictCount, identical: false };
}

/** Count remaining git-style conflict start markers in a document. */
export function countConflictMarkers(content: string): number {
  const matches = content.match(/^<<<<<<< /gm);
  return matches?.length ?? 0;
}
