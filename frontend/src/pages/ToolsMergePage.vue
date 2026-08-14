<script setup lang="ts">
/**
 * Ad-hoc two-file/dir merge using MergeService + workbench diffs for manual review.
 */
import { computed, onMounted, ref } from "vue";
import FileSelector from "../components/FileSelector.vue";
import { MergePreview, Merge } from "@services/mergeservice";
import { PreviewItem, FileMergeResult, MergerOptions } from "@services/models";
import { GetUserDownloadsDir } from "@services/fileservice";
import { reviewDiffInWorkbench } from "../composables/workbenchMerge";
import { useRouter } from "vue-router";

const router = useRouter();
const pathA = ref("");
const pathB = ref("");
const outputDir = ref("");
const loading = ref(false);
const error = ref("");
const previewItems = ref<PreviewItem[]>([]);
const mergeResults = ref<FileMergeResult[]>([]);
const manualMode = ref(false);

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
    previewItems.value =
      (await MergePreview(
        pathA.value,
        pathB.value,
        outputDir.value,
        mergeOptions.value,
      )) ?? [];
    if (!previewItems.value.length) {
      error.value = "No matching files found.";
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

/** Review one preview pair in the workbench. */
async function reviewItem(item: PreviewItem): Promise<void> {
  await reviewDiffInWorkbench({
    pathA: item.pathA,
    pathB: item.pathB,
    title: item.relPath,
    onBack: () => {
      void router.push({ name: "tools-merge" });
    },
  });
}

/** Run merge on previewed items (auto) or open first diff (manual). */
async function runMerge(): Promise<void> {
  if (!previewItems.value.length) return;
  if (manualMode.value) {
    const first = previewItems.value[0];
    if (first) await reviewItem(first);
    return;
  }
  loading.value = true;
  error.value = "";
  try {
    mergeResults.value =
      (await Merge(previewItems.value, mergeOptions.value)) ?? [];
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

onMounted(loadDefaults);
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
              :loading="loading"
              :disabled="!pathA || !pathB || !outputDir"
              @click="runPreview"
            />
            <UButton
              label="Merge"
              :loading="loading"
              :disabled="!previewItems.length"
              @click="runMerge"
            />
          </div>
        </div>
      </UCard>

      <UCard v-if="previewItems.length" :ui="{ body: 'max-h-48 overflow-auto' }">
        <template #header>
          <span class="font-semibold">
            {{ previewItems.length }} file(s) to merge
          </span>
        </template>
        <div class="space-y-1 text-sm">
          <div
            v-for="item in previewItems"
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
                @click="reviewItem(item)"
              />
            </div>
          </div>
        </div>
      </UCard>

      <UCard v-if="mergeResults.length" :ui="{ body: 'max-h-64 overflow-auto' }">
        <template #header>
          <span class="font-semibold">Merge Results</span>
        </template>
        <UTable
          :data="mergeResults"
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
