<script setup lang="ts">
/**
 * VS Code ViewsService attachPart grid. Always mounted at full size.
 * Tool pages cover it; never hide this host with display/visibility/opacity/inert.
 *
 * The monaco root is a class-free element so Vue re-renders cannot wipe
 * `monaco-workbench` (classList.add from LayoutService). Shell z-index lives
 * on the outer wrapper. Sidebar/panel size is host CSS vars + drag sashes.
 */
import { onMounted, shallowRef, useTemplateRef, watch } from "vue";
import { useIdeShellStore } from "../stores/ideShell";

const props = defineProps<{
  visible: boolean;
  theme?: string;
}>();

const SIDEBAR_KEY = "ide.sidebarWidth";
const PANEL_KEY = "ide.panelHeight";
const SIDEBAR_MIN = 170;
const PANEL_MIN = 120;
const SIDEBAR_DEFAULT = 300;
const PANEL_DEFAULT = 200;

const ideShell = useIdeShellStore();

const root = useTemplateRef<HTMLElement>("root");
const activityBar = useTemplateRef<HTMLElement>("activityBar");
const sidebar = useTemplateRef<HTMLElement>("sidebar");
const editor = useTemplateRef<HTMLElement>("editor");
const panel = useTemplateRef<HTMLElement>("panel");

/** Load a persisted pixel size, or the default. */
function loadPx(key: string, fallback: number): number {
  const n = Number(localStorage.getItem(key));
  return Number.isFinite(n) && n > 0 ? n : fallback;
}

const sidebarWidth = shallowRef(loadPx(SIDEBAR_KEY, SIDEBAR_DEFAULT));
const panelHeight = shallowRef(loadPx(PANEL_KEY, PANEL_DEFAULT));

/** Clamp explorer width to min 170px and 50% of the host. */
function clampSidebar(px: number, host: HTMLElement): number {
  const max = host.clientWidth * 0.5;
  return Math.round(Math.min(max, Math.max(SIDEBAR_MIN, px)));
}

/** Clamp problems height to min 120px and 50% of the host. */
function clampPanel(px: number, host: HTMLElement): number {
  const max = host.clientHeight * 0.5;
  return Math.round(Math.min(max, Math.max(PANEL_MIN, px)));
}

/** Drag the sidebar width or panel height sash; persist on pointer up. */
function startDrag(kind: "sidebar" | "panel", ev: PointerEvent): void {
  const host = root.value;
  if (!host) return;
  ev.preventDefault();
  const startX = ev.clientX;
  const startY = ev.clientY;
  const startW = sidebarWidth.value;
  const startH = panelHeight.value;
  const target = ev.currentTarget as HTMLElement;
  target.setPointerCapture(ev.pointerId);
  const onMove = (e: PointerEvent) => {
    if (kind === "sidebar") {
      sidebarWidth.value = clampSidebar(startW + (e.clientX - startX), host);
      return;
    }
    panelHeight.value = clampPanel(startH - (e.clientY - startY), host);
  };
  const onUp = () => {
    target.releasePointerCapture(ev.pointerId);
    target.removeEventListener("pointermove", onMove);
    target.removeEventListener("pointerup", onUp);
    if (kind === "sidebar") {
      localStorage.setItem(SIDEBAR_KEY, String(sidebarWidth.value));
      return;
    }
    localStorage.setItem(PANEL_KEY, String(panelHeight.value));
  };
  target.addEventListener("pointermove", onMove);
  target.addEventListener("pointerup", onUp);
}

watch(
  () => props.visible,
  (v) => {
    if (!v) return;
    void import("../ide/rootDecorations").then((m) => m.repaintDecoratedRoots());
  },
);

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
    :class="[
      visible ? 'z-10' : 'z-0 pointer-events-none',
      ideShell.mergeReview && 'ide-merge-review',
    ]"
  >
    <slot name="toolbar" />
    <div class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <div
        ref="root"
        :style="{
          '--ide-sidebar-width': `${sidebarWidth}px`,
          '--ide-panel-height': `${panelHeight}px`,
        }"
      >
        <div class="relative flex min-h-0 min-w-0 flex-1">
          <div ref="activityBar" class="ide-activity-bar w-12 shrink-0" />
          <div
            ref="sidebar"
            class="ide-sidebar w-[var(--ide-sidebar-width,300px)] min-w-[170px] max-w-[50%] rounded-xl my-1.5 me-1 overflow-hidden"
          />
          <div
            class="ide-sash ide-sash--v w-1 shrink-0 cursor-col-resize"
            @pointerdown="startDrag('sidebar', $event)"
          />
          <div ref="editor" class="ide-editor min-h-0 min-w-0 flex-1" />
        </div>
        <div
          class="ide-sash ide-sash--h h-1 shrink-0 cursor-row-resize"
          @pointerdown="startDrag('panel', $event)"
        />
        <div
          ref="panel"
          class="ide-panel h-[var(--ide-panel-height,200px)] min-h-[120px] max-h-[50%]"
        />
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

.ide-sidebar.ide-part-hidden + .ide-sash--v,
.ide-sash--h:has(+ .ide-panel.ide-part-hidden) {
  display: none;
}

.ide-merge-review .ide-activity-bar,
.ide-merge-review .ide-sidebar,
.ide-merge-review .ide-panel,
.ide-merge-review .ide-sash {
  display: none !important;
  width: 0 !important;
  min-width: 0 !important;
  height: 0 !important;
  min-height: 0 !important;
}
</style>
