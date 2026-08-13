<script setup lang="ts">
/**
 * Workspace IDE: multi-root tree, editable tabs, loc tools, semantics rebuild.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { CodeViewItem } from "@pierre/diffs";
import { parseDiffFromFile } from "@pierre/diffs";
import { Events } from "@wailsio/runtime";
import SplitPane from "../components/SplitPane.vue";
import EditorView from "../components/EditorView.vue";
import GuiPreviewPanel from "../components/GuiPreviewPanel.vue";
import { useWorkspaceContext } from "../composables/workspaceContext";
import { EDITOR_THEME_OPTIONS, setEditorTheme, editorTheme } from "../composables/editorTheme";
import { bumpTreeFontSize, treeFontSize } from "../composables/treeFontSize";
import { langForPath } from "../composables/langForPath";
import {
  GetWorkspace,
  ListWorkspaceMods,
  GetScriptRoot,
  EnsureStagingDir,
} from "@services/workspaceservice";
import { Workspace, WorkspaceMod } from "@services/internal/repos/models";
import { ListDirectory, ReadFileContent, WriteFileContent } from "@services/fileservice";
import { DirEntry, LocDiagnostic, GuiNode, AutoLocResult, SemanticsStatus } from "@services/models";
import { LintMissingLoc, AutoLocalize } from "@services/locservice";
import {
  RebuildWorkspaceSemantics,
  CancelSemantics,
  GetSemanticsStatus,
} from "@services/semanticsservice";
import { PreviewGui } from "@services/guiservice";
import { ReindexWorkspace, CancelIndex, CountIndexObjects } from "@services/indexerservice";

/** One open editor buffer (path + contents + dirty tracking). */
type EditorTab = {
  path: string;
  name: string;
  contents: string;
  savedContents: string;
  dirty: boolean;
};

type FileRoot = { label: string; path: string; type: "game" | "mod" | "staging" };
type TreeRow = { entry: DirEntry; depth: number; pathKey: string };

const route = useRoute();
const router = useRouter();
const ctx = useWorkspaceContext();

const workspace = ref<Workspace | null>(null);
const mods = ref<WorkspaceMod[]>([]);
const loading = ref(false);
const busy = ref("");
const progressLabel = ref("");
const progressPercent = ref(0);
const jobActive = ref(false);
const error = ref("");
const gameScriptRoot = ref("");
const compareMode = ref(false);
const showLocPanel = ref(false);
const locDiags = ref<LocDiagnostic[]>([]);
const guiRoots = ref<GuiNode[]>([]);
const selectedGuiName = ref<string | null>(null);
const semantics = ref<SemanticsStatus | null>(null);
const hadIndex = ref(false);
const autoLocOpen = ref(false);
const autoLocResult = ref<AutoLocResult | null>(null);

const fileRoots = ref<FileRoot[]>([]);
const expandedRoots = ref<Set<string>>(new Set());
const rootEntries = ref<Record<string, DirEntry[]>>({});
const childrenByPath = ref<Record<string, DirEntry[]>>({});
const expandedDirs = ref<Set<string>>(new Set());

const tabs = ref<EditorTab[]>([]);
const activeTabPath = ref<string | null>(null);
const compareFilePath = ref<string | null>(null);
const compareContent = ref("");

let loadGen = 0;
let offIndex: (() => void) | null = null;
let offSemantics: (() => void) | null = null;

const workspaceId = computed(() => String(route.params.id ?? ""));
const activeTab = computed(() => tabs.value.find((t) => t.path === activeTabPath.value) ?? null);
const isGuiFile = computed(() => (activeTabPath.value ?? "").toLowerCase().endsWith(".gui"));
const canSave = computed(() => !!activeTab.value?.dirty && !compareMode.value);
const indexVerb = computed(() => (hadIndex.value ? "Reindexing" : "Indexing"));

