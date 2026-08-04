<script setup lang="ts">
/**
 * Simplified item details drawer for the Inventory page.
 */
import { computed, ref, watch } from "vue";
import { CopyToClipboard } from "../../bindings/paradox-modding-tools/services/clipboardservice";
import { GetAttributes, GetItemDetails } from "../../bindings/paradox-modding-tools/services/inventoryservice";
import type { InventoryItemRow, ItemDetails } from "../../bindings/paradox-modding-tools/services/models";
import EditorView from "./EditorView.vue";

const props = defineProps<{
  inventoryId: string | null;
  itemType: string | null;
  itemKey: string | null;
  row: InventoryItemRow | null;
  game: string;
}>();

const details = ref<ItemDetails | null>(null);
const itemAttributes = ref<string[]>([]);
const loading = ref(false);

watch(
  () => [props.inventoryId, props.itemType, props.itemKey, props.game] as const,
  async ([inventoryId, itemType, itemKey, game]) => {
    if (!inventoryId || !itemType || !itemKey) {
      details.value = null;
      itemAttributes.value = [];
      return;
    }
    loading.value = true;
    try {
      const [detailResult, attributesResult] = await Promise.all([
        GetItemDetails(inventoryId, itemType, itemKey),
        GetAttributes(game, itemType),
      ]);
      details.value = detailResult ?? null;
      itemAttributes.value = attributesResult ?? [];
    } finally {
      loading.value = false;
    }
  },
  { immediate: true },
);

const presentSet = computed(() => {
  const attrs = details.value?.attributes;
  if (!attrs || typeof attrs !== "object") return new Set<string>();
  return new Set(Object.keys(attrs).filter((key) => attrs[key]));
});
</script>

<template>
  <div class="space-y-4 p-4">
    <div v-if="!row" class="text-muted">No item selected</div>
    <template v-else>
      <div class="flex flex-wrap items-center justify-between gap-2">
        <UBadge color="primary" class="font-medium">{{ row.type }}</UBadge>
        <div class="text-sm font-medium">{{ row.key }}</div>
        <div class="text-xs text-muted">Lines {{ row.lineStart }} - {{ row.lineEnd }}</div>
      </div>

      <UCard>
        <div class="text-xs text-muted">File path</div>
        <div class="font-mono text-sm">{{ row.filePath }}</div>
      </UCard>

      <div class="grid grid-cols-2 gap-2">
        <UButton color="neutral" variant="outline" label="Copy Key" @click="CopyToClipboard(row.key)" />
        <UButton color="neutral" variant="outline" label="Copy Path" @click="CopyToClipboard(row.filePath)" />
      </div>

      <UCard>
        <div class="text-base font-medium">Attributes</div>
        <div class="mt-2 space-y-1">
          <div
            v-for="attr in itemAttributes"
            :key="attr"
            class="flex items-center justify-between rounded-md border border-default px-2 py-1 text-sm"
          >
            <span>{{ attr }}</span>
            <UIcon
              :name="presentSet.has(attr) ? 'i-lucide-check' : 'i-lucide-minus'"
              :class="presentSet.has(attr) ? 'text-success' : 'text-muted'"
            />
          </div>
        </div>
      </UCard>

      <UCard>
        <div class="text-base font-medium">Raw Text</div>
        <EditorView
          :content="details?.rawText ?? ''"
          :file-name="row.filePath.split(/[/\\]/).pop() ?? ''"
          placeholder="Unavailable"
          :read-only="true"
          class="inventory-raw-scroll mt-2"
        />
      </UCard>
    </template>
  </div>
</template>
