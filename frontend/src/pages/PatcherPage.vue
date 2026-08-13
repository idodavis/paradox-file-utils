<script setup lang="ts">
/**
 * Mod Patcher: select mod, versions, run patch, preview, accept/skip, MergeEditorModal for conflicts.
 */
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import MergeEditorModal from "../components/MergeEditorModal.vue";
import { GetWorkspace, ListWorkspaceMods, ListGameInstalls } from "@services/workspaceservice";
import { Workspace, WorkspaceMod, GameInstall, PatchRun, PatchRunFile } from "@services/internal/repos/models";
import {
  StartPatchRun,
  PreviewPatchRun,
  GetPatchRunFiles,
  SetFileDecision,
  ApplyPatchRun,
} from "@services/patcherservice";
import { PatchRunPreview } from "@services/models";
import { ReadFileContent, WriteWithBOM } from "@services/fileservice";

type PatchRunFileStats = { added?: number; changed?: number; conflicts?: number };
import { buildConflictMarkedFile } from "../composables/textMerge";

const route = useRoute();
const router = useRouter();

const workspaceId = computed(() => route.params.id as string);
const workspace = ref<Workspace | null>(null);
const mods = ref<WorkspaceMod[]>([]);
const installs = ref<GameInstall[]>([]);
const selectedModId = ref<string | null>(null);
const baselineInstallId = ref<string | null>(null);
const targetInstallId = ref<string | null>(null);
const loading = ref(false);
const error = ref("");
const patchRun = ref<PatchRun | null>(null);
const preview = ref<PatchRunPreview | null>(null);
const patchFiles = ref<PatchRunFile[]>([]);

const mergeModalOpen = ref(false);
const currentMergeFile = ref<{
  file: PatchRunFile;
  contentA: string;
  contentB: string;
  markedContent: string;
  conflictCount: number;
} | null>(null);

const selectedMod = computed(() => mods.value.find((m) => m.id === selectedModId.value));
const baselineInstall = computed(() => installs.value.find((i) => i.id === baselineInstallId.value));
const targetInstall = computed(() => installs.value.find((i) => i.id === targetInstallId.value));
const canStart = computed(() => !!selectedModId.value && !!baselineInstallId.value && !!targetInstallId.value);

