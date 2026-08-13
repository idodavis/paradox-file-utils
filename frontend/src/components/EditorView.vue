<script setup lang="ts">
/**
 * Vue 3 wrapper around @pierre/diffs CodeView.
 * Renders files or diffs with optional per-item edit mode.
 */
import { computed, onBeforeUnmount, onMounted, useTemplateRef, watch } from "vue";
import { CodeView, type CodeViewItem } from "@pierre/diffs";
import { Editor } from "@pierre/diffs/edit";

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
    /** Enable Pierre edit mode for items with edit: true. */
    editable?: boolean;
  }>(),
  {
    placeholder: "Missing Code Content",
    labelClass: "bg-muted text-default",
    editable: false,
  },
);

const emit = defineEmits<{
  /** Live edited contents for an editable item. */
  "item-edit": [payload: { id: string; contents: string }];
}>();

const container = useTemplateRef<HTMLElement>("container");
let viewer: CodeView | null = null;

/** Active CodeView items derived from props. */
const activeItems = computed<CodeViewItem[]>(() => props.items ?? []);

/** Sync current items to the CodeView viewer instance. */
function updateViewerItems(): void {
  viewer?.setItems(activeItems.value);
}

onMounted(() => {
  if (!container.value) return;
  viewer = new CodeView({
    theme: { dark: "pierre-dark", light: "pierre-light" },
    stickyHeaders: true,
    enableLineSelection: true,
    enableGutterUtility: false,
    layout: { paddingTop: 0, paddingBottom: 0, gap: 12 },
    renderHeaderMetadata(_headerData, context) {
      return context.item.type === "diff" ? context.item.fileDiff.type : "file";
    },
    createEditor: props.editable ? () => new Editor() : undefined,
    onItemEditChange(item, file) {
      emit("item-edit", { id: item.id, contents: file.contents });
    },
  });
  viewer.setup(container.value);
  updateViewerItems();
});

watch(activeItems, updateViewerItems);

onBeforeUnmount(() => {
  viewer?.cleanUp();
  viewer = null;
});
</script>

<template>
  <div class="flex h-full min-h-0 flex-1 flex-col">
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
