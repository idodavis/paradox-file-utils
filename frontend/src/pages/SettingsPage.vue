<script setup lang="ts">
/**
 * Settings page: fonts, data reset, workspace library link.
 */
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { storeToRefs } from "pinia";
import { useMutation, useQuery } from "@pinia/colada";
import { ResetData } from "@services/settingsservice";
import {
  ListGameInstalls,
  UpdateGameInstall,
  DeleteGameInstall,
  SetInstallVersion,
  GetInstallCacheInfo,
  SetWorkspaceLocLang,
} from "@services/workspaceservice";
import { RebuildInstallSemantics } from "@services/sessionservice";
import FileSelector from "../components/FileSelector.vue";
import { useSettingsStore } from "../stores/settings";
import { LOC_LANG_ITEMS, useWorkspaceStore } from "../stores/workspace";

defineOptions({ name: "SettingsPage" });

const router = useRouter();
const settings = useSettingsStore();
const ws = useWorkspaceStore();
const toast = useToast();
const { fontScale, saving } = storeToRefs(settings);

type InstallDraft = {
  id: string;
  name: string;
  version: string;
  versionDetected: string;
  path: string;
  docsPath: string;
  scannedAt: string;
};

const drafts = ref<InstallDraft[]>([]);
const locLang = ref("english");
const resetOpen = ref(false);

const {
  data: listed,
  error: loadError,
  isPending: loadingInstalls,
  refetch,
} = useQuery({
  key: () => ["settings-installs", ws.currentGameId],
  query: async () => {
    await Promise.all([settings.load(), ws.loadActiveWorkspace()]);
    const rows = (await ListGameInstalls(ws.currentGameId)) ?? [];
    return Promise.all(
      rows.map(async (i) => ({
        id: i.id,
        name: i.name,
        version: i.version,
        versionDetected: i.versionDetected || "",
        path: i.path,
        docsPath: i.docsPath || "",
        scannedAt: (await GetInstallCacheInfo(i.id))?.scannedAt || "",
      })),
    );
  },
});

watch(listed, (rows) => {
  drafts.value = (rows ?? []).map((r) => ({ ...r }));
  locLang.value = ws.activeWorkspace?.defaultLocLang || "english";
}, { immediate: true });

/** Apply UI scale live while dragging; persistence happens on commit. */
function onScaleInput(value: number | undefined): void {
  if (value === undefined) return;
  void settings.set("ui.fontScale", String(value), false);
}

/** Toast success; optionally refetch install drafts. */
function toastOk(title: string, refresh = false): void {
  toast.add({ title, color: "success" });
  if (refresh) void refetch();
}

const { mutateAsync: saveInstall, isLoading: savingInstall } = useMutation({
  mutation: async (draft: InstallDraft) => {
    await UpdateGameInstall(draft.id, draft.path, draft.docsPath);
    await SetInstallVersion(draft.id, draft.version || "latest");
  },
  onSuccess: () => toastOk("Install updated.", true),
});

const { mutateAsync: rescanInstall, isLoading: scanning, variables: scanningId } =
  useMutation({
    mutation: async (id: string) => { await RebuildInstallSemantics(id); },
    onSuccess: () => toastOk("Install rescanned.", true),
  });

const { mutateAsync: deleteInstall } = useMutation({
  mutation: async (id: string) => { await DeleteGameInstall(id); },
  onSuccess: () => toastOk("Install deleted.", true),
});

const { mutateAsync: saveLocLang } = useMutation({
  mutation: async () => {
    const id = ws.activeWorkspaceId;
    if (!id) return;
    await SetWorkspaceLocLang(id, locLang.value);
    await ws.loadActiveWorkspace();
  },
  onSuccess: () => toastOk("Loc language saved."),
});

const { mutateAsync: resetData, isLoading: resetting } = useMutation({
  mutation: async () => {
    resetOpen.value = false;
    await ResetData();
    await refetch();
  },
  onSuccess: () => toastOk("Data reset complete."),
});

const error = computed(() => loadError.value?.message ?? "");

/** Navigate to library. */
function goToLibrary(): void {
  void router.push({ name: "library" });
}