const editorItems = computed<CodeViewItem[]>(() => {
  const tab = activeTab.value;
  if (!tab) return [];
  const lang = langForPath(tab.path);
  if (compareMode.value && compareFilePath.value && compareContent.value) {
    return [
      {
        id: `diff:${tab.path}:${compareFilePath.value}`,
        type: "diff",
        fileDiff: parseDiffFromFile(
          { name: tab.path, contents: tab.contents, lang },
          { name: compareFilePath.value, contents: compareContent.value, lang },
        ),
        version: tab.contents.length + compareContent.value.length,
      },
    ];
  }
  return [
    {
      id: `file:${tab.path}`,
      type: "file",
      file: {
        name: tab.path,
        contents: tab.contents,
        lang,
        cacheKey: tab.path,
      },
      version: tab.contents.length + (tab.dirty ? 1 : 0),
      edit: !compareMode.value,
    },
  ];
});

/** Soft-cancel messages are not user-facing failures. */
function isCancelled(msg: string): boolean {
  return /cancelled/i.test(msg);
}

/** Flatten lazy-expanded directory entries for one root. */
function flattenEntries(entries: DirEntry[], depth: number): TreeRow[] {
  const rows: TreeRow[] = [];
  for (const entry of entries) {
    const pathKey = entry.fullPath;
    rows.push({ entry, depth, pathKey });
    if (entry.isDir && expandedDirs.value.has(pathKey)) {
      const kids = childrenByPath.value[pathKey] ?? [];
      rows.push(...flattenEntries(kids, depth + 1));
    }
  }
  return rows;
}

/** Rows for a specific expanded root. */
function rowsForRoot(rootPath: string): TreeRow[] {
  return flattenEntries(rootEntries.value[rootPath] ?? [], 0);
}

async function loadWorkspace(): Promise<void> {
  const id = workspaceId.value;
  if (!id) return;
  const gen = ++loadGen;
  loading.value = true;
  error.value = "";
  try {
    const ws = await GetWorkspace(id);
    if (gen !== loadGen) return;
    workspace.value = ws;
    mods.value = (await ListWorkspaceMods(id)) ?? [];
    if (gen !== loadGen) return;
    if (ws?.installId) {
      gameScriptRoot.value = await GetScriptRoot(ws.installId);
    } else {
      gameScriptRoot.value = "";
    }
    if (gen !== loadGen) return;
    try {
      const staging = await EnsureStagingDir(id);
      if (ws) ws.stagingDir = staging;
    } catch {
      /* staging optional */
    }
    const [sem, count] = await Promise.all([
      GetSemanticsStatus(id).catch(() => null),
      CountIndexObjects(id).catch(() => 0),
    ]);
    if (gen !== loadGen) return;
    semantics.value = sem;
    hadIndex.value = count > 0;
    buildRoots();
  } catch (e) {
    if (gen !== loadGen) return;
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    if (gen === loadGen) loading.value = false;
  }
}

function buildRoots(): void {
  const roots: FileRoot[] = [];
  if (gameScriptRoot.value) {
    roots.push({ label: "Game", path: gameScriptRoot.value, type: "game" });
  }
  for (const mod of mods.value) {
    roots.push({ label: mod.name, path: mod.path, type: "mod" });
  }
  const staging = workspace.value?.stagingDir;
  if (staging) {
    roots.push({ label: "Staging", path: staging, type: "staging" });
  }
  fileRoots.value = roots;
}

/** Toggle one root without collapsing siblings. */
async function toggleRoot(root: FileRoot): Promise<void> {
  const next = new Set(expandedRoots.value);
  if (next.has(root.path)) {
    next.delete(root.path);
    expandedRoots.value = next;
    return;
  }
  if (!rootEntries.value[root.path]) {
    loading.value = true;
    try {
      rootEntries.value = {
        ...rootEntries.value,
        [root.path]: (await ListDirectory(root.path)) ?? [],
      };
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      return;
    } finally {
      loading.value = false;
    }
  }
  next.add(root.path);
  expandedRoots.value = next;
}

async function toggleDir(entry: DirEntry): Promise<void> {
  const key = entry.fullPath;
  const next = new Set(expandedDirs.value);
  if (next.has(key)) {
    next.delete(key);
    expandedDirs.value = next;
    return;
  }
  if (!childrenByPath.value[key]) {
    try {
      childrenByPath.value = {
        ...childrenByPath.value,
        [key]: (await ListDirectory(entry.fullPath)) ?? [],
      };
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
      return;
    }
  }
  next.add(key);
  expandedDirs.value = next;
}

