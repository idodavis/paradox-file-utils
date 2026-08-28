<script setup lang="ts">
/**
 * Vue Flow canvas: feeds Go nodes/edges through dagre, then renders.
 * Click selects; double-click re-roots. No graph derivation.
 */
import { computed, nextTick, watch } from "vue";
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
} from "@services/internal/graph/models";
import {
  NODE_H,
  NODE_W,
  useGraphLayout,
  type GraphLayoutMode,
} from "../../composables/useGraphLayout";
import EventNodeCard from "./EventNodeCard.vue";

const FLOW_ID = "pmt-graph-canvas";

const props = withDefaults(
  defineProps<{
    nodes: EventGraphNode[] | null;
    edges: EventGraphEdge[] | null;
    layout: GraphLayoutMode;
    selectedId?: string;
  }>(),
  { nodes: () => [], edges: () => [], selectedId: "" },
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

const { fitView } = useVueFlow(FLOW_ID);

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

const flowNodes = computed((): Node<EventGraphNode>[] => {
  const hp = handlePos(props.layout);
  return (props.nodes ?? []).map((n) => ({
    id: n.id,
    type: "event",
    position: positions.value.get(n.id) ?? { x: 0, y: 0 },
    data: n,
    selected: n.id === props.selectedId,
    sourcePosition: hp.source,
    targetPosition: hp.target,
    width: NODE_W,
    height: NODE_H,
    draggable: false,
    connectable: false,
  }));
});

const flowEdges = computed((): Edge[] => {
  const ids = new Set((props.nodes ?? []).map((n) => n.id));
  return (props.edges ?? [])
    .filter((e) => ids.has(e.from) && ids.has(e.to))
    .map((e, i) => ({
      id: `${e.from}->${e.to}:${e.via}:${i}`,
      source: e.from,
      target: e.to,
      label: e.label || e.via,
      markerEnd: MarkerType.ArrowClosed,
    }));
});

const layoutNonce = computed(
  () =>
    `${props.layout}:${(props.nodes ?? []).map((n) => n.id).join("|")}`,
);

watch(layoutNonce, async () => {
  if (!(props.nodes ?? []).length) return;
  await nextTick();
  void fitView({ padding: 0.2 });
});

/** Select the clicked Go node id. */
function onClick(ev: NodeMouseEvent): void {
  emit("select", ev.node.id);
}

/** Re-root the graph on the double-clicked node. */
function onDblclick(ev: NodeMouseEvent): void {
  emit("reroot", ev.node.id);
}
</script>

<template>
  <div class="h-full min-h-0 w-full text-muted">
    <VueFlow
      :id="FLOW_ID"
      :nodes="flowNodes"
      :edges="flowEdges"
      :nodes-draggable="false"
      :nodes-connectable="false"
      :edges-updatable="false"
      :fit-view-on-init="true"
      :min-zoom="0.2"
      :max-zoom="2"
      class="h-full w-full"
      @node-click="onClick"
      @node-double-click="onDblclick"
    >
      <template #node-event="np">
        <EventNodeCard v-bind="np" />
      </template>
      <Background />
      <Controls />
    </VueFlow>
  </div>
</template>

<style scoped>
:deep(.vue-flow__edge-path) {
  stroke: currentColor;
}
:deep(.vue-flow__edge-text) {
  fill: var(--ui-text, currentColor);
}
:deep(.vue-flow__controls) {
  box-shadow: none;
  border: 1px solid var(--ui-border);
  border-radius: 0.375rem;
  overflow: hidden;
}
</style>
