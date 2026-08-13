<script setup lang="ts">
/**
 * Vue/Nuxt UI port of the file compare workflow.
 */
import { computed, onMounted, ref, watch } from "vue";
import { BuildTree, ReadFileContent } from "@services/fileservice";
import { DirectoryCompare, VanillaCompare } from "@services/compareservice";
import type { PathMatch, TreeNode } from "@services/models";
import FileSelector from "../components/FileSelector.vue";
import SplitPane from "../components/SplitPane.vue";
import EditorView from "../components/EditorView.vue";
import { parseDiffFromFile } from "@pierre/diffs";
import { useCurrentGame } from "../composables/appContext";
import { normalizeSettings } from "../composables/settings";
import { toTreeItems } from "../composables/treeItems";
import { GetSettings } from "@services/settingsservice";

const currentGame = useCurrentGame();
const settings = ref<Record<string, string>>({});
const compareMode = ref<string | number>("vanilla");
const compareModeOptions = [
  { label: "Vanilla vs Mod", value: "vanilla", slot: "vanilla" },
  { label: "Directory vs Directory", value: "directory", slot: "directory" },
  { label: "File vs File", value: "file", slot: "file" },
];
const layoutItems = [
  { label: "Unified", value: false },
  { label: "Split", value: true },
];
const modPath = ref("");
const setAPath = ref("");
const setBPath = ref("");
const fileAPath = ref("");
const fileBPath = ref("");
const matchingFiles = ref<Record<string, PathMatch | undefined>>({});
const selectedIndex = ref<number | null>(null);
const showFullscreen = ref(false);
const renderSideBySide = ref(true);
const loading = ref(false);
const originalContent = ref("");
const modifiedContent = ref("");
const selectedLabel = ref("Select a row to view the diff");
const resultsTree = ref<TreeNode[]>([]);

/** Flatten compare matches into table-friendly rows. */
function toRows(
  entries: [string, PathMatch | undefined][],
): { relativePath: string; pathA: string; pathB: string }[] {
  const result: { relativePath: string; pathA: string; pathB: string }[] = [];
  for (const [relativePath, match] of entries) {
    if (!match) continue;
    result.push({ relativePath, pathA: match.pathA, pathB: match.pathB });
  }
  return result;
}

const rows = computed(() =>
  toRows(Object.entries(matchingFiles.value) as [string, PathMatch | undefined][]),
);
const selectedRow = computed(() =>
  selectedIndex.value === null ? null : (rows.value[selectedIndex.value] ?? null),
);
const rowIndexByPath = computed(
  () => new Map(rows.value.map((row, index) => [row.relativePath, index])),
);
const treeItems = computed(() =>
  toTreeItems(resultsTree.value, (node) => {
    const index = rowIndexByPath.value.get(node.relPath);
    if (index !== undefined) void openRow(index);
  }),
);
const installPath = computed(() =>
  currentGame.value === "CK3"
    ? (settings.value["ck3.install_path"] ?? "")
    : (settings.value["eu5.install_path"] ?? ""),
);
const scriptRootHint = computed(() => (currentGame.value === "CK3" ? "game" : "game/in_game"));
const canRunVanilla = computed(() => installPath.value.length > 0 && modPath.value.length > 0);
const canRunDirectory = computed(() => setAPath.value.length > 0 && setBPath.value.length > 0);
const canRunFile = computed(() => fileAPath.value.length > 0 && fileBPath.value.length > 0);
const selectedDiffItems = computed(() => {
  if (!selectedRow.value) return [];
  return [
    {
      id: "diff:" + selectedRow.value.relativePath,
      type: "diff" as const,
      fileDiff: parseDiffFromFile(
        { name: selectedRow.value.pathA, contents: originalContent.value, lang: "hcl" },
        { name: selectedRow.value.pathB, contents: modifiedContent.value, lang: "hcl" },
      ),
    },
  ];
});

/** Load game install paths from backend settings. */
async function loadSettings(): Promise<void> {
  settings.value = normalizeSettings(await GetSettings());
}

/** Clear compare results and the selected diff. */
function clearResults(): void {
  matchingFiles.value = {};
  resultsTree.value = [];
  selectedIndex.value = null;
  originalContent.value = "";
  modifiedContent.value = "";
  selectedLabel.value = "Select a row to view the diff";
}

/** Compare vanilla game files against a selected mod folder. */
async function runVanillaCompare(): Promise<void> {
  loading.value = true;
  try {
    matchingFiles.value =
      (await VanillaCompare(currentGame.value, installPath.value, modPath.value)) ?? {};
    selectedIndex.value = null;
    selectedLabel.value = "Select a row to view the diff";
  } finally {
    loading.value = false;
  }
}

/** Compare two arbitrary directories. */
async function runDirectoryCompare(): Promise<void> {
  loading.value = true;
  try {
    matchingFiles.value = (await DirectoryCompare(setAPath.value, setBPath.value)) ?? {};
    selectedIndex.value = null;
    selectedLabel.value = "Select a row to view the diff";
  } finally {
    loading.value = false;
  }
}

/** Compare two explicit files as a single result row. */
async function runFileCompare(): Promise<void> {
  matchingFiles.value = {
    "Comparing Two Files": { pathA: fileAPath.value, pathB: fileBPath.value },
  };
  selectedIndex.value = null;
  selectedLabel.value = "Select a row to view the diff";
}

/** Load file contents for a result row and show the diff. */
async function openRow(index: number): Promise<void> {
  selectedIndex.value = index;
  const row = rows.value[index];
  if (!row) return;
  const [original, modified] = await Promise.all([
    ReadFileContent(row.pathA),
    ReadFileContent(row.pathB),
  ]);
  originalContent.value = original;
  modifiedContent.value = modified;
  selectedLabel.value = row.relativePath;
}

