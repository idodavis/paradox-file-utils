<script setup lang="ts">
/**
 * Fullscreen merge editor with resizable result pane.
 */
import { computed, ref, watch } from "vue";
import DiffView from "./DiffView.vue";
import EditorView from "./EditorView.vue";
import LangThemeSelect from "./LangThemeSelect.vue";
import SplitPane from "./SplitPane.vue";
import type { MergeConflictChunk } from "../../bindings/paradox-modding-tools/services/models";

const props = defineProps<{
  fileAPath: string;
  fileBPath: string;
  relPath: string;
  chunks: MergeConflictChunk[];
  labelA: string;
  labelB: string;
  fileIndex: number;
  fileTotal: number;
  allowAdditions: boolean;
}>();

const emit = defineEmits<{
  (event: "save", payload: { content: string; stats: { changed: number; added: number } }): void;
  (event: "auto-merge"): void;
  (event: "skip"): void;
  (event: "cancel"): void;
}>();

const resultValues = ref<Record<number, string>>({});
const resolvedState = ref<Record<number, "A" | "B" | "Custom" | undefined>>({});
const includedAdditions = ref<Record<number, boolean>>({});
const editorTab = ref<"conflicts" | "additions">("conflicts");
const currentConflictNum = ref(1);
const currentAdditionNum = ref(1);
const mergeResultLayout = ref<"right" | "bottom">("right");

function getConflictIndices(chunks: MergeConflictChunk[]): number[] {
  return chunks.flatMap((chunk, index) => (chunk.type === "conflict" ? [index] : []));
}

function getAddedIndices(chunks: MergeConflictChunk[]): number[] {
  return chunks.flatMap((chunk, index) => (chunk.type === "added" ? [index] : []));
}

function stripNL(text: string) {
  const prefix = text.match(/^(\r?\n)+/)?.[0] ?? "";
  return { body: text.slice(prefix.length), offset: prefix.split("\n").length - 1, prefix };
}

function buildMergedContent(
  chunks: MergeConflictChunk[],
  values: Record<number, string>,
  additions: Record<number, boolean>,
): string {
  let output = "";
  let addHeader = false;
  for (const [index, chunk] of chunks.entries()) {
    if (chunk.type === "unchanged") {
      output += chunk.textA;
      continue;
    }
    if (chunk.type === "added") {
      if (additions[index] === false) continue;
      if (!addHeader) {
        if (output.length > 0 && !/\n$/.test(output)) output += "\n";
        output += "\n############# Additional Entries From B (PDX-Merge-Tools) #############\n";
        addHeader = true;
      }
      output += chunk.textB;
      continue;
    }
    output += values[index] ?? "";
  }
  return output;
}

function computeMergeStats(
  chunks: MergeConflictChunk[],
  resolved: Record<number, "A" | "B" | "Custom" | undefined>,
  additions: Record<number, boolean>,
) {
  return {
    changed: chunks.filter((chunk, index) => chunk.type === "conflict" && resolved[index] !== "A").length,
    added: chunks.filter((chunk, index) => chunk.type === "added" && additions[index] !== false).length,
  };
}

