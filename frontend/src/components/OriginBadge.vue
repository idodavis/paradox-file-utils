<script setup lang="ts">
/**
 * Origin identity chip painted with the workspace origin hex.
 */
defineProps<{
  label: string;
  hex: string;
}>();
</script>

<template>
  <span
    class="inline-flex max-w-full items-center truncate rounded-sm border px-1.5 py-0.5 text-xs leading-none"
    :style="{
      backgroundColor: `${hex}33`,
      borderColor: hex,
    }"
  >
    {{ label }}
  </span>
</template>
