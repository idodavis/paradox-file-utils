<script setup lang="ts">
/**
 * Merge workflow page.
 */
import { onMounted, watch } from "vue";
import MergeEditorModal from "../components/MergeEditorModal.vue";
import FileSelector from "../components/FileSelector.vue";
import DiffView from "../components/DiffView.vue";
import { createMergeWorkflow, provideMergeWorkflow } from "../composables/mergeWorkflow";

const workflow = createMergeWorkflow();
provideMergeWorkflow(workflow);

watch(workflow.currentGame, async () => {
  await workflow.resetForGameChange();
});

onMounted(workflow.initialize);
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-4">
    <UCard>
      <template #header>
        <div class="text-xs font-semibold uppercase tracking-wide text-primary">Merge workflow</div>
        <div class="text-2xl font-bold">Script Merger</div>
        <div class="mt-2 text-sm text-muted">
          Merge matching script files with presets, preview, results, and manual conflict resolution.
        </div>
      </template>

      <div class="space-y-4">
        <div class="flex flex-wrap items-center gap-2">
          <UBadge :color="workflow.config.value.manualConflictResolution ? 'secondary' : 'primary'" variant="subtle">
            {{ workflow.config.value.manualConflictResolution ? "Manual" : "Auto" }}
          </UBadge>
          <UBadge color="neutral" variant="outline">Preset: {{ workflow.currentPresetName.value || "None" }}</UBadge>
          <UBadge color="neutral" variant="outline">Out: {{ workflow.outputDir.value || "Downloads" }}</UBadge>
        </div>

        <details class="rounded-lg border border-default p-3">
          <summary class="cursor-pointer text-sm font-medium">More options</summary>
          <div class="mt-3 space-y-4">
            <div>
              <div class="mb-2 text-sm font-medium">Presets</div>
              <div class="flex flex-wrap items-center gap-2">
                <USelect
                  v-model="workflow.currentPresetName.value"
                  :items="workflow.presets.value.map((preset) => preset.name)"
                  class="min-w-[12rem]"
                  @update:model-value="workflow.applyPresetByName"
                />
                <UInput v-model="workflow.presetNameToSave.value" placeholder="Name" class="min-w-[10rem]" />
                <UButton
                  :disabled="!workflow.presetNameToSave.value.trim()"
                  label="Save"
                  variant="outline"
                  @click="workflow.saveCurrentPreset"
                />
              </div>
              <div class="mt-2 flex flex-wrap gap-2">
                <UButton
                  v-for="preset in workflow.presets.value"
                  :key="preset.name"
                  :label="`Delete ${preset.name}`"
                  color="error"
                  variant="ghost"
                  size="sm"
                  @click="workflow.deletePreset(preset.name)"
                />
              </div>
            </div>

            <div class="space-y-2">
              <div class="text-sm font-medium">Resolution mode</div>
              <USelect
                v-model="workflow.config.value.manualConflictResolution"
                :items="workflow.resolutionModeOptions"
                value-key="value"
                class="w-48"
              />
            </div>

            <div class="space-y-2">
              <label class="flex items-center gap-2 text-sm">
                <USwitch v-model="workflow.config.value.addAdditionalEntries" />
                <span>Add entries from B not in A</span>
              </label>
              <label class="flex items-center gap-2 text-sm">
                <USwitch
                  v-model="workflow.config.value.useKeyList"
                  :disabled="workflow.config.value.manualConflictResolution"
                />
                <span>Key list (B overrides A)</span>
              </label>
              <UTextarea
                v-if="workflow.config.value.useKeyList && !workflow.config.value.manualConflictResolution"
                v-model="workflow.config.value.customKeys"
                placeholder="key1&#10;key2"
                :rows="4"
              />
              <label class="flex items-center gap-2 text-sm">
                <USwitch v-model="workflow.config.value.matchByFilenameOnly" />
                <span>Match by filename only</span>
              </label>
              <UInput
                v-model="workflow.config.value.includePathPattern"
                placeholder="Include path (regex) e.g. events/"
              />
              <UInput
                v-model="workflow.config.value.excludePathPattern"
                placeholder="Exclude path (regex) e.g. common/"
              />
              <UInput v-model="workflow.config.value.outputFileSuffix" placeholder="Output file suffix e.g. _merged" />
            </div>

            <FileSelector
              v-model="workflow.outputDir.value"
              mode="folder"
              label="Output directory"
              dialog-title="Select output directory"
              placeholder="Output directory for merged files"
              hint="Defaults to the Downloads folder when empty."
            />
          </div>
        </details>
      </div>
    </UCard>

    <div class="space-y-4">
      <USelect v-model="workflow.activeTab.value" :items="workflow.activeTabOptions" value-key="value" class="w-64" />

      <UCard v-if="workflow.activeTab.value === 'vanilla'">
        <div class="space-y-3">
          <FileSelector
            v-model="workflow.modPath.value"
            mode="folder"
            label="Mod folder"
            dialog-title="Select mod folder"
            placeholder="Folder containing the mod"
          />
        </div>
        <template #footer>
          <div class="flex gap-2">
            <UButton
              :loading="workflow.loadingPreview.value"
              :disabled="!workflow.canRun.value.vanilla"
              label="Preview"
              @click="workflow.runPreview('vanilla')"
            />
            <UButton
              color="secondary"
              :loading="workflow.runningMerge.value"
              :disabled="!workflow.canRun.value.vanilla"
              label="Merge"
              @click="workflow.runDirMerge('vanilla')"
            />
          </div>
        </template>
      </UCard>

      <UCard v-else-if="workflow.activeTab.value === 'dirs'">
        <div class="space-y-3">
          <FileSelector
            v-model="workflow.pathA.value"
            mode="folder"
            label="Directory A"
            dialog-title="Select directory A"
            placeholder="First directory"
          />
          <FileSelector
            v-model="workflow.pathB.value"
            mode="folder"
            label="Directory B"
            dialog-title="Select directory B"
            placeholder="Second directory"
          />
        </div>
        <template #footer>
          <div class="flex gap-2">
            <UButton
              :loading="workflow.loadingPreview.value"
              :disabled="!workflow.canRun.value.dirs"
              label="Preview"
              @click="workflow.runPreview('dirs')"
            />
            <UButton
              color="secondary"
              :loading="workflow.runningMerge.value"
              :disabled="!workflow.canRun.value.dirs"
              label="Merge"
              @click="workflow.runDirMerge('dirs')"
            />
          </div>
        </template>
      </UCard>

      <UCard v-else>
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <div class="text-sm font-medium">Pairs</div>
            <UButton
              icon="i-lucide-plus"
              label="Add pair"
              variant="outline"
              @click="workflow.filePairs.value.push({ pathA: '', pathB: '', outputName: '' })"
            />
          </div>
          <UCard v-for="(pair, index) in workflow.filePairs.value" :key="index">
            <div class="grid grid-cols-1 gap-3 lg:grid-cols-12">
              <div class="lg:col-span-4">
                <FileSelector
                  v-model="pair.pathA"
                  mode="file"
                  label="File A"
                  dialog-title="Select file A"
                  placeholder="First file"
                />
              </div>
              <div class="lg:col-span-4">
                <FileSelector
                  v-model="pair.pathB"
                  mode="file"
                  label="File B"
                  dialog-title="Select file B"
                  placeholder="Second file"
                />
              </div>
              <div class="lg:col-span-3">
                <UFormField label="Output name">
                  <UInput v-model="pair.outputName" placeholder="merged.txt" />
                </UFormField>
              </div>
              <div class="lg:col-span-1 lg:pt-7">
                <UButton
                  color="error"
                  variant="ghost"
                  icon="i-lucide-trash-2"
                  @click="workflow.filePairs.value.splice(index, 1)"
                />
              </div>
            </div>
          </UCard>
        </div>
        <template #footer>
          <UButton
            color="secondary"
            :loading="workflow.runningMerge.value"
            :disabled="!workflow.canRun.value.pairs"
            label="Merge"
            @click="workflow.runPairMerge"
          />
        </template>
      </UCard>
    </div>

    <div class="space-y-4">
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

      <UCard v-if="workflow.previewItems.value.length">
        <template #header>
          <div class="flex items-center justify-between">
            <div class="text-base font-medium">Preview</div>
            <div class="flex items-center gap-2">
              <UButton label="All" variant="outline" size="sm" @click="workflow.setSelectedAll(true)" />
              <UButton label="None" variant="outline" size="sm" @click="workflow.setSelectedAll(false)" />
            </div>
          </div>
        </template>
        <UTable :data="workflow.previewItems.value" :columns="workflow.previewColumns">
          <template #selected-cell="{ row }">
            <UCheckbox v-model="workflow.selectedRelPaths.value[row.original.relPath]" color="primary" />
          </template>
          <template #wouldOverwrite-cell="{ row }">
            <UBadge :color="row.original.wouldOverwrite ? 'warning' : 'success'" variant="subtle">
              {{ row.original.wouldOverwrite ? "Yes" : "No" }}
            </UBadge>
          </template>
        </UTable>
      </UCard>

      <UCard v-if="workflow.summary.value.files">
        <template #header>
          <div class="flex items-center justify-between">
            <div class="text-base font-medium">Results</div>
            <div class="flex gap-2">
              <UButton
                :loading="workflow.savingReport.value"
                label="Save report"
                variant="outline"
                @click="workflow.saveReport"
              />
              <UButton
                :loading="workflow.validating.value"
                label="Validate"
                variant="outline"
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

        <div v-if="workflow.conflicts.value.length" class="mt-4 space-y-2">
          <div class="text-sm font-medium">Conflict audit</div>
          <UCard v-for="file in workflow.conflicts.value" :key="file.outputPath">
            <div class="mb-2 font-medium">{{ file.filePath }}</div>
            <div
              v-for="conflict in file.resolvedConflicts ?? []"
              :key="conflict.key"
              class="flex items-center gap-2 text-xs text-muted"
            >
              <UBadge color="neutral" variant="outline">{{ conflict.key }}</UBadge>
              <span>→</span>
              <span>{{ conflict.usedSide }}</span>
              <span>({{ conflict.reason }})</span>
            </div>
          </UCard>
        </div>
      </UCard>

      <UModal v-model:open="workflow.showResultDialog.value" :ui="{ content: 'sm:max-w-[96vw] max-h-[95vh]' }">
        <template #body>
          <div v-if="workflow.selectedResult.value" class="flex min-h-0 flex-col space-y-3">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="text-base font-medium">File comparison view</div>
              <USelect
                v-model="workflow.diffSide.value"
                :items="workflow.diffSideOptions.value"
                value-key="value"
                class="w-52"
              />
            </div>
            <div class="text-xs text-muted">
              {{
                workflow.diffSide.value === "A"
                  ? workflow.selectedResult.value.fileAPath
                  : workflow.selectedResult.value.fileBPath
              }}
              → {{ workflow.selectedResult.value.outputPath }}
            </div>
            <div class="min-h-[18rem] flex-1">
              <DiffView
                :original-content="
                  workflow.diffSide.value === 'A'
                    ? workflow.resultFileAContent.value
                    : workflow.resultFileBContent.value
                "
                :modified-content="workflow.resultMergedContent.value"
                :original-label="workflow.diffSide.value === 'A' ? workflow.labels.value.a : workflow.labels.value.b"
                modified-label="Merged"
                :render-side-by-side="true"
                class="h-full min-h-[18rem]"
              />
            </div>
          </div>
        </template>
      </UModal>
    </div>

    <MergeEditorModal
      v-if="workflow.currentManualFile.value"
      :file-a-path="workflow.currentManualFile.value.task.pathA"
      :file-b-path="workflow.currentManualFile.value.task.pathB"
      :rel-path="workflow.currentManualFile.value.task.relPath"
      :chunks="workflow.currentManualFile.value.chunks"
      :label-a="workflow.labels.value.a"
      :label-b="workflow.labels.value.b"
      :file-index="workflow.manualQueueCurrent.value"
      :file-total="workflow.manualQueueTotal.value"
      :allow-additions="workflow.showAdditionsTab.value"
      @save="workflow.saveManual"
      @auto-merge="workflow.autoMergeCurrentFile"
      @skip="workflow.skipFile"
      @cancel="workflow.cancelManualMerge"
    />
  </div>
</template>