/** Load workspace data. */
async function loadData(): Promise<void> {
  loading.value = true;
  error.value = "";
  try {
    workspace.value = await GetWorkspace(workspaceId.value);
    if (!workspace.value) throw new Error("Workspace not found");
    mods.value = (await ListWorkspaceMods(workspaceId.value)) ?? [];
    installs.value = (await ListGameInstalls(workspace.value.gameId)) ?? [];
    if (mods.value.length) selectedModId.value = mods.value[0].id;
    if (installs.value.length >= 2) {
      baselineInstallId.value = installs.value[0].id;
      targetInstallId.value = installs.value[installs.value.length - 1].id;
    } else if (installs.value.length === 1) {
      baselineInstallId.value = installs.value[0].id;
      targetInstallId.value = installs.value[0].id;
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

/** Start a new patch run and preview. */
async function startPatch(): Promise<void> {
  if (!canStart.value) return;
  loading.value = true;
  error.value = "";
  try {
    const run = await StartPatchRun(
      workspaceId.value,
      selectedModId.value!,
      baselineInstall.value?.version ?? "",
      targetInstall.value?.version ?? "",
      baselineInstallId.value!,
      targetInstallId.value!,
    );
    if (!run) throw new Error("Failed to start patch run");
    patchRun.value = run;
    preview.value = await PreviewPatchRun(run.id);
    patchFiles.value = preview.value?.files ?? [];
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

/** Parse file stats JSON. */
function parseStats(file: PatchRunFile): PatchRunFileStats | null {
  try {
    return JSON.parse(file.stats || "{}") as PatchRunFileStats;
  } catch {
    return null;
  }
}

/** Accept a file for application. */
async function acceptFile(file: PatchRunFile): Promise<void> {
  await SetFileDecision(file.id, "accept");
  file.decision = "accept";
}

/** Skip a file. */
async function skipFile(file: PatchRunFile): Promise<void> {
  await SetFileDecision(file.id, "skip");
  file.decision = "skip";
}

/** Open merge editor for a file with conflicts. */
async function openMergeEditor(file: PatchRunFile): Promise<void> {
  if (!file.previewPath) return;
  loading.value = true;
  try {
    const contentA = await ReadFileContent(file.previewPath);
    const contentB = await ReadFileContent(file.previewPath);
    const marked = buildConflictMarkedFile(contentA, contentB, {
      fileName: file.relPath,
      labelA: "Baseline",
      labelB: "Target",
    });
    currentMergeFile.value = {
      file,
      contentA,
      contentB,
      markedContent: marked.content,
      conflictCount: marked.conflictCount,
    };
    mergeModalOpen.value = true;
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

/** Save merge result. */
async function saveMerge(payload: { content: string }): Promise<void> {
  if (!currentMergeFile.value) return;
  const file = currentMergeFile.value.file;
  await WriteWithBOM(file.previewPath, payload.content);
  await SetFileDecision(file.id, "accept");
  file.decision = "accept";
  mergeModalOpen.value = false;
  currentMergeFile.value = null;
}

/** Apply accepted files to the mod. */
async function applyPatch(): Promise<void> {
  if (!patchRun.value) return;
  loading.value = true;
  error.value = "";
  try {
    await ApplyPatchRun(patchRun.value.id);
    patchFiles.value = (await GetPatchRunFiles(patchRun.value.id)) ?? [];
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

watch(workspaceId, loadData, { immediate: true });
onMounted(loadData);
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <div class="flex shrink-0 items-center justify-between gap-2 border-b border-default bg-muted/50 px-3 py-2">
      <div class="flex items-center gap-2">
        <UButton icon="i-lucide-arrow-left" variant="ghost" size="sm"
          @click="router.push({ name: 'workspace-ide', params: { id: workspaceId } })" />
        <span class="font-semibold">Mod Patcher</span>
      </div>
      <UButton label="Patch Center" icon="i-lucide-newspaper" variant="outline" size="sm"
        @click="router.push({ name: 'patch-center', params: { id: workspaceId } })" />
    </div>

    <UAlert v-if="error" color="error" variant="subtle" :description="error" class="m-2" />

    <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-auto p-3">
      <UCard>
        <template #header>
          <span class="font-semibold">Patch Configuration</span>
        </template>
        <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <UFormField label="Mod">
            <USelect :model-value="selectedModId ?? undefined"
              :items="mods.map((m) => ({ label: m.name, value: m.id }))" value-key="value" placeholder="Select mod"
              @update:model-value="(v: string) => selectedModId = v" />
          </UFormField>
          <UFormField label="Baseline Install">
            <USelect :model-value="baselineInstallId ?? undefined"
              :items="installs.map((i) => ({ label: `${i.name} (${i.version || '?'})`, value: i.id }))"
              value-key="value" placeholder="Baseline" @update:model-value="(v: string) => baselineInstallId = v" />
          </UFormField>
          <UFormField label="Target Install">
            <USelect :model-value="targetInstallId ?? undefined"
              :items="installs.map((i) => ({ label: `${i.name} (${i.version || '?'})`, value: i.id }))"
              value-key="value" placeholder="Target" @update:model-value="(v: string) => targetInstallId = v" />
          </UFormField>
          <div class="flex items-end">
            <UButton label="Start Patch" icon="i-lucide-play" :disabled="!canStart" :loading="loading"
              @click="startPatch" />
          </div>
        </div>
      </UCard>

      <UCard v-if="preview" class="min-h-0 flex-1" :ui="{ body: 'overflow-auto' }">
        <template #header>
          <div class="flex items-center justify-between">
            <div>
              <span class="font-semibold">Preview</span>
              <span class="ml-2 text-sm text-muted">
                {{ preview.totalFiles }} files | {{ preview.safeCount }} safe | {{ preview.reviewCount }} review
              </span>
            </div>
            <UButton label="Apply Accepted" icon="i-lucide-check" :loading="loading"
              :disabled="!patchFiles.some((f) => f.decision === 'accept')" @click="applyPatch" />
          </div>
        </template>
        <UTable :data="patchFiles" :columns="[
          { accessorKey: 'relPath', header: 'File' },
          { accessorKey: 'status', header: 'Status' },
          { id: 'stats', header: 'Stats' },
          { id: 'decision', header: 'Decision' },
          { id: 'actions', header: '' },
        ]">
          <template #status-cell="{ row }">
            <UBadge
              :color="row.original.status === 'safe' ? 'success' : row.original.status === 'review' ? 'warning' : 'neutral'"
              variant="subtle" size="xs">
              {{ row.original.status }}
            </UBadge>
          </template>
          <template #stats-cell="{ row }">
            <template v-if="parseStats(row.original)">
              +{{ parseStats(row.original)?.added ?? 0 }} / ~{{ parseStats(row.original)?.changed ?? 0 }}
              <span v-if="(parseStats(row.original)?.conflicts ?? 0) > 0" class="text-error">
                / {{ parseStats(row.original)?.conflicts }} conflicts
              </span>
            </template>
          </template>
          <template #decision-cell="{ row }">
            <UBadge v-if="row.original.decision" :color="row.original.decision === 'accept' ? 'success' : 'neutral'"
              variant="subtle" size="xs">
              {{ row.original.decision }}
            </UBadge>
          </template>
          <template #actions-cell="{ row }">
            <div class="flex gap-1">
              <UButton v-if="row.original.status === 'review'" label="Merge" size="xs" variant="outline"
                @click="openMergeEditor(row.original)" />
              <UButton label="Accept" size="xs" color="success" variant="ghost"
                :disabled="row.original.decision === 'accept'" @click="acceptFile(row.original)" />
              <UButton label="Skip" size="xs" color="neutral" variant="ghost"
                :disabled="row.original.decision === 'skip'" @click="skipFile(row.original)" />
            </div>
          </template>
        </UTable>
      </UCard>
    </div>

    <MergeEditorModal v-if="currentMergeFile && mergeModalOpen" :file-a-path="currentMergeFile.file.previewPath"
      :file-b-path="currentMergeFile.file.previewPath" :rel-path="currentMergeFile.file.relPath"
      :content-a="currentMergeFile.contentA" :content-b="currentMergeFile.contentB"
      :marked-content="currentMergeFile.markedContent" :initial-conflict-count="currentMergeFile.conflictCount"
      :identical="false" label-a="Baseline" label-b="Target" :file-index="1" :file-total="1" @save="saveMerge"
      @skip="mergeModalOpen = false" @cancel="mergeModalOpen = false" />
  </div>
</template>
