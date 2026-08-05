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
import { GetSettings } from "@services/settingsservice";

type CompareTreeItem = {
  label: string;
  icon?: string;
  defaultExpanded?: boolean;
  onSelect?: () => void;
  children?: CompareTreeItem[];
};

const currentGame = useCurrentGame();
const settings = ref<Record<string, string>>({});
const compareMode = ref("vanilla");
const compareModeOptions = [
  { label: "Vanilla vs Mod", value: "vanilla" },
  { label: "Directory vs Directory", value: "directory" },
  { label: "File vs File", value: "file" },
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

function toRows(entries: [string, PathMatch | undefined][]): { relativePath: string; pathA: string; pathB: string }[] {
  const result: { relativePath: string; pathA: string; pathB: string }[] = [];
  for (const [relativePath, match] of entries) {
    if (!match) continue;
    result.push({
      relativePath,
      pathA: match.pathA,
      pathB: match.pathB,
    });
  }
  return result;
}

const rows = computed(() => toRows(Object.entries(matchingFiles.value) as [string, PathMatch | undefined][]));
const selectedRow = computed(() => (selectedIndex.value === null ? null : (rows.value[selectedIndex.value] ?? null)));
const rowIndexByPath = computed(() => new Map(rows.value.map((row, index) => [row.relativePath, index])));
const treeItems = computed(() => toTreeItems(resultsTree.value));
const installPath = computed(() =>
  currentGame.value === "CK3" ? (settings.value["ck3.install_path"] ?? "") : (settings.value["eu5.install_path"] ?? ""),
);
const scriptRootHint = computed(() => (currentGame.value === "CK3" ? "game" : "game/in_game"));
const canRunVanilla = computed(() => installPath.value.length > 0 && modPath.value.length > 0);
const canRunDirectory = computed(() => setAPath.value.length > 0 && setBPath.value.length > 0);
const canRunFile = computed(() => fileAPath.value.length > 0 && fileBPath.value.length > 0);

async function loadSettings(): Promise<void> {
  settings.value = Object.fromEntries(
    Object.entries((await GetSettings()) ?? {}).filter(([, value]) => value !== undefined),
  ) as Record<string, string>;
}

function clearResults(): void {
  matchingFiles.value = {};
  resultsTree.value = [];
  selectedIndex.value = null;
  originalContent.value = "";
  modifiedContent.value = "";
  selectedLabel.value = "Select a row to view the diff";
}

async function runVanillaCompare(): Promise<void> {
  loading.value = true;
  try {
    matchingFiles.value = (await VanillaCompare(currentGame.value, installPath.value, modPath.value)) ?? {};
    selectedIndex.value = null;
    selectedLabel.value = "Select a row to view the diff";
  } finally {
    loading.value = false;
  }
}

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

async function runFileCompare(): Promise<void> {
  matchingFiles.value = {
    "Comparing Two Files": {
      pathA: fileAPath.value,
      pathB: fileBPath.value,
    },
  };
  selectedIndex.value = null;
  selectedLabel.value = "Select a row to view the diff";
}

async function openRow(index: number): Promise<void> {
  selectedIndex.value = index;
  const row = rows.value[index];
  if (!row) return;
  const [original, modified] = await Promise.all([ReadFileContent(row.pathA), ReadFileContent(row.pathB)]);
  originalContent.value = original;
  modifiedContent.value = modified;
  selectedLabel.value = row.relativePath;
}

function navigateTo(index: number): void {
  void openRow(index);
}

function openFullscreen(): void {
  if (selectedRow.value) showFullscreen.value = true;
}

async function rebuildResultsTree(): Promise<void> {
  resultsTree.value = (await BuildTree(rows.value.map((row) => row.relativePath))) ?? [];
}

function toTreeItems(nodes: TreeNode[]): CompareTreeItem[] {
  return nodes.map((node) => {
    const children = node.children?.length ? toTreeItems(node.children) : undefined;
    return {
      label: node.name,
      icon: children?.length ? "i-lucide-folder" : "i-lucide-file-text",
      defaultExpanded: true,
      onSelect: children?.length
        ? undefined
        : () => {
          const index = rowIndexByPath.value.get(node.relPath);
          if (index !== undefined) void openRow(index);
        },
      children,
    };
  });
}

watch(rows, async () => {
  await rebuildResultsTree();
});

onMounted(loadSettings);
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-4">
    <UCard>
      <template #header>
        <div class="text-xs font-semibold uppercase tracking-wide text-primary">File compare</div>
        <div class="text-2xl font-bold">Compare Tool</div>
        <div class="mt-2 text-sm text-muted">
          Compare vanilla against mods, directories against directories, or explicit file pairs.
        </div>
      </template>
    </UCard>

    <USelect v-model="compareMode" :items="compareModeOptions" />

    <UCard v-if="compareMode === 'vanilla'">
      <div class="space-y-4">
        <UInput :model-value="installPath" readonly placeholder="Vanilla (A)" />
        <div class="text-xs text-muted">Based on current game: {{ currentGame }} - {{ scriptRootHint }}</div>
        <FileSelector v-model="modPath" label="Mod (B)" dialog-title="Select Mod (B) files/folders" mode="folder"
          placeholder="Select folder or files to compare with Vanilla" />
      </div>
      <template #footer>
        <div class="flex items-center justify-between gap-2">
          <UButton label="Clear Results" color="error" variant="outline" @click="clearResults" />
          <UButton :loading="loading" :disabled="!canRunVanilla" label="Run Compare" @click="runVanillaCompare" />
        </div>
      </template>
    </UCard>

    <UCard v-else-if="compareMode === 'directory'">
      <div class="space-y-4">
        <FileSelector v-model="setAPath" label="Set A" dialog-title="Select Set A files/folders" mode="folder"
          placeholder="Select folder or files for Set A" />
        <FileSelector v-model="setBPath" label="Set B" dialog-title="Select Set B files/folders" mode="folder"
          placeholder="Select folder or files for Set B" />
      </div>
      <template #footer>
        <div class="flex items-center justify-between gap-2">
          <UButton label="Clear Results" color="error" variant="outline" @click="clearResults" />
          <UButton :loading="loading" :disabled="!canRunDirectory" label="Run Compare" @click="runDirectoryCompare" />
        </div>
      </template>
    </UCard>

    <UCard v-else>
      <div class="space-y-4">
        <FileSelector v-model="fileAPath" label="File A" dialog-title="Select File A" mode="file"
          placeholder="Select file for File A" />
        <FileSelector v-model="fileBPath" label="File B" dialog-title="Select File B" mode="file"
          placeholder="Select file for File B" />
      </div>
      <template #footer>
        <div class="flex items-center justify-between gap-2">
          <UButton label="Clear Results" color="error" variant="outline" @click="clearResults" />
          <UButton :disabled="!canRunFile" label="Run Compare" @click="runFileCompare" />
        </div>
      </template>
    </UCard>

    <UCard v-if="rows.length" class="min-h-[24rem] flex-1"
      :ui="{ root: 'flex min-h-[24rem] flex-col', body: 'flex-1 min-h-0 p-0' }">
      <template #header>
        <div class="flex items-center justify-between gap-2">
          <div class="text-base font-medium">Results</div>
          <div class="text-xs text-muted">{{ rows.length }} matched paths</div>
        </div>
      </template>

      <SplitPane :second-open="selectedIndex !== null" :default-second-size="580" class="h-full rounded-none border-0">
        <template #first>
          <div class="h-full overflow-auto">
            <UTree :items="treeItems" />
          </div>
        </template>

        <template #second>
          <div v-if="selectedRow" class="flex h-full min-h-0 flex-col overflow-hidden">
            <div class="flex items-center justify-between gap-2 border-b border-default bg-muted/50 px-3 py-2 shrink-0">
              <div class="min-w-0">
                <div class="truncate text-sm font-semibold text-default">{{ selectedRow.relativePath }}</div>
                <div class="truncate text-xs text-muted">{{ selectedRow.pathA }} → {{ selectedRow.pathB }}</div>
              </div>
              <div class="flex items-center gap-2">
                <UButton color="neutral" variant="outline" icon="i-lucide-chevron-left"
                  :disabled="selectedIndex === null || selectedIndex <= 0"
                  @click="selectedIndex !== null && navigateTo(selectedIndex - 1)" />
                <span class="text-xs text-muted">{{ (selectedIndex ?? 0) + 1 }} / {{ rows.length }}</span>
                <UButton color="neutral" variant="outline" icon="i-lucide-chevron-right"
                  :disabled="selectedIndex === null || selectedIndex >= rows.length - 1"
                  @click="selectedIndex !== null && navigateTo(selectedIndex + 1)" />
                <UButton color="neutral" variant="outline" icon="i-lucide-expand" @click="openFullscreen" />
              </div>
            </div>
            <EditorView :items="[
              {
                id: 'diff:' + selectedRow.relativePath,
                type: 'diff',
                fileDiff: parseDiffFromFile(
                  {
                    name: selectedRow.pathA,
                    contents: originalContent,
                  },
                  {
                    name: selectedRow.pathB,
                    contents: modifiedContent,
                  }),
              },
            ]" />
          </div>
        </template>
      </SplitPane>
    </UCard>

    <UModal v-model:open="showFullscreen" :title="selectedLabel" :ui="{ content: 'sm:max-w-[96vw] max-h-[95vh]' }">
      <template #body>
        <div class="flex min-h-0 flex-col space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="text-xs font-semibold uppercase tracking-wide text-primary">File Comparison View</div>
            <div class="flex items-center gap-2">
              <USelect v-model="renderSideBySide" :items="[
                { label: 'Unified', value: false },
                { label: 'Split', value: true },
              ]" value-key="value" class="w-28" />
              <UButton color="neutral" variant="outline" icon="i-lucide-chevron-left"
                :disabled="selectedIndex === null || selectedIndex <= 0"
                @click="selectedIndex !== null && navigateTo(selectedIndex - 1)" />
              <UButton color="neutral" variant="outline" icon="i-lucide-chevron-right"
                :disabled="selectedIndex === null || selectedIndex >= rows.length - 1"
                @click="selectedIndex !== null && navigateTo(selectedIndex + 1)" />
            </div>
          </div>

          <div class="min-h-[18rem] flex-1">
            <EditorView :items="[
              {
                id: 'diff:' + selectedRow?.relativePath,
                type: 'diff',
                fileDiff: parseDiffFromFile(
                  {
                    name: selectedRow?.pathA || '',
                    contents: originalContent,
                  },
                  {
                    name: selectedRow?.pathB || '',
                    contents: modifiedContent,
                  }),
              },
            ]" />
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
