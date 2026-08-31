/**
 * Dagre layout wrapper: Go graph payloads in, Vue Flow {x,y} out.
 * Rankdir/ranker are the only knobs; no force-sim or ranking math.
 */
import { computed, toValue, type ComputedRef, type MaybeRefOrGetter } from "vue";
import { Graph, layout } from "@dagrejs/dagre";
import type { GraphLabel } from "@dagrejs/dagre";

/** Caller (inbound) card size. */
export const CALLER_W = 168;
/** Caller (inbound) card size. */
export const CALLER_H = 56;
/** Query-root card size. */
export const ROOT_W = 260;
/** Query-root card size. */
export const ROOT_H = 96;
/** Default event / more-stub card size. */
export const NODE_W = 240;
/** Default event / more-stub card size. */
export const NODE_H = 86;

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

/** Minimal directed edge dagre needs (Go `from`/`to`). */
export interface LayoutEdge {
  from: string;
  to: string;
}

/** Pixel box for a node given its role (caller/root/default). */
export function nodeBox(n: { role?: string; kind?: string }): {
  width: number;
  height: number;
} {
  if (n.role === "caller") return { width: CALLER_W, height: CALLER_H };
  if (n.role === "root") return { width: ROOT_W, height: ROOT_H };
  return { width: NODE_W, height: NODE_H };
}

/** Dagre graph label for a layout preset. */
function graphLabel(mode: GraphLayoutMode): GraphLabel {
  switch (mode) {
    case "lr":
      return {
        rankdir: "LR",
        ranker: "network-simplex",
        nodesep: 56,
        ranksep: 200,
        edgesep: 20,
      };
    case "tree":
      return {
        rankdir: "TB",
        ranker: "tight-tree",
        nodesep: 48,
        ranksep: 120,
      };
    default: {
      const _never: never = mode;
      throw new Error(`unknown layout ${_never}`);
    }
  }
}

/**
 * Run dagre once on a snapshot of nodes and edges.
 * Dagre uses center coordinates; this returns Vue Flow top-left positions.
 */
export function layoutPositions(
  nodes: readonly LayoutNode[],
  edges: readonly LayoutEdge[],
  mode: GraphLayoutMode,
): Map<string, GraphPosition> {
  const g = new Graph({ directed: true });
  g.setGraph(graphLabel(mode));
  g.setDefaultEdgeLabel(() => ({}));
  const ids = new Set<string>();
  const boxes = new Map<string, { width: number; height: number }>();
  for (const n of nodes) {
    if (!n.id || ids.has(n.id)) continue;
    ids.add(n.id);
    const box = nodeBox(n);
    boxes.set(n.id, box);
    g.setNode(n.id, box);
  }
  for (const e of edges) {
    if (!ids.has(e.from) || !ids.has(e.to)) continue;
    g.setEdge(e.from, e.to);
  }
  layout(g);
  const out = new Map<string, GraphPosition>();
  for (const id of ids) {
    const n = g.node(id);
    const box = boxes.get(id) ?? { width: NODE_W, height: NODE_H };
    out.set(id, {
      x: (n?.x ?? 0) - box.width / 2,
      y: (n?.y ?? 0) - box.height / 2,
    });
  }
  return out;
}

/**
 * Reactive dagre positions for maybe-ref Go nodes/edges and a layout preset.
 */
export function useGraphLayout(
  nodes: MaybeRefOrGetter<readonly LayoutNode[] | null | undefined>,
  edges: MaybeRefOrGetter<readonly LayoutEdge[] | null | undefined>,
  mode: MaybeRefOrGetter<GraphLayoutMode>,
): { positions: ComputedRef<Map<string, GraphPosition>> } {
  const positions = computed(() =>
    layoutPositions(toValue(nodes) ?? [], toValue(edges) ?? [], toValue(mode)),
  );
  return { positions };
}
