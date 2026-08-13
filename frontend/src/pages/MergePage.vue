<script setup lang="ts">
/**
 * Merge workflow page.
 */
import { computed, onMounted, watch } from "vue";
import type { DropdownMenuItem } from "@nuxt/ui";
import type { CodeViewItem } from "@pierre/diffs";
import { parseDiffFromFile } from "@pierre/diffs";
import MergeEditorModal from "../components/MergeEditorModal.vue";
import FileSelector from "../components/FileSelector.vue";
import EditorView from "../components/EditorView.vue";
import { createMergeWorkflow, provideMergeWorkflow } from "../composables/mergeWorkflow";

const workflow = createMergeWorkflow();
provideMergeWorkflow(workflow);

const moreOptionsItems = [{ label: "More options", icon: "i-lucide-settings" }];
const presetDeleteItems = computed<DropdownMenuItem[]>(() =>
  workflow.presets.value.map((preset) => ({
    label: preset.name,
    icon: "i-lucide-trash-2",
    color: "error" as const,
    onSelect: () => workflow.deletePreset(preset.name),
  })),
);

/** Diff items for the post-merge result dialog. */
const resultDiffItems = computed<CodeViewItem[]>(() => {
  const result = workflow.selectedResult.value;
  if (!result) return [];
  const original =
    workflow.diffSide.value === "A"
      ? workflow.resultFileAContent.value
      : workflow.resultFileBContent.value;
  const originalLabel =
    workflow.diffSide.value === "A" ? workflow.labels.value.a : workflow.labels.value.b;
  return [
    {
      id: `result-diff:${result.outputPath}`,
      type: "diff",
      fileDiff: parseDiffFromFile(
        { name: originalLabel, contents: original, lang: "hcl" },
        { name: "Merged", contents: workflow.resultMergedContent.value, lang: "hcl" },
      ),
    },
  ];
});

watch(workflow.currentGame, async () => {
  await workflow.resetForGameChange();
});

onMounted(workflow.initialize);
</script>

