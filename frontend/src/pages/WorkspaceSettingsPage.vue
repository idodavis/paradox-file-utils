<script setup lang="ts">
/**
 * Per-workspace settings: section sidebar and two-column form.
 */
import { computed, ref, shallowRef, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useDebounceFn } from "@vueuse/core";
import { useSortable } from "@vueuse/integrations/useSortable";
import { useMutation, useQuery } from "@pinia/colada";
import type { NavigationMenuItem, RadioGroupItem } from "@nuxt/ui";
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
import { refreshIdeRootDecorations } from "../ide/workbenchHost";
import { originHex } from "../ide/rootDecorations";
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
const locLang = ref("english");
const rememberOpen = ref(true);
const defaultTool = ref("workspace-ide");
const stagingDir = ref("");
const gameColor = ref("");
const stagingColor = ref("");
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

/** Copy the last loaded workspace into the form (initial load and Cancel). */
function applyWorkspace(): void {
  const p = page.value;
  if (!p) return;
  const w = p.workspace;
  name.value = w.name;
  locLang.value = w.defaultLocLang || "english";
  rememberOpen.value = !w.resetIdeOnOpen;
  defaultTool.value = w.defaultTool || "workspace-ide";
  stagingDir.value = w.stagingDir;
  gameColor.value = w.gameColor ?? "";
  stagingColor.value = w.stagingColor ?? "";
  installId.value = w.installId;
  mods.value = p.mods.map((m) => ({ ...m }));
  const inst = p.installs.find((i) => i.id === w.installId);
  draftVersion.value = inst?.version || "latest";
  draftPath.value = inst?.path ?? "";
  draftDocs.value = inst?.docsPath ?? "";
}

watch(page, applyWorkspace, { immediate: true });

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
  watchElement: true,
  onEnd: () => {
    mods.value.forEach((m, i) => {
      m.sortOrder = i;
    });
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
    await UpdateWorkspace(wsId, name.value.trim(), w.installId, w.stagingDir);
    if (locLang.value !== (w.defaultLocLang || "english")) {
      await SetWorkspaceLocLang(wsId, locLang.value);
    }
    await UpdateWorkspacePrefs(
      wsId, !rememberOpen.value, defaultTool.value,
      gameColor.value, stagingColor.value,
    );
    await wsStore.refresh();
  },
  onSuccess: () => { toastOk("Overview saved."); void refetch(); },
});

const { mutateAsync: changeInstall, isLoading: changingInstall } = useMutation({
  mutation: async (nextId: string) => {
    const w = page.value?.workspace;
    if (!w) return;
    installId.value = nextId;
    await UpdateWorkspace(id.value, w.name, nextId, w.stagingDir);
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
    await UpdateWorkspace(id.value, w.name, w.installId, dir);
    if (!dir) await EnsureStagingDir(id.value);
    await wsStore.refresh();
  },
  onSuccess: () => { toastOk("Staging saved."); void refetch(); },
});

const persistMod = useDebounceFn(async (mod: WorkspaceMod) => {
  await UpdateWorkspaceMod(
    id.value, mod.id, mod.name, mod.color ?? "",
    mod.thumbnail ?? "",
  );
  await wsStore.refresh();
  const roots = ((await GetIdeRoots(id.value)) ?? []) as IdeRoot[];
  refreshIdeRootDecorations(roots);
}, 400);

async function addMod(): Promise<void> {
  const path = await pickDirectory("Select mod folder");
  if (!path) return;
  const label = path.split(/[/\\]/).pop() || "Mod";
  await AddWorkspaceMod(id.value, label, path, "");
  toastOk("Mod added.");
  await refetch();
}

