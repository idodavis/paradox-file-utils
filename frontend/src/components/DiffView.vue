<script setup lang="ts">
/**
 * Monaco-backed diff viewer used for compare and merge workflows.
 */
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { markRaw, shallowRef } from "vue";
import { codeTheme, getMonaco, type MonacoApi } from "../stores/monaco";

const props = withDefaults(
  defineProps<{
    originalContent?: string;
    modifiedContent?: string;
    originalLabel?: string;
    modifiedLabel?: string;
    originalFileName?: string;
    modifiedFileName?: string;
    originalLabelClass?: string;
    modifiedLabelClass?: string;
    origFirstLine?: number;
    modFirstLine?: number;
    renderSideBySide?: boolean;
  }>(),
  {
    originalContent: "",
    modifiedContent: "",
    originalLabelClass: "bg-primary/10 text-primary",
    modifiedLabelClass: "bg-secondary/10 text-secondary",
    origFirstLine: 1,
    modFirstLine: 1,
    renderSideBySide: true,
  },
);

const host = ref<HTMLElement | null>(null);
const monaco = shallowRef<MonacoApi | null>(null);
const diffEditor = shallowRef<ReturnType<MonacoApi["editor"]["createDiffEditor"]> | null>(null);
const originalModel = shallowRef<ReturnType<MonacoApi["editor"]["createModel"]> | null>(null);
const modifiedModel = shallowRef<ReturnType<MonacoApi["editor"]["createModel"]> | null>(null);

function syncModels(): void {
  if (!monaco.value || !diffEditor.value) return;
  const original = props.originalContent ?? "";
  const modified = props.modifiedContent ?? "";

  if (!originalModel.value) {
    originalModel.value = markRaw(monaco.value.editor.createModel(original, "plaintext"));
  } else if (originalModel.value.getValue() !== original) {
    originalModel.value.setValue(original);
  }

  if (!modifiedModel.value) {
    modifiedModel.value = markRaw(monaco.value.editor.createModel(modified, "plaintext"));
  } else if (modifiedModel.value.getValue() !== modified) {
    modifiedModel.value.setValue(modified);
  }

  diffEditor.value.setModel({ original: originalModel.value, modified: modifiedModel.value });
  diffEditor.value.getOriginalEditor()?.updateOptions({
    lineNumbers: props.origFirstLine > 1 ? (line: number) => String(line + props.origFirstLine - 1) : "on",
  });
  diffEditor.value.getModifiedEditor()?.updateOptions({
    lineNumbers: props.modFirstLine > 1 ? (line: number) => String(line + props.modFirstLine - 1) : "on",
  });
}

function syncOptions(): void {
  diffEditor.value?.updateOptions({ renderSideBySide: props.renderSideBySide });
}

onMounted(async () => {
  monaco.value = markRaw(await getMonaco());
  if (!host.value) return;
  diffEditor.value = markRaw(
    monaco.value.editor.createDiffEditor(host.value, {
      originalEditable: false,
      readOnly: true,
      automaticLayout: true,
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      padding: { top: 4, bottom: 4 },
      renderSideBySide: props.renderSideBySide,
    } as Parameters<MonacoApi["editor"]["createDiffEditor"]>[1]),
  );
  monaco.value.editor.setTheme(codeTheme.value);
  syncModels();
  syncOptions();
});

watch(
  [() => props.originalContent, () => props.modifiedContent, () => props.origFirstLine, () => props.modFirstLine],
  syncModels,
  { immediate: true },
);
watch(() => props.renderSideBySide, syncOptions, { immediate: true });
watch(
  codeTheme,
  (theme) => {
    monaco.value?.editor.setTheme(theme);
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  originalModel.value?.dispose();
  modifiedModel.value?.dispose();
  diffEditor.value?.dispose();
});
</script>

<template>
  <div class="flex flex-col h-full overflow-hidden">
    <div v-if="originalLabel || modifiedLabel" class="flex shrink-0 border-b-2 border-default">
      <div class="flex-1 py-2 px-3 text-sm font-semibold" :class="originalLabelClass">
        {{ originalLabel ?? "" }}
        <span v-if="originalFileName" class="ml-1 font-normal text-muted">{{ originalFileName }}</span>
      </div>
      <div class="flex-1 py-2 px-3 text-sm font-semibold" :class="modifiedLabelClass">
        {{ modifiedLabel ?? "" }}
        <span v-if="modifiedFileName" class="ml-1 font-normal text-muted">{{ modifiedFileName }}</span>
      </div>
    </div>
    <div class="min-h-0 flex-1 relative">
      <div
        v-if="!originalContent && !modifiedContent"
        class="pointer-events-none absolute inset-0 flex select-none items-center justify-center text-sm text-muted"
      >
        Select a row to view the diff
      </div>
      <div ref="host" class="absolute inset-0"></div>
    </div>
  </div>
</template>
