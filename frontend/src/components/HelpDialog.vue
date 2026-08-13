<script setup lang="ts">
/**
 * Lightweight help dialog for the Vue app.
 */
const open = defineModel<boolean>({ required: true });

defineProps<{
  title: string;
  description: string;
}>();
</script>

<template>
  <UModal
    v-model:open="open"
    :title="title"
    :description="description"
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #footer="{ close }">
      <UButton label="Close" color="neutral" variant="outline" icon="i-lucide-x" @click="close" />
    </template>
  </UModal>
</template>