async function onModCreated(path: string, thumbnail = ""): Promise<void> {
  const label = path.split(/[/\\]/).pop() || "Mod";
  await AddWorkspaceMod(id.value, label, path, thumbnail);
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

const persistColors = useDebounceFn(async () => {
  await UpdateWorkspacePrefs(
    id.value, !rememberOpen.value, defaultTool.value,
    gameColor.value, stagingColor.value,
  );
  await wsStore.refresh();
  const roots = ((await GetIdeRoots(id.value)) ?? []) as IdeRoot[];
  refreshIdeRootDecorations(roots);
}, 400);

/** Field descriptions for the settings page (filter + UFormField). */
const COPY = {
  name: "Display name in the library and title bar.",
  loc: "Language used for localization coverage, hover text, and new-mod localization files.",
  remember: "Maintain IDE tabs status across workspace switches and app closes.",
  defaultPage: "Page opened when you enter this workspace.",
  gameColor:
    "Game/vanilla chips in IDE, Conflicts, LocCoverage, and Graph. Default is teal.",
  installList:
    "Which scanned game install this workspace reads. Shared installs warn before you switch.",
  versionPin:
    "Version of the game that this install is. Your pin wins over auto-detection.",
  installPath:
    "Top level Folder of the game; needed for scanning the game and supporting most features.",
  docsPath:
    "Script_docs folder, used for certain language features.",
  addInstall:
    "Register another copy of the same game (Steam vs Paradox, version folders).",
  mods: "Generally, Last listed wins (LISO). Drag the grip. Color and thumbnail paint explorer and origin chips.",
  modColor:
    "Same color in explorer roots, conflict/loc/graph origin chips. Empty uses the palette.",
  modThumb:
    "Explorer folder icon and origin menus. Not used on OriginBadge chips.",
  staging: "Scratch folder for patch/merge output. Not a playset.",
  stagingColor:
    "Staging root in explorer and origin chips. Staging is not a playset. Empty uses amber.",
  remove: "Drops the workspace record. Mod folders on disk stay.",
} as const;

const filter = shallowRef("");

/** Case-insensitive match against group headings and field copy. */
function matchesFilter(...parts: string[]): boolean {
  const q = filter.value.trim().toLowerCase();
  if (!q) return true;
  return parts.some((p) => p.toLowerCase().includes(q));
}

const showOverview = computed(() =>
  matchesFilter(
    "Overview",
    "Name", COPY.name,
    "Default loc language", COPY.loc,
    "Remember open files", COPY.remember,
    "Default page", COPY.defaultPage,
  ),
);
const showGame = computed(() =>
  matchesFilter(
    "Game",
    "Explorer color (game)", COPY.gameColor,
    "Install list", COPY.installList,
    "Version pin", COPY.versionPin,
    "Install path", "Docs path", COPY.installPath,
    "Add install", COPY.addInstall,
    "Install name", "Version",
  ),
);
const showMods = computed(() =>
  matchesFilter("Mods", COPY.mods, "Color", COPY.modColor, "Thumbnail", COPY.modThumb),
);

/** First-wins kinds for this workspace's game (mirrors game.IsFIOS). */
function firstWinsNote(gameId: string): string {
  switch (gameId) {
    case "ck3":
      return "GUI types and templates are first-wins.";
    case "vic3":
      return "GUI types, templates, and events are first-wins.";
    case "eu5":
      return "Events are first-wins; GUI types last-wins.";
    default:
      return "GUI types and templates are first-wins.";
  }
}

const loadOrderCopy = computed(() => {
  const extra = firstWinsNote(page.value?.workspace.gameId ?? "");
  return (
    `Last listed wins for most kinds. ${extra} ` +
    "Vanilla loses to any mod. Staging is not in this list. Conflicts labels each row."
  );
});
const showStaging = computed(() =>
  matchesFilter(
    "Staging",
    "Staging directory", COPY.staging,
    "Explorer color (staging)", COPY.stagingColor,
  ),
);
const showDanger = computed(() =>
  matchesFilter("Danger", "Remove workspace", COPY.remove),
);

type SettingsSection = "overview" | "game" | "mods" | "staging" | "danger";

const section = shallowRef<SettingsSection>("overview");

const visibleSections = computed<SettingsSection[]>(() => {
  const rows: [SettingsSection, boolean][] = [
    ["overview", showOverview.value],
    ["game", showGame.value],
    ["mods", showMods.value],
    ["staging", showStaging.value],
    ["danger", showDanger.value],
  ];
  return rows.filter(([, on]) => on).map(([id]) => id);
});

watch(visibleSections, (ids) => {
  if (!ids.includes(section.value) && ids[0]) section.value = ids[0];
});

const sectionMeta: Record<SettingsSection, { label: string; icon: string }> = {
  overview: { label: "Overview", icon: "i-lucide-layout-dashboard" },
  game: { label: "Game", icon: "i-lucide-gamepad-2" },
  mods: { label: "Mods", icon: "i-lucide-puzzle" },
  staging: { label: "Staging", icon: "i-lucide-package" },
  danger: { label: "Danger", icon: "i-lucide-triangle-alert" },
};

const navItems = computed<NavigationMenuItem[]>(() =>
  visibleSections.value.map((id) => ({
    ...sectionMeta[id],
    value: id,
    active: section.value === id,
    onSelect(e: Event) {
      e.preventDefault();
      section.value = id;
    },
  })),
);

const showFooter = computed(
  () => section.value === "overview" || section.value === "staging",
);

/** Save the section that uses the sticky footer (overview / staging). */
function saveSection(): void {
  switch (section.value) {
    case "overview":
      void saveOverview();
      return;
    case "staging":
      void saveStaging();
      return;
    case "game":
    case "mods":
    case "danger":
      return;
    default: {
      const _exhaustive: never = section.value;
      void _exhaustive;
    }
  }
}

function onModColor(mod: WorkspaceMod, hex: string): void {
  mod.color = hex;
  void persistMod(mod);
}

/** Palette or custom hex for a mod card's color picker. */
function modOriginHex(mod: WorkspaceMod, idx: number): string {
  return originHex({
    kind: "mod", path: mod.path, color: mod.color, wrapIndex: idx,
  });
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
    <div class="flex min-h-0 flex-1">
      <aside class="flex w-52 shrink-0 flex-col gap-3 border-e border-default p-3">
        <UInput v-model="filter" icon="i-lucide-search" placeholder="Filter settings…" class="w-full" />
        <UNavigationMenu :items="navItems" orientation="vertical" highlight class="data-[orientation=vertical]:w-full"
          :ui="{
            link: [
              'after:absolute after:-start-1.5 after:inset-y-0.5 after:block',
              'after:w-px after:rounded-full data-[active]:after:bg-primary',
            ],
          }" />
      </aside>
      <div class="flex min-w-0 flex-1 flex-col">
        <div class="min-h-0 flex-1 overflow-auto p-6">
          <UAlert v-if="error" color="error" variant="subtle" :description="error" />
          <p v-if="loading" class="text-sm text-muted">Loading…</p>

          <div v-else-if="section === 'overview'" class="grid grid-cols-1 gap-x-6 gap-y-5 md:grid-cols-2">
            <UFormField label="Name" :description="COPY.name">
              <UInput v-model="name" class="w-full" />
            </UFormField>
            <UFormField label="Default loc language" :description="COPY.loc">
              <USelect v-model="locLang" :items="LOC_LANG_ITEMS" value-key="value" class="w-full" />
            </UFormField>
            <UFormField label="Default page" :description="COPY.defaultPage">
              <USelect v-model="defaultTool" :items="pageItems" value-key="value" class="w-full"
                :ui="{ content: 'min-w-max' }" />
            </UFormField>
            <UFormField label="Remember open files" :description="COPY.remember">
              <USwitch v-model="rememberOpen" label="Enabled" />
            </UFormField>
          </div>

          <div v-else-if="section === 'game'" class="space-y-5">
            <UAlert v-if="sharingWarn" color="warning" variant="subtle" title="Shared install"
              :description="`Also used by: ${sharingWarn}`" />
            <p v-if="selectedInstall" class="text-sm text-muted">
              {{ selectedInstall.path }}
              · pin {{ selectedInstall.version || "latest" }}
              <template v-if="selectedInstall.versionDetected">
                · detected {{ selectedInstall.versionDetected }}
              </template>
            </p>
            <div class="grid grid-cols-1 gap-x-6 gap-y-5 md:grid-cols-2">
              <UFormField label="Explorer color (game)" :description="COPY.gameColor">
                <div class="flex items-center gap-2">
                  <input type="color" class="size-8 cursor-pointer rounded border border-default"
                    :value="originHex({ kind: 'game', path: '', color: gameColor })"
                    @input="gameColor = ($event.target as HTMLInputElement).value; persistColors()">
                  <UButton v-if="gameColor" label="Reset color" size="xs" variant="ghost"
                    @click="gameColor = ''; persistColors()" />
                </div>
              </UFormField>
              <UFormField v-if="selectedInstall" label="Version pin" :description="COPY.versionPin">
                <div class="flex gap-2">
                  <UInput v-model="draftVersion" placeholder="latest" class="w-full" />
                  <UButton label="latest" size="xs" variant="outline" @click="draftVersion = 'latest'" />
                  <UButton label="Save" size="xs" variant="outline" :loading="savingVersion" @click="saveVersion()" />
                </div>
              </UFormField>
              <UFormField label="Install list" :description="COPY.installList" class="md:col-span-2">
                <URadioGroup v-if="installItems.length" :model-value="installId" :items="installItems" variant="card"
                  :disabled="changingInstall" @update:model-value="onInstallPick" />
              </UFormField>
              <template v-if="selectedInstall">
                <p class="text-xs text-muted md:col-span-2">
                  {{ page?.cacheAt[selectedInstall.id]
                    ? `cache scanned ${page.cacheAt[selectedInstall.id]}`
                    : "cache not scanned" }}
                </p>
                <FileSelector v-model="draftPath" mode="folder" label="Install path" :description="COPY.installPath"
                  dialog-title="Select game install folder"
                  placeholder="C:\Program Files (x86)\Steam\steamapps\common\GAME_NAME" />
                <FileSelector v-model="draftDocs" mode="folder" label="Docs path" :description="COPY.docsPath"
                  dialog-title="Select script_docs folder"
                  placeholder="C:\Users\idhis\Documents\Paradox Interactive\GAME_NAME\docs(or logs)" />
                <div class="md:col-span-2">
                  <UButton label="Save paths" size="xs" variant="outline" :loading="savingPaths"
                    @click="saveInstallPaths()" />
                </div>
              </template>
              <USeparator class="md:col-span-2" />
              <FileSelector v-model="newInstallPath" mode="folder" label="Install path" :description="COPY.addInstall"
                dialog-title="Select game install folder" class="md:col-span-2" />
              <UFormField label="Install name" :description="COPY.addInstall">
                <UInput v-model="newInstallName" placeholder="e.g. Steam 1.14.0" />
              </UFormField>
              <UFormField label="Version" :description="COPY.addInstall">
                <UInput v-model="newInstallVersion" placeholder="latest" />
              </UFormField>
              <p class="text-xs text-muted md:col-span-2">
                <template v-if="newDetected">detected: {{ newDetected }}</template>
                <template v-else>detected: none — default latest</template>
              </p>
              <div class="md:col-span-2">
                <UButton label="Add Install" icon="i-lucide-plus" size="sm"
                  :disabled="!newInstallPath || !newInstallName" :loading="addingInstall" @click="addInstall()" />
              </div>
            </div>
          </div>

          <div v-else-if="section === 'mods'" class="space-y-3">
            <UAlert color="neutral" variant="subtle" title="Generally, Last listed wins (LISO). Drag the grip."
              :description="loadOrderCopy" />
            <div ref="modListEl" class="space-y-2">
              <div v-for="(mod, idx) in mods" :key="mod.id" class="space-y-2 rounded border border-default p-2">
                <div class="flex items-center gap-2">
                  <UBadge :label="String((mod.sortOrder ?? idx) + 1)" color="neutral" variant="subtle" size="xs"
                    class="w-6 justify-center tabular-nums" />
                  <UButton icon="i-lucide-grip-vertical" color="neutral" variant="ghost" size="xs"
                    class="mod-handle cursor-grab" />
                  <UInput v-model="mod.name" class="flex-1" @update:model-value="persistMod(mod)" />
                  <UBadge v-if="mod.isBroken" color="error" variant="subtle" size="xs">
                    Missing
                  </UBadge>
                  <UButton icon="i-lucide-trash-2" color="error" variant="ghost" size="xs" @click="removeMod(mod)" />
                </div>
                <p class="truncate text-xs text-muted">{{ mod.path }}</p>
                <div class="grid grid-cols-1 gap-x-6 gap-y-3 md:grid-cols-2">
                  <UFormField label="Color" :description="COPY.modColor">
                    <div class="flex items-center gap-2">
                      <input type="color" class="size-8 cursor-pointer rounded border border-default"
                        :value="modOriginHex(mod, idx)"
                        @input="onModColor(mod, ($event.target as HTMLInputElement).value)">
                      <UButton v-if="mod.color" label="Reset color" size="xs" variant="ghost"
                        @click="clearModColor(mod)" />
                    </div>
                  </UFormField>
                  <UFormField label="Thumbnail" :description="COPY.modThumb">
                    <div class="flex items-center gap-2">
                      <img v-if="mod.thumbnail && wsStore.thumbUrls[mod.id]" :src="wsStore.thumbUrls[mod.id]" alt=""
                        class="size-6 rounded-sm object-cover">
                      <FileSelector :model-value="mod.thumbnail ?? ''" mode="file" label="Thumbnail"
                        dialog-title="Select thumbnail" file-filter="*.png; *.jpg; *.jpeg; *.svg"
                        class="min-w-48 flex-1" @update:model-value="onModThumb(mod, $event)" />
                      <UButton v-if="mod.thumbnail" label="Clear" size="xs" variant="ghost"
                        @click="clearModThumb(mod)" />
                    </div>
                  </UFormField>
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

          <div v-else-if="section === 'staging'" class="grid grid-cols-1 gap-x-6 gap-y-5 md:grid-cols-2">
            <FileSelector v-model="stagingDir" mode="folder" label="Staging directory" :description="COPY.staging"
              dialog-title="Select staging folder" />
            <UFormField label="Explorer color (staging)" :description="COPY.stagingColor">
              <div class="flex items-center gap-2">
                <input type="color" class="size-8 cursor-pointer rounded border border-default"
                  :value="originHex({ kind: 'staging', path: '', color: stagingColor })"
                  @input="stagingColor = ($event.target as HTMLInputElement).value; persistColors()">
                <UButton v-if="stagingColor" label="Reset color" size="xs" variant="ghost"
                  @click="stagingColor = ''; persistColors()" />
              </div>
            </UFormField>
          </div>

          <div v-else-if="section === 'danger'" class="grid grid-cols-1 gap-x-6 gap-y-5 md:grid-cols-2">
            <UFormField label="Remove workspace" :description="COPY.remove">
              <UButton label="Remove workspace" color="error" variant="outline" size="sm" @click="removeOpen = true" />
            </UFormField>
          </div>
        </div>
        <div v-if="showFooter" class="flex justify-end gap-2 border-t border-default px-6 py-3">
          <UButton label="Cancel" color="neutral" variant="outline" @click="applyWorkspace" />
          <UButton label="Save" :loading="section === 'overview' ? savingOverview : savingStaging"
            @click="saveSection" />
        </div>
      </div>
    </div>
    <RemoveWorkspaceModal v-model:open="removeOpen" :name="name" :loading="deleting" @confirm="deleteThis()" />
  </div>
</template>
