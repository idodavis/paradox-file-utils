<script setup lang="ts">
/**
 * Fullscreen Pierre-based manual merge editor (UnresolvedFile + layout toggle).
 */
import { computed, ref, watch } from "vue";
import { parseDiffFromFile, type CodeViewItem } from "@pierre/diffs";
import EditorView from "./EditorView.vue";
import UnresolvedFileView from "./UnresolvedFileView.vue";
import SplitPane from "./SplitPane.vue";
import { countConflictMarkers } from "../composables/textMerge";

const props = defineProps<{
  fileAPath: string;
  fileBPath: string;
  relPath: string;
  contentA: string;
  contentB: string;
  markedContent: string;
  initialConflictCount: number;
  identical: boolean;
  labelA: string;
  labelB: string;
  fileIndex: number;
  fileTotal: number;
}>();

const emit = defineEmits<{
  save: [payload: { content: string; stats: { changed: number; added: number } }];
  "auto-merge": [];
  skip: [];
  cancel: [];
}>();

type LayoutMode = "middle" | "right" | "bottom";

const layout = ref<LayoutMode>("middle");
const workingContent = ref(props.markedContent);
const editVersion = ref(0);
const unresolvedCount = ref(props.initialConflictCount);

const layoutItems: { label: string; value: LayoutMode }[] = [
  { label: "A|R|B", value: "middle" },
  { label: "Right", value: "right" },
  { label: "Bottom", value: "bottom" },
];

/** Narrow UTabs model updates to LayoutMode. */
function onLayoutChange(value: string | number): void {
  if (value === "middle" || value === "right" || value === "bottom") {
    layout.value = value;
  }
}

watch(
  () => [props.relPath, props.markedContent, props.initialConflictCount] as const,
  () => {
    workingContent.value = props.markedContent;
    unresolvedCount.value = props.initialConflictCount;
    editVersion.value += 1;
  },
);

const fileName = computed(() => props.relPath.split(/[/\\]/).pop() ?? props.relPath);

const markedFile = computed(() => ({
  name: fileName.value,
  contents: props.markedContent,
  lang: "hcl" as const,
}));

const sideAItems = computed<CodeViewItem[]>(() => [
  {
    id: `file-a:${props.relPath}:${editVersion.value}`,
    type: "file",
    file: { name: `${props.labelA}: ${fileName.value}`, contents: props.contentA, lang: "hcl" },
    version: editVersion.value,
  },
]);

const sideBItems = computed<CodeViewItem[]>(() => [
  {
    id: `file-b:${props.relPath}:${editVersion.value}`,
    type: "file",
    file: { name: `${props.labelB}: ${fileName.value}`, contents: props.contentB, lang: "hcl" },
    version: editVersion.value,
  },
]);

const abDiffItems = computed<CodeViewItem[]>(() => [
  {
    id: `diff-ab:${props.relPath}:${editVersion.value}`,
    type: "diff",
    fileDiff: parseDiffFromFile(
      { name: props.labelA, contents: props.contentA, lang: "hcl" },
      { name: props.labelB, contents: props.contentB, lang: "hcl" },
    ),
    version: editVersion.value,
  },
]);

const editableResultItems = computed<CodeViewItem[]>(() => [
  {
    id: `result:${props.relPath}:${editVersion.value}`,
    type: "file",
    edit: true,
    file: {
      name: `Result: ${fileName.value}`,
      contents: workingContent.value,
      lang: "hcl",
      cacheKey: `merge-result:${props.relPath}`,
    },
    version: editVersion.value,
  },
]);

const showUnresolved = computed(() => unresolvedCount.value > 0 && !props.identical);

/** Track Accept resolutions from UnresolvedFile. */
function onResolve(contents: string): void {
  workingContent.value = contents;
  unresolvedCount.value = countConflictMarkers(contents);
  if (unresolvedCount.value === 0) editVersion.value += 1;
}

/** Track freeform edits on the resolved result file. */
function onItemEdit(payload: { id: string; contents: string }): void {
  workingContent.value = payload.contents;
  unresolvedCount.value = countConflictMarkers(payload.contents);
}

