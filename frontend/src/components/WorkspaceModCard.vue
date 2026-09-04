<script lang="ts">
/**
 * One workspace mod row: load-order header, expandable fields, nested ignore.
 */
import type { WorkspaceMod } from "@services/models";

/** Field help used by this card and the settings filter. */
export const MOD_HELP = {
  thumb: "Explorer folder icon and origin filters. Not always used on Origin/Mod Badges.",
  descMd: "Markdown listing description. Empty uses mod-description.md in the mod folder. The file must already exist.",
  descBb: "BBCode sent to Steam. Empty uses mod-description.bbcode. The file must already exist.",
  workshopIgnore:
    "Gitignore syntax, stored in this workspace. Steam never uploads: dotfiles (except .metadata/), .gitignore, and patterns in a .gitignore in the mod folder. Export writes .workshop-ignore; Import reads it. Editing that file in the IDE does nothing until Import.",
} as const;

/** Fields the parent persists via UpdateWorkspaceMod. */
export type ModPatch = Partial<
  Pick<WorkspaceMod, "name" | "color" | "thumbnail" | "descMdRel" | "descBbRel" | "workshopIgnore">
>;
</script>

<script setup lang="ts">
import { computed, shallowRef } from "vue";
import FileSelector from "./FileSelector.vue";
import FieldHelpTip from "./FieldHelpTip.vue";
import OriginColorField from "./OriginColorField.vue";
import { originHex } from "../ide/rootDecorations";

const FILE_FIELDS = [
  {
    key: "thumbnail",
    label: "Thumbnail",
    help: MOD_HELP.thumb,
    dialog: "Select thumbnail",
    filter: "*.png; *.jpg; *.jpeg; *.svg",
  },
  {
    key: "descMdRel",
    label: "Markdown file",
    help: MOD_HELP.descMd,
    dialog: "Select description Markdown",
    filter: "*.md; *.markdown; *.txt",
  },
  {
    key: "descBbRel",
    label: "BBCode file",
    help: MOD_HELP.descBb,
    dialog: "Select description BBCode",
    filter: "*.bbcode; *.txt",
  },
] as const;

const props = defineProps<{
  mod: WorkspaceMod;
  index: number;
  expanded: boolean;
  thumbUrl?: string;
}>();

const emit = defineEmits<{
  "update:expanded": [open: boolean];
  patch: [patch: ModPatch];
  importIgnore: [];
  exportIgnore: [];
  remove: [];
}>();

const ignoreOpen = shallowRef(false);

/** Palette or stored hex for the collapsed header swatch. */
const swatch = computed(() =>
  originHex({
    kind: "mod",
    path: props.mod.path,
    color: props.mod.color,
    wrapIndex: props.index,
  }).toLowerCase(),
);
</script>

<template>
  <div class="flex overflow-hidden rounded border border-default">
    <div
      class="flex shrink-0 flex-col items-center gap-1 self-stretch border-e border-default bg-elevated/50 px-1 py-1.5"
    >
      <UBadge
        :label="String((mod.sortOrder ?? index) + 1)"
        color="neutral"
        variant="subtle"
        size="xs"
        class="w-6 justify-center tabular-nums"
      />
      <button
        type="button"
        class="mod-handle inline-flex size-8 cursor-grab items-center justify-center text-muted hover:text-default active:cursor-grabbing"
        aria-label="Drag to reorder"
        title="Drag to reorder"
      >
        <UIcon name="i-lucide-grip-vertical" class="size-4" />
      </button>
    </div>

    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2 px-2 py-1.5">
        <span
          class="size-5 shrink-0 rounded border border-default"
          :style="{ backgroundColor: swatch }"
          :title="swatch"
        />
        <div class="min-w-0 flex-1">
          <UInput
            :model-value="mod.name"
            size="sm"
            class="w-full"
            @update:model-value="emit('patch', { name: String($event ?? '') })"
          />
          <p class="truncate text-xs text-muted">{{ mod.path }}</p>
        </div>
        <UBadge v-if="mod.isBroken" color="error" variant="subtle" size="xs"> Missing </UBadge>
        <UTooltip text="Remove mod">
          <UButton icon="i-lucide-trash-2" color="error" variant="ghost" size="xs" square @click="emit('remove')" />
        </UTooltip>
        <UTooltip :text="expanded ? 'Collapse settings' : 'Expand settings'">
          <UButton
            :icon="expanded ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
            color="neutral"
            variant="ghost"
            size="xs"
            square
            @click="emit('update:expanded', !expanded)"
          />
        </UTooltip>
      </div>

      <div v-show="expanded" class="space-y-3 border-t border-default px-2 py-2">
        <div class="grid grid-cols-1 gap-x-6 gap-y-3 md:grid-cols-2">
          <OriginColorField
            :model-value="mod.color ?? ''"
            kind="mod"
            compact
            :wrap-index="index"
            @update:model-value="emit('patch', { color: $event })"
          />
          <UFormField v-for="f in FILE_FIELDS" :key="f.key" size="sm">
            <template #label>
              <FieldHelpTip :label="f.label" :text="f.help" />
            </template>
            <div class="flex items-center gap-2">
              <img
                v-if="f.key === 'thumbnail' && mod.thumbnail && thumbUrl"
                :src="thumbUrl"
                alt=""
                class="size-6 rounded-sm object-cover"
              />
              <FileSelector
                :model-value="mod[f.key] ?? ''"
                mode="file"
                :dialog-title="f.dialog"
                :file-filter="f.filter"
                class="min-w-0 flex-1"
                @update:model-value="emit('patch', { [f.key]: $event })"
              />
              <UButton
                v-if="mod[f.key]"
                label="Clear"
                size="xs"
                variant="ghost"
                @click="emit('patch', { [f.key]: '' })"
              />
            </div>
          </UFormField>
        </div>

        <div class="space-y-2">
          <div class="flex flex-wrap items-center gap-1">
            <UButton
              label="Workshop ignore"
              color="neutral"
              variant="ghost"
              size="xs"
              :trailing-icon="ignoreOpen ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
              @click="ignoreOpen = !ignoreOpen"
            />
            <FieldHelpTip :text="MOD_HELP.workshopIgnore" />
            <div class="ms-auto flex flex-wrap gap-2">
              <UButton
                label="Import .workshop-ignore"
                icon="i-lucide-download"
                size="xs"
                variant="outline"
                @click="emit('importIgnore')"
              />
              <UButton
                label="Export .workshop-ignore"
                icon="i-lucide-upload"
                size="xs"
                variant="outline"
                @click="emit('exportIgnore')"
              />
            </div>
          </div>
          <UTextarea
            v-show="ignoreOpen"
            :model-value="mod.workshopIgnore ?? ''"
            :rows="3"
            class="w-full font-mono text-sm"
            placeholder="# e.g. docs/&#10;*.psd"
            @update:model-value="emit('patch', { workshopIgnore: String($event ?? '') })"
          />
        </div>
      </div>
    </div>
  </div>
</template>
