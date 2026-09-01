<script setup lang="ts">
/**
 * Per-workspace settings: overview, game install, mods, and staging.
 */
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useDebounceFn } from "@vueuse/core";
import { useSortable } from "@vueuse/integrations/useSortable";
import { useMutation, useQuery } from "@pinia/colada";
import type { RadioGroupItem } from "@nuxt/ui";
import {
  AddGameInstall,
  AddWorkspaceMod,
  CountWorkspacesUsingInstall,
  DetectGameVersion,
  EnsureStagingDir,
  GetInstallCacheInfo,
  ListGameInstalls,
  ListWorkspaceMods,
  GetWorkspace,
  RemoveWorkspaceMod,
  ReorderWorkspaceMods,
  SetInstallVersion,
  SetWorkspaceLocLang,
  UpdateGameInstall,
  UpdateWorkspace,
  UpdateWorkspaceMod,
  UpdateWorkspacePrefs,
  GetIdeRoots,
  DeleteWorkspace,
} from "@services/workspaceservice";
import type { WorkspaceMod } from "@services/models";
import FileSelector, { pickDirectory } from "../components/FileSelector.vue";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import CreateModForm from "../components/CreateModForm.vue";
import RemoveWorkspaceModal from "../components/RemoveWorkspaceModal.vue";
import { LOC_LANG_ITEMS, useWorkspaceStore } from "../stores/workspace";
import { WORKSPACE_TOOLS } from "../workspaceTools";
import {
  isWorkbenchReady,
  refreshIdeRootDecorations,
} from "../ide/workbenchHost";
import type { IdeRoot } from "../ide/fsBridge";

defineOptions({ name: "WorkspaceSettingsPage" });

const route = useRoute();
const router = useRouter();
const wsStore = useWorkspaceStore();
const toast = useToast();
const id = computed(() => String(route.params.id ?? ""));
const creatingMod = ref(false);
const removeOpen = ref(false);

const name = ref("");
const tags = ref<string[]>([]);
const locLang = ref("english");
const rememberOpen = ref(true);
const defaultTool = ref("workspace-ide");
const stagingDir = ref("");
const installId = ref("");
const mods = ref<WorkspaceMod[]>([]);
const modListEl = ref<HTMLElement | null>(null);

const newInstallPath = ref("");
const newInstallName = ref("");
const newInstallVersion = ref("latest");
const newDetected = ref("");
const draftVersion = ref("latest");
const draftPath = ref("");
const draftDocs = ref("");

const pageItems: { label: string; value: string }[] = WORKSPACE_TOOLS.map((t) => ({
  label: t.label,
  value: t.name,
}));

const {
  data: page,
  error: loadError,
  isPending: loading,
  refetch,
} = useQuery({
  key: () => ["workspace-settings", id.value],
  query: async () => {
    const wsId = id.value;
    if (!wsId) throw new Error("workspace id is required");
    wsStore.setActiveWorkspace(wsId);
    const workspace = await GetWorkspace(wsId);
    if (!workspace) throw new Error("workspace not found");
    const [modRows, installs, sharing] = await Promise.all([
      ListWorkspaceMods(wsId),
      ListGameInstalls(workspace.gameId),
      CountWorkspacesUsingInstall(workspace.installId),
    ]);
    const listed = installs ?? [];
    const cacheAt = Object.fromEntries(
      await Promise.all(
        listed.map(async (i) => {
          const info = await GetInstallCacheInfo(i.id);
          return [i.id, info?.scannedAt ?? ""] as const;
        }),
      ),
    );
    return {
      workspace,
      mods: modRows ?? [],
      installs: listed,
      sharing: sharing ?? [],
      cacheAt,
    };
  },
  enabled: () => !!id.value,
});

watch(page, (p) => {
  if (!p) return;
  const w = p.workspace;
  name.value = w.name;
  tags.value = w.tags ?? [];
  locLang.value = w.defaultLocLang || "english";
  rememberOpen.value = !w.resetIdeOnOpen;
  defaultTool.value = w.defaultTool || "workspace-ide";
  stagingDir.value = w.stagingDir;
  installId.value = w.installId;
  mods.value = p.mods.map((m) => ({ ...m, tags: m.tags ?? [] }));
  const inst = p.installs.find((i) => i.id === w.installId);
  draftVersion.value = inst?.version || "latest";
  draftPath.value = inst?.path ?? "";
  draftDocs.value = inst?.docsPath ?? "";
}, { immediate: true });

