<script setup lang="ts">
/**
 * Workspace creation wizard: game → install → mods → name/tags → staging → create.
 */
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import FileSelector from "../components/FileSelector.vue";
import { GAME_OPTIONS, useWorkspaceStore, type GameId } from "../stores/workspace";
import {
  ListGameInstalls,
  AddGameInstall,
  CreateWorkspace,
  AddWorkspaceMod,
  DetectGameVersion,
  EnsureStagingDir,
  UpdateWorkspace,
  FindGameInstalls,
} from "@services/workspaceservice";
import { GameInstall } from "@services/internal/repos/models";
import { DetectedInstall } from "@services/internal/game/models";
import { SelectDirectory } from "@services/fileservice";

const router = useRouter();
const ctx = useWorkspaceStore();

const step = ref(1);
const selectedGame = ref<GameId>(ctx.currentGameId);
const installs = ref<GameInstall[]>([]);
const selectedInstallId = ref<string | null>(null);
const newInstallPath = ref("");
const newInstallName = ref("");
const detectedInstalls = ref<DetectedInstall[]>([]);
const modPaths = ref<string[]>([]);
const workspaceName = ref("");
const tagsText = ref("");
const thumbnailPath = ref("");
const stagingDir = ref("");
const loading = ref(false);
const error = ref("");

const canContinue = computed(() => {
  switch (step.value) {
    case 1:
      return !!selectedGame.value;
    case 2:
      return !!selectedInstallId.value || (!!newInstallPath.value && !!newInstallName.value);
    case 3:
      return true;
    case 4:
      return !!workspaceName.value.trim();
    case 5:
      return true;
    default:
      return false;
  }
});

