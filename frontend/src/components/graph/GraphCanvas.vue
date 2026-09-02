<script setup lang="ts">
/**
 * Vue Flow canvas: dagre-placed event cards and default edges.
 * Click selects; double-click re-roots. Drag is local until layout/root changes.
 */
import { computed, nextTick, onActivated, shallowRef, watch } from "vue";
import {
  BaseEdge,
  ConnectionMode,
  EdgeLabelRenderer,
  MarkerType,
  Position,
  VueFlow,
  getSmoothStepPath,
  useVueFlow,
  type Edge,
  type EdgeProps,
  type Node,
  type NodeDragEvent,
  type NodeMouseEvent,
} from "@vue-flow/core";
import { Background } from "@vue-flow/background";
import { Controls } from "@vue-flow/controls";
import { MiniMap } from "@vue-flow/minimap";
import "@vue-flow/minimap/dist/style.css";
import type {
  EventGraphEdge,
  EventGraphNode,
} from "@services/internal/views/models";
import {
  LABEL_SHOW_CAP,
  layoutEdgeId,
  nodeBox,
  useGraphLayout,
  type GraphLayoutMode,
  type GraphPosition,
} from "../../composables/useGraphLayout";
import EventNodeCard from "./EventNodeCard.vue";

const FLOW_ID = "pmt-graph-canvas";

const props = withDefaults(
  defineProps<{
    nodes: EventGraphNode[] | null;
    edges: EventGraphEdge[] | null;
    layout: GraphLayoutMode;
    selectedId?: string;
    rootId?: string;
  }>(),
  {
    nodes: () => [],
    edges: () => [],
    selectedId: "",
    rootId: "",
  },
);

const emit = defineEmits<{
  select: [id: string];
  reroot: [id: string];
  clearSelection: [];
}>();

const { positions } = useGraphLayout(
  () => props.nodes,
  () => props.edges,
  () => props.layout,
);

const placed = shallowRef(new Map<string, GraphPosition>());
watch(positions, (p) => {
  placed.value = new Map(p);
});

const { fitView } = useVueFlow(FLOW_ID);

/** Handle sides for the active dagre direction. */
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

/** Truncate only when the label is longer than the spacing cap heuristic. */
function edgeText(text?: string): { label?: string; title?: string } {
  const full = text ?? "";
  if (!full) return {};
  if (full.length <= LABEL_SHOW_CAP) return { label: full, title: full };
  return { label: `${full.slice(0, LABEL_SHOW_CAP - 1)}…`, title: full };
}

/** Tooltip text for a truncated edge label. */
type EdgeData = { title?: string };

/** Smoothstep path plus label anchor for the default edge slot. */
function smoothGeom(p: EdgeProps) {
  const [path, labelX, labelY] = getSmoothStepPath({
    sourceX: p.sourceX,
    sourceY: p.sourceY,
    targetX: p.targetX,
    targetY: p.targetY,
    sourcePosition: p.sourcePosition,
    targetPosition: p.targetPosition,
  });
  return { path, labelX, labelY };
}

/** Displayed edge label (Vue Flow also allows object labels). */
function edgeLabelText(ep: EdgeProps<EdgeData>): string {
  return typeof ep.label === "string" ? ep.label : "";
}

/** Full label for the tooltip; falls back to the truncated text. */
function edgeTooltip(ep: EdgeProps<EdgeData>): string {
  return ep.data?.title || edgeLabelText(ep);
}

const eventNodes = computed((): Node<EventGraphNode>[] => {
  const list = (props.nodes ?? []).filter((n) => n.id && placed.value.has(n.id));
  const hp = handlePos(props.layout);
  const focus = focusIds.value;
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
      class: focus && !focus.has(n.id) ? "pmt-dim" : "",
    };
  });
});

const focusIds = computed((): Set<string> | null => {
  const id = props.selectedId;
  if (!id) return null;
  const adj = new Set<string>([id]);
  for (const e of props.edges ?? []) {
    if (e.from === id || e.to === id) {
      adj.add(e.from);
      adj.add(e.to);
    }
  }
  return adj;
});

