<script setup lang="ts">
/**
 * Vue port of the Modding Docs workflow.
 */
import { computed, onMounted, ref, watch } from "vue";
import { OpenURL } from "@services/browserservice";
import { BuildTree } from "@services/fileservice";
import { GetDocContent, GetDocPathCache, Scan } from "@services/moddocservice";
import { GetSettings } from "@services/settingsservice";
import type { TreeNode } from "@services/models";
import { useCurrentGame } from "../composables/appContext";
import { normalizeSettings } from "../composables/settings";
import { toTreeItems } from "../composables/treeItems";
import EditorView from "../components/EditorView.vue";

const currentGame = useCurrentGame();
const filterText = ref("");
const docFiles = ref<string[]>([]);
const docTree = ref<TreeNode[]>([]);
const selectedEntry = ref<{ name: string; content: string }>({ name: "Select a file", content: "" });
const settings = ref<Record<string, string>>({});
const loading = ref(false);
const tabItems = [
  { label: "Script docs", value: "docs", slot: "docs" },
  { label: "Modding Wiki", value: "wiki", slot: "wiki" },
];

const installPath = computed(() =>
  currentGame.value === "CK3"
    ? (settings.value["ck3.install_path"] ?? "")
    : (settings.value["eu5.install_path"] ?? ""),
);
const wikiUrl = computed(() =>
  currentGame.value === "CK3"
    ? (settings.value["ck3.ck3_wikiUrl"] ?? "")
    : (settings.value["eu5.eu5_wikiUrl"] ?? ""),
);
const canScan = computed(() => installPath.value.trim().length > 0);
const filteredDocFiles = computed(() => {
  const text = filterText.value.trim().toLowerCase();
  if (!text) return docFiles.value;
  return docFiles.value.filter((path) => path.toLowerCase().includes(text));
});
const treeItems = computed(() =>
  toTreeItems(docTree.value, (node) => void selectFile(node), !!filterText.value),
);
const selectedFileItems = computed(() => [
  {
    id: "file:" + selectedEntry.value.name,
    type: "file" as const,
    file: {
      name: selectedEntry.value.name,
      contents: selectedEntry.value.content,
      lang: "hcl",
    },
  },
]);

/** Load game install paths and wiki URLs from backend settings. */
async function loadSettings(): Promise<void> {
  settings.value = normalizeSettings(await GetSettings());
}

/** Restore cached doc paths for the current game, if any. */
async function loadCachedDocs(): Promise<void> {
  selectedEntry.value = { name: "Select a file", content: "" };
  const cache = await GetDocPathCache(currentGame.value, installPath.value);
  if (cache?.paths?.length) {
    docFiles.value = cache.paths;
    await rebuildDocTree();
    return;
  }
  docFiles.value = [];
  docTree.value = [];
}

/** Scan the game install for script documentation files. */
async function refreshDocs(): Promise<void> {
  if (!canScan.value) return;
  loading.value = true;
  try {
    selectedEntry.value = { name: "Select a file", content: "" };
    docFiles.value = (await Scan(currentGame.value, installPath.value)) ?? [];
    await rebuildDocTree();
  } finally {
    loading.value = false;
  }
}

/** Load a documentation file's contents into the viewer. */
async function selectFile(file: TreeNode): Promise<void> {
  const content = await GetDocContent(currentGame.value, installPath.value, file.relPath);
  selectedEntry.value = { name: file.name, content: content ?? "" };
}

/** Open the configured wiki URL in the system browser. */
async function openWiki(): Promise<void> {
  if (wikiUrl.value) await OpenURL(wikiUrl.value);
}

/** Rebuild the file tree from the current filtered path list. */
async function rebuildDocTree(): Promise<void> {
  docTree.value = (await BuildTree(filteredDocFiles.value)) ?? [];
}

watch(
  [currentGame, installPath],
  async () => {
    await loadCachedDocs();
  },
  { immediate: false },
);

watch(filterText, async () => {
  await rebuildDocTree();
});

onMounted(async () => {
  await loadSettings();
  await loadCachedDocs();
});
</script>

<template>
  <div class="flex h-full min-h-0 flex-1 flex-col gap-2 overflow-hidden p-3">
    <UTabs
      :items="tabItems"
      value-key="value"
      variant="link"
      color="primary"
      size="sm"
      class="flex min-h-0 flex-1 flex-col"
      :ui="{ content: 'flex-1 min-h-0' }"
    >
      <template #docs>
        <div class="flex h-full min-h-0 flex-col gap-2">
          <UFieldGroup class="w-full shrink-0">
            <UInput
              :model-value="installPath"
              readonly
              class="w-full"
              size="sm"
              placeholder="Install path (set in Settings)"
            />
            <UButton
              label="Scan"
              color="secondary"
              variant="outline"
              size="sm"
              :loading="loading"
              :disabled="!canScan"
              @click="refreshDocs"
            />
          </UFieldGroup>

          <div class="grid min-h-0 flex-1 grid-cols-1 gap-2 lg:grid-cols-12">
            <div class="min-h-0 overflow-hidden rounded-lg border border-default lg:col-span-4">
              <div class="border-b border-default p-2">
                <UInput
                  v-model="filterText"
                  placeholder="Filter files..."
                  icon="i-lucide-search"
                  size="sm"
                />
              </div>
              <div class="h-[calc(100%-3rem)] overflow-auto p-1">
                <UTree :items="treeItems" />
              </div>
            </div>
            <div class="min-h-0 overflow-hidden rounded-lg border border-default lg:col-span-8">
              <EditorView
                :label="selectedEntry.name"
                :items="selectedFileItems"
                placeholder="Select a file to view content"
              />
            </div>
          </div>
        </div>
      </template>

      <template #wiki>
        <div class="flex h-full min-h-0 flex-col gap-2">
          <div class="flex shrink-0 justify-end">
            <UButton
              label="Open in Browser"
              icon="i-lucide-external-link"
              color="secondary"
              variant="outline"
              size="sm"
              :disabled="!wikiUrl"
              @click="openWiki"
            />
          </div>
          <iframe v-if="wikiUrl" :src="wikiUrl" title="Modding Wiki" class="min-h-0 w-full flex-1 rounded-lg border border-default" />
          <UEmpty
            v-else
            class="flex-1"
            icon="i-lucide-globe"
            title="No wiki URL configured"
            description="Set the wiki URL in Settings."
          />
        </div>
      </template>
    </UTabs>
  </div>
</template>
