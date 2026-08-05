<script setup lang="ts">
/**
 * Vue 3 wrapper around @pierre/diffs CodeView.
 * Supports rendering single files, diffs, or custom CodeViewItem arrays with built-in virtualization.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { CodeView, type CodeViewItem } from "@pierre/diffs";

const props = withDefaults(
  defineProps<{
    /** Direct list of CodeView items. Takes precedence if provided. */
    items?: CodeViewItem[];
    /** Placeholder message when no content or items are available. */
    placeholder?: string;
    /** Optional top header label text. */
    label?: string;
    /** CSS class for top header label. */
    labelClass?: string;
  }>(),
  {
    placeholder: "Missing Code Content",
    labelClass: "bg-muted text-default",
  }
);

const container = ref<HTMLElement | null>(null);
let viewer: CodeView | null = null;

/**
 * Computes the active CodeView items to render based on props.
 */
const activeItems = computed<CodeViewItem[]>(() => {
  if (props.items && props.items.length > 0) {
    return props.items;
  }
  return [];
});

/**
 * Syncs current items to the CodeView viewer instance.
 */
function updateViewerItems() {
  if (!viewer) return;
  viewer.setItems(activeItems.value);
}

onMounted(() => {
  if (!container.value) return;

  // Initialize CodeView instance
  viewer = new CodeView({
    theme: { dark: "pierre-dark", light: "pierre-light" },
    stickyHeaders: true,
    enableLineSelection: true,
    enableGutterUtility: false,
    layout: { paddingTop: 0, paddingBottom: 0, gap: 12 },
    renderHeaderMetadata(_headerData, context) {
      return context.item.type === 'diff' ? context.item.fileDiff.type : 'file';
    },
    // renderGutterUtility(getHoveredLine, context) {
    //   const hoveredLine = getHoveredLine();
    //   if (hoveredLine == null || context.item.type !== 'diff') {
    //     return undefined;
    //   }

    //   // const button = document.createElement('button');
    //   // button.type = 'button';
    //   // button.textContent = 'Comment on line ' + hoveredLine.lineNumber;
    //   // return button;
    // }
  });

  // Attach virtualized viewer to host element
  viewer.setup(container.value);

  // Set initial items
  updateViewerItems();
});

// Reactively update items whenever props change
watch(activeItems, () => {
  updateViewerItems();
});

onBeforeUnmount(() => {
  viewer?.cleanUp();
  viewer = null;
});
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col h-full">
    <!-- Viewer Host Container -->
    <div class="relative min-h-0 flex-1 overflow-hidden">
      <!-- CodeView mounts inside this container and manages scrolling -->
      <div ref="container" class="absolute inset-0 overflow-auto"></div>

      <!-- Placeholder overlay when empty -->
      <div v-if="activeItems.length === 0"
        class="pointer-events-none absolute inset-0 flex select-none items-center justify-center text-sm text-muted">
        {{ placeholder }}
      </div>
    </div>
  </div>
</template>