const selectedInstall = computed(() =>
  page.value?.installs.find((i) => i.id === installId.value),
);
const supportedVersion = computed(() => {
  const i = selectedInstall.value;
  if (!i) return "";
  const v = i.versionDetected || i.version || "";
  return v === "latest" ? "" : v;
});
const installItems = computed<RadioGroupItem[]>(() =>
  (page.value?.installs ?? []).map((i) => ({
    label: `${i.name} (${i.version || "latest"})`,
    description: i.path,
    value: i.id,
  })),
);
const sharingWarn = computed(() => {
  const names = page.value?.sharing ?? [];
  return names.length > 1 ? names.join(", ") : "";
});
const error = computed(() => loadError.value?.message ?? "");

useSortable(modListEl, mods, {
  handle: ".mod-handle",
  animation: 150,
  onEnd: () => {
    void ReorderWorkspaceMods(id.value, mods.value.map((m) => m.id));
  },
});

/** Switch this workspace onto another install (rebuilds the session). */
function onInstallPick(v: unknown): void {
  if (typeof v === "string" && v) void changeInstall(v);
}

function toastOk(title: string): void {
  toast.add({ title, color: "success" });
}

watch(newInstallPath, async (p) => {
  if (!p) {
    newDetected.value = "";
    return;
  }
  newDetected.value = await DetectGameVersion(p);
  if (!newInstallVersion.value || newInstallVersion.value === "latest") {
    newInstallVersion.value = newDetected.value || "latest";
  }
});

const { mutateAsync: saveOverview, isLoading: savingOverview } = useMutation({
  mutation: async () => {
    const wsId = id.value;
    const w = page.value?.workspace;
    if (!w) return;
    await UpdateWorkspace(wsId, name.value.trim(), w.installId, w.stagingDir, tags.value);
    if (locLang.value !== (w.defaultLocLang || "english")) {
      await SetWorkspaceLocLang(wsId, locLang.value);
    }
    await UpdateWorkspacePrefs(wsId, !rememberOpen.value, defaultTool.value);
    await wsStore.refresh();
  },
  onSuccess: () => { toastOk("Overview saved."); void refetch(); },
});

const { mutateAsync: changeInstall, isLoading: changingInstall } = useMutation({
  mutation: async (nextId: string) => {
    const w = page.value?.workspace;
    if (!w) return;
    installId.value = nextId;
    await UpdateWorkspace(id.value, w.name, nextId, w.stagingDir, w.tags ?? []);
    await wsStore.refresh();
  },
  onSuccess: () => { toastOk("Install updated."); void refetch(); },
});

const { mutateAsync: saveVersion, isLoading: savingVersion } = useMutation({
  mutation: async () => {
    if (!installId.value) return;
    await SetInstallVersion(installId.value, draftVersion.value || "latest");
  },
  onSuccess: () => { toastOk("Version pin saved."); void refetch(); },
});

const { mutateAsync: saveInstallPaths, isLoading: savingPaths } = useMutation({
  mutation: async () => {
    if (!installId.value) return;
    await UpdateGameInstall(installId.value, draftPath.value, draftDocs.value);
  },
  onSuccess: () => { toastOk("Install paths saved."); void refetch(); },
});

const { mutateAsync: addInstall, isLoading: addingInstall } = useMutation({
  mutation: async () => {
    const gameId = page.value?.workspace.gameId;
    if (!gameId || !newInstallPath.value || !newInstallName.value) return;
    const inst = await AddGameInstall(
      gameId,
      newInstallName.value,
      newInstallPath.value,
      newInstallVersion.value || "latest",
    );
    if (!inst) throw new Error("Failed to create install");
    newInstallPath.value = "";
    newInstallName.value = "";
    newInstallVersion.value = "latest";
    newDetected.value = "";
    await changeInstall(inst.id);
  },
});

const { mutateAsync: saveStaging, isLoading: savingStaging } = useMutation({
  mutation: async () => {
    const w = page.value?.workspace;
    if (!w) return;
    const dir = stagingDir.value.trim();
    await UpdateWorkspace(id.value, w.name, w.installId, dir, w.tags ?? []);
    if (!dir) await EnsureStagingDir(id.value);
    await wsStore.refresh();
  },
  onSuccess: () => { toastOk("Staging saved."); void refetch(); },
});

const persistMod = useDebounceFn(async (mod: WorkspaceMod) => {
  await UpdateWorkspaceMod(
    id.value, mod.id, mod.name, mod.tags ?? [], mod.color ?? "",
    mod.thumbnail ?? "",
  );
  await wsStore.refresh();
  if (isWorkbenchReady()) {
    const roots = ((await GetIdeRoots(id.value)) ?? []) as IdeRoot[];
    refreshIdeRootDecorations(roots);
  }
}, 400);

