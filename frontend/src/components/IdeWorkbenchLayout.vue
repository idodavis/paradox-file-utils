<script setup lang="ts">
/**
 * VS Code ViewsService attachPart grid. Always mounted at full size.
 * Tool pages cover it; never hide this host with display/visibility/opacity/inert.
 *
 * The monaco root is a class-free element so Vue re-renders cannot wipe
 * `monaco-workbench` (classList.add from LayoutService). Shell z-index lives
 * on the outer wrapper.
 */
import { onMounted, ref } from "vue";

defineProps<{
  visible: boolean;
  theme?: string;
}>();

const root = ref<HTMLElement | null>(null);
const activityBar = ref<HTMLElement | null>(null);
const sidebar = ref<HTMLElement | null>(null);
const editor = ref<HTMLElement | null>(null);
const panel = ref<HTMLElement | null>(null);

onMounted(async () => {
  if (!root.value || !activityBar.value || !sidebar.value || !editor.value || !panel.value) {
    return;
  }
  root.value.classList.add("ide-monaco-host");
  const { registerIdeParts } = await import("../ide/workbenchHost");
  registerIdeParts({
    root: root.value,
    activityBar: activityBar.value,
    sidebar: sidebar.value,
    editor: editor.value,
    panel: panel.value,
  });
});
</script>

<template>
  <div
    class="absolute inset-0 flex min-h-0 flex-col overflow-hidden bg-default"
    :class="visible ? 'z-10' : 'z-0 pointer-events-none'"
  >
    <slot name="toolbar" />
    <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <div ref="root">
        <div class="ide-layout__body flex min-h-0 min-w-0 flex-1">
          <div ref="activityBar" class="ide-activity-bar shrink-0" />
          <div ref="sidebar" class="ide-sidebar shrink-0" />
          <div ref="editor" class="ide-editor min-h-0 min-w-0 flex-1" />
        </div>
        <div ref="panel" class="ide-panel shrink-0" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.ide-monaco-host {
  display: flex;
  flex: 1 1 0%;
  min-height: 0;
  min-width: 0;
  flex-direction: column;
  overflow: hidden;
}

.ide-layout__body {
  position: relative;
}

.ide-activity-bar {
  width: 48px;
}

.ide-sidebar {
  width: 300px;
  min-width: 170px;
  max-width: 50%;
}

.ide-panel {
  height: 200px;
  min-height: 120px;
  max-height: 50%;
}
</style>
