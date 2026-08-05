<script setup lang="ts">
/**
 * Monaco-backed code editor view used throughout the Vue frontend.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { markRaw, shallowRef } from "vue";
import { codeLanguage, codeTheme, getMonaco, type CodeLanguage, type MonacoApi } from "../stores/monaco";

const props = withDefaults(
  defineProps<{
    content: string;
    firstLineNumber?: number;
    placeholder?: string;
    label?: string;
    labelClass?: string;
    fileName?: string;
    readOnly?: boolean;
    lang?: CodeLanguage;
  }>(),
  {
    firstLineNumber: 1,
    placeholder: "Missing Code Content",
    labelClass: "bg-muted text-default",
    readOnly: true,
  },
);

const emit = defineEmits<{
  (event: "content-change", value: string): void;
}>();

const host = ref<HTMLElement | null>(null);
const monaco = shallowRef<MonacoApi | null>(null);
const editor = shallowRef<ReturnType<MonacoApi["editor"]["create"]> | null>(null);
const model = shallowRef<ReturnType<MonacoApi["editor"]["createModel"]> | null>(null);
let contentSubscription: { dispose: () => void } | null = null;
let programmatic = false;

const effectiveLang = computed(() => props.lang ?? codeLanguage.value);

function syncModel(): void {
  if (!monaco.value || !editor.value) return;
  const value = props.content ?? "";
  const lang = effectiveLang.value;

  if (!model.value || model.value.getLanguageId() !== lang) {
    contentSubscription?.dispose();
    model.value?.dispose();
    model.value = markRaw(monaco.value.editor.createModel(value, lang));
    editor.value.setModel(model.value);
    contentSubscription = model.value.onDidChangeContent(() => {
      if (!programmatic && !props.readOnly) {
        emit("content-change", model.value?.getValue() ?? "");
      }
    });
    return;
  }

  if (model.value.getValue() !== value) {
    programmatic = true;
    model.value.setValue(value);
    programmatic = false;
  }
}

function syncOptions(): void {
  if (!editor.value) return;
  editor.value.updateOptions({
    readOnly: props.readOnly,
    lineNumbers: props.firstLineNumber > 1 ? (line: number) => String(line + props.firstLineNumber - 1) : "on",
  });
}

onMounted(async () => {
  monaco.value = markRaw(await getMonaco());
  if (!host.value) return;
  editor.value = markRaw(
    monaco.value.editor.create(host.value, {
      automaticLayout: true,
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      wordWrap: "off",
      glyphMargin: false,
      folding: true,
      lineNumbersMinChars: 3,
      lineDecorationsWidth: 8,
      padding: { top: 8, bottom: 8 },
      readOnly: props.readOnly,
    } as Parameters<MonacoApi["editor"]["create"]>[1]),
  );
  monaco.value.editor.setTheme(codeTheme.value);
  syncModel();
  syncOptions();
});

watch([() => props.content, effectiveLang], syncModel, { immediate: true });
watch([() => props.readOnly, () => props.firstLineNumber], syncOptions, { immediate: true });
watch(
  codeTheme,
  (theme) => {
    monaco.value?.editor.setTheme(theme);
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  contentSubscription?.dispose();
  model.value?.dispose();
  editor.value?.dispose();
});
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col h-full">
    <div v-if="label" class="shrink-0 border-b-2 border-default px-3 py-2 text-sm font-semibold" :class="labelClass">
      {{ label }}
      <span v-if="fileName" class="ml-1 font-normal text-muted">{{ fileName }}</span>
    </div>
    <div class="relative min-h-0 flex-1 overflow-hidden">
      <div ref="host" class="absolute inset-0"></div>
      <div
        v-if="!content"
        class="pointer-events-none absolute inset-0 flex select-none items-center justify-center text-sm text-muted"
      >
        {{ placeholder }}
      </div>
    </div>
  </div>
</template>