async function openFile(path: string): Promise<void> {
  const existing = tabs.value.find((t) => t.path === path);
  if (existing) {
    activeTabPath.value = path;
  } else {
    const contents = await ReadFileContent(path);
    const name = path.split(/[/\\]/).pop() || path;
    tabs.value = [
      ...tabs.value,
      { path, name, contents, savedContents: contents, dirty: false },
    ];
    activeTabPath.value = path;
  }
  selectedGuiName.value = null;
  if (path.toLowerCase().endsWith(".gui")) {
    const preview = await PreviewGui(path);
    guiRoots.value = preview?.roots ?? [];
  } else {
    guiRoots.value = [];
  }
}

async function selectFile(entry: DirEntry): Promise<void> {
  if (entry.isDir) {
    await toggleDir(entry);
    return;
  }
  const fullPath = entry.fullPath;
  if (compareMode.value && activeTabPath.value) {
    compareFilePath.value = fullPath;
    compareContent.value = await ReadFileContent(fullPath);
    return;
  }
  compareFilePath.value = null;
  compareContent.value = "";
  await openFile(fullPath);
}

function closeTab(path: string): void {
  const idx = tabs.value.findIndex((t) => t.path === path);
  if (idx < 0) return;
  const next = tabs.value.filter((t) => t.path !== path);
  tabs.value = next;
  if (activeTabPath.value === path) {
    activeTabPath.value = next[Math.max(0, idx - 1)]?.path ?? null;
  }
}

function onItemEdit(payload: { id: string; contents: string }): void {
  const path = payload.id.replace(/^file:/, "");
  tabs.value = tabs.value.map((t) =>
    t.path === path
      ? { ...t, contents: payload.contents, dirty: payload.contents !== t.savedContents }
      : t,
  );
}

async function saveActive(): Promise<void> {
  const tab = activeTab.value;
  if (!tab?.dirty) return;
  busy.value = "Saving…";
  error.value = "";
  try {
    await WriteFileContent(tab.path, tab.contents);
    tabs.value = tabs.value.map((t) =>
      t.path === tab.path
        ? { ...t, savedContents: t.contents, dirty: false }
        : t,
    );
    busy.value = "";
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
    busy.value = "";
  }
}

function onGlobalKey(ev: KeyboardEvent): void {
  if ((ev.ctrlKey || ev.metaKey) && ev.key.toLowerCase() === "s") {
    ev.preventDefault();
    void saveActive();
  }
}

async function runLocLint(): Promise<void> {
  if (!workspaceId.value) return;
  busy.value = "Linting localization…";
  error.value = "";
  try {
    locDiags.value = (await LintMissingLoc(workspaceId.value, "l_english")) ?? [];
    showLocPanel.value = true;
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    busy.value = "";
  }
}

