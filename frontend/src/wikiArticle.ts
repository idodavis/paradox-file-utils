/**
 * Tailwind classes for sanitized wiki HTML, plus mount/highlight helpers.
 */
import * as monaco from "monaco-editor";

/** Per-tag utilities. Keep strings here so Tailwind scans them. */
export const WIKI_TAG_CLASS: Record<string, string> = {
  h1: "mt-2 mb-3 text-2xl font-semibold tracking-tight text-highlighted",
  h2: "mt-6 mb-2 text-xl font-semibold tracking-tight text-highlighted",
  h3: "mt-5 mb-1.5 text-lg font-semibold text-highlighted",
  h4: "mt-4 mb-1 text-base font-semibold text-highlighted",
  h5: "mt-3 mb-1 text-sm font-semibold text-toned",
  h6: "mt-3 mb-1 text-sm font-semibold text-toned",
  p: "my-2 text-sm leading-relaxed text-default",
  ul: "my-2 list-disc ps-5 space-y-1 text-sm",
  ol: "my-2 list-decimal ps-5 space-y-1 text-sm",
  li: "leading-relaxed text-default",
  pre: "my-3 overflow-x-auto rounded-md bg-elevated px-3 py-2 text-xs leading-relaxed",
  code: "rounded-sm bg-elevated px-1 py-px font-mono text-[0.8em]",
  a: "text-primary underline underline-offset-2",
  sup: "text-[0.7em] align-super text-primary",
  sub: "text-[0.7em] align-sub",
  small: "text-xs text-muted",
  table: "my-3 w-full border-collapse text-xs",
  th: "border border-default px-2 py-1",
  td: "border border-default px-2 py-1",
  blockquote: "my-3 border-s-2 border-accented ps-3 text-sm text-muted",
  mark: "rounded-sm bg-warning/30 px-0.5 text-default",
};

/** Stamp Tailwind utilities onto sanitized wiki tags. */
export function applyWikiTailwind(root: HTMLElement): void {
  for (const [tag, cls] of Object.entries(WIKI_TAG_CLASS)) {
    root.querySelectorAll(tag).forEach((el) => {
      el.className = cls;
    });
  }
  const cite = `${WIKI_TAG_CLASS.a} ${WIKI_TAG_CLASS.sup}`;
  root.querySelectorAll("sup a").forEach((el) => {
    el.className = cite;
  });
}

/** Colorize wiki <pre> blocks with the Paradox TextMate grammar. */
export async function colorizePres(root: HTMLElement): Promise<void> {
  const nodes = [...root.querySelectorAll("pre")];
  await Promise.all(
    nodes.map(async (pre) => {
      if (pre.dataset.hl === "1") return;
      const text = pre.textContent ?? "";
      if (!text.trim()) return;
      try {
        pre.innerHTML = await monaco.editor.colorize(text, "paradox", {});
        pre.dataset.hl = "1";
      } catch {
        /* tokenizer not ready yet */
      }
    }),
  );
}

/** Unwrap previous search marks. */
export function clearWikiMarks(root: HTMLElement): void {
  root.querySelectorAll("mark").forEach((mark) => {
    const parent = mark.parentNode;
    if (!parent) return;
    while (mark.firstChild) parent.insertBefore(mark.firstChild, mark);
    parent.removeChild(mark);
    parent.normalize();
  });
}

/** Highlight case-insensitive hits. Skips pre/code. */
export function highlightWiki(root: HTMLElement, needle: string): void {
  clearWikiMarks(root);
  const q = needle.trim();
  if (!q) return;
  const lower = q.toLowerCase();
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      const p = (node as Text).parentElement;
      if (!p || p.closest("pre, code")) return NodeFilter.FILTER_REJECT;
      return (node.textContent ?? "").toLowerCase().includes(lower)
        ? NodeFilter.FILTER_ACCEPT
        : NodeFilter.FILTER_SKIP;
    },
  });
  const hits: Text[] = [];
  while (walker.nextNode()) hits.push(walker.currentNode as Text);
  for (const text of hits) wrapHits(text, q);
}

/** First TOC anchor whose heading contains needle, or empty. */
export function firstSectionHit(
  sections: { line: string; anchor: string }[] | null | undefined,
  needle: string,
): string {
  const q = needle.trim().toLowerCase();
  if (!q) return "";
  return sections?.find((s) => s.line.toLowerCase().includes(q))?.anchor ?? "";
}

/** How a wiki <a> should be handled inside Guide / Patch Notes. */
export type WikiLinkHit =
  | { kind: "anchor"; anchor: string }
  | { kind: "page"; index: number; anchor: string }
  | { kind: "external" };

/** Resolve a wiki href against the pages currently loaded in the pane. */
export function resolveWikiLink(
  href: string,
  pages: { title: string; url: string }[],
  currentIndex: number,
): WikiLinkHit {
  const raw = href.trim();
  if (!raw) return { kind: "external" };
  const base = pages[currentIndex]?.url || "https://local.invalid/";
  let url: URL;
  try {
    url = new URL(raw, base);
  } catch {
    return { kind: "external" };
  }
  const anchor = decodeURIComponent(url.hash.replace(/^#/, ""));
  const pathTitle = wikiPathTitle(url);
  const idx = pages.findIndex((p) => wikiSamePage(p, url, pathTitle));
  if (idx >= 0) {
    if (idx === currentIndex) return { kind: "anchor", anchor };
    return { kind: "page", index: idx, anchor };
  }
  if (!pathTitle && anchor) return { kind: "anchor", anchor };
  return { kind: "external" };
}

function wikiPathTitle(url: URL): string {
  const parts = url.pathname.split("/").filter(Boolean);
  const last = parts.at(-1) ?? "";
  if (!last || last === "wiki" || last === "index.php") return "";
  try {
    return decodeURIComponent(last).replace(/_/g, " ");
  } catch {
    return last.replace(/_/g, " ");
  }
}

function wikiSamePage(
  p: { title: string; url: string },
  href: URL,
  pathTitle: string,
): boolean {
  const want = normWiki(pathTitle);
  if (want && normWiki(p.title) === want) return true;
  try {
    const pu = new URL(p.url);
    return normWikiPath(pu) === normWikiPath(href);
  } catch {
    return false;
  }
}

function normWiki(s: string): string {
  return s.trim().toLowerCase().replace(/_/g, " ");
}

function normWikiPath(u: URL): string {
  return u.pathname.replace(/\/+$/, "").toLowerCase();
}

function wrapHits(node: Text, needle: string): void {
  const src = node.textContent ?? "";
  const lower = src.toLowerCase();
  const q = needle.toLowerCase();
  const frag = document.createDocumentFragment();
  let i = 0;
  while (i < src.length) {
    const at = lower.indexOf(q, i);
    if (at < 0) {
      frag.appendChild(document.createTextNode(src.slice(i)));
      break;
    }
    if (at > i) frag.appendChild(document.createTextNode(src.slice(i, at)));
    const mark = document.createElement("mark");
    mark.className = WIKI_TAG_CLASS.mark ?? "";
    mark.textContent = src.slice(at, at + needle.length);
    frag.appendChild(mark);
    i = at + needle.length;
  }
  node.parentNode?.replaceChild(frag, node);
}
