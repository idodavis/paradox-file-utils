<script setup lang="ts">
/**
 * Saved inventory card used on the Inventory page.
 */
import { computed } from "vue";
import type { DropdownMenuItem } from "@nuxt/ui";
import type { InventorySummary } from "@services/models";

const props = defineProps<{
  inv: InventorySummary;
  active?: boolean;
}>();

const emit = defineEmits<{
  load: [inv: InventorySummary];
  rename: [inv: InventorySummary];
  delete: [inv: InventorySummary];
}>();

const actionItems = computed<DropdownMenuItem[]>(() => [
  {
    label: "Rename",
    icon: "i-lucide-pencil",
    onSelect: () => emit("rename", props.inv),
  },
  {
    label: "Delete",
    icon: "i-lucide-trash-2",
    color: "error",
    onSelect: () => emit("delete", props.inv),
  },
]);
</script>

<template>
  <UCard class="p-3" :class="active ? 'border-primary bg-primary/5' : 'bg-default'">
    <div class="flex items-start justify-between gap-1">
      <div class="min-w-0 flex-1">
        <button
          class="text-left text-weight-medium ellipsis inventory-name-btn"
          :title="inv.name"
          @click="emit('load', inv)"
        >
          {{ inv.name }}
        </button>
      </div>
      <UDropdownMenu :items="actionItems">
        <UButton color="neutral" variant="ghost" icon="i-lucide-ellipsis" size="xs" />
      </UDropdownMenu>
    </div>
    <div class="mt-2 flex items-center gap-2">
      <UBadge variant="outline" color="neutral">{{ inv.game }}</UBadge>
      <div class="text-xs text-muted">{{ inv.totalCount ?? 0 }} items</div>
    </div>
    <div class="mt-1 text-xs text-muted">{{ inv.createdAt }}</div>

    <template #footer>
      <div class="flex justify-end">
        <UButton label="Load" variant="outline" @click="emit('load', inv)" />
      </div>
    </template>
  </UCard>
</template>