async function runAutoLoc(): Promise<void> {
  if (!workspaceId.value) return;
  busy.value = "Auto-localizing…";
  error.value = "";
  try {
    const result = await AutoLocalize(workspaceId.value, "l_english", "l_english", "");
    autoLocResult.value = result;
    autoLocOpen.value = true;
    await runLocLint();
    busy.value = "";
    if (result?.path) {
      await openFile(result.path);
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
    busy.value = "";
  }
}

async function rebuildSemantics(): Promise<void> {
  if (!workspaceId.value) return;
  jobActive.value = true;
  busy.value = "Rebuilding semantics…";
  progressLabel.value = "";
  progressPercent.value = 0;
  error.value = "";
  try {
    const cache = await RebuildWorkspaceSemantics(workspaceId.value);
    if (!jobActive.value) return;
    // Soft cancel returns null cache with no error.
    if (!cache) {
      busy.value = "";
      progressLabel.value = "";
      progressPercent.value = 0;
      return;
    }
    busy.value = `${indexVerb.value}…`;
    await ReindexWorkspace(workspaceId.value);
    if (!jobActive.value) return;
    semantics.value = await GetSemanticsStatus(workspaceId.value);
    hadIndex.value = true;
    busy.value = "Semantics rebuilt";
    progressLabel.value = "";
    progressPercent.value = 0;
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    if (!isCancelled(msg)) error.value = msg;
    busy.value = "";
    progressLabel.value = "";
    progressPercent.value = 0;
  } finally {
    jobActive.value = false;
  }
}

function cancelJobs(): void {
  void CancelSemantics();
  void CancelIndex();
  jobActive.value = false;
  busy.value = "";
  progressLabel.value = "";
  progressPercent.value = 0;
}

function jumpToDiag(d: LocDiagnostic): void {
  void openFile(d.filePath);
}

function applyProgress(raw: unknown, fallback: string): void {
  const wrapped = raw as { data?: Record<string, unknown> } & Record<string, unknown>;
  const d = wrapped.data ?? wrapped;
  if (!d) return;
  if (d.phase === "cancelled") {
    jobActive.value = false;
    busy.value = "";
    progressLabel.value = "";
    progressPercent.value = 0;
    return;
  }
  const pct = typeof d.percent === "number" ? d.percent : undefined;
  if (pct != null) progressPercent.value = pct;
  const pctText = pct != null ? ` ${Math.round(pct)}%` : "";
  progressLabel.value = `${String(d.message ?? d.phase ?? fallback)}${pctText}`;
}

watch(workspaceId, loadWorkspace, { immediate: true });

onMounted(() => {
  ctx.setActiveWorkspace(workspaceId.value);
  window.addEventListener("keydown", onGlobalKey);
  offIndex = Events.On("index:progress", (ev) => applyProgress(ev, "Indexing"));
  offSemantics = Events.On("semantics:progress", (ev) => applyProgress(ev, "Semantics"));
});

onBeforeUnmount(() => {
  loadGen++;
  window.removeEventListener("keydown", onGlobalKey);
  offIndex?.();
  offSemantics?.();
  cancelJobs();
});
</script>

<template>
  <div class="relative flex h-full min-h-0 flex-col overflow-hidden">
    <div class="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-default bg-muted/50 px-3 py-2">
      <div class="flex min-w-0 items-center gap-2">
        <UTooltip text="Back to library"><UButton
            icon="i-lucide-arrow-left"
            variant="ghost"
            size="sm"
            @click="router.push({ name: 'library' })"  /></UTooltip>
        <span class="truncate font-semibold">{{ workspace?.name ?? "Workspace" }}</span>
        <UBadge v-if="workspace" color="neutral" variant="outline" size="xs">
          {{ workspace.gameId.toUpperCase() }}
        </UBadge>
        <UBadge
          v-if="semantics"
          :color="semantics.present ? 'success' : 'warning'"
          variant="subtle"
          size="xs"
        >
          {{
            semantics.present
              ? `Semantics ${semantics.scannedAt?.slice(0, 10) ?? "ready"}`
              : "No semantics"
          }}
        </UBadge>
        <UTooltip v-if="jobActive || busy" text="Cancel rebuild / index">
          <UButton
            label="Cancel"
            size="xs"
            color="neutral"
            variant="ghost"
            @click="cancelJobs"
          />
        </UTooltip>
      </div>
      <div class="flex flex-wrap items-center gap-1.5">
        <div class="flex items-center gap-0.5">
          <UTooltip text="Smaller explorer font"><UButton icon="i-lucide-minus" size="xs" variant="ghost" @click="bumpTreeFontSize(-1)"  /></UTooltip>
          <span class="w-7 text-center text-xs text-muted">{{ treeFontSize }}</span>
          <UTooltip text="Larger explorer font"><UButton icon="i-lucide-plus" size="xs" variant="ghost" @click="bumpTreeFontSize(1)"  /></UTooltip>
        </div>
        <USelect
          :model-value="editorTheme"
          :items="[...EDITOR_THEME_OPTIONS]"
          value-key="value"
          class="w-36"
          size="sm"
          @update:model-value="(v: string) => setEditorTheme(v)"
        />
        <UTooltip text="Compare two files side-by-side"><USwitch v-model="compareMode" label="Compare" size="sm"  /></UTooltip>
        <UTooltip text="Save active file (Ctrl+S)"><UButton
            label="Save"
            icon="i-lucide-save"
            variant="outline"
            size="sm"
            :disabled="!canSave"
            @click="saveActive"  /></UTooltip>
        <UTooltip text="Find missing localization keys"><UButton
            label="Loc lint"
            icon="i-lucide-languages"
            variant="outline"
            size="sm"
            @click="runLocLint"  /></UTooltip>
        <UTooltip text="Stub missing loc keys into mod file"><UButton
            label="Auto-loc"
            icon="i-lucide-wand-sparkles"
            variant="outline"
            size="sm"
            @click="runAutoLoc"  /></UTooltip>
        <UTooltip text="Rebuild install semantics then index"><UButton
            label="Rebuild semantics"
            icon="i-lucide-refresh-cw"
            variant="outline"
            size="sm"
            :disabled="jobActive"
            @click="rebuildSemantics"  /></UTooltip>
        <UTooltip text="Open Patch Center"><UButton
            label="Patch Center"
            icon="i-lucide-external-link"
            variant="soft"
            size="sm"
            @click="router.push({ name: 'patch-center', params: { id: workspaceId } })"  /></UTooltip>
        <UTooltip text="Open Mod Patcher"><UButton
            label="Patcher"
            icon="i-lucide-external-link"
            variant="soft"
            size="sm"
            @click="router.push({ name: 'patcher', params: { id: workspaceId } })"  /></UTooltip>
        <UTooltip text="Open Event Graph"><UButton
            label="Graph"
            icon="i-lucide-external-link"
            variant="soft"
            size="sm"
            @click="router.push({ name: 'event-graph', params: { id: workspaceId } })"  /></UTooltip>
      </div>
    </div>

    <UAlert v-if="error" color="error" variant="subtle" :description="error" class="m-2 shrink-0" />
    <UAlert
      v-else-if="semantics && !semantics.present"
      color="warning"
      variant="subtle"
      class="m-2 shrink-0"
      title="Semantics cache missing"
      description="Rebuild semantics before relying on type-aware tools."
    >
      <template #actions>
        <UButton size="xs" label="Rebuild now" @click="rebuildSemantics" />
      </template>
    </UAlert>

    <div
      v-if="jobActive"
      class="absolute inset-0 z-20 flex items-center justify-center bg-default/70 backdrop-blur-[1px]"
    >
      <div class="w-full max-w-md rounded-lg border border-default bg-default p-5 shadow-lg">
        <div class="mb-1 text-sm font-semibold">{{ busy || indexVerb + "…" }}</div>
        <div class="mb-3 text-xs text-muted">{{ progressLabel || "Working…" }}</div>
        <div class="h-2 w-full overflow-hidden rounded bg-muted">
          <div
            class="h-full rounded bg-primary transition-all"
            :style="{
              width: progressPercent > 0 ? `${Math.min(100, progressPercent)}%` : '35%',
            }"
          />
        </div>
        <div class="mt-3 flex justify-end">
          <UButton label="Cancel" size="xs" color="neutral" variant="outline" @click="cancelJobs" />
        </div>
      </div>
    </div>

    <div class="flex min-h-0 flex-1 overflow-hidden">
      <SplitPane
        class="h-full min-h-0 flex-1 rounded-none border-0"
        fixed-side="first"
        :default-second-size="280"
      >
        <template #first>
          <div class="flex h-full min-h-0 flex-col overflow-hidden">
            <div class="shrink-0 border-b border-default px-2 py-1 text-xs font-semibold text-muted">
              Explorer
            </div>
            <div class="min-h-0 flex-1 overflow-auto" :style="{ fontSize: `${treeFontSize}px` }">
              <div v-for="root in fileRoots" :key="root.path" class="border-b border-default">
                <button
                  class="flex w-full items-center gap-2 px-2 py-1.5 text-left hover:bg-muted/50"
                  :class="{ 'bg-muted/40': expandedRoots.has(root.path) }"
                  @click="toggleRoot(root)"
                >
                  <UIcon
                    :name="expandedRoots.has(root.path) ? 'i-lucide-folder-open' : 'i-lucide-folder'"
                    class="text-muted"
                  />
                  <span class="flex-1 truncate">{{ root.label }}</span>
                  <UBadge
                    :color="root.type === 'game' ? 'primary' : root.type === 'mod' ? 'secondary' : 'warning'"
                    variant="subtle"
                    size="xs"
                  >
                    {{ root.type }}
                  </UBadge>
                </button>
                <div v-if="expandedRoots.has(root.path)" class="bg-default/50">
                  <button
                    v-for="{ entry, depth, pathKey } in rowsForRoot(root.path)"
                    :key="pathKey"
                    class="flex w-full items-center gap-1 px-2 py-0.5 text-left hover:bg-muted/50"
                    :class="{ 'bg-primary/10': activeTabPath === pathKey }"
                    :style="{ paddingLeft: `${depth * 12 + 8}px` }"
                    @click="selectFile(entry)"
                  >
                    <UIcon
                      :name="
                        entry.isDir
                          ? expandedDirs.has(pathKey)
                            ? 'i-lucide-folder-open'
                            : 'i-lucide-folder'
                          : 'i-lucide-file-text'
                      "
                      class="shrink-0 text-muted"
                    />
                    <span class="truncate">{{ entry.name }}</span>
                  </button>
                  <div
                    v-if="!(rootEntries[root.path]?.length) && !loading"
                    class="px-3 py-2 text-muted"
                  >
                    Empty folder
                  </div>
                </div>
              </div>
            </div>
            <div v-if="showLocPanel" class="max-h-48 shrink-0 overflow-auto border-t border-default">
              <div class="flex items-center justify-between px-2 py-1 text-xs font-semibold">
                <span>Missing loc ({{ locDiags.length }})</span>
                <UButton size="xs" variant="ghost" icon="i-lucide-x" @click="showLocPanel = false" />
              </div>
              <button
                v-for="(d, i) in locDiags"
                :key="`${d.key}-${i}`"
                class="flex w-full flex-col px-2 py-1 text-left text-xs hover:bg-muted/50"
                @click="jumpToDiag(d)"
              >
                <span class="font-medium">{{ d.key }}</span>
                <span class="truncate text-muted">{{ d.filePath }}:{{ d.line }}</span>
              </button>
            </div>
          </div>
        </template>
        <template #second>
          <div class="flex h-full min-h-0 min-w-0 flex-col">
            <div
              v-if="tabs.length"
              class="flex shrink-0 items-center gap-0.5 overflow-x-auto border-b border-default bg-muted/30 px-1"
            >
              <button
                v-for="tab in tabs"
                :key="tab.path"
                class="group flex max-w-48 items-center gap-1 rounded-t px-2 py-1.5 text-xs hover:bg-muted"
                :class="{ 'bg-default font-medium': tab.path === activeTabPath }"
                @click="activeTabPath = tab.path"
              >
                <span class="truncate">{{ tab.dirty ? "● " : "" }}{{ tab.name }}</span>
                <span
                  class="inline-flex size-3.5 shrink-0 items-center justify-center rounded opacity-60 hover:bg-muted hover:opacity-100"
                  role="button"
                  title="Close tab"
                  @click.stop="closeTab(tab.path)"
                >
                  <UIcon name="i-lucide-x" class="size-3" />
                </span>
              </button>
            </div>
            <div class="flex min-h-0 flex-1">
              <EditorView
                class="min-w-0 flex-1"
                :items="editorItems"
                :editable="!compareMode"
                :placeholder="compareMode ? 'Select two files to compare' : 'Select a file to edit'"
                @item-edit="onItemEdit"
              />
              <div v-if="isGuiFile" class="w-80 shrink-0 border-l border-default">
                <GuiPreviewPanel
                  :roots="guiRoots"
                  :selected-name="selectedGuiName"
                  @select="(n) => (selectedGuiName = n)"
                />
              </div>
            </div>
          </div>
        </template>
      </SplitPane>
    </div>

    <UModal v-model:open="autoLocOpen" title="Auto-loc result">
      <template #body>
        <div v-if="autoLocResult" class="space-y-2 text-sm">
          <p>
            Wrote <strong>{{ autoLocResult.count }}</strong> key(s) to:
          </p>
          <code class="block break-all rounded bg-muted px-2 py-1 text-xs">
            {{ autoLocResult.path }}
          </code>
          <pre class="max-h-64 overflow-auto rounded border border-default p-2 text-xs">{{
            autoLocResult.preview || "(empty)"
          }}</pre>
        </div>
      </template>
      <template #footer>
        <UButton label="Close" color="neutral" variant="outline" @click="autoLocOpen = false" />
        <UButton
          v-if="autoLocResult?.path"
          label="Open file"
          @click="
            () => {
              autoLocOpen = false;
              void openFile(autoLocResult!.path);
            }
          "
        />
      </template>
    </UModal>
  </div>
</template>