/** Persist settings map. */
function save(): void {
  void settings.save().then(() => {
    toast.add({ title: "Settings saved.", color: "success" });
  });
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-auto p-4">
    <div class="mx-auto w-full max-w-xl">
      <UCard>
        <template #header>
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h1 class="text-lg font-semibold">Settings</h1>
              <p class="text-xs text-muted">App configuration and data management</p>
            </div>
            <UButton
              label="Library"
              icon="i-lucide-library"
              color="neutral"
              variant="outline"
              size="sm"
              @click="goToLibrary"
            />
          </div>
        </template>

        <div class="space-y-4">
          <UAlert v-if="error" color="error" variant="subtle" :description="error" />

          <UCard variant="subtle">
            <template #header>
              <span class="font-medium">Appearance</span>
            </template>
            <div class="space-y-4">
              <UFormField label="UI scale"
                :description="`App chrome at ${fontScale}% (editor font is set in the workbench)`">
                <div class="flex items-center gap-3">
                  <USlider :model-value="fontScale" :min="settings.FONT_SCALE_MIN"
                    :max="settings.FONT_SCALE_MAX" :step="5" class="flex-1"
                    @update:model-value="onScaleInput" @change="settings.save()" />
                  <span class="w-12 text-center text-sm tabular-nums">{{ fontScale }}%</span>
                </div>
              </UFormField>
            </div>
          </UCard>

          <UCard variant="subtle">
            <template #header>
              <span class="font-medium">Game Installs & Workspaces</span>
            </template>
            <div class="space-y-3">
              <p class="text-sm text-muted">
                Override install and script_docs paths for {{ ws.currentGameId.toUpperCase() }}.
              </p>
              <div
                v-for="inst in drafts"
                :key="inst.id"
                class="space-y-2 rounded border border-default p-2"
              >
                <p class="text-sm font-medium">{{ inst.name }}</p>
                <UFormField label="Version">
                  <UInput v-model="inst.version" placeholder="latest" />
                </UFormField>
                <p class="text-xs text-muted">
                  <template v-if="inst.versionDetected">
                    detected: {{ inst.versionDetected }} from
                    launcher/launcher-settings.json
                  </template>
                  <template v-else>
                    no file — default latest, pin optional
                  </template>
                </p>
                <p class="text-xs text-muted">
                  {{ inst.scannedAt ? `cache scanned ${inst.scannedAt}` : "cache not scanned" }}
                </p>
                <FileSelector v-model="inst.path" mode="folder" label="Install path"
                  dialog-title="Select game install folder" />
                <FileSelector
                  v-model="inst.docsPath"
                  mode="folder"
                  label="Docs path (empty = detected)"
                  dialog-title="Select script_docs folder"
                />
                <div class="flex flex-wrap gap-2">
                  <UButton
                    label="Save"
                    size="xs"
                    variant="outline"
                    :loading="savingInstall"
                    @click="saveInstall(inst)"
                  />
                  <UButton label="Rescan" size="xs" variant="outline"
                    :loading="scanning && scanningId === inst.id"
                    @click="rescanInstall(inst.id)" />
                  <UButton label="Delete" size="xs" color="error" variant="ghost"
                    @click="deleteInstall(inst.id)" />
                </div>
              </div>
              <UFormField v-if="ws.activeWorkspaceId" label="Default loc language" class="pt-2">
                <div class="flex gap-2">
                  <USelect v-model="locLang" :items="LOC_LANG_ITEMS" value-key="value"
                    class="flex-1" />
                  <UButton label="Save" size="xs" variant="outline" @click="saveLocLang()" />
                </div>
              </UFormField>
              <p v-if="!drafts.length" class="text-sm text-muted">
                No installs yet — add one from the workspace wizard.
              </p>
            </div>
            <template #footer>
              <UButton
                label="Open Library"
                icon="i-lucide-library"
                size="sm"
                @click="goToLibrary"
              />
            </template>
          </UCard>

          <hr class="border-default" />

          <div class="flex flex-wrap items-center justify-between gap-2">
            <UButton
              label="Reset all data"
              color="error"
              variant="outline"
              size="sm"
              :loading="resetting"
              @click="resetOpen = true"
            />
            <div class="flex gap-2">
              <UButton
                label="Reload"
                color="neutral"
                variant="outline"
                size="sm"
                :loading="loadingInstalls"
                @click="refetch()"
              />
              <UButton label="Save" size="sm" :loading="saving" @click="save" />
            </div>
          </div>
        </div>
      </UCard>
    </div>

    <UModal
      v-model:open="resetOpen"
      title="Reset all data?"
      description="This will delete workspaces, indexes, the semantic cache,
        and patch runs. Settings will be kept."
    >
      <template #footer="{ close }">
        <UButton label="Cancel" color="neutral" variant="outline" @click="close" />
        <UButton label="Reset" color="error" :loading="resetting" @click="resetData()" />
      </template>
    </UModal>
  </div>
</template>
