<script setup lang="ts">
/**
 * Header Display popover: UI scale, editor font, workbench UI font, tools.
 */
import { ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useSettingsStore } from "../stores/settings";
import { WORKSPACE_TOOLS, type WorkspaceToolName } from "../workspaceTools";
import { applyEditorFontSize } from "../ide/themeBridge";

const settings = useSettingsStore();
const {
  fontScale, editorFontSize, ideUiFontSize, visibleTools, saving,
} = storeToRefs(settings);

const scale = ref(fontScale.value);
const font = ref(editorFontSize.value);
const uiFont = ref(ideUiFontSize.value);
const tools = ref<WorkspaceToolName[]>([...visibleTools.value]);

watch(fontScale, (v) => { scale.value = v; });
watch(editorFontSize, (v) => { font.value = v; });
watch(ideUiFontSize, (v) => { uiFont.value = v; });
watch(visibleTools, (v) => { tools.value = [...v]; });

/** Apply UI scale live while dragging; persistence happens on commit. */
function onScaleInput(value: number | number[] | undefined): void {
  const n = Array.isArray(value) ? value[0] : value;
  if (n === undefined) return;
  scale.value = n;
  void settings.set("ui.fontScale", String(n), false);
}

/** Apply editor font live while dragging; persistence happens on commit. */
function onFontInput(value: number | number[] | undefined): void {
  const n = Array.isArray(value) ? value[0] : value;
  if (n === undefined) return;
  font.value = n;
  void settings.set("ui.editorFontSize", String(n), false);
  applyEditorFontSize(n);
}

/** Apply workbench UI font live while dragging. */
function onUiFontInput(value: number | number[] | undefined): void {
  const n = Array.isArray(value) ? value[0] : value;
  if (n === undefined) return;
  uiFont.value = n;
  void settings.set("ui.ideUiFontSize", String(n), false);
}

/** Toggle one tool in the visible-tools list. */
function setToolVisible(name: WorkspaceToolName, on: boolean): void {
  const next = tools.value.filter((n) => n !== name);
  if (on) next.push(name);
  tools.value = next;
  void settings.setVisibleTools(next);
}

/** Checkbox change handler (ignores indeterminate). */
function onToolCheck(name: WorkspaceToolName, v: boolean | "indeterminate"): void {
  setToolVisible(name, v === true);
}

/** Reload the webview so chrome and the workbench remount. */
function reloadUi(): void {
  window.location.reload();
}
</script>

<template>
  <UPopover :content="{ align: 'end', side: 'bottom' }">
    <UTooltip text="Display">
      <UButton
        icon="i-lucide-monitor"
        color="neutral"
        variant="ghost"
      />
    </UTooltip>
    <template #content>
      <div class="w-72 space-y-4 p-3">
        <UFormField
          label="UI scale"
          :description="`App chrome at ${scale}%`"
        >
          <div class="flex items-center gap-3">
            <USlider
              :model-value="scale"
              :min="settings.FONT_SCALE_MIN"
              :max="settings.FONT_SCALE_MAX"
              :step="5"
              class="flex-1"
              @update:model-value="onScaleInput"
              @change="settings.save()"
            />
            <span class="w-12 text-center text-sm tabular-nums">{{ scale }}%</span>
          </div>
        </UFormField>
        <UFormField
          label="Editor font"
          :description="`${font}px in the workbench`"
        >
          <div class="flex items-center gap-3">
            <USlider
              :model-value="font"
              :min="settings.EDITOR_FONT_MIN"
              :max="settings.EDITOR_FONT_MAX"
              :step="1"
              class="flex-1"
              @update:model-value="onFontInput"
              @change="settings.save()"
            />
            <span class="w-12 text-center text-sm tabular-nums">{{ font }}px</span>
          </div>
        </UFormField>
        <UFormField
          label="Workbench UI"
          description="Explorer, Problems, panels, menus. Root folders are 2px larger."
        >
          <div class="flex items-center gap-3">
            <USlider
              :model-value="uiFont"
              :min="settings.IDE_UI_FONT_MIN"
              :max="settings.IDE_UI_FONT_MAX"
              :step="1"
              class="flex-1"
              @update:model-value="onUiFontInput"
              @change="settings.save()"
            />
            <span class="w-12 text-center text-sm tabular-nums">{{ uiFont }}px</span>
          </div>
        </UFormField>
        <UFormField label="Visible tools">
          <div class="space-y-1">
            <UCheckbox
              v-for="tool in WORKSPACE_TOOLS"
              :key="tool.name"
              :label="tool.label"
              :model-value="tools.includes(tool.name)"
              :disabled="saving"
              @update:model-value="onToolCheck(tool.name, $event)"
            />
          </div>
        </UFormField>
        <UButton
          label="Reload UI"
          icon="i-lucide-refresh-cw"
          color="neutral"
          variant="outline"
          size="sm"
          block
          @click="reloadUi"
        />
      </div>
    </template>
  </UPopover>
</template>
