<script setup lang="ts">
/**
 * Vue 3 thin host for @pierre/diffs CodeView (+ optional edit mode).
 * Relies on Pierre built-ins for clipboard, find, undo — no custom menus.
 * Editor factory is always mounted so find works when item.edit is true.
 */
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from "vue";
import { CodeView, type CodeViewItem } from "@pierre/diffs";
import { Editor } from "@pierre/diffs/edit";
import { editorTheme, pierreThemeOption } from "../composables/editorTheme";
import { paradoxLanguagesReady } from "../composables/registerSyntax";
import { useSettingsStore } from "../stores/settings";
import { ReadFromClipboard } from "@services/clipboardservice";

const props = withDefaults(
  defineProps<{
    /** Direct list of CodeView items. */
    items?: CodeViewItem[];
    /** Placeholder when no items are available. */
    placeholder?: string;
    /** Optional top header label text. */
    label?: string;
    /** CSS class for top header label. */
    labelClass?: string;
    /** When true, ignore item-edit emissions (caller still controls item.edit). */
    readonly?: boolean;
  }>(),
  {
    placeholder: "Missing Code Content",
    labelClass: "bg-muted text-default",
    readonly: false,
  },
);

const emit = defineEmits<{
  /** Live edited contents for an editable item. */
  "item-edit": [payload: { id: string; contents: string }];
}>();

const settings = useSettingsStore();
const container = useTemplateRef<HTMLElement>("container");
const langsReady = ref(false);
let viewer: CodeView | null = null;

/** Active CodeView items derived from props. */
const activeItems = computed<CodeViewItem[]>(() => props.items ?? []);

/** Host style applies Pierre font size from settings. */
const hostStyle = computed(() => ({
  "--diffs-font-size": `${settings.editorFontSize}px`,
}));

/** Sync current items to the CodeView viewer instance. */
function updateViewerItems(): void {
  viewer?.setItems(activeItems.value);
}

/** Create or recreate the CodeView host with the active theme. */
function mountViewer(): void {
  if (!container.value || !langsReady.value) return;
  viewer?.cleanUp();
  container.value.replaceChildren();
  viewer = new CodeView({
    theme: pierreThemeOption(),
    stickyHeaders: true,
    enableLineSelection: true,
    enableGutterUtility: false,
    layout: { paddingTop: 0, paddingBottom: 0, gap: 12 },
    renderHeaderMetadata(_headerData, context) {
      return context.item.type === "diff" ? context.item.fileDiff.type : "file";
    },
    createEditor: (options) =>
      new Editor({
        ...options,
        persistState: true,
        clipboard: {
          readText: async () => {
            try {
              return await ReadFromClipboard();
            } catch {
              return navigator.clipboard.readText();
            }
          },
        },
      }),
    onItemEditChange(item, file) {
      if (props.readonly) return;
      emit("item-edit", { id: item.id, contents: file.contents });
    },
  });
  viewer.setup(container.value);
  updateViewerItems();
}

onMounted(() => {
  void paradoxLanguagesReady().then(() => {
    langsReady.value = true;
    mountViewer();
  });
});

watch(activeItems, updateViewerItems);
watch(editorTheme, mountViewer);
watch(
  () => props.readonly,
  () => mountViewer(),
);
watch(
  () => settings.editorFontSize,
  () => mountViewer(),
);

onBeforeUnmount(() => {
  viewer?.cleanUp();
  viewer = null;
});
</script>

<template>
  <div class="relative flex h-full min-h-0 flex-1 flex-col" :style="hostStyle">
    <div v-if="label" class="shrink-0 px-3 py-2 text-sm font-semibold" :class="labelClass">
      {{ label }}
    </div>
    <div class="relative min-h-0 flex-1 overflow-hidden">
      <div ref="container" class="absolute inset-0 overflow-auto" />
      <UEmpty
        v-if="activeItems.length === 0"
        class="pointer-events-none absolute inset-0"
        icon="i-lucide-file-code"
        :title="placeholder"
      />
    </div>
  </div>
</template>