/** Load installs for selected game. */
async function loadInstalls(): Promise<void> {
  loading.value = true;
  try {
    installs.value = (await ListGameInstalls(selectedGame.value)) ?? [];
    if (installs.value.length) {
      selectedInstallId.value = installs.value[0].id;
    }
    detectedInstalls.value = (await FindGameInstalls(selectedGame.value)) ?? [];
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

/** Prefill add-install fields from a Steam-detected path. */
function useDetected(path: string, version: string): void {
  newInstallPath.value = path;
  if (!newInstallName.value) {
    newInstallName.value = version ? `Steam ${version}` : "Steam";
  }
}
async function addNewInstall(): Promise<void> {
  if (!newInstallPath.value || !newInstallName.value) return;
  loading.value = true;
  error.value = "";
  try {
    await DetectGameVersion(newInstallPath.value);
    const install = await AddGameInstall(selectedGame.value, newInstallName.value, newInstallPath.value);
    if (!install) throw new Error("Failed to create install");
    installs.value.push(install);
    selectedInstallId.value = install.id;
    newInstallPath.value = "";
    newInstallName.value = "";
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

/** Add a mod path entry. */
async function addModPath(): Promise<void> {
  const path = await SelectDirectory("Select mod folder");
  if (path) modPaths.value.push(path);
}

/** Remove a mod path. */
function removeModPath(index: number): void {
  modPaths.value.splice(index, 1);
}

/** Navigate steps. */
function nextStep(): void {
  if (step.value === 1) {
    void loadInstalls();
  }
  step.value++;
}

function prevStep(): void {
  step.value--;
}

/** Create workspace with all settings. */
async function createWorkspace(): Promise<void> {
  if (!workspaceName.value.trim() || !selectedInstallId.value) return;
  loading.value = true;
  error.value = "";
  try {
    const tagsArray = tagsText.value
      .split(",")
      .map((t) => t.trim())
      .filter(Boolean);
    const ws = await CreateWorkspace(
      selectedGame.value,
      workspaceName.value.trim(),
      selectedInstallId.value,
      JSON.stringify(tagsArray),
      thumbnailPath.value,
    );
    if (!ws) throw new Error("Failed to create workspace");
    if (stagingDir.value.trim()) {
      await UpdateWorkspace(
        ws.id,
        ws.name,
        ws.installId,
        stagingDir.value.trim(),
        ws.tags,
        ws.thumbnailPath,
      );
    }
    await EnsureStagingDir(ws.id);
    for (const modPath of modPaths.value) {
      const name = modPath.split(/[/\\]/).pop() || "Mod";
      await AddWorkspaceMod(ws.id, name, modPath);
    }
    ctx.setActiveWorkspace(ws.id);
    await ctx.refresh();
    void router.push({ name: "workspace-ide", params: { id: ws.id } });
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <!-- min-h-full + items-center so short wizards sit in the middle; overflow on the outer for tall steps -->
  <div class="h-full w-full min-h-0 overflow-y-auto">
    <div class="flex min-h-full w-full items-center justify-center p-4">
      <div class="w-full max-w-2xl">
      <div class="mb-6 text-center">
        <h1 class="text-xl font-bold">Create Workspace</h1>
        <p class="text-sm text-muted">Step {{ step }} of 5</p>
      </div>

      <UAlert v-if="error" color="error" variant="subtle" :description="error" class="mb-4" />

      <UCard>
        <template v-if="step === 1">
          <div class="space-y-4">
            <h2 class="font-semibold">Select Game</h2>
            <div class="flex flex-col gap-2">
              <label
                v-for="opt in GAME_OPTIONS"
                :key="opt.value"
                class="flex cursor-pointer items-center gap-2 rounded border border-default p-2 hover:bg-muted"
                :class="{ 'border-primary bg-primary/10': selectedGame === opt.value }"
              >
                <input
                  type="radio"
                  :value="opt.value"
                  :checked="selectedGame === opt.value"
                  class="accent-primary"
                  @change="selectedGame = opt.value"
                />
                <span>{{ opt.label }}</span>
              </label>
            </div>
          </div>
        </template>

        <template v-else-if="step === 2">
          <div class="space-y-4">
            <h2 class="font-semibold">Game Install</h2>
            <div v-if="installs.length" class="flex flex-col gap-2">
              <label
                v-for="inst in installs"
                :key="inst.id"
                class="flex cursor-pointer items-center gap-2 rounded border border-default p-2 hover:bg-muted"
                :class="{ 'border-primary bg-primary/10': selectedInstallId === inst.id }"
              >
                <input
                  type="radio"
                  :value="inst.id"
                  :checked="selectedInstallId === inst.id"
                  class="accent-primary"
                  @change="selectedInstallId = inst.id"
                />
                <span>{{ inst.name }} ({{ inst.version || 'unknown' }})</span>
              </label>
            </div>
            <p v-else class="text-sm text-muted">No installs configured yet.</p>
            <div v-if="detectedInstalls.length" class="space-y-2">
              <p class="text-sm font-medium">Found on this machine</p>
              <div
                v-for="d in detectedInstalls"
                :key="d.path"
                class="flex items-center justify-between gap-2 rounded border border-default p-2"
              >
                <span class="text-sm">
                  Found {{ selectedGame }} at {{ d.path }}
                  <span v-if="d.version" class="text-muted">({{ d.version }})</span>
                </span>
                <UButton
                  label="Use"
                  size="xs"
                  variant="outline"
                  @click="useDetected(d.path, d.version)"
                />
              </div>
            </div>
            <div class="flex items-center gap-2 py-2">
              <hr class="flex-1 border-default" />
              <span class="text-xs text-muted">Or add new</span>
              <hr class="flex-1 border-default" />
            </div>
            <div class="space-y-2">
              <FileSelector
                v-model="newInstallPath"
                mode="folder"
                label="Install path"
                dialog-title="Select game install folder"
              />
              <UFormField label="Install name">
                <UInput v-model="newInstallName" placeholder="e.g. Steam 1.14.0" />
              </UFormField>
              <UButton
                label="Add Install"
                icon="i-lucide-plus"
                size="sm"
                :disabled="!newInstallPath || !newInstallName"
                :loading="loading"
                @click="addNewInstall"
              />
            </div>
          </div>
        </template>

        <template v-else-if="step === 3">
          <div class="space-y-4">
            <h2 class="font-semibold">Mod Folders</h2>
            <p class="text-sm text-muted">Add one or more mod folders to include in this workspace.</p>
            <div v-for="(path, idx) in modPaths" :key="idx" class="flex items-center gap-2">
              <UInput :model-value="path" readonly class="flex-1" />
              <UButton icon="i-lucide-trash-2" color="error" variant="ghost" size="sm" @click="removeModPath(idx)" />
            </div>
            <UButton label="Add Mod Folder" icon="i-lucide-folder-plus" variant="outline" @click="addModPath" />
          </div>
        </template>

        <template v-else-if="step === 4">
          <div class="space-y-4">
            <h2 class="font-semibold">Workspace Details</h2>
            <UFormField label="Workspace name" required>
              <UInput v-model="workspaceName" placeholder="My CK3 Mod Project" />
            </UFormField>
            <UFormField label="Tags (comma-separated)">
              <UInput v-model="tagsText" placeholder="wip, balance, events" />
            </UFormField>
            <FileSelector
              v-model="thumbnailPath"
              mode="file"
              label="Thumbnail (optional)"
              dialog-title="Select thumbnail image"
              file-filter="*.png; *.jpg; *.jpeg"
            />
          </div>
        </template>

        <template v-else-if="step === 5">
          <div class="space-y-4">
            <h2 class="font-semibold">Staging Directory</h2>
            <p class="text-sm text-muted">
              Where merged/patched files are staged. Leave empty for the default location.
            </p>
            <FileSelector v-model="stagingDir" mode="folder" label="Staging dir" dialog-title="Select staging folder" />
          </div>
        </template>

        <template #footer>
          <div class="flex justify-between">
            <UButton v-if="step > 1" label="Back" variant="outline" @click="prevStep" />
            <div v-else />
            <UButton
              v-if="step < 5"
              label="Continue"
              :disabled="!canContinue"
              @click="nextStep"
            />
            <UButton
              v-else
              label="Create Workspace"
              :disabled="!canContinue"
              :loading="loading"
              @click="createWorkspace"
            />
          </div>
        </template>
      </UCard>

      <div class="mt-4 flex justify-center">
        <UButton
          label="Cancel"
          variant="ghost"
          color="neutral"
          @click="router.push({ name: 'library' })"
        />
      </div>
      </div>
    </div>
  </div>
</template>