const flowEdges = computed((): Edge<EdgeData>[] => {
  const focus = focusIds.value;
  return (props.edges ?? [])
    .map((e, i) => ({ e, i, eid: layoutEdgeId(e, i) }))
    .filter(({ e }) => placed.value.has(e.from) && placed.value.has(e.to))
    .map(({ e, eid }) => {
      const kind = e.kind || "events";
      const stub = e.via === "more" || e.via === "more-in";
      const dim = !!(focus && !focus.has(e.from) && !focus.has(e.to));
      const text = edgeText(e.label);
      return {
        id: eid,
        source: e.from,
        target: e.to,
        type: "smoothstep",
        label: text.label,
        data: { title: text.title || e.via },
        class: `pmt-edge pmt-edge--${kind}${stub ? " pmt-edge--stub" : ""}${
          dim ? " pmt-dim" : ""
        }`,
        markerEnd: { type: MarkerType.ArrowClosed },
      } satisfies Edge<EdgeData>;
    });
});

const eventNodeIds = computed(() => eventNodes.value.map((n) => n.id));

const layoutNonce = computed(
  () => `${props.layout}:${eventNodeIds.value.join("|")}:${props.rootId}`,
);

/** Fit the canvas to event cards. Does not re-run dagre. */
async function recenter(): Promise<void> {
  const ids = eventNodeIds.value;
  if (!ids.length) return;
  await nextTick();
  await nextTick();
  void fitView({ padding: 0.28, nodes: ids, duration: 200, maxZoom: 1 });
}

watch(layoutNonce, () => {
  void recenter();
});

onActivated(() => {
  window.dispatchEvent(new Event("resize"));
  void recenter();
});

defineExpose({ recenter });

/** Keep dragged coordinates until the next dagre result. */
function onNodeDragStop(ev: NodeDragEvent): void {
  if (ev.node.type !== "event") return;
  placed.value.set(ev.node.id, { ...ev.node.position });
}

/** Select the clicked Go node id. */
function onClick(ev: NodeMouseEvent): void {
  emit("select", ev.node.id);
}

/** Re-root the graph on the double-clicked node. */
function onDblclick(ev: NodeMouseEvent): void {
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
      :nodes="eventNodes"
      :edges="flowEdges"
      :connection-mode="ConnectionMode.Loose"
      :nodes-draggable="true"
      :nodes-connectable="false"
      :edges-updatable="false"
      :fit-view-on-init="true"
      :only-render-visible-elements="false"
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
      <template #edge-smoothstep="ep">
        <BaseEdge
          :id="ep.id"
          :path="smoothGeom(ep).path"
          :marker-end="ep.markerEnd"
          :interaction-width="ep.interactionWidth"
        />
        <EdgeLabelRenderer v-if="edgeLabelText(ep)">
          <div
            class="nodrag nopan pointer-events-auto absolute"
            :style="{
              transform:
                `translate(-50%, -50%) translate(${smoothGeom(ep).labelX}px,` +
                `${smoothGeom(ep).labelY}px)`,
            }"
          >
            <UTooltip :text="edgeTooltip(ep)">
              <span
                class="rounded-sm bg-elevated px-1 text-[11px] text-default"
              >{{ edgeLabelText(ep) }}</span>
            </UTooltip>
          </div>
        </EdgeLabelRenderer>
      </template>
      <Background />
      <MiniMap pannable zoomable />
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
  stroke-width: 2;
  vector-effect: non-scaling-stroke;
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
:deep(.vue-flow__edge.pmt-edge--events .vue-flow__edge-path) {
  stroke: var(--ui-text-muted);
}
:deep(.vue-flow__edge.pmt-edge--stub .vue-flow__edge-path) {
  stroke-dasharray: 6 4;
}
:deep(.vue-flow__node.pmt-dim),
:deep(.vue-flow__edge.pmt-dim) {
  opacity: 0.55;
}
:deep(.vue-flow__minimap) {
  background-color: var(--ui-bg-elevated, var(--ui-bg));
  border: 1px solid var(--ui-border);
}
</style>
