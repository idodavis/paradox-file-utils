<script setup lang="ts">
/**
 * Vue Flow canvas: feeds Go nodes/edges through dagre, then renders.
 * Click selects; double-click re-roots. No graph derivation.
 */
import { computed, nextTick, onActivated, watch } from "vue";
import {
  MarkerType,
  Position,
  VueFlow,
  useVueFlow,
  type Edge,
  type Node,
  type NodeMouseEvent,
} from "@vue-flow/core";
import { Background } from "@vue-flow/background";
import { Controls } from "@vue-flow/controls";
import type {
  EventGraphEdge,
  EventGraphNode,
} from "@services/internal/views/models";
import {
  nodeBox,
  useGraphLayout,
  type GraphLayoutMode,
} from "../../composables/useGraphLayout";
import EventNodeCard from "./EventNodeCard.vue";

const FLOW_ID = "pmt-graph-canvas";
const HULL_PAD = 28;
const LABEL_ZOOM = 0.85;

type EdgeKind =
  | "via"
  | "option"
  | "immediate"
  | "trigger"
  | "on_action"
  | "events"
  | "effect";

const props = withDefaults(
  defineProps<{
    nodes: EventGraphNode[] | null;
    edges: EventGraphEdge[] | null;
    layout: GraphLayoutMode;
    selectedId?: string;
    rootId?: string;
    originColors?: Record<string, string>;
  }>(),
  {
    nodes: () => [],
    edges: () => [],
    selectedId: "",
    rootId: "",
    originColors: () => ({}),
  },
);

const emit = defineEmits<{
  select: [id: string];
  reroot: [id: string];
}>();

const { positions } = useGraphLayout(
  () => props.nodes,
  () => props.edges,
  () => props.layout,
);

const { fitView, viewport } = useVueFlow(FLOW_ID);

/** Handle sides for the active dagre rankdir. */
function handlePos(mode: GraphLayoutMode): {
  source: Position;
  target: Position;
} {
  switch (mode) {
    case "lr":
      return { source: Position.Right, target: Position.Left };
    case "tree":
      return { source: Position.Bottom, target: Position.Top };
    default: {
      const _never: never = mode;
      throw new Error(`unknown layout ${_never}`);
    }
  }
}

/** Origin key used for hull grouping and color lookup. */
function originKey(n: EventGraphNode): string {
  return n.origin || "vanilla";
}

/** 8-digit hex with alpha for a soft origin wash. */
function hexAlpha(hex: string, a: number): string {
  const n = Math.round(a * 255).toString(16).padStart(2, "0");
  return `${hex}${n}`;
}

/** Map a payload kind to the closed EdgeKind union. */
function asEdgeKind(kind: string | undefined): EdgeKind {
  switch (kind) {
    case "via":
    case "option":
    case "immediate":
    case "trigger":
    case "on_action":
    case "events":
    case "effect":
      return kind;
    default:
      return "events";
  }
}

/** Theme token stroke for an edge kind. */
function edgeStroke(kind: string | undefined): string {
  const k = asEdgeKind(kind);
  switch (k) {
    case "immediate":
      return "var(--ui-success)";
    case "option":
      return "var(--ui-warning)";
    case "via":
      return "var(--ui-secondary)";
    case "trigger":
      return "var(--ui-info)";
    case "on_action":
      return "var(--ui-accent)";
    case "events":
    case "effect":
      return "var(--ui-text-muted)";
    default: {
      const _never: never = k;
      return _never;
    }
  }
}

/** Short edge tag shown only when zoomed in. */
function shortLabel(kind: string | undefined): string {
  const k = asEdgeKind(kind);
  switch (k) {
    case "immediate":
      return "imm";
    case "option":
      return "opt";
    case "via":
      return "via";
    case "trigger":
      return "trig";
    case "on_action":
      return "oa";
    case "events":
    case "effect":
      return "evt";
    default: {
      const _never: never = k;
      return _never;
    }
  }
}

const eventNodes = computed((): Node<EventGraphNode>[] => {
  const hp = handlePos(props.layout);
  return (props.nodes ?? []).map((n) => {
    const box = nodeBox(n);
    return {
      id: n.id,
      type: "event",
      position: positions.value.get(n.id) ?? { x: 0, y: 0 },
      data: n,
      selected: n.id === props.selectedId,
      sourcePosition: hp.source,
      targetPosition: hp.target,
      width: box.width,
      height: box.height,
      draggable: false,
      connectable: false,
    };
  });
});

