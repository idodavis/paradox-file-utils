<script setup lang="ts">
/**
 * Compact Vue Flow card: shared rectangle, kind header strip, origin hex badge.
 */
import { computed } from "vue";
import { Handle, Position } from "@vue-flow/core";
import type { EventGraphNode } from "@services/internal/views/models";
import { nodeBox } from "../../composables/useGraphLayout";
import { originHexByOriginId } from "../../ide/rootDecorations";
import OriginBadge from "../OriginBadge.vue";

const props = defineProps<{
  id: string;
  data: EventGraphNode;
  selected?: boolean;
  sourcePosition?: Position;
  targetPosition?: Position;
}>();

const box = computed(() => nodeBox(props.data ?? {}));
const isRoot = computed(() => props.data?.role === "root");
const isMore = computed(() => props.data?.kind === "more");
const hex = computed(() => originHexByOriginId(props.data?.origin ?? ""));

/** Header strip tint and kind glyph. */
const header = computed(() => {
  switch (props.data?.kind) {
    case "event":
      return {
        tint: "color-mix(in oklab, var(--ui-info) 20%, transparent)",
        icon: "i-lucide-scroll-text",
      };
    case "on_action":
      return {
        tint: "color-mix(in oklab, var(--ui-accent) 20%, transparent)",
        icon: "i-lucide-timer",
      };
    case "decision":
      return {
        tint: "color-mix(in oklab, var(--ui-warning) 20%, transparent)",
        icon: "i-lucide-scale",
      };
    case "scripted_effect":
      return {
        tint: "color-mix(in oklab, var(--ui-success) 20%, transparent)",
        icon: "i-lucide-zap",
      };
    case "more":
      return { icon: "i-lucide-ellipsis" };
    default:
      return { icon: "i-lucide-circle-dashed" };
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
      class="flex h-5 items-center justify-between gap-1 border-b px-1.5 text-[10px] text-muted"
      :class="isMore ? 'border-dashed border-muted' : 'border-default'"
      :style="header.tint ? { backgroundColor: header.tint } : undefined"
    >
      <span class="flex min-w-0 items-center gap-1">
        <UIcon :name="header.icon" class="size-2.5 shrink-0" />
        <span class="truncate">{{ isMore ? "" : props.data?.kind }}</span>
      </span>
      <span
        v-if="props.data?.kind === 'event' && props.data.namespace"
        class="max-w-[50%] shrink-0 truncate"
      >{{ props.data.namespace }}</span>
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
