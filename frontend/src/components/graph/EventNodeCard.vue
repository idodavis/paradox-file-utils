<script setup lang="ts">
/**
 * Compact Vue Flow card for one Go EventGraphNode. No layout or graph logic.
 */
import { Handle, Position, type NodeProps } from "@vue-flow/core";
import type { EventGraphNode } from "@services/internal/graph/models";
import { NODE_W } from "../../composables/useGraphLayout";

const props = defineProps<NodeProps<EventGraphNode>>();
</script>

<template>
  <div
    class="rounded-md border px-2 py-1.5 text-left shadow-sm"
    :class="
      props.selected
        ? 'border-primary bg-elevated ring-1 ring-primary'
        : props.data.source === 'mod'
          ? 'border-primary/60 bg-elevated'
          : 'border-default bg-default'
    "
    :style="{ width: `${NODE_W}px` }"
  >
    <Handle
      type="target"
      :position="props.targetPosition ?? Position.Left"
      :connectable="false"
    />
    <div class="truncate text-sm font-medium text-default">
      {{ props.data.title || props.id }}
    </div>
    <div class="mt-0.5 flex flex-wrap items-center gap-1 text-xs text-muted">
      <UBadge color="neutral" variant="subtle" size="xs">
        {{ props.data.kind }}
      </UBadge>
      <UBadge
        :color="props.data.source === 'mod' ? 'primary' : 'neutral'"
        variant="subtle"
        size="xs"
      >
        {{ props.data.source }}
      </UBadge>
      <span v-if="props.data.fires">{{ props.data.fires }} fires</span>
      <span v-if="props.data.theme" class="truncate">{{ props.data.theme }}</span>
    </div>
    <Handle
      type="source"
      :position="props.sourcePosition ?? Position.Right"
      :connectable="false"
    />
  </div>
</template>
