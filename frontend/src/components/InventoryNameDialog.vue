<script setup lang="ts">
/**
 * Save/rename dialog for inventory names.
 */
import { computed, ref, watch } from "vue";

const open = defineModel<boolean>({ required: true });

const props = defineProps<{
  mode: "save" | "rename";
  initialName: string;
}>();

const emit = defineEmits<{
  save: [name: string];
}>();

const name = ref(props.initialName);

watch(
  () => props.initialName,
  (value) => {
    name.value = value;
  },
  { immediate: true },
);

const title = computed(() => (props.mode === "save" ? "Save inventory" : "Rename inventory"));

/** Persist the current name and close the dialog. */
function submit(): void {
  emit("save", name.value);
  open.value = false;
}
</script>

<template>
  <UModal v-model:open="open" :title="title" description="Inventory" :ui="{ content: 'sm:max-w-lg' }">
    <template #body>
      <UInput v-model="name" autofocus placeholder="Inventory name" />
    </template>

    <template #footer="{ close }">
      <UButton label="Cancel" color="neutral" variant="outline" @click="close" />
      <UButton label="Save" @click="submit" />
    </template>
  </UModal>
</template>
