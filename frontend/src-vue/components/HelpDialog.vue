<script setup lang="ts">
/**
 * Lightweight help dialog for the Vue app.
 */
import { computed } from "vue";

const props = defineProps<{
  modelValue: boolean;
  title: string;
  description: string;
}>();

const emit = defineEmits<{ (event: "update:modelValue", value: boolean): void }>();

const open = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});
</script>

<template>
  <UModal v-model:open="open" :title="title" :description="description" :ui="{ content: 'sm:max-w-2xl' }">
    <template #body>
      <div class="space-y-3 text-sm text-muted">
        <p>{{ description }}</p>
      </div>
    </template>

    <template #footer="{ close }">
      <UButton label="Close" color="neutral" variant="outline" icon="i-lucide-x" @click="close" />
    </template>
  </UModal>
</template>
