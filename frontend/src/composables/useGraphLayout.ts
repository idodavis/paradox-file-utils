/**
 * ELK layered layout: Go graph payloads in, Vue Flow {x,y} out.
 * Origin clusters pack in ELK; positions are flattened so edges paint on top.
 */
import {
  onScopeDispose,
  shallowRef,
  toValue,
  watch,
  type MaybeRefOrGetter,
  type ShallowRef,
} from "vue";
import ELK, { type ElkExtendedEdge, type ElkNode } from "elkjs/lib/elk.bundled.js";

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

/** ELK preset: left→right "happens after", or top-down tree. */
export type GraphLayoutMode = "lr" | "tree";

/** Top-left pixel position for one Vue Flow node. */
export interface GraphPosition {
  x: number;
  y: number;
}

/** Minimal node identity ELK needs, plus role/kind/origin for box and grouping. */
export interface LayoutNode {
  id: string;
  role?: string;
  kind?: string;
  origin?: string;
  originName?: string;
}

/** Minimal directed edge ELK needs (Go `from`/`to`). */
export interface LayoutEdge {
  from: string;
  to: string;
}

/** Laid-out origin parent for a Vue Flow nested group. */
export interface OriginLayout {
  id: string;
  origin: string;
  label: string;
  x: number;
  y: number;
  width: number;
  height: number;
}

const elk = new ELK();

/** Vue Flow parent id for a payload origin (empty = vanilla). */
export function originGroupId(origin?: string): string {
  return `origin:${origin || "vanilla"}`;
}

/** Pixel box for a node given its role, then kind (callers stay small). */
export function nodeBox(n: { role?: string; kind?: string }): {
  width: number;
  height: number;
} {
  if (n.role === "caller") return { width: CALLER_W, height: CALLER_H };
  if (n.role === "root") return { width: ROOT_W, height: ROOT_H };
  if (n.kind && n.kind !== "event" && n.kind !== "more") {
    return { width: 280, height: 110 };
  }
  return { width: NODE_W, height: NODE_H };
}

/** ELK layered direction for a layout preset. */
function elkDirection(mode: GraphLayoutMode): string {
  switch (mode) {
    case "lr":
      return "RIGHT";
    case "tree":
      return "DOWN";
    default: {
      const _never: never = mode;
      throw new Error(`unknown layout ${_never}`);
    }
  }
}

/** Node and layer spacing for a layout preset. */
function elkSpacing(mode: GraphLayoutMode): {
  node: string; edge: string; layer: string
} {
  switch (mode) {
    case "lr":
      return { node: "96", edge: "48", layer: "340" };
    case "tree":
      return { node: "80", edge: "40", layer: "180" };
    default: {
      const _never: never = mode;
      throw new Error(`unknown layout ${_never}`);
    }
  }
}

interface LayoutResult {
  positions: Map<string, GraphPosition>;
  origins: Map<string, OriginLayout>;
}

/** Run ELK once on a snapshot of nodes and edges. */
async function layoutElk(
  nodes: readonly LayoutNode[],
  edges: readonly LayoutEdge[],
  mode: GraphLayoutMode,
): Promise<LayoutResult> {
  const empty: LayoutResult = { positions: new Map(), origins: new Map() };
  const ids = new Set<string>();
  const groups = new Map<string, ElkNode>();
  const labels = new Map<string, string>();
  const originsByGroup = new Map<string, string>();

  for (const n of nodes) {
    if (!n.id || ids.has(n.id)) continue;
    ids.add(n.id);
    const gid = originGroupId(n.origin);
    const origin = n.origin || "vanilla";
    let group = groups.get(gid);
    if (!group) {
      group = { id: gid, children: [] };
      groups.set(gid, group);
      originsByGroup.set(gid, origin);
    }
    if (n.originName) labels.set(gid, n.originName);
    else if (!labels.has(gid)) labels.set(gid, origin);
    const box = nodeBox(n);
    group.children!.push({ id: n.id, width: box.width, height: box.height });
  }
  if (!ids.size) return empty;

  for (const [gid, origin] of originsByGroup) {
    if (!labels.has(gid)) labels.set(gid, origin);
  }

  const elkEdges: ElkExtendedEdge[] = [];
  edges.forEach((e, i) => {
    if (!ids.has(e.from) || !ids.has(e.to)) return;
    elkEdges.push({
      id: `e${i}:${e.from}->${e.to}`,
      sources: [e.from],
      targets: [e.to],
    });
  });

  const space = elkSpacing(mode);
  const dir = elkDirection(mode);
  const graph: ElkNode = {
    id: "root",
    layoutOptions: {
      "elk.algorithm": "layered",
      "elk.direction": dir,
      "elk.hierarchyHandling": "INCLUDE_CHILDREN",
      "elk.spacing.nodeNode": space.node,
      "elk.spacing.edgeNode": space.edge,
      "elk.layered.spacing.nodeNodeBetweenLayers": space.layer,
      "elk.edgeRouting": "ORTHOGONAL",
      "elk.layered.crossingMinimization.strategy": "LAYER_SWEEP",
    },
    children: [...groups.values()].map((g) => ({
      ...g,
      layoutOptions: {
        "elk.padding": "[top=28,left=12,bottom=12,right=12]",
        "elk.spacing.nodeNode": space.node,
      },
    })),
    edges: elkEdges,
  };

  const laid = await elk.layout(graph);
  const positions = new Map<string, GraphPosition>();
  const origins = new Map<string, OriginLayout>();
  for (const g of laid.children ?? []) {
    const origin = originsByGroup.get(g.id) ?? "vanilla";
    origins.set(g.id, {
      id: g.id,
      origin,
      label: labels.get(g.id) ?? origin,
      x: g.x ?? 0,
      y: g.y ?? 0,
      width: g.width ?? 0,
      height: g.height ?? 0,
    });
    for (const child of g.children ?? []) {
      positions.set(child.id, {
        x: (g.x ?? 0) + (child.x ?? 0),
        y: (g.y ?? 0) + (child.y ?? 0),
      });
    }
  }
  return { positions, origins };
}

/**
 * Reactive ELK positions for maybe-ref Go nodes/edges and a layout preset.
 * Layout is async; ignore stale runs with a generation counter.
 */
export function useGraphLayout(
  nodes: MaybeRefOrGetter<readonly LayoutNode[] | null | undefined>,
  edges: MaybeRefOrGetter<readonly LayoutEdge[] | null | undefined>,
  mode: MaybeRefOrGetter<GraphLayoutMode>,
): {
  positions: ShallowRef<Map<string, GraphPosition>>;
  origins: ShallowRef<Map<string, OriginLayout>>;
} {
  const positions = shallowRef(new Map<string, GraphPosition>());
  const origins = shallowRef(new Map<string, OriginLayout>());
  let gen = 0;

  watch(
    () => [toValue(nodes), toValue(edges), toValue(mode)] as const,
    async ([ns, es, m]) => {
      const my = ++gen;
      if (!ns?.length) {
        positions.value = new Map();
        origins.value = new Map();
        return;
      }
      const result = await layoutElk(ns ?? [], es ?? [], m);
      if (my !== gen) return;
      positions.value = result.positions;
      origins.value = result.origins;
    },
    { immediate: true },
  );

  onScopeDispose(() => {
    gen += 1;
  });

  return { positions, origins };
}
