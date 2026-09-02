<script setup lang="ts">
/**
 * Compact Vue Flow card: shared rectangle, kind header strip, origin hex badge.
 */
import { computed } from "vue";
import { Handle, Position, type NodeProps } from "@vue-flow/core";
import type { EventGraphNode } from "@services/internal/views/models";
import { nodeBox } from "../../composables/useGraphLayout";
import { originHexByOriginId } from "../../ide/rootDecorations";
import OriginBadge from "../OriginBadge.vue";

const props = defineProps<NodeProps<EventGraphNode>>();

const box = computed(() => nodeBox(props.data ?? {}));
const isRoot = computed(() => props.data?.role === "root");
const isMore = computed(() => props.data?.kind === "more");
const hex = computed(() => originHexByOriginId(props.data?.origin ?? ""));
const headerTint = computed(() => {
  switch (props.data?.kind) {
    case "event":
      return "color-mix(in oklab, var(--ui-info) 20%, transparent)";
    case "on_action":
      return "color-mix(in oklab, var(--ui-accent) 20%, transparent)";
    case "decision":
      return "color-mix(in oklab, var(--ui-warning) 20%, transparent)";
    case "more":
      return undefined;
    default:
      return undefined;
  }
});
const headerIcon = computed(() => {
  switch (props.data?.kind) {
    case "event":
      return "i-lucide-circle";
    case "on_action":
      return "i-lucide-hexagon";
    case "decision":
      return "i-lucide-diamond";
    case "more":
      return "";
    default:
      return "i-lucide-dot";
  }
});
const badge = computed(() => props.data?.originName || "");
</script>

<template>
  <div
    class="overflow-hidden rounded-md border bg-default text-left shadow-sm"
    :class="{
      'border-2 border-default': isRoot && !props.selected,
      'border-dashed border-muted': isMore && !props.selected,
      'border-primary ring-1 ring-primary': props.selected,
      'border-default': !props.selected && !isRoot && !isMore,
    }"
    :style="{ width: `${box.width}px`, height: `${box.height}px` }"
  >
    <Handle
      type="target"
      :position="props.targetPosition ?? Position.Left"
      :connectable="false"
    />
    <div
      class="flex h-5 items-center gap-1 border-b px-2 text-[10px] text-muted"
      :class="isMore ? 'border-dashed border-muted' : 'border-default'"
      :style="headerTint ? { backgroundColor: headerTint } : undefined"
    >
      <span v-if="headerIcon" :class="headerIcon" class="size-3 shrink-0" />
      <span class="truncate">{{ isMore ? "" : props.data?.kind }}</span>
    </div>
    <div class="px-2 py-1">
      <div class="truncate text-sm font-medium text-default">
        {{ isMore ? props.data.title : props.id }}
      </div>
      <div
        v-if="!isMore && props.data.title"
        class="truncate text-xs text-muted"
      >
        {{ props.data.title }}
      </div>
      <div
        v-if="!isMore"
        class="mt-0.5 flex flex-wrap items-center gap-1 text-xs text-muted"
      >
        <OriginBadge v-if="badge" :label="badge" :hex="hex" />
        <span v-if="props.data.fires">{{ props.data.fires }} fires</span>
      </div>
    </div>
    <Handle
      type="source"
      :position="props.sourcePosition ?? Position.Right"
      :connectable="false"
    />
  </div>
</template>
