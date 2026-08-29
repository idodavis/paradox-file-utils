<script setup lang="ts">
/**
 * Shared overview pane: summary chips plus a clickable file/mod tree.
 */
export interface OverviewChip {
  id: string;
  label: string;
  count: number;
  color?: "error" | "warning" | "info" | "primary" | "neutral" | "success";
}

export interface OverviewNode {
  id: string;
  label: string;
  count: number;
}

const props = defineProps<{
  chips: OverviewChip[];
  nodes: OverviewNode[];
  activeChip?: string;
  activeNode?: string;
  treeLabel?: string;
}>();

const emit = defineEmits<{
  selectChip: [id: string | undefined];
  selectNode: [id: string | undefined];
}>();

/** Toggle a chip filter, or clear when clicking the active one. */
function onChip(id: string): void {
  emit("selectChip", props.activeChip === id ? undefined : id);
}

/** Toggle a tree-node filter, or clear when clicking the active one. */
function onNode(id: string): void {
  emit("selectNode", props.activeNode === id ? undefined : id);
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col gap-3 overflow-auto p-2">
    <div class="flex flex-wrap gap-1.5">
      <UButton
        v-for="chip in chips"
        :key="chip.id"
        :label="chip.label"
        size="xs"
        :color="chip.color ?? 'neutral'"
        :variant="activeChip === chip.id ? 'solid' : 'subtle'"
        @click="onChip(chip.id)"
      >
        <template #trailing>
          <UBadge
            :label="String(chip.count)"
            size="xs"
            color="neutral"
            variant="subtle"
          />
        </template>
      </UButton>
    </div>
    <div class="space-y-0.5">
      <div class="text-xs font-medium text-muted">{{ treeLabel ?? "By file" }}</div>
      <button
        v-for="node in nodes"
        :key="node.id"
        type="button"
        class="flex w-full items-center justify-between gap-2 rounded-md px-1.5
          py-0.5 text-left text-xs hover:bg-elevated"
        :class="activeNode === node.id ? 'bg-elevated text-default' : 'text-muted'"
        @click="onNode(node.id)"
      >
        <span class="min-w-0 truncate">{{ node.label }}</span>
        <UBadge
          :label="String(node.count)"
          size="xs"
          color="neutral"
          variant="subtle"
        />
      </button>
    </div>
  </div>
</template>
