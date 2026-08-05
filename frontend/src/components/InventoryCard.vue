<script setup lang="ts">
/**
 * Saved inventory card used on the Inventory page.
 */
import type { InventorySummary } from "@services/models";

defineProps<{
  inv: InventorySummary;
  active?: boolean;
}>();

const emit = defineEmits<{
  (event: "load", inv: InventorySummary): void;
  (event: "rename", inv: InventorySummary): void;
  (event: "delete", inv: InventorySummary): void;
}>();
</script>

<template>
  <UCard class="p-3" :class="active ? 'border-primary bg-primary/5' : 'bg-default'">
    <div class="flex items-start justify-between gap-1">
      <div class="min-w-0 flex-1">
        <button class="text-left text-weight-medium ellipsis inventory-name-btn" :title="inv.name"
          @click="emit('load', inv)">
          {{ inv.name }}
        </button>
      </div>
      <div class="flex items-center gap-1">
        <UButton color="neutral" variant="ghost" icon="i-lucide-pencil" size="xs" @click.stop="emit('rename', inv)" />
        <UButton color="error" variant="ghost" icon="i-lucide-trash-2" size="xs" @click.stop="emit('delete', inv)" />
      </div>
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
