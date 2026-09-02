<script setup lang="ts">
/**
 * Multi-select origin menu for loc/conflicts: workspace mods only, color dots.
 */
export type OriginMenuItem = {
  id: string;
  label: string;
  color: string;
  thumbnail?: string;
};

defineProps<{
  items: OriginMenuItem[];
}>();

const selected = defineModel<string[]>({ default: () => [] });

const MENU_UI = { content: "min-w-96 w-[420px]", viewport: "max-h-96" };
</script>

<template>
  <USelectMenu
    v-model="selected"
    :items="items"
    value-key="id"
    label-key="label"
    multiple
    placeholder="Origins"
    size="md"
    class="w-72"
    :ui="MENU_UI"
  >
    <template #item-leading="{ item }">
      <img
        v-if="item.thumbnail"
        :src="item.thumbnail"
        alt=""
        class="size-4 rounded-sm object-cover"
      />
      <span
        v-else
        class="size-2 shrink-0 rounded-full"
        :style="{ backgroundColor: item.color }"
      />
    </template>
  </USelectMenu>
</template>
