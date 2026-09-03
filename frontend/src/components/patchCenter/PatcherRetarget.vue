<script setup lang="ts">
/**
 * File retarget: pick mod and target install, preview, accept/skip, apply.
 */
import { computed, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useRoute, useRouter } from "vue-router";
import { useMutation, useQuery } from "@pinia/colada";
import { ListGameInstalls } from "@services/workspaceservice";
import { PatchRunFile } from "@services/models";
import {
  StartPatchRun,
  PreviewPatchRun,
  SetFileDecision,
  ApplyPatchRun,
  CancelPatchRun,
} from "@services/patcherservice";
import { openMergeEditor, startMergeOverlay } from "../../ide/commands";
import { useWorkspaceStore } from "../../stores/workspace";

const ws = useWorkspaceStore();
const route = useRoute();
const router = useRouter();
const toast = useToast();

const workspaceId = computed(() => String(route.params.id ?? ""));
const { activeWorkspace: workspace, workspaceMods: mods } = storeToRefs(ws);
const modId = ref<string | null>(null);
const targetInstallId = ref<string | null>(null);
const patchFiles = ref<PatchRunFile[]>([]);

const {
  data: installs,
  error: loadError,
} = useQuery({
  key: () => ["patcher-installs", workspaceId.value, workspace.value?.gameId ?? ""],
  query: async () => {
    await ws.loadActiveWorkspace();
    if (!workspace.value) throw new Error("Workspace not found");
    return (await ListGameInstalls(workspace.value.gameId)) ?? [];
  },
  enabled: () => !!workspaceId.value,
});

watch(
  [installs, mods, () => route.query.mod, () => route.query.install],
  () => {
    const qMod = String(route.query.mod ?? "");
    const qInst = String(route.query.install ?? "");
    if (qMod && mods.value.some((m) => m.id === qMod)) {
      modId.value = qMod;
    } else if (mods.value.length && !modId.value) {
      modId.value = mods.value[0]!.id;
    }
    if (qInst && installs.value?.some((i) => i.id === qInst)) {
      targetInstallId.value = qInst;
      return;
    }
    if (targetInstallId.value || !installs.value?.length) return;
    const preferred = workspace.value?.installId;
    targetInstallId.value =
      installs.value.find((i) => i.id === preferred)?.id ??
      installs.value.at(-1)?.id ??
      null;
  },
  { immediate: true },
);

const targetInstall = computed(() =>
  installs.value?.find((i) => i.id === targetInstallId.value),
);
const canStart = computed(() => !!modId.value && !!targetInstallId.value);

const {
  mutateAsync: startPatchMut,
  reset: resetStart,
  isLoading: starting,
  error: startError,
  data: startData,
} = useMutation({
  mutation: async () => {
    const run = await StartPatchRun(
      workspaceId.value,
      modId.value!,
      targetInstall.value?.version ?? "",
      targetInstallId.value!,
    );
    if (!run) throw new Error("Failed to start patch run");
    const preview = await PreviewPatchRun(run.id);
    return { run, preview };
  },
});

watch(startData, (d) => {
  patchFiles.value = d?.preview?.files ?? [];
});

const {
  mutateAsync: applyPatchMut,
  isLoading: applying,
  error: applyError,
} = useMutation({
  mutation: async (runId: string) => {
    await ApplyPatchRun(runId);
    for (const f of patchFiles.value) {
      if (f.decision === "accept") f.status = "applied";
    }
  },
});

const error = computed(
  () =>
    loadError.value?.message ??
    startError.value?.message ??
    applyError.value?.message ??
    "",
);
const preview = computed(() => startData.value?.preview ?? null);
const patchRun = computed(() => startData.value?.run ?? null);

