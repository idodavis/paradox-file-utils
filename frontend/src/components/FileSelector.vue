<script setup lang="ts">
/**
 * File/folder selector used by pages that need Wails dialogs.
 */
import { ref, watch } from "vue";
import { SelectDirectory, SelectSingleFile } from "@services/fileservice";

const props = withDefaults(
  defineProps<{
    label: string;
    modelValue: string;
    dialogTitle: string;
    mode: "file" | "folder";
    placeholder?: string;
    hint?: string;
    fileFilter?: string;
  }>(),
  {
    placeholder: "",
    hint: "",
    fileFilter: "*.txt; *.json",
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
  const path =
    props.mode === "folder"
      ? await SelectDirectory(props.dialogTitle)
      : await SelectSingleFile(props.dialogTitle, props.fileFilter);
  selectedPath.value = path;
  emit("update:modelValue", path);
}
</script>

<template>
  <UFormField :label="label" :help="hint || undefined">
    <div class="flex gap-2">
      <UInput v-model="selectedPath" readonly :placeholder="placeholder" class="flex-1" />
      <UButton :icon="mode === 'folder' ? 'i-lucide-folder-open' : 'i-lucide-file-text'" label="Browse"
        @click="browse" />
    </div>
  </UFormField>
</template>