/** Persist the working merge result, or auto-merge when files are identical. */
function save(): void {
  if (props.identical) {
    emit("auto-merge");
    return;
  }
  if (unresolvedCount.value > 0) return;
  emit("save", {
    content: workingContent.value,
    stats: {
      changed: props.initialConflictCount,
      added: 0,
    },
  });
}
</script>

<template>
  <UModal
    :open="true"
    fullscreen
    :close="false"
    :dismissible="false"
    :ui="{ content: 'bg-default flex flex-col', body: 'p-0 flex-1 min-h-0' }"
  >
    <template #body>
      <div class="flex h-full min-h-0 flex-col overflow-hidden">
        <div class="flex shrink-0 flex-wrap items-center gap-3 border-b border-default bg-muted/80 px-4 py-2">
          <h2 class="min-w-0 flex-1 truncate text-base font-semibold" :title="relPath">
            Resolving: <span class="text-primary">{{ relPath }}</span>
          </h2>
          <span class="text-sm tabular-nums text-muted">File {{ fileIndex }} of {{ fileTotal }}</span>
          <UBadge :color="unresolvedCount > 0 ? 'warning' : 'success'" variant="subtle">
            {{ identical ? "Identical" : unresolvedCount > 0 ? `${unresolvedCount} unresolved` : "Resolved" }}
          </UBadge>
          <UTabs
            :model-value="layout"
            :items="layoutItems"
            :content="false"
            value-key="value"
            size="xs"
            variant="pill"
            class="w-52"
            @update:model-value="onLayoutChange"
          />
          <UButton size="sm" variant="outline" label="Skip" @click="emit('skip')" />
          <UButton
            size="sm"
            color="primary"
            label="Save & Continue"
            :disabled="!identical && unresolvedCount > 0"
            @click="save"
          />
          <UButton color="error" variant="ghost" size="sm" label="Cancel" @click="emit('cancel')" />
        </div>

        <div class="min-h-0 flex-1 overflow-hidden">
          <UEmpty
            v-if="identical"
            class="h-full"
            icon="i-lucide-check-circle"
            title="Files are identical"
            description="Nothing to merge — continue to write the file as-is."
          />

          <!-- Middle: A | Result | B -->
          <div v-else-if="layout === 'middle'" class="flex h-full min-h-0">
            <div class="min-h-0 min-w-0 flex-1 overflow-hidden border-r border-default">
              <EditorView :items="sideAItems" :label="labelA" label-class="bg-primary/10 text-primary" />
            </div>
            <div class="min-h-0 min-w-0 flex-[1.4] overflow-hidden border-r border-default">
              <UnresolvedFileView
                v-if="showUnresolved"
                :key="`u-${relPath}-${editVersion}`"
                :file="markedFile"
                label="Result (resolve conflicts)"
                @resolve="onResolve"
                @update:contents="onResolve"
              />
              <EditorView
                v-else
                :key="`e-${relPath}-${editVersion}`"
                editable
                :items="editableResultItems"
                label="Result (editable)"
                label-class="bg-accent/10 text-accent"
                @item-edit="onItemEdit"
              />
            </div>
            <div class="min-h-0 min-w-0 flex-1 overflow-hidden">
              <EditorView :items="sideBItems" :label="labelB" label-class="bg-secondary/10 text-secondary" />
            </div>
          </div>

          <!-- Right / Bottom: A↔B + Result -->
          <SplitPane
            v-else
            :orientation="layout === 'right' ? 'horizontal' : 'vertical'"
            :default-second-size="layout === 'right' ? 520 : 320"
            fixed-side="second"
            class="h-full rounded-none border-0"
          >
            <template #first>
              <EditorView :items="abDiffItems" label="A ↔ B" />
            </template>
            <template #second>
              <UnresolvedFileView
                v-if="showUnresolved"
                :key="`u2-${relPath}-${editVersion}`"
                :file="markedFile"
                label="Result (resolve conflicts)"
                @resolve="onResolve"
                @update:contents="onResolve"
              />
              <EditorView
                v-else
                :key="`e2-${relPath}-${editVersion}`"
                editable
                :items="editableResultItems"
                label="Result (editable)"
                label-class="bg-accent/10 text-accent"
                @item-edit="onItemEdit"
              />
            </template>
          </SplitPane>
        </div>
      </div>
    </template>
  </UModal>
</template>
