<script setup lang="ts">
/**
 * Compact Vue Flow card for one Go EventGraphNode. No layout or graph logic.
 */
import { computed } from "vue";
import { Handle, Position, type NodeProps } from "@vue-flow/core";
import type { EventGraphNode } from "@services/internal/views/models";
import { nodeBox } from "../../composables/useGraphLayout";

const props = defineProps<NodeProps<EventGraphNode>>();

const box = computed(() => nodeBox(props.data ?? {}));
const isRoot = computed(() => props.data?.role === "root");
const isCaller = computed(() => props.data?.role === "caller");
const isMore = computed(() => props.data?.kind === "more");
const isGate = computed(
  () => props.data?.kind === "on_action" || props.data?.kind === "decision",
);
const isMod = computed(() => !!props.data?.origin);
const badge = computed(() => props.data?.originName || "");
</script>

<template>
  <div
    class="rounded-md border px-2 py-1.5 text-left shadow-sm"
    :class="{
      'ring-2 ring-primary bg-elevated': isRoot && !props.selected,
      'border-muted text-muted border-l-4 border-l-warning':
        isCaller && !props.selected,
      'border-dashed border-muted bg-default': isMore && !props.selected,
      'rounded-lg border-dashed border-secondary bg-default':
        isGate && !isMore && !props.selected,
      'border-primary bg-elevated ring-1 ring-primary': props.selected,
      'border-primary/60 bg-elevated':
        !props.selected && !isRoot && !isCaller && !isMore && !isGate && isMod,
      'border-default bg-default':
        !props.selected && !isRoot && !isCaller && !isMore && !isGate && !isMod,
    }"
    :style="{ width: `${box.width}px`, height: `${box.height}px` }"
  >
    <Handle
      type="target"
      :position="props.targetPosition ?? Position.Left"
      :connectable="false"
    />
    <div
      class="truncate font-medium text-default"
      :class="isCaller ? 'text-xs' : 'text-sm'"
    >
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
      <UBadge
        v-if="isRoot"
        color="primary"
        variant="subtle"
        size="xs"
      >
        root
      </UBadge>
      <UBadge
        v-if="isCaller"
        color="warning"
        variant="subtle"
        size="xs"
      >
        fires this
      </UBadge>
      <UBadge
        :color="isGate ? 'secondary' : 'neutral'"
        variant="subtle"
        size="xs"
      >
        {{ props.data.kind }}
      </UBadge>
      <UBadge
        v-if="badge"
        :color="isMod ? 'primary' : 'neutral'"
        variant="subtle"
        size="xs"
      >
        {{ badge }}
      </UBadge>
      <span v-if="props.data.fires">{{ props.data.fires }} fires</span>
    </div>
    <Handle
      type="source"
      :position="props.sourcePosition ?? Position.Right"
      :connectable="false"
    />
  </div>
</template>
