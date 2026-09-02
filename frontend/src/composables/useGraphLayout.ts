/**
 * Dagre layered layout: Go graph payloads in, Vue Flow {x,y} out.
 * Origin is shown on pills only; there are no group boxes.
 */
import {
  onScopeDispose,
  shallowRef,
  toValue,
  watch,
  type MaybeRefOrGetter,
  type ShallowRef,
} from "vue";
import { Graph, layout } from "@dagrejs/dagre";

/** Query-root card size. */
export const ROOT_W = 260;
/** Query-root card size. */
export const ROOT_H = 118;
/** Default event / more-stub card size. */
export const NODE_W = 248;
/** Default event / more-stub card size. */
export const NODE_H = 108;

/** Chars shown on an edge before ellipsis; dagre width uses this cap. */
export const LABEL_SHOW_CAP = 40;

/** Pixel width dagre should reserve for a (possibly truncated) label. */
function displayedLabelWidth(label: string): number {
  const n = Math.min((label || "").length, LABEL_SHOW_CAP);
  if (!n) return 24;
  return Math.max(24, Math.round(n * 7));
}

/** Dagre preset: left→right "happens after", or top-down tree. */
export type GraphLayoutMode = "lr" | "tree";

/** Top-left pixel position for one Vue Flow node. */
export interface GraphPosition {
  x: number;
  y: number;
}

/** Minimal node identity dagre needs, plus role/kind for box size. */
export interface LayoutNode {
  id: string;
  role?: string;
  kind?: string;
}

/** Minimal directed edge dagre needs (Go `from`/`to` plus label for spacing). */
export interface LayoutEdge {
  from: string;
  to: string;
  label?: string;
  via?: string;
}

/** Pixel box for a node given its role, then kind. */
export function nodeBox(n: { role?: string; kind?: string }): {
  width: number;
  height: number;
} {
  if (n.role === "root") return { width: ROOT_W, height: ROOT_H };
  if (n.kind && n.kind !== "event" && n.kind !== "more") {
    return { width: 280, height: 120 };
  }
  return { width: NODE_W, height: NODE_H };
}

/** Dagre rankdir for a layout preset. */
function dagreRankdir(mode: GraphLayoutMode): "LR" | "TB" {
  switch (mode) {
    case "lr":
      return "LR";
    case "tree":
      return "TB";
    default: {
      const _never: never = mode;
      throw new Error(`unknown layout ${_never}`);
    }
  }
}

/** Spacing for the axis labels sit on; the other axis stays modest. */
function labelGap(
  edges: readonly LayoutEdge[],
  mode: GraphLayoutMode,
): {
  nodesep: number;
  ranksep: number;
} {
  let max = 24;
  for (const e of edges) {
    max = Math.max(max, displayedLabelWidth(e.label ?? ""));
  }
  switch (mode) {
    case "lr":
      return { nodesep: 80, ranksep: Math.max(120, max + 24) };
    case "tree":
      return { nodesep: Math.max(80, max + 24), ranksep: 120 };
    default: {
      const _never: never = mode;
      throw new Error(`unknown layout ${_never}`);
    }
  }
}

/** Stable Vue Flow id for a payload edge. */
export function layoutEdgeId(e: LayoutEdge, i: number): string {
  return `${e.from}->${e.to}:${e.via ?? ""}:${i}`;
}

/** Run dagre on a snapshot of nodes and edges. Positions are top-left. */
function layoutDagre(
  nodes: readonly LayoutNode[],
  edges: readonly LayoutEdge[],
  mode: GraphLayoutMode,
): Map<string, GraphPosition> {
  const g = new Graph({ directed: true });
  g.setDefaultEdgeLabel(() => ({}));
  const gap = labelGap(edges, mode);
  g.setGraph({
    rankdir: dagreRankdir(mode),
    nodesep: gap.nodesep,
    ranksep: gap.ranksep,
    marginx: 24,
    marginy: 24,
  });
  const ids = new Set<string>();
  for (const n of nodes) {
    if (!n.id || ids.has(n.id)) continue;
    ids.add(n.id);
    const box = nodeBox(n);
    g.setNode(n.id, { width: box.width, height: box.height });
  }
  if (!ids.size) return new Map();
  edges.forEach((e, i) => {
    if (!ids.has(e.from) || !ids.has(e.to)) return;
    const w = displayedLabelWidth(e.label ?? "");
    g.setEdge(e.from, e.to, {
      label: e.label ?? layoutEdgeId(e, i),
      width: w,
      height: 16,
    });
  });
  layout(g);
  const positions = new Map<string, GraphPosition>();
  for (const id of g.nodes()) {
    const n = g.node(id);
    if (!n) continue;
    positions.set(id, {
      x: (n.x ?? 0) - (n.width ?? 0) / 2,
      y: (n.y ?? 0) - (n.height ?? 0) / 2,
    });
  }
  return positions;
}

/**
 * Reactive dagre positions for maybe-ref Go nodes/edges and a layout preset.
 */
export function useGraphLayout(
  nodes: MaybeRefOrGetter<readonly LayoutNode[] | null | undefined>,
  edges: MaybeRefOrGetter<readonly LayoutEdge[] | null | undefined>,
  mode: MaybeRefOrGetter<GraphLayoutMode>,
): {
  positions: ShallowRef<Map<string, GraphPosition>>;
} {
  const positions = shallowRef(new Map<string, GraphPosition>());
  let gen = 0;

  watch(
    () => [toValue(nodes), toValue(edges), toValue(mode)] as const,
    ([ns, es, m]) => {
      const my = ++gen;
      if (!ns?.length) {
        positions.value = new Map();
        return;
      }
      const next = layoutDagre(ns ?? [], es ?? [], m);
      if (my !== gen) return;
      positions.value = next;
    },
    { immediate: true },
  );

  onScopeDispose(() => {
    gen += 1;
  });

  return { positions };
}
