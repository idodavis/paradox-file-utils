<script setup lang="ts">
/**
 * Workspace creation wizard: game → install → mods → name → staging → create.
 */
import { computed, ref, useTemplateRef, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useMutation, useQuery } from "@pinia/colada";
import type { StepperItem } from "@nuxt/ui";
import { useSortable } from "@vueuse/integrations/useSortable";
import FileSelector, { pickDirectory } from "../components/FileSelector.vue";
import CreateModForm from "../components/CreateModForm.vue";
import { LOC_LANG_ITEMS, useWorkspaceStore } from "../stores/workspace";
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
  SetWorkspaceLocLang,
} from "@services/workspaceservice";

const router = useRouter();
const route = useRoute();
const ctx = useWorkspaceStore();
const wantsNewMod = computed(() => String(route.query.newMod ?? "") === "1");
const creatingMod = ref(false);

const step = ref(1);
const selectedGame = ref(ctx.currentGameId);
const selectedInstallId = ref<string | undefined>(undefined);
const newInstallPath = ref("");
const newInstallName = ref("");
const newInstallVersion = ref("latest");
const newDetected = ref("");
const draftVersion = ref("latest");
const defaultLocLang = ref("english");
const modEntries = ref<{ path: string; thumbnail: string }[]>([]);
const workspaceName = ref("");
const stagingDir = ref("");
const wizardModsEl = useTemplateRef<HTMLElement>("wizardModsEl");

useSortable(wizardModsEl, modEntries, {
  handle: ".mod-handle",
  animation: 150,
  watchElement: true,
});

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
    label: `${i.name} (${i.version || "latest"})${i.scannedAt ? " — cache ready" : ""}`,
    value: i.id,
  })),
);

const selectedInstall = computed(() =>
  installs.value.find((i) => i.id === selectedInstallId.value),
);
const supportedVersion = computed(() => {
  const i = selectedInstall.value;
  if (!i) return "";
  const v = i.versionDetected || i.version || "";
  return v === "latest" ? "" : v;
});

const canContinue = computed(() => {
  switch (step.value) {
    case 1: return !!selectedGame.value;
    case 2: return !!selectedInstallId.value;
    case 3: return wantsNewMod.value ? modEntries.value.length > 0 : true;
    case 4: return !!workspaceName.value.trim();
    case 5: return !!selectedInstallId.value;
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
    return {
      installs: listed ?? [],
      detected: detected ?? [],
    };
  },
  enabled: () => step.value >= 2,
});

const installs = computed(() => installPack.value?.installs ?? []);
const detectedInstalls = computed(() => installPack.value?.detected ?? []);

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
  if (path) modEntries.value.push({ path, thumbnail: "" });
}

/** Attach a newly created mod folder. */
function onModCreated(path: string, thumbnail = ""): void {
  if (!modEntries.value.some((m) => m.path === path)) {
    modEntries.value.push({ path, thumbnail });
  }
  creatingMod.value = false;
}

/** Remove a mod path. */
function removeModPath(index: number): void {
  modEntries.value.splice(index, 1);
}

/** Move the stepper; later steps stay locked until an install is selected. */
function onStep(v: number | string | undefined): void {
  if (typeof v !== "number") return;
  if (v > 2 && !selectedInstallId.value) {
    step.value = 2;
    return;
  }
  step.value = v;
}

function nextStep(): void { onStep(step.value + 1); }
function prevStep(): void { step.value--; }

