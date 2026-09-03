<script lang="ts">
export { pickDirectory, pickFile, pickSave } from "../lib/nativeDialog";
</script>

<script setup lang="ts">
/**
 * Read-only path input + Browse button. Wrap in UFormField for label/description.
 */
import { pickDirectory, pickFile } from "../lib/nativeDialog";

const selectedPath = defineModel<string>({ required: true });

const props = withDefaults(
  defineProps<{
    dialogTitle: string;
    mode: "file" | "folder";
    placeholder?: string;
    fileFilter?: string;
  }>(),
  {
    placeholder: "",
    fileFilter: "*.txt; *.json",
  },
);

/** Open a native file or folder dialog and write the result into the model. */
async function browse(): Promise<void> {
  const path =
    props.mode === "folder"
      ? await pickDirectory(props.dialogTitle)
      : await pickFile(props.dialogTitle, props.fileFilter);
  if (path) selectedPath.value = path;
}
</script>

<template>
  <UFieldGroup class="w-full">
    <UInput v-model="selectedPath" readonly :placeholder="placeholder" class="w-full" />
    <UButton
      :icon="mode === 'folder' ? 'i-lucide-folder-open' : 'i-lucide-file-text'"
      label="Browse"
      @click="browse"
    />
  </UFieldGroup>
</template>
