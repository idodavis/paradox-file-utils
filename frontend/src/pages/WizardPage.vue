<script setup lang="ts">
/**
 * Workspace creation wizard: game → install → mods → name/tags → staging → create.
 */
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useMutation, useQuery } from "@pinia/colada";
import type { StepperItem } from "@nuxt/ui";
import FileSelector from "../components/FileSelector.vue";
import { GAME_OPTIONS, LOC_LANG_ITEMS, useWorkspaceStore, type GameId } from "../stores/workspace";
import {
  ListGameInstalls,
  AddGameInstall,
  CreateWorkspace,
  AddWorkspaceMod,
  DetectGameVersion,
  EnsureStagingDir,
  UpdateWorkspace,
  FindGameInstalls,
  DeleteGameInstall,
  SetInstallVersion,
  GetInstallCacheInfo,
  SetWorkspaceLocLang,
} from "@services/workspaceservice";
import { pickDirectory } from "../composables/nativeDialog";

const router = useRouter();
const ctx = useWorkspaceStore();

const step = ref(1);
const selectedGame = ref<GameId>(ctx.currentGameId);
const selectedInstallId = ref<string | undefined>(undefined);
const newInstallPath = ref("");
const newInstallName = ref("");
const newInstallVersion = ref("latest");
const newDetected = ref("");
const draftVersion = ref("latest");
const defaultLocLang = ref("english");
const modPaths = ref<string[]>([]);
const workspaceName = ref("");
const tagsText = ref("");
const stagingDir = ref("");

/** Wizard steps; `value` matches the 1-based `step` ref. */
const stepItems: StepperItem[] = [
  { title: "Game", icon: "i-lucide-gamepad-2", value: 1 },
  { title: "Install", icon: "i-lucide-folder", value: 2 },
  { title: "Mods", icon: "i-lucide-package", value: 3 },
  { title: "Details", icon: "i-lucide-file-text", value: 4 },
  { title: "Staging", icon: "i-lucide-folder-output", value: 5 },
];

/** Install options for the radio group. */
const installItems = computed(() =>
  installs.value.map((i) => ({
    label: `${i.name} (${i.version || "latest"})${cacheAt.value[i.id] ? " — cache ready" : ""}`,
    value: i.id,
  })),
);

const selectedInstall = computed(() =>
  installs.value.find((i) => i.id === selectedInstallId.value),
);

const canContinue = computed(() => {
  switch (step.value) {
    case 1: return !!selectedGame.value;
    case 2:
      return !!selectedInstallId.value
        || (!!newInstallPath.value && !!newInstallName.value);
    case 3:
    case 5: return true;
    case 4: return !!workspaceName.value.trim();
    default: return false;
  }
});

/** Load installs for selected game. */
const {
  data: installPack,
  isPending: installsPending,
  error: installError,
  refetch: refetchInstalls,
} = useQuery({
  key: () => ["wizard-installs", selectedGame.value],
  query: async () => {
    const [listed, detected] = await Promise.all([
      ListGameInstalls(selectedGame.value),
      FindGameInstalls(selectedGame.value),
    ]);
    const listedInstalls = listed ?? [];
    const infos = await Promise.all(
      listedInstalls.map(async (i) => {
        const info = await GetInstallCacheInfo(i.id);
        return [i.id, info?.scannedAt ?? ""] as const;
      }),
    );
    return {
      installs: listedInstalls,
      detected: detected ?? [],
      cacheAt: Object.fromEntries(infos),
    };
  },
  enabled: () => step.value === 2,
});

const installs = computed(() => installPack.value?.installs ?? []);
const detectedInstalls = computed(() => installPack.value?.detected ?? []);
const cacheAt = computed(() => installPack.value?.cacheAt ?? {});

watch(installs, (list) => {
  const keep = selectedInstallId.value;
  const next = list.find((i) => i.id === keep) ?? list[0];
  selectedInstallId.value = next?.id;
  draftVersion.value = next?.version || "latest";
});

/** Prefill add-install fields from a Steam-detected path. */
function useDetected(path: string, version: string): void {
  newInstallPath.value = path;
  newInstallVersion.value = version || "latest";
  newDetected.value = version;
  if (!newInstallName.value) {
    newInstallName.value = version ? `Steam ${version}` : "Steam";
  }
}

