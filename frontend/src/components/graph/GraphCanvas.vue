<script setup lang="ts">
/**
 * Vue Flow canvas: flattened origin boxes under event nodes; payload edges on top.
 * Click selects; double-click re-roots. Drag is local until layout/root changes.
 */
import { computed, nextTick, onActivated, ref, watch } from "vue";
import {
  ConnectionMode,
  MarkerType,
  Position,
  VueFlow,
  useVueFlow,
  type Edge,
  type Node,
  type NodeDragEvent,
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
  type GraphPosition,
} from "../../composables/useGraphLayout";
import EventNodeCard from "./EventNodeCard.vue";

const FLOW_ID = "pmt-graph-canvas";
const LABEL_ZOOM = 1;

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
  clearSelection: [];
}>();

const { positions, origins } = useGraphLayout(
  () => props.nodes,
  () => props.edges,
  () => props.layout,
);

const placed = ref(new Map<string, GraphPosition>());
watch(positions, (p) => {
  placed.value = new Map(p);
});

const { fitView, viewport } = useVueFlow(FLOW_ID);

/** Handle sides for the active ELK direction. */
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

const originNodes = computed((): Node[] => {
  const out: Node[] = [];
  for (const g of origins.value.values()) {
    const hex = props.originColors[g.origin] ?? "#5B9A8B";
    out.push({
      id: g.id,
      type: "origin",
      position: { x: g.x, y: g.y },
      data: { label: g.label },
      selectable: false,
      draggable: false,
      connectable: false,
      focusable: false,
      zIndex: 0,
      style: {
        width: `${g.width}px`,
        height: `${g.height}px`,
        backgroundColor: `${hex}29`,
        zIndex: 0,
      },
      class: "pointer-events-none rounded-xl",
    });
  }
  return out;
});

const eventNodes = computed((): Node<EventGraphNode>[] => {
  const list = props.nodes ?? [];
  if (list.some((n) => n.id && !placed.value.has(n.id))) return [];
  const hp = handlePos(props.layout);
  return list.map((n) => {
    const box = nodeBox(n);
    return {
      id: n.id,
      type: "event",
      position: placed.value.get(n.id) ?? { x: 0, y: 0 },
      data: n,
      selected: n.id === props.selectedId,
      sourcePosition: hp.source,
      targetPosition: hp.target,
      width: box.width,
      height: box.height,
      draggable: true,
      connectable: false,
      zIndex: 2,
    };
  });
});

const flowNodes = computed((): Node[] => [...originNodes.value, ...eventNodes.value]);

const labelsVisible = computed(
  () => (viewport.value?.zoom ?? 1) >= LABEL_ZOOM,
);

const flowEdges = computed((): Edge[] => {
  const ids = new Set((props.nodes ?? []).map((n) => n.id));
  const labels = labelsVisible.value;
  return (props.edges ?? [])
    .filter((e) => ids.has(e.from) && ids.has(e.to))
    .map((e, i) => {
      const kind = e.kind || "events";
      return {
        id: `${e.from}->${e.to}:${e.via}:${i}`,
        source: e.from,
        target: e.to,
        type: "smoothstep",
        pathOptions: { borderRadius: 8 },
        label: labels ? e.label : undefined,
        title: e.label || e.via,
        class: `pmt-edge pmt-edge--${kind}`,
        markerEnd: { type: MarkerType.ArrowClosed },
      };
    });
});

const eventNodeIds = computed(() => eventNodes.value.map((n) => n.id));

const layoutNonce = computed(
  () =>
    `${props.layout}:${eventNodeIds.value.join("|")}:${props.rootId}:${origins.value.size}`,
);

/** Fit the canvas to event cards. Does not re-run ELK. */
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

/** Keep dragged coordinates until the next ELK result. */
function onNodeDragStop(ev: NodeDragEvent): void {
  if (ev.node.type !== "event") return;
  placed.value.set(ev.node.id, { ...ev.node.position });
}

/** Select the clicked Go node id. */
function onClick(ev: NodeMouseEvent): void {
  if (ev.node.type === "origin") return;
  emit("select", ev.node.id);
}

/** Re-root the graph on the double-clicked node. */
function onDblclick(ev: NodeMouseEvent): void {
  if (ev.node.type === "origin") return;
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
      :connection-mode="ConnectionMode.Loose"
      :nodes-draggable="true"
      :nodes-connectable="false"
      :edges-updatable="false"
      :fit-view-on-init="true"
      :only-render-visible-elements="true"
      :min-zoom="0.35"
      :max-zoom="2"
      class="h-full w-full"
      @node-click="onClick"
      @node-double-click="onDblclick"
      @node-drag-stop="onNodeDragStop"
      @pane-click="emit('clearSelection')"
    >
      <template #node-event="np">
        <EventNodeCard v-bind="np" />
      </template>
      <template #node-origin="np">
        <div class="pointer-events-none relative h-full w-full">
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
  font-size: 11px;
}
:deep(.vue-flow__edge-textbg) {
  fill: var(--ui-bg-elevated, var(--ui-bg));
}
:deep(.vue-flow__controls) {
  box-shadow: none;
  border: 1px solid var(--ui-border);
  border-radius: 0.375rem;
  overflow: hidden;
}
:deep(.vue-flow__edge.pmt-edge .vue-flow__edge-path) {
  stroke: var(--ui-text-muted);
}
:deep(.vue-flow__edge.pmt-edge--immediate .vue-flow__edge-path) {
  stroke: var(--ui-success);
}
:deep(.vue-flow__edge.pmt-edge--option .vue-flow__edge-path) {
  stroke: var(--ui-warning);
}
:deep(.vue-flow__edge.pmt-edge--via .vue-flow__edge-path) {
  stroke: var(--ui-secondary);
}
:deep(.vue-flow__edge.pmt-edge--trigger .vue-flow__edge-path) {
  stroke: var(--ui-info);
}
:deep(.vue-flow__edge.pmt-edge--on_action .vue-flow__edge-path) {
  stroke: var(--ui-accent);
}
</style>
