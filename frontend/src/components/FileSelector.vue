<script setup lang="ts">
/**
 * File/folder selector used by pages that need Wails dialogs.
 */
import { SelectDirectory, SelectSingleFile } from "@services/fileservice";

const selectedPath = defineModel<string>({ required: true });

const props = withDefaults(
  defineProps<{
    label: string;
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

/** Open a native file or folder dialog and write the result into the model. */
async function browse(): Promise<void> {
  const path =
    props.mode === "folder"
      ? await SelectDirectory(props.dialogTitle)
      : await SelectSingleFile(props.dialogTitle, props.fileFilter);
  selectedPath.value = path;
}
</script>

<template>
  <UFormField :label="label" :help="hint || undefined">
    <UFieldGroup class="w-full">
      <UInput v-model="selectedPath" readonly :placeholder="placeholder" class="w-full" />
      <UButton
        :icon="mode === 'folder' ? 'i-lucide-folder-open' : 'i-lucide-file-text'"
        label="Browse"
        @click="browse"
      />
    </UFieldGroup>
  </UFormField>
</template>
