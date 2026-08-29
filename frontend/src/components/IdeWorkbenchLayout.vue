<script setup lang="ts">
/**
 * VS Code ViewsService attachPart grid. Always mounted; visibility via class/inert —
 * attachPart containers must never sit under parent display:none.
 */
import { onMounted, ref, watch } from "vue";

const props = defineProps<{
  visible: boolean;
  theme?: string;
}>();

const root = ref<HTMLElement | null>(null);
const activityBar = ref<HTMLElement | null>(null);
const sidebar = ref<HTMLElement | null>(null);
const editor = ref<HTMLElement | null>(null);
const panel = ref<HTMLElement | null>(null);

let initStarted = false;
let initDone = false;

/** Initialize monaco workbench only when the IDE shell is shown. */
async function tryInitWorkbench(): Promise<void> {
  if (initStarted || !props.visible) return;
  if (!root.value || !activityBar.value || !sidebar.value || !editor.value || !panel.value) {
    return;
  }
  initStarted = true;
  try {
    const { ensureWorkbench } = await import("../ide/workbenchHost");
    await ensureWorkbench(
      {
        root: root.value,
        activityBar: activityBar.value,
        sidebar: sidebar.value,
        editor: editor.value,
        panel: panel.value,
      },
      { theme: props.theme },
    );
    initDone = true;
  } catch (error) {
    initStarted = false;
    console.error("workbench init", error);
  }
}

watch(
  () => props.visible,
  (visible) => {
    if (!visible) return;
    if (initDone) {
      void import("../ide/workbenchHost").then((m) => m.revealWorkbench());
      return;
    }
    void tryInitWorkbench();
  },
  { immediate: true },
);

onMounted(() => void tryInitWorkbench());
</script>

<template>
  <div
    ref="root"
    :class="[
      'absolute inset-0 flex min-h-0 flex-col bg-default',
      visible ? 'z-0 overflow-hidden' : 'pointer-events-none invisible -z-10 overflow-hidden',
    ]"
    :inert="visible ? undefined : true"
  >
    <slot name="toolbar" />
    <div class="ide-layout__body flex min-h-0 min-w-0 flex-1">
      <div ref="activityBar" class="ide-activity-bar shrink-0" />
      <div ref="sidebar" class="ide-sidebar shrink-0" />
      <div ref="editor" class="ide-editor min-h-0 min-w-0 flex-1" />
    </div>
    <div ref="panel" class="ide-panel shrink-0" />
  </div>
</template>

<style scoped>
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