async function addMod(): Promise<void> {
  const path = await pickDirectory("Select mod folder");
  if (!path) return;
  const label = path.split(/[/\\]/).pop() || "Mod";
  await AddWorkspaceMod(id.value, label, path, [], "");
  toastOk("Mod added.");
  await refetch();
}

async function onModCreated(path: string, thumbnail = ""): Promise<void> {
  const label = path.split(/[/\\]/).pop() || "Mod";
  await AddWorkspaceMod(id.value, label, path, [], thumbnail);
  creatingMod.value = false;
  toastOk("Mod created.");
  await refetch();
}

const { mutateAsync: deleteThis, isLoading: deleting } = useMutation({
  mutation: async () => {
    await DeleteWorkspace(id.value);
    removeOpen.value = false;
    await wsStore.refresh();
    void router.push({ name: "library" });
  },
  onSuccess: () => toastOk("Workspace removed."),
});

async function removeMod(mod: WorkspaceMod): Promise<void> {
  await RemoveWorkspaceMod(id.value, mod.id);
  toastOk("Mod removed.");
  await refetch();
}

function onModColor(mod: WorkspaceMod, hex: string): void {
  mod.color = hex;
  void persistMod(mod);
}

function clearModColor(mod: WorkspaceMod): void {
  mod.color = "";
  void persistMod(mod);
}

function onModThumb(mod: WorkspaceMod, path: string): void {
  mod.thumbnail = path;
  void persistMod(mod);
}

