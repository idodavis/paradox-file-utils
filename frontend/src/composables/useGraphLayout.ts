/**
 * Dagre layout wrapper: Go graph payloads in, Vue Flow {x,y} out.
 * Rankdir/ranker are the only knobs; no force-sim or ranking math.
 */
import { computed, toValue, type ComputedRef, type MaybeRefOrGetter } from "vue";
import { Graph, layout } from "@dagrejs/dagre";
import type { GraphLabel } from "@dagrejs/dagre";

/** Vue Flow node box used as dagre's width/height. */
export const NODE_W = 240;
/** Vue Flow node box used as dagre's width/height. */
export const NODE_H = 72;

/** Dagre preset: left→right "happens after", or top-down tree. */
export type GraphLayoutMode = "lr" | "tree";

/** Top-left pixel position for one Vue Flow node. */
export interface GraphPosition {
  x: number;
  y: number;
}

/** Minimal node identity dagre needs. */
export interface LayoutNode {
  id: string;
}

/** Minimal directed edge dagre needs (Go `from`/`to`). */
export interface LayoutEdge {
  from: string;
  to: string;
}

/** Dagre graph label for a layout preset. */
function graphLabel(mode: GraphLayoutMode): GraphLabel {
  switch (mode) {
    case "lr":
      return {
        rankdir: "LR",
        ranker: "network-simplex",
        nodesep: 40,
        ranksep: 80,
      };
    case "tree":
      return {
        rankdir: "TB",
        ranker: "tight-tree",
        nodesep: 40,
        ranksep: 80,
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
  for (const n of nodes) {
    if (!n.id || ids.has(n.id)) continue;
    ids.add(n.id);
    g.setNode(n.id, { width: NODE_W, height: NODE_H });
  }
  for (const e of edges) {
    if (!ids.has(e.from) || !ids.has(e.to)) continue;
    g.setEdge(e.from, e.to);
  }
  layout(g);
  const out = new Map<string, GraphPosition>();
  for (const id of ids) {
    const n = g.node(id);
    out.set(id, {
      x: (n?.x ?? 0) - NODE_W / 2,
      y: (n?.y ?? 0) - NODE_H / 2,
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