const conflictIndices = computed(() => getConflictIndices(props.chunks));
const addedIndices = computed(() => getAddedIndices(props.chunks));
const conflictCount = computed(() => conflictIndices.value.length);
const addedCount = computed(() => addedIndices.value.length);
const resolvedCount = computed(() => Object.values(resolvedState.value).filter(Boolean).length);
const unresolvedCount = computed(() => conflictCount.value - resolvedCount.value);
const includedCount = computed(() => addedIndices.value.filter((index) => includedAdditions.value[index]).length);
const currentChunkIndex = computed(() =>
  currentConflictNum.value >= 1 && currentConflictNum.value <= conflictCount.value
    ? (conflictIndices.value[currentConflictNum.value - 1] ?? -1)
    : -1,
);
const currentAdditionIndex = computed(() =>
  currentAdditionNum.value >= 1 && currentAdditionNum.value <= addedCount.value
    ? (addedIndices.value[currentAdditionNum.value - 1] ?? -1)
    : -1,
);
const currentChunk = computed(() =>
  currentChunkIndex.value >= 0 ? (props.chunks[currentChunkIndex.value] ?? null) : null,
);
const currentAdditionChunk = computed(() =>
  currentAdditionIndex.value >= 0 ? (props.chunks[currentAdditionIndex.value] ?? null) : null,
);
const currentChoice = computed(() =>
  currentChunkIndex.value >= 0 ? resolvedState.value[currentChunkIndex.value] : undefined,
);
const choiceBadge = computed(() =>
  currentChoice.value === "A"
    ? { color: "primary", text: `Chose ${props.labelA}` }
    : currentChoice.value === "B"
      ? { color: "secondary", text: `Chose ${props.labelB}` }
      : currentChoice.value === "Custom"
        ? { color: "warning", text: "Custom" }
        : { color: "warning", text: "Unresolved" },
);
const diffDisplay = computed(() => {
  if (!currentChunk.value) return null;
  const a = stripNL(currentChunk.value.textA);
  const b = stripNL(currentChunk.value.textB);
  return {
    textA: a.body,
    textB: b.body,
    startLineA: currentChunk.value.startLineA + a.offset,
    startLineB: currentChunk.value.startLineB + b.offset,
  };
});
const additionsDiffDisplay = computed(() => {
  if (!currentAdditionChunk.value) return null;
  const b = stripNL(currentAdditionChunk.value.textB);
  return {
    textA: "(not in A)",
    textB: b.body,
    startLineA: 1,
    startLineB: currentAdditionChunk.value.startLineB + b.offset,
  };
});
const resultDisplay = computed(() => {
  if (editorTab.value === "additions") {
    if (!currentAdditionChunk.value) return "";
    return includedAdditions.value[currentAdditionIndex.value]
      ? stripNL(currentAdditionChunk.value.textB).body
      : "(excluded)";
  }
  if (currentChunkIndex.value < 0 || !resolvedState.value[currentChunkIndex.value]) return "";
  return stripNL(resultValues.value[currentChunkIndex.value] ?? "").body;
});

watch(
  () => [props.relPath, props.chunks, props.allowAdditions] as const,
  () => {
    resultValues.value = {};
    resolvedState.value = {};
    currentConflictNum.value = 1;
    currentAdditionNum.value = 1;
    editorTab.value = conflictCount.value > 0 ? "conflicts" : "additions";
    includedAdditions.value = props.allowAdditions
      ? Object.fromEntries(addedIndices.value.map((index) => [index, true]))
      : {};
  },
  { immediate: true },
);

function choose(index: number, side: "A" | "B"): void {
  const chunk = props.chunks[index];
  if (!chunk || chunk.type !== "conflict") return;
  resultValues.value[index] = side === "A" ? chunk.textA : chunk.textB;
  resolvedState.value[index] = side;
}

function chooseRest(side: "A" | "B"): void {
  for (const index of conflictIndices.value) {
    if (!resolvedState.value[index]) choose(index, side);
  }
}

function setAdditionIncluded(index: number, value: boolean): void {
  includedAdditions.value = { ...includedAdditions.value, [index]: value };
}

function setAllAdditions(value: boolean): void {
  includedAdditions.value = Object.fromEntries(addedIndices.value.map((index) => [index, value]));
}

function onResultChange(value: string): void {
  if (currentChunkIndex.value < 0 || !currentChunk.value) return;
  resultValues.value[currentChunkIndex.value] = stripNL(currentChunk.value.textA).prefix + value;
  resolvedState.value[currentChunkIndex.value] = "Custom";
}

