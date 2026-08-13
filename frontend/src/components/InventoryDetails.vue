<script setup lang="ts">
/**
 * Item details panel for the Inventory page.
 */
import { computed, ref, watch } from "vue";
import type { TableColumn } from "@nuxt/ui";
import { CopyToClipboard } from "@services/clipboardservice";
import { GetAttributes, GetItemDetails } from "@services/inventoryservice";
import type { InventoryItemRow, ItemDetails } from "@services/models";
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

type AttributeRow = { name: string; present: boolean };

const attributeRows = computed<AttributeRow[]>(() =>
  itemAttributes.value.map((name) => ({ name, present: presentSet.value.has(name) })),
);

const attributeColumns: TableColumn<AttributeRow>[] = [
  { accessorKey: "name", header: "Attribute" },
  { id: "present", header: "Present" },
];

const rawFileName = computed(() => props.row?.filePath.split(/[/\\]/).pop() ?? "item.txt");
</script>

<template>
  <UEmpty v-if="!row" icon="i-lucide-inbox" title="No item selected" />
  <div v-else class="space-y-4 p-4">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <UBadge color="primary" class="font-medium">{{ row.type }}</UBadge>
      <div class="text-sm font-medium">{{ row.key }}</div>
      <div class="text-xs text-muted">Lines {{ row.lineStart }} - {{ row.lineEnd }}</div>
    </div>

    <UCard>
      <div class="text-xs text-muted">File path</div>
      <div class="font-mono text-sm">{{ row.filePath }}</div>
    </UCard>

    <div class="flex w-full gap-2">
      <UButton
        class="flex-1"
        color="neutral"
        variant="outline"
        label="Copy Key"
        @click="CopyToClipboard(row.key)"
      />
      <UButton
        class="flex-1"
        color="neutral"
        variant="outline"
        label="Copy Path"
        @click="CopyToClipboard(row.filePath)"
      />
    </div>

    <UCard>
      <div class="text-base font-medium">Attributes</div>
      <UTable class="mt-2" :data="attributeRows" :columns="attributeColumns">
        <template #present-cell="{ row: attrRow }">
          <UIcon
            :name="attrRow.original.present ? 'i-lucide-check' : 'i-lucide-minus'"
            :class="attrRow.original.present ? 'text-success' : 'text-muted'"
          />
        </template>
      </UTable>
    </UCard>

    <UCard>
      <div class="text-base font-medium">Raw Text</div>
      <EditorView
        class="inventory-raw-scroll mt-2"
        :label="rawFileName"
        placeholder="Unavailable"
        :items="[
          {
            id: 'file:' + rawFileName,
            type: 'file',
            file: {
              name: rawFileName,
              contents: details?.rawText ?? '',
              lang: 'hcl',
            },
          },
        ]"
      />
    </UCard>
  </div>
</template>