watch(newInstallPath, async (p) => {
  if (!p) {
    newDetected.value = "";
    return;
  }
  const d = await DetectGameVersion(p);
  newDetected.value = d;
  if (!newInstallVersion.value || newInstallVersion.value === "latest") {
    newInstallVersion.value = d || "latest";
  }
});

watch(selectedInstallId, (id) => {
  const i = installs.value.find((x) => x.id === id);
  draftVersion.value = i?.version || "latest";
});

const {
  mutateAsync: addNewInstall,
  isLoading: adding,
  error: addError,
} = useMutation({
  mutation: async () => {
    if (!newInstallPath.value || !newInstallName.value) return;
    const install = await AddGameInstall(
      selectedGame.value,
      newInstallName.value,
      newInstallPath.value,
      newInstallVersion.value || "latest",
    );
    if (!install) throw new Error("Failed to create install");
    selectedInstallId.value = install.id;
    newInstallPath.value = "";
    newInstallName.value = "";
    newInstallVersion.value = "latest";
    newDetected.value = "";
    await refetchInstalls();
  },
});

const {
  mutateAsync: saveSelectedVersion,
  isLoading: savingVersion,
  error: saveVerError,
} = useMutation({
  mutation: async () => {
    if (!selectedInstallId.value) return;
    await SetInstallVersion(selectedInstallId.value, draftVersion.value || "latest");
    await refetchInstalls();
  },
});

const {
  mutateAsync: deleteSelectedInstall,
  isLoading: deleting,
  error: deleteError,
} = useMutation({
  mutation: async () => {
    if (!selectedInstallId.value) return;
    await DeleteGameInstall(selectedInstallId.value);
    selectedInstallId.value = undefined;
    await refetchInstalls();
  },
});

/** Add a mod path entry. */
async function addModPath(): Promise<void> {
  const path = await pickDirectory("Select mod folder");
  if (path) modPaths.value.push(path);
}

/** Remove a mod path. */
function removeModPath(index: number): void {
  modPaths.value.splice(index, 1);
}

function nextStep(): void { step.value++; }
function prevStep(): void { step.value--; }