const {
  mutateAsync: createWorkspace,
  isLoading: creating,
  error: createError,
} = useMutation({
  mutation: async () => {
    if (!workspaceName.value.trim()) {
      throw new Error("Workspace name is required.");
    }
    if (!selectedInstallId.value) {
      step.value = 2;
      throw new Error("Select or add a game install before creating.");
    }
    const ws = await CreateWorkspace(
      selectedGame.value,
      workspaceName.value.trim(),
      selectedInstallId.value,
    );
    if (!ws) throw new Error("Failed to create workspace");
    await SetWorkspaceLocLang(ws.id, defaultLocLang.value);
    if (stagingDir.value.trim()) {
      await UpdateWorkspace(
        ws.id,
        ws.name,
        ws.installId,
        stagingDir.value.trim(),
      );
    }
    await EnsureStagingDir(ws.id);
    for (const mod of modEntries.value) {
      const name = mod.path.split(/[/\\]/).pop() || "Mod";
      await AddWorkspaceMod(ws.id, name, mod.path, mod.thumbnail);
    }
    ctx.setActiveWorkspace(ws.id);
    await ctx.refresh();
    void router.replace({ name: "workspace-ide", params: { id: ws.id } });
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
          @update:model-value="onStep" />

        <UCard>
          <template v-if="step === 1">
            <div class="space-y-4">
              <h2 class="font-semibold">Select Game</h2>
              <URadioGroup v-model="selectedGame" :items="ctx.gameOptions" variant="card" />
            </div>
          </template>

          <template v-else-if="step === 2">
            <div class="space-y-4">
              <h2 class="font-semibold">Game Install</h2>
              <p class="text-sm text-muted">
                Select an existing install or click Add Install before continuing.
              </p>
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
                  {{ selectedInstall.scannedAt
                    ? `cache ready (${selectedInstall.scannedAt})` : "cache not scanned" }}
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
              <UAlert
                v-if="wantsNewMod"
                color="info"
                variant="subtle"
                title="Creating a new mod"
                description="Add at least one new or existing mod folder, then continue."
              />
              <p v-else class="text-sm text-muted">
                Add one or more mod folders to include in this workspace.
              </p>
              <div ref="wizardModsEl" class="space-y-2">
                <div
                  v-for="(mod, idx) in modEntries"
                  :key="mod.path"
                  class="space-y-2 rounded border border-default p-2"
                >
                  <div class="flex items-center gap-2">
                    <UButton
                      icon="i-lucide-grip-vertical"
                      color="neutral"
                      variant="ghost"
                      size="xs"
                      class="mod-handle cursor-grab"
                    />
                    <UInput :model-value="mod.path" readonly class="flex-1" />
                    <UButton
                      icon="i-lucide-trash-2"
                      color="error"
                      variant="ghost"
                      size="sm"
                      @click="removeModPath(idx)"
                    />
                  </div>
                  <div class="flex items-end gap-2">
                    <FileSelector
                      v-model="mod.thumbnail"
                      mode="file"
                      label="Thumbnail"
                      dialog-title="Select thumbnail"
                      file-filter="*.png; *.jpg; *.jpeg; *.svg"
                      class="flex-1"
                    />
                    <UButton
                      v-if="mod.thumbnail"
                      label="Clear"
                      size="xs"
                      color="neutral"
                      variant="ghost"
                      @click="mod.thumbnail = ''"
                    />
                  </div>
                </div>
              </div>
              <div class="flex flex-wrap gap-2">
                <UButton
                  label="Add existing folder"
                  icon="i-lucide-folder-plus"
                  variant="outline"
                  @click="addModPath"
                />
                <UButton
                  label="Create new mod"
                  icon="i-lucide-package-plus"
                  variant="outline"
                  @click="creatingMod = !creatingMod"
                />
              </div>
              <CreateModForm
                v-if="creatingMod"
                :game-id="selectedGame"
                :loc-lang="defaultLocLang"
                :supported-version="supportedVersion"
                @created="onModCreated"
              />
            </div>
          </template>

          <template v-else-if="step === 4">
            <div class="space-y-4">
              <h2 class="font-semibold">Workspace Details</h2>
              <UFormField label="Workspace name" required>
                <UInput v-model="workspaceName" placeholder="My CK3 Mod Project" />
              </UFormField>
              <UFormField label="Default loc language">
                <USelect v-model="defaultLocLang" :items="LOC_LANG_ITEMS" value-key="value" />
              </UFormField>
            </div>
          </template>

          <template v-else-if="step === 5">
            <div class="space-y-4">
              <h2 class="font-semibold">Staging Directory</h2>
              <UAlert
                v-if="!selectedInstallId"
                color="warning"
                variant="subtle"
                title="No game install selected"
                description="Go back to Install and select or add an install before creating."
              />
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
            @click="router.replace({ name: 'library' })"
          />
        </div>
      </div>
    </div>
  </div>
</template>
