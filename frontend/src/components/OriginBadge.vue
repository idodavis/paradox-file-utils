<script lang="ts">
/**
 * Origin identity chip painted with the workspace origin hex.
 * PILL_UI matches this density on view-page UBadge tags.
 */
export const PILL_DENSITY =
  "rounded-sm px-1.5 py-0.5 text-xs leading-none";
export const PILL_UI = { base: `${PILL_DENSITY} font-normal` } as const;
</script>

<script setup lang="ts">
defineProps<{
  label: string;
  hex: string;
}>();
</script>

<template>
  <span
    :class="['inline-flex max-w-full items-center truncate border', PILL_DENSITY]"
    :style="{
      backgroundColor: `${hex}33`,
      borderColor: hex,
    }"
  >
    {{ label }}
  </span>
</template>
