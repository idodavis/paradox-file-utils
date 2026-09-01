<script lang="ts">
/**
 * Native Wails v3 file/folder dialogs used by the wizard and this selector.
 */
import { Dialogs } from "@wailsio/runtime";

/** Open a folder picker. Cancel yields an empty path. */
export async function pickDirectory(title: string): Promise<string> {
  const path = await Dialogs.OpenFile({
    Title: title,
    CanChooseDirectories: true,
    CanChooseFiles: false,
  });
  return typeof path === "string" ? path : "";
}

/** Open a file picker. Cancel yields an empty path. */
export async function pickFile(title: string, filter: string): Promise<string> {
  const path = await Dialogs.OpenFile({
    Title: title,
    CanChooseDirectories: false,
    CanChooseFiles: true,
    Filters: filter ? [{ DisplayName: filter, Pattern: filter }] : undefined,
  });
  return typeof path === "string" ? path : "";
}
</script>

<script setup lang="ts">
/**
 * File/folder selector used by pages that need native Wails dialogs.
 */
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
      ? await pickDirectory(props.dialogTitle)
      : await pickFile(props.dialogTitle, props.fileFilter);
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