<template>
  <div class="flex h-full min-h-0 flex-1 flex-col gap-2 overflow-hidden p-3">
    <div class="flex shrink-0 flex-wrap items-center gap-2">
      <div class="min-w-0 flex-1">
        <h1 class="text-lg font-semibold">Script Merger</h1>
        <p class="text-xs text-muted">Presets, preview, and conflict resolution</p>
      </div>
      <UBadge
        :color="workflow.config.value.manualConflictResolution ? 'secondary' : 'primary'"
        variant="subtle"
        size="sm"
      >
        {{ workflow.config.value.manualConflictResolution ? "Manual" : "Auto" }}
      </UBadge>
      <UBadge color="neutral" variant="outline" size="sm">
        {{ workflow.currentPresetName.value || "No preset" }}
      </UBadge>
    </div>

    <UAccordion :items="moreOptionsItems" :unmount-on-hide="false" class="shrink-0">
      <template #body>
        <div class="space-y-3 pb-2">
          <div class="flex flex-wrap items-center gap-2">
            <USelect
              v-model="workflow.currentPresetName.value"
              :items="workflow.presets.value.map((preset) => preset.name)"
              class="min-w-40"
              size="sm"
              @update:model-value="workflow.applyPresetByName"
            />
            <UFieldGroup>
              <UInput v-model="workflow.presetNameToSave.value" placeholder="Preset name" size="sm" class="min-w-32" />
              <UButton
                :disabled="!workflow.presetNameToSave.value.trim()"
                label="Save"
                variant="outline"
                size="sm"
                @click="workflow.saveCurrentPreset"
              />
            </UFieldGroup>
            <UDropdownMenu v-if="presetDeleteItems.length" :items="presetDeleteItems">
              <UButton label="Delete" color="error" variant="ghost" size="sm" trailing-icon="i-lucide-chevron-down" />
            </UDropdownMenu>
            <USelect
              v-model="workflow.config.value.manualConflictResolution"
              :items="workflow.resolutionModeOptions"
              value-key="value"
              class="w-36"
              size="sm"
            />
          </div>
          <div class="flex flex-wrap gap-4">
            <USwitch v-model="workflow.config.value.addAdditionalEntries" label="Add B-only entries" size="sm" />
            <USwitch
              v-model="workflow.config.value.useKeyList"
              :disabled="workflow.config.value.manualConflictResolution"
              label="Key list"
              size="sm"
            />
            <USwitch v-model="workflow.config.value.matchByFilenameOnly" label="Filename match" size="sm" />
          </div>
          <UTextarea
            v-if="workflow.config.value.useKeyList && !workflow.config.value.manualConflictResolution"
            v-model="workflow.config.value.customKeys"
            placeholder="key1&#10;key2"
            :rows="3"
          />
          <div class="grid gap-2 md:grid-cols-3">
            <UInput v-model="workflow.config.value.includePathPattern" placeholder="Include regex" size="sm" />
            <UInput v-model="workflow.config.value.excludePathPattern" placeholder="Exclude regex" size="sm" />
            <UInput v-model="workflow.config.value.outputFileSuffix" placeholder="Output suffix" size="sm" />
          </div>
          <FileSelector
            v-model="workflow.outputDir.value"
            mode="folder"
            label="Output directory"
            dialog-title="Select output directory"
            placeholder="Defaults to Downloads"
          />
        </div>
      </template>
    </UAccordion>

    <div class="flex min-h-0 flex-1 flex-col gap-2 overflow-hidden">
      <UTabs
        v-model="workflow.activeTab.value"
        :items="workflow.activeTabOptions"
        value-key="value"
        :unmount-on-hide="false"
        variant="link"
        color="primary"
        size="sm"
        class="shrink-0"
      >
        <template #vanilla>
          <div class="flex flex-wrap items-end gap-2 pb-2">
            <div class="min-w-64 flex-1">
              <FileSelector
                v-model="workflow.modPath.value"
                mode="folder"
                label="Mod folder"
                dialog-title="Select mod folder"
              />
            </div>
            <UButton
              :loading="workflow.loadingPreview.value"
              :disabled="!workflow.canRun.value.vanilla"
              label="Preview"
              size="sm"
              @click="workflow.runPreview('vanilla')"
            />
            <UButton
              color="secondary"
              :loading="workflow.runningMerge.value"
              :disabled="!workflow.canRun.value.vanilla"
              label="Merge"
              size="sm"
              @click="workflow.runDirMerge('vanilla')"
            />
          </div>
        </template>
        <template #dirs>
          <div class="space-y-2 pb-2">
            <div class="grid gap-2 md:grid-cols-2">
              <FileSelector
                v-model="workflow.pathA.value"
                mode="folder"
                label="Directory A"
                dialog-title="Select directory A"
              />
              <FileSelector
                v-model="workflow.pathB.value"
                mode="folder"
                label="Directory B"
                dialog-title="Select directory B"
              />
            </div>
            <div class="flex gap-2">
              <UButton
                :loading="workflow.loadingPreview.value"
                :disabled="!workflow.canRun.value.dirs"
                label="Preview"
                size="sm"
                @click="workflow.runPreview('dirs')"
              />
              <UButton
                color="secondary"
                :loading="workflow.runningMerge.value"
                :disabled="!workflow.canRun.value.dirs"
                label="Merge"
                size="sm"
                @click="workflow.runDirMerge('dirs')"
              />
            </div>
          </div>
        </template>
        <template #pairs>
          <div class="space-y-2 pb-2">
            <div class="flex justify-end">
              <UButton
                icon="i-lucide-plus"
                label="Add pair"
                variant="outline"
                size="sm"
                @click="workflow.filePairs.value.push({ pathA: '', pathB: '', outputName: '' })"
              />
            </div>
            <div
              v-for="(pair, index) in workflow.filePairs.value"
              :key="index"
              class="grid grid-cols-1 gap-2 rounded border border-default p-2 lg:grid-cols-12"
            >
              <div class="lg:col-span-4">
                <FileSelector v-model="pair.pathA" mode="file" label="File A" dialog-title="Select file A" />
              </div>
              <div class="lg:col-span-4">
                <FileSelector v-model="pair.pathB" mode="file" label="File B" dialog-title="Select file B" />
              </div>
              <div class="lg:col-span-3">
                <UFormField label="Output name">
                  <UInput v-model="pair.outputName" placeholder="merged.txt" size="sm" />
                </UFormField>
              </div>
              <div class="flex items-end lg:col-span-1">
                <UButton
                  color="error"
                  variant="ghost"
                  icon="i-lucide-trash-2"
                  size="sm"
                  @click="workflow.filePairs.value.splice(index, 1)"
                />
              </div>
            </div>
            <UButton
              color="secondary"
              :loading="workflow.runningMerge.value"
              :disabled="!workflow.canRun.value.pairs"
              label="Merge"
              size="sm"
              @click="workflow.runPairMerge"
            />
          </div>
        </template>
      </UTabs>

      <div class="min-h-0 flex-1 space-y-2 overflow-auto">
        <UAlert
          v-if="workflow.errorMsg.value"
          color="error"
          variant="subtle"
          title="Merge error"
          :description="workflow.errorMsg.value"
        />
        <UAlert
          v-if="workflow.validationErrors.value.length"
          color="warning"
          variant="subtle"
          :title="`${workflow.validationErrors.value.length} validation error(s)`"
        />

        <UCard
          v-if="workflow.previewItems.value.length"
          :ui="{ root: 'flex flex-col', body: 'max-h-64 overflow-auto p-0' }"
        >
          <template #header>
            <div class="flex items-center justify-between">
              <div class="text-sm font-medium">Preview</div>
              <div class="flex gap-1">
                <UButton label="All" variant="outline" size="xs" @click="workflow.setSelectedAll(true)" />
                <UButton label="None" variant="outline" size="xs" @click="workflow.setSelectedAll(false)" />
              </div>
            </div>
          </template>
          <UTable :data="workflow.previewItems.value" :columns="workflow.previewColumns">
            <template #selected-cell="{ row }">
              <UCheckbox
                :model-value="Boolean(workflow.selectedRelPaths.value[row.original.relPath])"
                color="primary"
                @update:model-value="
                  (value) => (workflow.selectedRelPaths.value[row.original.relPath] = value === true)
                "
              />
            </template>
            <template #wouldOverwrite-cell="{ row }">
              <UBadge :color="row.original.wouldOverwrite ? 'warning' : 'success'" variant="subtle" size="sm">
                {{ row.original.wouldOverwrite ? "Yes" : "No" }}
              </UBadge>
            </template>
          </UTable>
        </UCard>

        <UCard
          v-if="workflow.summary.value.files"
          class="min-h-0 flex-1"
          :ui="{ root: 'flex min-h-0 flex-col', body: 'min-h-0 flex-1 overflow-auto p-0' }"
        >
          <template #header>
            <div class="flex items-center justify-between">
              <div class="text-sm font-medium">Results</div>
              <div class="flex gap-1">
                <UButton
                  :loading="workflow.savingReport.value"
                  label="Save report"
                  variant="outline"
                  size="xs"
                  @click="workflow.saveReport"
                />
                <UButton
                  :loading="workflow.validating.value"
                  label="Validate"
                  variant="outline"
                  size="xs"
                  @click="workflow.runValidation"
                />
              </div>
            </div>
          </template>
          <UTable
            :data="workflow.mergeResults.value"
            :columns="workflow.resultColumns"
            @select="(_event, row) => workflow.openResult(row.index)"
          >
            <template #outputPath-cell="{ row }">
              {{ workflow.truncatePath(row.original.outputPath) }}
            </template>
          </UTable>
        </UCard>
      </div>
    </div>

    <UModal v-model:open="workflow.showResultDialog.value" :ui="{ content: 'sm:max-w-[96vw] max-h-[95vh]' }">
      <template #body>
        <div v-if="workflow.selectedResult.value" class="flex min-h-[70vh] flex-col gap-2">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="text-sm font-medium">File comparison</div>
            <USelect
              v-model="workflow.diffSide.value"
              :items="workflow.diffSideOptions.value"
              value-key="value"
              class="w-52"
              size="sm"
            />
          </div>
          <EditorView class="min-h-0 flex-1" :items="resultDiffItems" />
        </div>
      </template>
    </UModal>

    <MergeEditorModal
      v-if="workflow.currentManualFile.value"
      :file-a-path="workflow.currentManualFile.value.task.pathA"
      :file-b-path="workflow.currentManualFile.value.task.pathB"
      :rel-path="workflow.currentManualFile.value.task.relPath"
      :content-a="workflow.currentManualFile.value.contentA"
      :content-b="workflow.currentManualFile.value.contentB"
      :marked-content="workflow.currentManualFile.value.markedContent"
      :initial-conflict-count="workflow.currentManualFile.value.conflictCount"
      :identical="workflow.currentManualFile.value.identical"
      :label-a="workflow.labels.value.a"
      :label-b="workflow.labels.value.b"
      :file-index="workflow.manualQueueCurrent.value"
      :file-total="workflow.manualQueueTotal.value"
      @save="workflow.saveManual"
      @auto-merge="workflow.autoMergeCurrentFile"
      @skip="workflow.skipFile"
      @cancel="workflow.cancelManualMerge"
    />
  </div>
</template>