/** Start a new patch run and preview. */
function startPatch(): void {
  if (!canStart.value) return;
  void startPatchMut();
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

/** Open the VS Code merge editor for a review file. */
async function reviewFile(file: PatchRunFile): Promise<void> {
  const modPath = file.modPath;
  const targetPath = file.targetPath;
  const previewPath = file.previewPath;
  if (!modPath || !targetPath || !previewPath) {
    toast.add({ title: "Missing merge paths for this file.", color: "error" });
    return;
  }
  await startMergeOverlay({
    files: [modPath, targetPath, previewPath],
    label: "Back to Patch Center",
    back: () => {
      void router.push({
        name: "patcher",
        params: { id: workspaceId.value },
        query: { tab: "patcher" },
      });
    },
    open: () => openMergeEditor({
      input1: modPath,
      input2: targetPath,
      result: previewPath,
    }),
  });
}

/** Apply accepted files to the mod. */
function applyPatch(): void {
  if (!patchRun.value) return;
  void applyPatchMut(patchRun.value.id);
}

/** Drop the run, staging, and preview state. */
async function cancelPatch(): Promise<void> {
  const id = patchRun.value?.id;
  if (id) {
    try {
      await CancelPatchRun(id);
    } catch {
      /* already gone */
    }
  }
  resetStart();
  patchFiles.value = [];
}
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-auto p-3">
    <UAlert v-if="error" color="error" variant="subtle" :description="error" />

    <UCard>
      <template #header>
        <span class="font-semibold">Patch Configuration</span>
      </template>
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <UFormField label="Mod">
          <USelect
            :model-value="modId ?? undefined"
            :items="mods.map((m) => ({ label: m.name, value: m.id }))"
            value-key="value"
            placeholder="Select mod"
            @update:model-value="(v: string) => modId = v"
          />
        </UFormField>
        <UFormField label="Target Install">
          <USelect
            :model-value="targetInstallId ?? undefined"
            :items="(installs ?? []).map((i) => ({
              label: `${i.name} (${i.version || '?'})`,
              value: i.id,
            }))"
            value-key="value"
            placeholder="Target"
            @update:model-value="(v: string) => targetInstallId = v"
          />
        </UFormField>
        <div class="flex items-end">
          <UButton
            label="Start Patch"
            icon="i-lucide-play"
            :disabled="!canStart"
            :loading="starting"
            @click="startPatch"
          />
        </div>
      </div>
    </UCard>

    <UCard v-if="preview" class="min-h-0 flex-1" :ui="{ body: 'overflow-auto' }">
      <template #header>
        <div class="flex items-center justify-between">
          <div>
            <span class="font-semibold">Preview</span>
            <span class="ml-2 text-sm text-muted">
              {{ preview.totalFiles }} files |
              {{ preview.safeCount }} safe |
              {{ preview.reviewCount }} review
            </span>
          </div>
          <div class="flex gap-2">
            <UButton
              label="Cancel"
              color="neutral"
              variant="outline"
              @click="cancelPatch"
            />
            <UButton
              label="Apply Accepted"
              icon="i-lucide-check"
              :loading="applying"
              :disabled="!patchFiles.some((f) => f.decision === 'accept')"
              @click="applyPatch"
            />
          </div>
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
            :color="row.original.status === 'safe' ? 'success'
              : row.original.status === 'review' ? 'warning' : 'neutral'"
            variant="subtle"
            size="xs"
          >
            {{ row.original.status }}
          </UBadge>
        </template>
        <template #stats-cell="{ row }">
          <template v-if="row.original.stats?.error">
            {{ row.original.stats.error }}
          </template>
          <template v-else-if="row.original.stats">
            +{{ row.original.stats.added ?? 0 }} /
            ~{{ row.original.stats.changed ?? 0 }}
            <span
              v-if="(row.original.stats.conflicts ?? 0) > 0"
              class="text-error"
            >
              / {{ row.original.stats.conflicts }} conflicts
            </span>
          </template>
        </template>
        <template #decision-cell="{ row }">
          <UBadge
            v-if="row.original.decision"
            :color="row.original.decision === 'accept' ? 'success' : 'neutral'"
            variant="subtle"
            size="xs"
          >
            {{ row.original.decision }}
          </UBadge>
        </template>
        <template #actions-cell="{ row }">
          <div class="flex gap-1">
            <UButton
              v-if="row.original.status === 'review'"
              label="Merge"
              size="xs"
              variant="outline"
              @click="reviewFile(row.original)"
            />
            <UButton label="Accept" size="xs" color="success" variant="ghost"
              :disabled="row.original.decision === 'accept'"
              @click="acceptFile(row.original)" />
            <UButton label="Skip" size="xs" color="neutral" variant="ghost"
              :disabled="row.original.decision === 'skip'"
              @click="skipFile(row.original)" />
          </div>
        </template>
      </UTable>
    </UCard>
  </div>
</template>
