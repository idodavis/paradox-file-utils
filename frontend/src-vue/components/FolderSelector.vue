<script setup lang="ts">
/**
 * Folder selector that mirrors the legacy PMT browse-field behavior.
 */
import { ref, watch } from "vue";
import { SelectDirectory } from "../../bindings/paradox-modding-tools/services/fileservice";

const props = withDefaults(
  defineProps<{
    label: string;
    modelValue: string;
    dialogTitle: string;
    hint?: string;
    placeholder?: string;
  }>(),
  {
    hint: "",
    placeholder: "",
  },
);

const emit = defineEmits<{ (event: "update:modelValue", value: string): void }>();
const selectedPath = ref(props.modelValue);

watch(
  () => props.modelValue,
  (value) => {
    selectedPath.value = value;
  },
);

async function browse(): Promise<void> {
  const path = await SelectDirectory(props.dialogTitle);
  selectedPath.value = path;
  emit("update:modelValue", path);
}
</script>

<template>
  <UFormField :label="label" :help="hint || undefined">
    <div class="flex gap-2">
      <UInput v-model="selectedPath" readonly :placeholder="placeholder" class="flex-1" />
      <UButton icon="i-lucide-folder-open" label="Browse" @click="browse" />
    </div>
  </UFormField>
</template>