const hullNodes = computed((): Node[] => {
  const list = (props.nodes ?? []).filter((n) => n.kind !== "more");
  const by = new Map<
    string,
    { minX: number; minY: number; maxX: number; maxY: number; label: string }
  >();
  for (const n of list) {
    const p = positions.value.get(n.id);
    if (!p) continue;
    const key = originKey(n);
    const box = nodeBox(n);
    const cur = by.get(key);
    const maxX = p.x + box.width;
    const maxY = p.y + box.height;
    const label = n.originName || "Vanilla";
    if (!cur) {
      by.set(key, { minX: p.x, minY: p.y, maxX, maxY, label });
      continue;
    }
    cur.minX = Math.min(cur.minX, p.x);
    cur.minY = Math.min(cur.minY, p.y);
    cur.maxX = Math.max(cur.maxX, maxX);
    cur.maxY = Math.max(cur.maxY, maxY);
  }
  if (by.size < 2) return [];
  const out: Node[] = [];
  for (const [key, b] of by) {
    const hex = props.originColors[key] ?? "#5B9A8B";
    const w = b.maxX - b.minX + HULL_PAD * 2;
    const h = b.maxY - b.minY + HULL_PAD * 2;
    out.push({
      id: `hull:${key}`,
      type: "hull",
      position: { x: b.minX - HULL_PAD, y: b.minY - HULL_PAD },
      data: { fill: hexAlpha(hex, 0.16), label: b.label },
      selectable: false,
      draggable: false,
      connectable: false,
      focusable: false,
      style: { width: `${w}px`, height: `${h}px`, zIndex: -1 },
      class: "pointer-events-none",
    });
  }
  return out;
});

const flowNodes = computed((): Node[] => [...hullNodes.value, ...eventNodes.value]);

const labelsVisible = computed(
  () => (viewport.value?.zoom ?? 1) >= LABEL_ZOOM,
);

const flowEdges = computed((): Edge[] => {
  const ids = new Set((props.nodes ?? []).map((n) => n.id));
  const labels = labelsVisible.value;
  return (props.edges ?? [])
    .filter((e) => ids.has(e.from) && ids.has(e.to))
    .map((e, i) => {
      const stroke = edgeStroke(e.kind);
      return {
        id: `${e.from}->${e.to}:${e.via}:${i}`,
        source: e.from,
        target: e.to,
        type: "smoothstep",
        label: labels ? shortLabel(e.kind) : undefined,
        title: e.label || e.via,
        style: { stroke },
        markerEnd: { type: MarkerType.ArrowClosed, color: stroke },
      };
    });
});

const eventNodeIds = computed(() => eventNodes.value.map((n) => n.id));

const layoutNonce = computed(
  () =>
    `${props.layout}:${eventNodeIds.value.join("|")}:${props.rootId}`,
);

/** Fit the canvas to event nodes (not hulls). */
async function recenter(): Promise<void> {
  const ids = eventNodeIds.value;
  if (!ids.length) return;
  await nextTick();
  await nextTick();
  void fitView({ padding: 0.2, nodes: ids, duration: 200 });
}

watch(layoutNonce, () => {
  void recenter();
});

onActivated(() => {
  window.dispatchEvent(new Event("resize"));
  void recenter();
});

defineExpose({ recenter });

/** Select the clicked Go node id. */
function onClick(ev: NodeMouseEvent): void {
  if (ev.node.type === "hull") return;
  emit("select", ev.node.id);
}

/** Re-root the graph on the double-clicked node. */
function onDblclick(ev: NodeMouseEvent): void {
  if (ev.node.type === "hull") return;
  const data = ev.node.data as EventGraphNode | undefined;
  if (data?.kind === "more") {
    emit("select", ev.node.id);
    return;
  }
  emit("reroot", ev.node.id);
}
</script>

<template>
  <div class="h-full min-h-0 w-full text-default">
    <VueFlow
      :id="FLOW_ID"
      :nodes="flowNodes"
      :edges="flowEdges"
      :nodes-draggable="false"
      :nodes-connectable="false"
      :edges-updatable="false"
      :fit-view-on-init="true"
      :only-render-visible-elements="true"
      :min-zoom="0.35"
      :max-zoom="2"
      class="h-full w-full"
      @node-click="onClick"
      @node-double-click="onDblclick"
    >
      <template #node-event="np">
        <EventNodeCard v-bind="np" />
      </template>
      <template #node-hull="np">
        <div
          class="pointer-events-none relative h-full w-full rounded-xl"
          :style="{ backgroundColor: np.data.fill }"
        >
          <span
            class="pointer-events-none absolute top-1 left-2 text-xs text-muted"
          >
            {{ np.data.label }}
          </span>
        </div>
      </template>
      <Background />
      <Controls />
    </VueFlow>
  </div>
</template>

<style scoped>
:deep(.vue-flow__edge-text) {
  fill: var(--ui-text, currentColor);
}
:deep(.vue-flow__edge-textbg) {
  fill: var(--ui-bg, transparent);
}
:deep(.vue-flow__controls) {
  box-shadow: none;
  border: 1px solid var(--ui-border);
  border-radius: 0.375rem;
  overflow: hidden;
}
</style>
