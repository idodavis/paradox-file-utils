<script setup lang="ts">
/**
 * Ad-hoc two-file/dir merge using existing MergeService + FileSelector.
 */
import { computed, onMounted, ref } from "vue";
import type { CodeViewItem } from "@pierre/diffs";
import { parseDiffFromFile } from "@pierre/diffs";
import FileSelector from "../components/FileSelector.vue";
import EditorView from "../components/EditorView.vue";
import MergeEditorModal from "../components/MergeEditorModal.vue";
import { MergePreview, Merge } from "@services/mergeservice";
import { PreviewItem, FileMergeResult, MergerOptions } from "@services/models";
import { GetUserDownloadsDir, ReadFileContent, WriteWithBOM } from "@services/fileservice";
import { langForPath } from "../composables/langForPath";
import { buildConflictMarkedFile } from "../composables/textMerge";

const pathA = ref("");
const pathB = ref("");
const outputDir = ref("");
const loading = ref(false);
const error = ref("");
const previewItems = ref<PreviewItem[]>([]);
const mergeResults = ref<FileMergeResult[]>([]);
const manualMode = ref(false);

const currentManualFile = ref<{
  task: PreviewItem;
  contentA: string;
  contentB: string;
  markedContent: string;
  conflictCount: number;
  identical: boolean;
} | null>(null);

const mergeOptions = computed<MergerOptions>(() => ({
  addAdditionalEntries: true,
  manualConflictResolution: manualMode.value,
  keyList: [],
  matchByFilenameOnly: false,
  includePathPattern: "",
  excludePathPattern: "",
  outputFileSuffix: "",
  outputDir: outputDir.value,
}));

const resultDiffItems = computed<CodeViewItem[]>(() => {
  if (!mergeResults.value.length) return [];
  return mergeResults.value.slice(0, 5).map((r) => ({
    id: `result:${r.outputPath}`,
    type: "file" as const,
    file: { name: r.outputPath, contents: `Changed: ${r.changed}, Added: ${r.added}`, lang: "text" },
    version: r.changed + r.added,
  }));
});

/** Load default output directory. */
async function loadDefaults(): Promise<void> {
  outputDir.value = (await GetUserDownloadsDir()) ?? "";
}

/** Run preview to find matching files. */
async function runPreview(): Promise<void> {
  if (!pathA.value || !pathB.value || !outputDir.value) return;
  loading.value = true;
  error.value = "";
  try {
    previewItems.value = (await MergePreview(pathA.value, pathB.value, outputDir.value, mergeOptions.value)) ?? [];
    if (!previewItems.value.length) {
      error.value = "No matching files found.";
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

/** Run merge on previewed items. */
async function runMerge(): Promise<void> {
  if (!previewItems.value.length) return;
  if (manualMode.value) {
    await processManualQueue([...previewItems.value]);
    return;
  }
  loading.value = true;
  error.value = "";
  try {
    mergeResults.value = (await Merge(previewItems.value, mergeOptions.value)) ?? [];
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

/** Process manual merge queue. */
async function processManualQueue(queue: PreviewItem[]): Promise<void> {
  if (!queue.length) {
    currentManualFile.value = null;
    return;
  }
  const task = queue[0];
  try {
    const [contentA, contentB] = await Promise.all([
      ReadFileContent(task.pathA),
      ReadFileContent(task.pathB),
    ]);
    const marked = buildConflictMarkedFile(contentA, contentB, {
      fileName: task.relPath,
      labelA: "File A",
      labelB: "File B",
    });
    currentManualFile.value = {
      task,
      contentA,
      contentB,
      markedContent: marked.content,
      conflictCount: marked.conflictCount,
      identical: marked.identical,
    };
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

/** Save manual merge result. */
async function saveManual(payload: { content: string }): Promise<void> {
  if (!currentManualFile.value) return;
  const task = currentManualFile.value.task;
  await WriteWithBOM(task.outputPath, payload.content);
  mergeResults.value.push({
    filePath: task.relPath,
    fileAPath: task.pathA,
    fileBPath: task.pathB,
    outputPath: task.outputPath,
    changed: 1,
    added: 0,
  });
  const remaining = previewItems.value.filter((p) => p.relPath !== task.relPath);
  currentManualFile.value = null;
  if (remaining.length) {
    await processManualQueue(remaining);
  }
}

/** Skip current file in manual mode. */
function skipManual(): void {
  if (!currentManualFile.value) return;
  const remaining = previewItems.value.filter((p) => p.relPath !== currentManualFile.value!.task.relPath);
  currentManualFile.value = null;
  if (remaining.length) {
    void processManualQueue(remaining);
  }
}

/** Cancel manual merge. */
function cancelManual(): void {
  currentManualFile.value = null;
}

onMounted(loadDefaults);
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-auto p-4">
    <div class="mx-auto w-full max-w-4xl space-y-4">
      <div>
        <h1 class="text-xl font-bold">Ad-hoc Merge</h1>
        <p class="text-sm text-muted">Merge two files or directories without a workspace</p>
      </div>

      <UAlert v-if="error" color="error" variant="subtle" :description="error" />

      <UCard>
        <div class="grid gap-4 md:grid-cols-2">
          <FileSelector v-model="pathA" mode="folder" label="Path A" dialog-title="Select folder A" />
          <FileSelector v-model="pathB" mode="folder" label="Path B" dialog-title="Select folder B" />
        </div>
        <div class="mt-4">
          <FileSelector v-model="outputDir" mode="folder" label="Output directory"
            dialog-title="Select output folder" />
        </div>
        <div class="mt-4 flex items-center justify-between">
          <USwitch v-model="manualMode" label="Manual conflict resolution" />
          <div class="flex gap-2">
            <UButton label="Preview" variant="outline" :loading="loading" :disabled="!pathA || !pathB || !outputDir"
              @click="runPreview" />
            <UButton label="Merge" :loading="loading" :disabled="!previewItems.length" @click="runMerge" />
          </div>
        </div>
      </UCard>

      <UCard v-if="previewItems.length" :ui="{ body: 'max-h-48 overflow-auto' }">
        <template #header>
          <span class="font-semibold">{{ previewItems.length }} file(s) to merge</span>
        </template>
        <div class="space-y-1 text-sm">
          <div v-for="item in previewItems" :key="item.relPath" class="flex items-center justify-between">
            <span class="truncate">{{ item.relPath }}</span>
            <UBadge v-if="item.wouldOverwrite" color="warning" variant="subtle" size="xs">Overwrite</UBadge>
          </div>
        </div>
      </UCard>

      <UCard v-if="mergeResults.length" :ui="{ body: 'max-h-64 overflow-auto' }">
        <template #header>
          <span class="font-semibold">Merge Results</span>
        </template>
        <UTable :data="mergeResults" :columns="[
          { accessorKey: 'filePath', header: 'File' },
          { accessorKey: 'changed', header: 'Changed' },
          { accessorKey: 'added', header: 'Added' },
        ]" />
      </UCard>
    </div>

    <MergeEditorModal v-if="currentManualFile" :file-a-path="currentManualFile.task.pathA"
      :file-b-path="currentManualFile.task.pathB" :rel-path="currentManualFile.task.relPath"
      :content-a="currentManualFile.contentA" :content-b="currentManualFile.contentB"
      :marked-content="currentManualFile.markedContent" :initial-conflict-count="currentManualFile.conflictCount"
      :identical="currentManualFile.identical" label-a="File A" label-b="File B" :file-index="1"
      :file-total="previewItems.length" @save="saveManual" @skip="skipManual" @cancel="cancelManual" />
  </div>
</template>
