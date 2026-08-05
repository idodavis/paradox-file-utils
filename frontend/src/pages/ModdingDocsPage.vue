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
import EditorView from "../components/EditorView.vue";

type DocTreeItem = {
  label: string;
  icon?: string;
  defaultExpanded?: boolean;
  onSelect?: () => void;
  children?: DocTreeItem[];
};

const currentGame = useCurrentGame();
const filterText = ref("");
const docFiles = ref<string[]>([]);
const docTree = ref<TreeNode[]>([]);
const selectedEntry = ref<{ name: string; content: string }>({ name: "Select a file", content: "" });
const settings = ref<Record<string, string>>({});
const loading = ref(false);
const tabItems = [
  { label: "Script docs", slot: "docs" },
  { label: "Modding Wiki", slot: "wiki" },
];

const installPath = computed(() =>
  currentGame.value === "CK3" ? (settings.value["ck3.install_path"] ?? "") : (settings.value["eu5.install_path"] ?? ""),
);
const wikiUrl = computed(() =>
  currentGame.value === "CK3" ? (settings.value["ck3.ck3_wikiUrl"] ?? "") : (settings.value["eu5.eu5_wikiUrl"] ?? ""),
);
const canScan = computed(() => installPath.value.trim().length > 0);
const filteredDocFiles = computed(() => {
  const text = filterText.value.trim().toLowerCase();
  if (!text) return docFiles.value;
  return docFiles.value.filter((path) => path.toLowerCase().includes(text));
});
const treeItems = computed(() => toTreeItems(docTree.value));

async function loadSettings(): Promise<void> {
  settings.value = Object.fromEntries(
    Object.entries((await GetSettings()) ?? {}).filter(([, value]) => value !== undefined),
  ) as Record<string, string>;
}

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

async function selectFile(file: TreeNode): Promise<void> {
  const content = await GetDocContent(currentGame.value, installPath.value, file.relPath);
  selectedEntry.value = {
    name: file.name,
    content: content ?? "",
  };
}

async function openWiki(): Promise<void> {
  if (wikiUrl.value) {
    await OpenURL(wikiUrl.value);
  }
}

async function rebuildDocTree(): Promise<void> {
  docTree.value = (await BuildTree(filteredDocFiles.value)) ?? [];
}

function toTreeItems(nodes: TreeNode[]): DocTreeItem[] {
  return nodes.map((node) => {
    const children = node.children?.length ? toTreeItems(node.children) : undefined;
    return {
      label: node.name,
      icon: children?.length ? "i-lucide-folder" : "i-lucide-file-text",
      defaultExpanded: !!filterText.value,
      onSelect: children?.length ? undefined : () => void selectFile(node),
      children,
    };
  });
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
  <div class="relative flex min-h-0 flex-1 flex-col p-4 max-w-full min-w-0 overflow-auto">
    <UTabs :items="tabItems" variant="link" color="primary" class="mb-4">

      <template #docs>
        <div class="flex min-h-0 flex-1 flex-col gap-4 rounded-lg border border-default bg-muted/60 p-2">
          <UCard>
            <UFormField label="Game install path">
              <div class="flex gap-2">
                <UInput :model-value="installPath" readonly class="flex-1 min-w-0"
                  placeholder="Set in Settings (gear icon in header)" />
                <UButton label="Scan" color="secondary" variant="outline" :loading="loading" :disabled="!canScan"
                  @click="refreshDocs" />
              </div>
            </UFormField>
          </UCard>

          <div class="grid flex-1 min-h-0 grid-cols-1 gap-4 lg:grid-cols-12">
            <div class="lg:col-span-4 min-h-0">
              <UCard class="flex h-full min-h-0 flex-col overflow-hidden"
                :ui="{ root: 'h-full flex flex-col', body: 'flex-1 min-h-0 p-0' }">
                <template #header>
                  <div class="space-y-2">
                    <div class="text-sm font-semibold">Filter Files</div>
                    <UInput v-model="filterText" placeholder="Type to filter by filename..." icon="i-lucide-search" />
                  </div>
                </template>
                <div class="flex-1 min-h-0 overflow-auto p-2">
                  <UTree :items="treeItems" />
                </div>
              </UCard>
            </div>

            <div class="lg:col-span-8 min-h-0">
              <EditorView :label="selectedEntry.name" :items="[
                {
                  id: 'file:' + selectedEntry.name,
                  type: 'file',
                  file: {
                    name: selectedEntry.name,
                    contents: selectedEntry.content,
                  },
                },
              ]" placeholder="Select a file to view content" />
            </div>
          </div>
        </div>
      </template>

      <template #wiki>
        <div class="flex min-h-0 flex-1 flex-col rounded-lg border border-default bg-muted/60 p-2">
          <div class="flex min-h-0 flex-1 flex-col gap-2">
            <div class="flex justify-end shrink-0">
              <UButton label="Open in Browser" icon="i-lucide-external-link" color="secondary" variant="outline"
                :disabled="!wikiUrl" @click="openWiki" />
            </div>
            <iframe v-if="wikiUrl" :src="wikiUrl" title="Modding Wiki" class="w-full flex-1 min-h-0" />
            <div v-else class="flex flex-1 items-center justify-center text-muted">No wiki URL configured.</div>
          </div>
        </div>
      </template>
    </UTabs>
  </div>
</template>
