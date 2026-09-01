<script setup lang="ts">
/**
 * Ad-hoc two-file/dir merge using MergeService + workbench diffs for manual review.
 */
import { computed, ref, watch } from "vue";
import { useMutation, useQuery } from "@pinia/colada";
import FileSelector from "../components/FileSelector.vue";
import { MergePreview, Merge } from "@services/mergeservice";
import { PreviewItem, MergerOptions } from "@services/models";
import { GetUserDownloadsDir } from "@services/fileservice";
import { openDiff, openMergeEditor, startMergeOverlay } from "../ide/commands";
import { useRouter } from "vue-router";

defineOptions({ name: "ToolsMergePage" });

const router = useRouter();
const pathA = ref("");
const pathB = ref("");
const outputDir = ref("");
const manualMode = ref(false);

const { data: downloadsDir } = useQuery({
  key: () => ["downloads-dir"],
  query: async () => (await GetUserDownloadsDir()) ?? "",
});
watch(downloadsDir, (d) => {
  if (d && !outputDir.value) outputDir.value = d;
}, { immediate: true });

const mergeOptions = computed<MergerOptions>(() => ({
  addAdditionalEntries: true,
}));

const {
  mutateAsync: previewMut,
  isLoading: previewing,
  error: previewError,
  data: previewItems,
} = useMutation({
  mutation: () => MergePreview(pathA.value, pathB.value, outputDir.value),
});

const {
  mutateAsync: mergeMut,
  isLoading: merging,
  error: mergeError,
  data: mergeResults,
} = useMutation({
  mutation: () => Merge(previewItems.value ?? [], mergeOptions.value),
});

const error = computed(
  () => previewError.value?.message ?? mergeError.value?.message ?? "",
);

/** Run preview to find matching files. */
function runPreview(): void {
  if (!pathA.value || !pathB.value || !outputDir.value) return;
  void previewMut();
}

/** 2-way preview of one pair in the chrome-hidden workbench. */
async function previewItem(item: PreviewItem): Promise<void> {
  await startMergeOverlay({
    files: [item.pathA, item.pathB, item.outputPath],
    label: "Back to Merge",
    back: () => { void router.push({ name: "tools-merge" }); },
    open: () => openDiff(item.pathA, item.pathB, item.relPath),
  });
}

/** Review one pair in the VS Code merge editor. */
async function reviewItem(item: PreviewItem): Promise<void> {
  await startMergeOverlay({
    files: [item.pathA, item.pathB, item.outputPath],
    label: "Back to Merge",
    back: () => { void router.push({ name: "tools-merge" }); },
    open: () => openMergeEditor({
      input1: item.pathA,
      input2: item.pathB,
      result: item.outputPath || item.pathA,
    }),
  });
}

/** Run merge on previewed items (auto) or open first diff (manual). */
async function runMerge(): Promise<void> {
  const items = previewItems.value ?? [];
  if (!items.length) return;
  if (manualMode.value) {
    const first = items[0];
    if (first) await reviewItem(first);
    return;
  }
  void mergeMut();
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-auto p-4">
    <div class="mx-auto w-full max-w-4xl space-y-4">
      <div>
        <h1 class="text-xl font-bold">Ad-hoc Merge</h1>
        <p class="text-sm text-muted">
          Merge two paths; manual mode opens workbench diffs
        </p>
      </div>

      <UAlert
        v-if="error"
        color="error"
        variant="subtle"
        :description="error"
      />

      <UCard>
        <div class="grid gap-4 md:grid-cols-2">
          <FileSelector
            v-model="pathA"
            mode="folder"
            label="Path A"
            dialog-title="Select folder A"
          />
          <FileSelector
            v-model="pathB"
            mode="folder"
            label="Path B"
            dialog-title="Select folder B"
          />
        </div>
        <div class="mt-4">
          <FileSelector
            v-model="outputDir"
            mode="folder"
            label="Output directory"
            dialog-title="Select output folder"
          />
        </div>
        <div class="mt-4 flex items-center justify-between">
          <USwitch v-model="manualMode" label="Manual conflict resolution" />
          <div class="flex gap-2">
            <UButton
              label="Preview"
              variant="outline"
              :loading="previewing"
              :disabled="!pathA || !pathB || !outputDir"
              @click="runPreview"
            />
            <UButton
              label="Merge"
              :loading="merging"
              :disabled="!(previewItems ?? []).length"
              @click="runMerge"
            />
          </div>
        </div>
      </UCard>

      <UCard v-if="(previewItems ?? []).length" :ui="{ body: 'max-h-48 overflow-auto' }">
        <template #header>
          <span class="font-semibold">
            {{ (previewItems ?? []).length }} file(s) to merge
          </span>
        </template>
        <div class="space-y-1 text-sm">
          <div
            v-for="item in previewItems ?? []"
            :key="item.relPath"
            class="flex items-center justify-between gap-2"
          >
            <span class="truncate">{{ item.relPath }}</span>
            <div class="flex shrink-0 items-center gap-1">
              <UBadge
                v-if="item.wouldOverwrite"
                color="warning"
                variant="subtle"
                size="xs"
              >
                Overwrite
              </UBadge>
              <UButton
                v-if="manualMode"
                label="Diff"
                size="xs"
                variant="outline"
                @click="previewItem(item)"
              />
            </div>
          </div>
        </div>
      </UCard>

      <UCard v-if="(mergeResults ?? []).length" :ui="{ body: 'max-h-64 overflow-auto' }">
        <template #header>
          <span class="font-semibold">Merge Results</span>
        </template>
        <UTable
          :data="mergeResults ?? []"
          :columns="[
            { accessorKey: 'filePath', header: 'File' },
            { accessorKey: 'changed', header: 'Changed' },
            { accessorKey: 'added', header: 'Added' },
          ]"
        />
      </UCard>
    </div>
  </div>
</template>
