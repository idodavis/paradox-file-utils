<script setup lang="ts">
/**
 * Mod Patcher: pick mod and target, preview, accept/skip, apply.
 */
import { computed, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useRoute, useRouter } from "vue-router";
import { useMutation, useQuery } from "@pinia/colada";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import { ListGameInstalls } from "@services/workspaceservice";
import { PatchRunFile } from "@services/models";
import {
  StartPatchRun,
  PreviewPatchRun,
  SetFileDecision,
  ApplyPatchRun,
} from "@services/patcherservice";
import { openFile } from "../ide/commands";
import { useIdeShellStore } from "../stores/ideShell";
import { useWorkspaceStore } from "../stores/workspace";

defineOptions({ name: "PatcherPage" });

const ideShell = useIdeShellStore();
const ws = useWorkspaceStore();
const route = useRoute();
const router = useRouter();

const workspaceId = computed(() => route.params.id as string);
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
  [installs, mods],
  () => {
    if (mods.value.length && !modId.value) {
      modId.value = mods.value[0]!.id;
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

/** Open preview / conflict file in the workbench for review. */
async function openMergeEditor(file: PatchRunFile): Promise<void> {
  if (!file.previewPath) return;
  ideShell.beginMergeReview(() => {
    void router.push({
      name: "patcher",
      params: { id: workspaceId.value },
    });
  });
  await openFile(file.previewPath);
}

/** Apply accepted files to the mod. */
function applyPatch(): void {
  if (!patchRun.value) return;
  void applyPatchMut(patchRun.value.id);
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <WorkspaceToolBar
      :workspace-id="workspaceId"
      title="Mod Patcher"
      active="patcher"
    />

    <UAlert v-if="error" color="error" variant="subtle" :description="error" class="m-2" />

    <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-auto p-3">
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
            <UButton
              label="Apply Accepted"
              icon="i-lucide-check"
              :loading="applying"
              :disabled="!patchFiles.some((f) => f.decision === 'accept')"
              @click="applyPatch"
            />
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
                @click="openMergeEditor(row.original)"
              />
              <UButton label="Accept" size="xs" color="success" variant="ghost"
                :disabled="row.original.decision === 'accept'" @click="acceptFile(row.original)" />
              <UButton label="Skip" size="xs" color="neutral" variant="ghost"
                :disabled="row.original.decision === 'skip'" @click="skipFile(row.original)" />
            </div>
          </template>
        </UTable>
      </UCard>
    </div>

  </div>
</template>