const {
  mutateAsync: createWorkspace,
  isLoading: creating,
  error: createError,
} = useMutation({
  mutation: async () => {
    if (!workspaceName.value.trim() || !selectedInstallId.value) return;
    const tagsArray = tagsText.value
      .split(",")
      .map((t) => t.trim())
      .filter(Boolean);
    const ws = await CreateWorkspace(
      selectedGame.value,
      workspaceName.value.trim(),
      selectedInstallId.value,
      tagsArray,
    );
    if (!ws) throw new Error("Failed to create workspace");
    await SetWorkspaceLocLang(ws.id, defaultLocLang.value);
    if (stagingDir.value.trim()) {
      await UpdateWorkspace(
        ws.id,
        ws.name,
        ws.installId,
        stagingDir.value.trim(),
        ws.tags ?? [],
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
  },
});

const busy = computed(
  () =>
    installsPending.value ||
    adding.value ||
    savingVersion.value ||
    deleting.value ||
    creating.value,
);
const error = computed(
  () =>
    installError.value?.message ??
    addError.value?.message ??
    saveVerError.value?.message ??
    deleteError.value?.message ??
    createError.value?.message ??
    "",
);
</script>

<template>
  <div class="h-full w-full min-h-0 overflow-y-auto">
    <div class="flex min-h-full w-full items-center justify-center p-4">
      <div class="w-full max-w-2xl">
        <h1 class="mb-6 text-center text-xl font-bold">Create Workspace</h1>
        <UAlert v-if="error" color="error" variant="subtle" :description="error" class="mb-4" />
        <UStepper :model-value="step" :items="stepItems" class="mb-6"
          @update:model-value="(v) => { if (typeof v === 'number') step = v; }" />

        <UCard>
          <template v-if="step === 1">
            <div class="space-y-4">
              <h2 class="font-semibold">Select Game</h2>
              <URadioGroup v-model="selectedGame" :items="GAME_OPTIONS" variant="card" />
            </div>
          </template>

          <template v-else-if="step === 2">
            <div class="space-y-4">
              <h2 class="font-semibold">Game Install</h2>
              <URadioGroup
                v-if="installs.length"
                v-model="selectedInstallId"
                :items="installItems"
                variant="card"
              />
              <p v-else class="text-sm text-muted">No installs configured yet.</p>
              <div v-if="selectedInstall" class="space-y-2 rounded border border-default p-2">
                <p class="truncate text-xs text-muted">{{ selectedInstall.path }}</p>
                <UFormField label="Version">
                  <div class="flex gap-2">
                    <UInput v-model="draftVersion" placeholder="latest" />
                    <UButton
                      label="latest"
                      size="xs"
                      variant="outline"
                      @click="draftVersion = 'latest'"
                    />
                    <UButton
                      label="Save"
                      size="xs"
                      variant="outline"
                      :loading="busy"
                      @click="saveSelectedVersion()"
                    />
                  </div>
                </UFormField>
                <p class="text-xs text-muted">
                  <template v-if="selectedInstall.versionDetected">
                    detected: {{ selectedInstall.versionDetected }} from
                    launcher/launcher-settings.json
                  </template>
                  <template v-else>no file — default latest, pin optional</template>
                </p>
                <p class="text-xs text-muted">
                  {{ cacheAt[selectedInstall.id]
                    ? `cache ready (${cacheAt[selectedInstall.id]})` : "cache not scanned" }}
                </p>
                <UButton label="Delete" size="xs" color="error" variant="ghost" :loading="busy"
                  @click="deleteSelectedInstall()" />
              </div>
              <div v-if="detectedInstalls.length" class="space-y-2">
                <p class="text-sm font-medium">Found on this machine</p>
                <div v-for="d in detectedInstalls" :key="d.path"
                  class="flex items-center justify-between gap-2 rounded border border-default p-2">
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
              <USeparator label="Or add new" />
              <div class="space-y-2">
                <FileSelector v-model="newInstallPath" mode="folder" label="Install path"
                  dialog-title="Select game install folder" />
                <UFormField label="Install name">
                  <UInput v-model="newInstallName" placeholder="e.g. Steam 1.14.0" />
                </UFormField>
                <UFormField label="Version">
                  <UInput v-model="newInstallVersion" placeholder="latest" />
                </UFormField>
                <p class="text-xs text-muted">
                  <template v-if="newDetected">
                    detected: {{ newDetected }} from launcher/launcher-settings.json
                  </template>
                  <template v-else>
                    detected: none — default latest, pin optional
                  </template>
                </p>
                <UButton
                  label="Add Install"
                  icon="i-lucide-plus"
                  size="sm"
                  :disabled="!newInstallPath || !newInstallName"
                  :loading="busy"
                  @click="addNewInstall()"
                />
              </div>
            </div>
          </template>

          <template v-else-if="step === 3">
            <div class="space-y-4">
              <h2 class="font-semibold">Mod Folders</h2>
              <p class="text-sm text-muted">
                Add one or more mod folders to include in this workspace.
              </p>
              <div v-for="(path, idx) in modPaths" :key="idx" class="flex items-center gap-2">
                <UInput :model-value="path" readonly class="flex-1" />
                <UButton
                  icon="i-lucide-trash-2"
                  color="error"
                  variant="ghost"
                  size="sm"
                  @click="removeModPath(idx)"
                />
              </div>
              <UButton
                label="Add Mod Folder"
                icon="i-lucide-folder-plus"
                variant="outline"
                @click="addModPath"
              />
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
              <UFormField label="Default loc language">
                <USelect v-model="defaultLocLang" :items="LOC_LANG_ITEMS" value-key="value" />
              </UFormField>
            </div>
          </template>

          <template v-else-if="step === 5">
            <div class="space-y-4">
              <h2 class="font-semibold">Staging Directory</h2>
              <p class="text-sm text-muted">
                Where merged/patched files are staged. Leave empty for the default location.
              </p>
              <FileSelector v-model="stagingDir" mode="folder" label="Staging dir"
                dialog-title="Select staging folder" />
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
                :loading="busy"
                @click="createWorkspace()"
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
