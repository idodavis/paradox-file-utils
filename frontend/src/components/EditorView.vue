<script setup lang="ts">
/**
 * Vue 3 wrapper around @pierre/diffs CodeView.
 * Renders files or diffs with optional edit mode, copy, and context menu.
 */
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from "vue";
import { CodeView, type CodeViewItem } from "@pierre/diffs";
import { Editor } from "@pierre/diffs/edit";
import { editorTheme, pierreThemeOption } from "../composables/editorTheme";
import { paradoxLanguagesReady } from "../composables/registerSyntax";
import { CopyToClipboard } from "@services/clipboardservice";

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
const langsReady = ref(false);
const menuOpen = ref(false);
const menuX = ref(0);
const menuY = ref(0);
let viewer: CodeView | null = null;

/** Active CodeView items derived from props. */
const activeItems = computed<CodeViewItem[]>(() => props.items ?? []);

/** Selected text from the DOM selection (Pierre uses native Selection). */
function selectedText(): string {
  return window.getSelection()?.toString() ?? "";
}

/** Copy current selection or full file contents to the clipboard. */
async function copySelection(fallbackAll = false): Promise<void> {
  let text = selectedText();
  if (!text && fallbackAll) {
    const item = activeItems.value[0];
    if (item?.type === "file") text = item.file.contents;
  }
  if (!text) return;
  try {
    await CopyToClipboard(text);
  } catch {
    try {
      await navigator.clipboard.writeText(text);
    } catch {
      /* ignore */
    }
  }
  menuOpen.value = false;
}

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
    createEditor: props.editable ? (options) => new Editor(options) : undefined,
    onItemEditChange(item, file) {
      emit("item-edit", { id: item.id, contents: file.contents });
    },
  });
  viewer.setup(container.value);
  updateViewerItems();
}

/** Handle Ctrl/Cmd+C when focus is inside the editor host. */
function onKeyDown(ev: KeyboardEvent): void {
  if (!(ev.ctrlKey || ev.metaKey) || ev.key.toLowerCase() !== "c") return;
  if (!container.value?.contains(ev.target as Node)) return;
  const text = selectedText();
  if (!text) return;
  ev.preventDefault();
  void copySelection();
}

/** Open a minimal context menu at the pointer. */
function onContextMenu(ev: MouseEvent): void {
  ev.preventDefault();
  menuX.value = ev.clientX;
  menuY.value = ev.clientY;
  menuOpen.value = true;
}

function closeMenu(): void {
  menuOpen.value = false;
}

onMounted(() => {
  void paradoxLanguagesReady().then(() => {
    langsReady.value = true;
    mountViewer();
  });
  window.addEventListener("keydown", onKeyDown);
  window.addEventListener("click", closeMenu);
});

watch(activeItems, updateViewerItems);
watch(editorTheme, mountViewer);
watch(
  () => props.editable,
  () => mountViewer(),
);

onBeforeUnmount(() => {
  window.removeEventListener("keydown", onKeyDown);
  window.removeEventListener("click", closeMenu);
  viewer?.cleanUp();
  viewer = null;
});
</script>

<template>
  <div class="relative flex h-full min-h-0 flex-1 flex-col">
    <div v-if="label" class="shrink-0 px-3 py-2 text-sm font-semibold" :class="labelClass">
      {{ label }}
    </div>
    <div class="relative min-h-0 flex-1 overflow-hidden" @contextmenu="onContextMenu">
      <div ref="container" class="absolute inset-0 overflow-auto" />
      <UEmpty
        v-if="activeItems.length === 0"
        class="pointer-events-none absolute inset-0"
        icon="i-lucide-file-code"
        :title="placeholder"
      />
    </div>
    <div
      v-if="menuOpen"
      class="fixed z-50 min-w-36 rounded-md border border-default bg-default py-1 shadow-lg"
      :style="{ left: `${menuX}px`, top: `${menuY}px` }"
      @click.stop
    >
      <button
        class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm hover:bg-muted"
        @click="copySelection(true)"
      >
        <UIcon name="i-lucide-copy" class="size-3.5" />
        Copy
      </button>
    </div>
  </div>
</template>