function save(): void {
  if (conflictCount.value > 0 && unresolvedCount.value > 0) return;
  if (conflictCount.value === 0 && addedCount.value === 0) {
    emit("auto-merge");
    return;
  }
  emit("save", {
    content: buildMergedContent(props.chunks, resultValues.value, includedAdditions.value),
    stats: computeMergeStats(props.chunks, resolvedState.value, includedAdditions.value),
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
        <div class="shrink-0 border-b border-default bg-muted/80 px-4 py-2.5">
          <div class="flex flex-col gap-2">
            <div class="flex items-center justify-between gap-3">
              <h2 class="min-w-0 truncate text-lg font-semibold" :title="relPath">
                Resolving: <span class="text-primary">{{ relPath }}</span>
              </h2>
              <UButton color="error" variant="ghost" size="sm" label="Cancel Merge" @click="emit('cancel')" />
            </div>

            <UTabs
              v-if="conflictCount > 0 || allowAdditions"
              v-model="editorTab"
              :content="false"
              value-key="value"
              variant="link"
              color="primary"
              :items="[
                { label: `Conflicts (${conflictCount})`, value: 'conflicts' },
                ...(allowAdditions ? [{ label: `Additions (${addedCount})`, value: 'additions' }] : []),
              ]"
            />

            <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted">
              <span class="text-primary/70" :title="fileAPath">{{ labelA }}: ...{{ fileAPath.slice(-50) }}</span>
              <span class="text-secondary/70" :title="fileBPath">{{ labelB }}: ...{{ fileBPath.slice(-50) }}</span>
              <span v-if="conflictCount > 0" class="text-default"
                >{{ resolvedCount }}/{{ conflictCount }} resolved</span
              >
              <span v-else-if="allowAdditions" class="text-default">{{ includedCount }}/{{ addedCount }} included</span>
              <div class="ml-auto flex items-center gap-3">
                <span class="text-default/60">Result</span>
                <div class="flex gap-1">
                  <UButton
                    size="xs"
                    variant="outline"
                    :color="mergeResultLayout === 'right' ? 'primary' : 'neutral'"
                    label="Right"
                    @click="mergeResultLayout = 'right'"
                  />
                  <UButton
                    size="xs"
                    variant="outline"
                    :color="mergeResultLayout === 'bottom' ? 'primary' : 'neutral'"
                    label="Bottom"
                    @click="mergeResultLayout = 'bottom'"
                  />
                </div>
                <LangThemeSelect />
              </div>
            </div>

            <div class="flex flex-wrap items-center gap-3 border-t border-default pt-2">
              <span class="shrink-0 text-sm font-medium tabular-nums text-default/80">
                File {{ fileIndex }} of {{ fileTotal }}
              </span>
              <UButton size="sm" variant="outline" label="Skip File" @click="emit('skip')" />

              <div
                v-if="editorTab === 'conflicts' && conflictCount > 0"
                class="flex flex-1 flex-wrap items-center justify-center gap-2"
              >
                <UButton
                  size="sm"
                  variant="outline"
                  label="Prev"
                  :disabled="currentConflictNum <= 1"
                  @click="currentConflictNum -= 1"
                />
                <UButton
                  size="sm"
                  variant="outline"
                  label="Next"
                  :disabled="currentConflictNum >= conflictCount"
                  @click="currentConflictNum += 1"
                />
                <span class="text-sm font-medium tabular-nums text-default"
                  >Conflict {{ currentConflictNum }} of {{ conflictCount }}</span
                >
                <UBadge :color="choiceBadge.color" variant="subtle">{{ choiceBadge.text }}</UBadge>
                <UButton
                  size="sm"
                  :color="currentChoice === 'A' ? 'primary' : 'neutral'"
                  :variant="currentChoice === 'A' ? 'solid' : 'outline'"
                  :label="`Choose ${labelA}`"
                  @click="choose(currentChunkIndex, 'A')"
                />
                <UButton
                  size="sm"
                  :color="currentChoice === 'B' ? 'secondary' : 'neutral'"
                  :variant="currentChoice === 'B' ? 'solid' : 'outline'"
                  :label="`Choose ${labelB}`"
                  @click="choose(currentChunkIndex, 'B')"
                />
                <template v-if="unresolvedCount > 0">
                  <UButton size="sm" variant="outline" :label="`Remaining → ${labelA}`" @click="chooseRest('A')" />
                  <UButton size="sm" variant="outline" :label="`Remaining → ${labelB}`" @click="chooseRest('B')" />
                </template>
              </div>

              <div
                v-else-if="editorTab === 'additions' && addedCount > 0"
                class="flex flex-1 flex-wrap items-center justify-center gap-2"
              >
                <UButton
                  size="sm"
                  variant="outline"
                  label="Prev"
                  :disabled="currentAdditionNum <= 1"
                  @click="currentAdditionNum -= 1"
                />
                <UButton
                  size="sm"
                  variant="outline"
                  label="Next"
                  :disabled="currentAdditionNum >= addedCount"
                  @click="currentAdditionNum += 1"
                />
                <span class="text-sm font-medium tabular-nums text-default"
                  >Addition {{ currentAdditionNum }} of {{ addedCount }}</span
                >
                <UBadge :color="includedAdditions[currentAdditionIndex] ? 'success' : 'neutral'" variant="subtle">
                  {{ includedAdditions[currentAdditionIndex] ? "Included" : "Excluded" }}
                </UBadge>
                <UButton
                  size="xs"
                  variant="ghost"
                  :label="includedAdditions[currentAdditionIndex] ? 'Exclude' : 'Include'"
                  @click="setAdditionIncluded(currentAdditionIndex, !includedAdditions[currentAdditionIndex])"
                />
                <UButton size="xs" variant="ghost" label="Include all" @click="setAllAdditions(true)" />
                <UButton size="xs" variant="ghost" label="Exclude all" @click="setAllAdditions(false)" />
              </div>

              <div v-else class="flex-1 text-center text-sm text-muted">
                <template v-if="editorTab === 'additions' && addedCount === 0">No additions from {{ labelB }}</template>
                <template v-else-if="addedCount > 0 && !allowAdditions"
                  >No shared-key conflicts — {{ addedCount }} additions from {{ labelB }} will be appended</template
                >
                <template v-else-if="addedCount === 0 && conflictCount === 0"
                  >Files are identical — nothing to merge</template
                >
              </div>

              <UButton
                class="shrink-0"
                size="sm"
                color="primary"
                label="Save & Continue"
                :disabled="conflictCount > 0 && resolvedCount < conflictCount"
                @click="save"
              />
            </div>
          </div>
        </div>

        <div class="min-h-0 flex-1 overflow-hidden">
          <template
            v-if="(editorTab === 'conflicts' && diffDisplay) || (editorTab === 'additions' && additionsDiffDisplay)"
          >
            <SplitPane
              :orientation="mergeResultLayout === 'right' ? 'horizontal' : 'vertical'"
              :default-second-size="mergeResultLayout === 'right' ? 480 : 300"
              fixed-side="second"
              class="h-full rounded-none border-0"
            >
              <template #first>
                <DiffView
                  :original-content="(editorTab === 'additions' ? additionsDiffDisplay : diffDisplay)?.textA ?? ''"
                  :modified-content="(editorTab === 'additions' ? additionsDiffDisplay : diffDisplay)?.textB ?? ''"
                  :original-label="editorTab === 'additions' ? '(not in A)' : labelA"
                  :modified-label="labelB"
                  :orig-first-line="(editorTab === 'additions' ? additionsDiffDisplay : diffDisplay)?.startLineA ?? 1"
                  :mod-first-line="(editorTab === 'additions' ? additionsDiffDisplay : diffDisplay)?.startLineB ?? 1"
                  class="h-full"
                />
              </template>
              <template #second>
                <EditorView
                  :content="resultDisplay"
                  label="Result"
                  label-class="bg-accent/10 text-accent"
                  :first-line-number="(editorTab === 'additions' ? additionsDiffDisplay : diffDisplay)?.startLineA ?? 1"
                  :read-only="editorTab === 'additions'"
                  class="h-full"
                  @content-change="onResultChange"
                />
              </template>
            </SplitPane>
          </template>

          <div v-else class="flex h-full flex-col items-center justify-center gap-3 text-muted">
            <template v-if="conflictCount === 0 && addedCount > 0">
              <span class="text-lg font-medium text-default">No shared-key conflicts</span>
              <span class="text-sm"
                >{{ addedCount }} entry additions from {{ labelB }} will be appended to the merged output.</span
              >
            </template>
            <template v-else-if="conflictCount === 0">
              <span class="text-lg font-medium text-default">Files are identical</span>
              <span class="text-sm">No differences found between the two files.</span>
            </template>
            <template v-else>
              <span class="text-sm">No conflict selected</span>
            </template>
          </div>
        </div>
      </div>
    </template>
  </UModal>
</template>