function clearModThumb(mod: WorkspaceMod): void {
  mod.thumbnail = "";
  void persistMod(mod);
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <WorkspaceToolBar :workspace-id="id" title="Workspace Settings" />
    <div class="min-h-0 flex-1 overflow-auto p-4">
      <div class="mx-auto w-full max-w-2xl space-y-4">
        <UAlert v-if="error" color="error" variant="subtle" :description="error" />
        <p v-if="loading" class="text-sm text-muted">Loading…</p>

        <UCard title="Overview">
          <div class="space-y-3">
            <UFormField label="Name">
              <UInput v-model="name" />
            </UFormField>
            <UFormField label="Tags">
              <UInputTags v-model="tags" placeholder="Add tag" />
            </UFormField>
            <UFormField label="Default loc language">
              <USelect v-model="locLang" :items="LOC_LANG_ITEMS" value-key="value" />
            </UFormField>
            <USwitch v-model="rememberOpen" label="Remember open files"
              description="Keep IDE tabs across workspace switch and app close." />
            <UFormField label="Default page">
              <USelect v-model="defaultTool" :items="pageItems" value-key="value" class="w-40" />
            </UFormField>
          </div>
          <template #footer>
            <UButton label="Save" size="sm" :loading="savingOverview" @click="saveOverview()" />
          </template>
        </UCard>

        <UCard title="Game">
          <div class="space-y-3">
            <UAlert v-if="sharingWarn" color="warning" variant="subtle" title="Shared install"
              :description="`Also used by: ${sharingWarn}`" />
            <p v-if="selectedInstall" class="text-sm text-muted">
              {{ selectedInstall.path }}
              · pin {{ selectedInstall.version || "latest" }}
              <template v-if="selectedInstall.versionDetected">
                · detected {{ selectedInstall.versionDetected }}
              </template>
            </p>
            <URadioGroup v-if="installItems.length" :model-value="installId" :items="installItems" variant="card"
              :disabled="changingInstall" @update:model-value="onInstallPick" />
            <div v-if="selectedInstall" class="space-y-2 rounded border border-default p-2">
              <UFormField label="Version pin">
                <div class="flex gap-2">
                  <UInput v-model="draftVersion" placeholder="latest" />
                  <UButton label="latest" size="xs" variant="outline" @click="draftVersion = 'latest'" />
                  <UButton label="Save" size="xs" variant="outline" :loading="savingVersion" @click="saveVersion()" />
                </div>
              </UFormField>
              <p class="text-xs text-muted">
                {{ page?.cacheAt[selectedInstall.id]
                  ? `cache scanned ${page.cacheAt[selectedInstall.id]}`
                  : "cache not scanned" }}
              </p>
              <FileSelector v-model="draftPath" mode="folder" label="Install path"
                dialog-title="Select game install folder" />
              <FileSelector v-model="draftDocs" mode="folder" label="Docs path (empty = detected)"
                dialog-title="Select script_docs folder" />
              <UButton label="Save paths" size="xs" variant="outline" :loading="savingPaths"
                @click="saveInstallPaths()" />
            </div>
            <USeparator label="Add install" />
            <FileSelector v-model="newInstallPath" mode="folder" label="Install path"
              dialog-title="Select game install folder" />
            <UFormField label="Install name">
              <UInput v-model="newInstallName" placeholder="e.g. Steam 1.14.0" />
            </UFormField>
            <UFormField label="Version">
              <UInput v-model="newInstallVersion" placeholder="latest" />
            </UFormField>
            <p class="text-xs text-muted">
              <template v-if="newDetected">detected: {{ newDetected }}</template>
              <template v-else>detected: none — default latest</template>
            </p>
            <UButton label="Add Install" icon="i-lucide-plus" size="sm" :disabled="!newInstallPath || !newInstallName"
              :loading="addingInstall" @click="addInstall()" />
          </div>
        </UCard>

        <UCard title="Mods">
          <div class="space-y-3">
            <div ref="modListEl" class="space-y-2">
              <div v-for="mod in mods" :key="mod.id" class="space-y-2 rounded border border-default p-2">
                <div class="flex items-center gap-2">
                  <UButton icon="i-lucide-grip-vertical" color="neutral" variant="ghost" size="xs"
                    class="mod-handle cursor-grab" />
                  <UInput v-model="mod.name" class="flex-1" @update:model-value="persistMod(mod)" />
                  <UBadge v-if="mod.isBroken" color="error" variant="subtle" size="xs">
                    Missing
                  </UBadge>
                  <UButton icon="i-lucide-trash-2" color="error" variant="ghost" size="xs" @click="removeMod(mod)" />
                </div>
                <p class="truncate text-xs text-muted">{{ mod.path }}</p>
                <UInputTags :model-value="mod.tags ?? []" placeholder="Mod tags"
                  @update:model-value="(v: string[]) => { mod.tags = v; persistMod(mod); }" />
                <div class="flex items-center gap-2">
                  <input type="color" class="size-8 cursor-pointer rounded border border-default"
                    :value="mod.color || '#5B9A8B'" @input="onModColor(mod, ($event.target as HTMLInputElement).value)">
                  <UButton v-if="mod.color" label="Use palette" size="xs" variant="ghost" @click="clearModColor(mod)" />
                  <img v-if="mod.thumbnail && wsStore.thumbUrls[mod.id]" :src="wsStore.thumbUrls[mod.id]" alt=""
                    class="size-6 rounded-sm object-cover" />
                  <FileSelector :model-value="mod.thumbnail ?? ''" mode="file" label="Thumbnail"
                    dialog-title="Select thumbnail" file-filter="*.png; *.jpg; *.jpeg; *.svg" class="min-w-48 flex-1"
                    @update:model-value="onModThumb(mod, $event)" />
                  <UButton v-if="mod.thumbnail" label="Clear" size="xs" variant="ghost" @click="clearModThumb(mod)" />
                </div>
              </div>
            </div>
            <p v-if="!mods.length" class="text-sm text-muted">No mods yet.</p>
            <div class="flex flex-wrap gap-2">
              <UButton label="Add existing folder" icon="i-lucide-folder-plus" variant="outline" @click="addMod" />
              <UButton label="Create new mod" icon="i-lucide-package-plus" variant="outline"
                @click="creatingMod = !creatingMod" />
            </div>
            <CreateModForm v-if="creatingMod && page?.workspace" :game-id="page.workspace.gameId" :loc-lang="locLang"
              :supported-version="supportedVersion" @created="onModCreated" />
          </div>
        </UCard>

        <UCard title="Staging">
          <FileSelector v-model="stagingDir" mode="folder" label="Staging directory"
            dialog-title="Select staging folder" />
          <template #footer>
            <UButton label="Save" size="sm" :loading="savingStaging" @click="saveStaging()" />
          </template>
        </UCard>

        <UCard title="Remove this workspace">
          <p class="text-sm text-muted">
            Mod folders on disk are not deleted. You can attach them to another
            workspace later.
          </p>
          <template #footer>
            <UButton label="Remove workspace" color="error" variant="outline" size="sm" @click="removeOpen = true" />
          </template>
        </UCard>
      </div>
    </div>
    <RemoveWorkspaceModal v-model:open="removeOpen" :name="name" :loading="deleting" @confirm="deleteThis()" />
  </div>
</template>