/** Navigate to a result row by index. */
function navigateTo(index: number): void {
  void openRow(index);
}

/** Open the fullscreen diff modal for the selected row. */
function openFullscreen(): void {
  if (selectedRow.value) showFullscreen.value = true;
}

/** Rebuild the results tree from current matched paths. */
async function rebuildResultsTree(): Promise<void> {
  resultsTree.value = (await BuildTree(rows.value.map((row) => row.relativePath))) ?? [];
}

watch(rows, async () => {
  await rebuildResultsTree();
});

onMounted(loadSettings);
</script>

<template>
  <div class="flex h-full min-h-0 flex-1 flex-col gap-2 overflow-hidden p-3">
    <div class="flex shrink-0 flex-wrap items-center justify-between gap-2">
      <div>
        <h1 class="text-lg font-semibold">Compare</h1>
        <p class="text-xs text-muted">Vanilla, directories, or file pairs</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton
          v-if="rows.length"
          label="Clear"
          color="error"
          variant="ghost"
          size="sm"
          @click="clearResults"
        />
        <span v-if="rows.length" class="text-xs text-muted">{{ rows.length }} matches</span>
      </div>
    </div>

    <UAccordion
      :items="[{ label: 'Setup', icon: 'i-lucide-sliders-horizontal', value: 'setup' }]"
      :default-value="rows.length ? undefined : 'setup'"
      :unmount-on-hide="false"
      class="shrink-0"
    >
      <template #body>
        <UTabs
          v-model="compareMode"
          :items="compareModeOptions"
          value-key="value"
          :unmount-on-hide="false"
          variant="link"
          color="primary"
          size="sm"
        >
          <template #vanilla>
            <div class="space-y-2 pb-2">
              <UInput :model-value="installPath" readonly placeholder="Vanilla (A)" size="sm" />
              <div class="text-xs text-muted">{{ currentGame }} · {{ scriptRootHint }}</div>
              <FileSelector
                v-model="modPath"
                label="Mod (B)"
                dialog-title="Select Mod (B)"
                mode="folder"
                placeholder="Mod folder"
              />
              <div class="flex justify-end">
                <UButton
                  :loading="loading"
                  :disabled="!canRunVanilla"
                  label="Run Compare"
                  size="sm"
                  @click="runVanillaCompare"
                />
              </div>
            </div>
          </template>
          <template #directory>
            <div class="space-y-2 pb-2">
              <FileSelector v-model="setAPath" label="Set A" dialog-title="Select Set A" mode="folder" />
              <FileSelector v-model="setBPath" label="Set B" dialog-title="Select Set B" mode="folder" />
              <div class="flex justify-end">
                <UButton
                  :loading="loading"
                  :disabled="!canRunDirectory"
                  label="Run Compare"
                  size="sm"
                  @click="runDirectoryCompare"
                />
              </div>
            </div>
          </template>
          <template #file>
            <div class="space-y-2 pb-2">
              <FileSelector v-model="fileAPath" label="File A" dialog-title="Select File A" mode="file" />
              <FileSelector v-model="fileBPath" label="File B" dialog-title="Select File B" mode="file" />
              <div class="flex justify-end">
                <UButton :disabled="!canRunFile" label="Run Compare" size="sm" @click="runFileCompare" />
              </div>
            </div>
          </template>
        </UTabs>
      </template>
    </UAccordion>

    <div class="min-h-0 flex-1 overflow-hidden rounded-lg border border-default">
      <SplitPane
        v-if="rows.length"
        :second-open="selectedIndex !== null"
        :default-second-size="580"
        class="h-full rounded-none border-0"
      >
        <template #first>
          <div class="h-full overflow-auto p-1">
            <UTree :items="treeItems" />
          </div>
        </template>
        <template #second>
          <div v-if="selectedRow" class="flex h-full min-h-0 flex-col overflow-hidden">
            <div class="flex shrink-0 items-center justify-between gap-2 border-b border-default bg-muted/50 px-3 py-1.5">
              <div class="min-w-0 truncate text-sm font-medium">{{ selectedRow.relativePath }}</div>
              <div class="flex items-center gap-1">
                <UButton
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  icon="i-lucide-chevron-left"
                  :disabled="selectedIndex === null || selectedIndex <= 0"
                  @click="selectedIndex !== null && navigateTo(selectedIndex - 1)"
                />
                <UButton
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  icon="i-lucide-chevron-right"
                  :disabled="selectedIndex === null || selectedIndex >= rows.length - 1"
                  @click="selectedIndex !== null && navigateTo(selectedIndex + 1)"
                />
                <UButton
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  icon="i-lucide-expand"
                  @click="openFullscreen"
                />
              </div>
            </div>
            <EditorView :items="selectedDiffItems" />
          </div>
        </template>
      </SplitPane>
      <UEmpty v-else class="h-full" icon="i-lucide-git-compare" title="No results" description="Configure setup and run compare." />
    </div>

    <UModal v-model:open="showFullscreen" :title="selectedLabel" :ui="{ content: 'sm:max-w-[96vw] max-h-[95vh]' }">
      <template #body>
        <div class="flex min-h-[70vh] flex-col gap-2">
          <div class="flex items-center justify-end gap-2">
            <USelect v-model="renderSideBySide" :items="layoutItems" value-key="value" class="w-28" size="sm" />
          </div>
          <EditorView class="min-h-0 flex-1" :items="selectedDiffItems" />
        </div>
      </template>
    </UModal>
  </div>
</template>
