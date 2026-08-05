<script setup lang="ts">
/**
 * Vue port of the inventory extraction and browsing workflow.
 */
import { computed, onMounted, ref, watch } from "vue";
import type { TableRow } from "@nuxt/ui";
import { GetSettings } from "@services/settingsservice";
import {
  DeleteInventory,
  ExtractInventory,
  GetInventoryItems,
  GetSupportedTypes,
  ListInventoriesForGame,
  RenameInventory,
  SaveInventory,
} from "@services/inventoryservice";
import type { InventoryItemRow, InventorySummary } from "@services/models";
import FileSelector from "../components/FileSelector.vue";
import InventoryCard from "../components/InventoryCard.vue";
import InventoryDetails from "../components/InventoryDetails.vue";
import InventoryNameDialog from "../components/InventoryNameDialog.vue";
import { useCurrentGame } from "../composables/appContext";

const currentGame = useCurrentGame();
const settings = ref<Record<string, string>>({});
const file = ref("");
const selectedTypes = ref<string[]>([]);
const supportedTypes = ref<string[]>([]);
const savedInventories = ref<InventorySummary[]>([]);
const allItems = ref<InventoryItemRow[]>([]);
const currentInventoryId = ref<string | null>(null);
const currentInventoryGame = ref<string | null>(null);
const selectedRow = ref<InventoryItemRow | null>(null);
const itemDetailsOpen = ref(false);
const loading = ref(false);
const currentPage = ref(1);
const rowsPerPage = 20;
const extractionErrors = ref<string[]>([]);
const hasExtraction = ref(false);
const isCurrentTemp = ref(false);
const nameModal = ref<{ open: boolean; mode: "save" | "rename"; invId: string | null }>({
  open: false,
  mode: "save",
  invId: null,
});
let extractionPromise: (Promise<unknown> & { cancel?: () => void }) | null = null;

const typesDisabled = computed(() => supportedTypes.value.length === 0);
const extractDisabled = computed(() => !file.value || selectedTypes.value.length === 0);
const nameModalInitialName = computed(() => {
  if (nameModal.value.mode === "save") {
    return `${currentGame.value} - ${new Date().toISOString().slice(0, 10)} - ${Math.random().toString(36).slice(2, 8)}`;
  }
  return savedInventories.value.find((inv) => inv.id === nameModal.value.invId)?.name ?? "";
});
const gameInstallPath = computed(() =>
  currentGame.value === "CK3" ? (settings.value["ck3.install_path"] ?? "") : (settings.value["eu5.install_path"] ?? ""),
);
const appConstantsHint = computed(() =>
  currentGame.value === "CK3"
    ? (settings.value["ck3.ck3_scriptRootFolder"] ?? "")
    : (settings.value["eu5.eu5_scriptRootFolder"] ?? ""),
);
const totalPages = computed(() => Math.max(1, Math.ceil(allItems.value.length / rowsPerPage)));
const pagedItems = computed(() => {
  const start = (currentPage.value - 1) * rowsPerPage;
  return allItems.value.slice(start, start + rowsPerPage);
});
const itemColumns = [
  { accessorKey: "type", header: "Type" },
  { accessorKey: "key", header: "Key" },
  { accessorKey: "filePath", header: "File" },
  { id: "lines", header: "Lines" },
  { accessorKey: "referencesCount", header: "References" },
  { accessorKey: "referrersCount", header: "Referrers" },
];

function clearAll(): void {
  hasExtraction.value = false;
  extractionErrors.value = [];
  allItems.value = [];
  currentInventoryId.value = null;
  currentInventoryGame.value = null;
  isCurrentTemp.value = false;
  selectedRow.value = null;
  itemDetailsOpen.value = false;
}

async function loadSettings(): Promise<void> {
  settings.value = Object.fromEntries(
    Object.entries((await GetSettings()) ?? {}).filter(([, value]) => value !== undefined),
  ) as Record<string, string>;
}

async function refresh(): Promise<void> {
  const game = currentGame.value;
  const [types, list] = await Promise.all([GetSupportedTypes(game), ListInventoriesForGame(game)]);
  supportedTypes.value = types ?? [];
  savedInventories.value = list ?? [];
  if (currentInventoryId.value && currentInventoryGame.value !== null && currentInventoryGame.value !== game) {
    clearAll();
  }
}

async function doExtract(): Promise<void> {
  if (extractDisabled.value) return;
  loading.value = true;
  clearAll();
  const game = currentGame.value;
  try {
    extractionPromise = ExtractInventory(game, file.value, selectedTypes.value);
    const inventoryId = (await extractionPromise) as string | null;
    if (inventoryId) {
      currentInventoryId.value = inventoryId;
      currentInventoryGame.value = game;
      hasExtraction.value = true;
      isCurrentTemp.value = true;
      allItems.value = (await GetInventoryItems(inventoryId)) ?? [];
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    if (!message.toLowerCase().includes("cancel")) extractionErrors.value = [message];
  } finally {
    loading.value = false;
    extractionPromise = null;
  }
}

function cancelExtraction(): void {
  extractionPromise?.cancel?.();
  extractionPromise = null;
  loading.value = false;
}

async function loadInventory(inv: InventorySummary): Promise<void> {
  currentInventoryId.value = inv.id;
  currentInventoryGame.value = inv.game;
  hasExtraction.value = true;
  isCurrentTemp.value = false;
  allItems.value = (await GetInventoryItems(inv.id)) ?? [];
}

async function handleNameModalSave(name: string): Promise<void> {
  if (!nameModal.value.invId) return;
  if (nameModal.value.mode === "save") {
    await SaveInventory(nameModal.value.invId, name);
    isCurrentTemp.value = false;
  } else {
    await RenameInventory(nameModal.value.invId, name);
  }
  await refresh();
}

async function handleDelete(inv: InventorySummary): Promise<void> {
  await DeleteInventory(inv.id);
  if (currentInventoryId.value === inv.id) clearAll();
  await refresh();
}

function openModal(mode: "save" | "rename", inv?: InventorySummary): void {
  nameModal.value = { open: true, mode, invId: currentInventoryId.value ?? inv?.id ?? null };
}

async function loadSelectedRow(row: InventoryItemRow): Promise<void> {
  selectedRow.value = row;
  itemDetailsOpen.value = true;
}

function selectInventoryRow(_event: Event, row: TableRow<InventoryItemRow>): void {
  void loadSelectedRow(row.original);
}

watch(currentGame, async () => {
  await refresh();
  clearAll();
});

watch(
  allItems,
  () => {
    if (currentPage.value > totalPages.value) currentPage.value = totalPages.value;
    if (currentPage.value < 1) currentPage.value = 1;
  },
  { deep: true },
);

onMounted(async () => {
  await loadSettings();
  await refresh();
});
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-4">
    <UCard>
      <template #header>
        <div class="text-xs font-semibold uppercase tracking-wide text-primary">Inventory explorer</div>
        <div class="text-2xl font-bold">Inventory Tool</div>
        <div class="mt-2 text-sm text-muted">
          Extract and browse game objects from script files with saved inventories and row details.
        </div>
      </template>

      <div class="space-y-4">
        <FileSelector v-model="file" label="Folder to extract" dialog-title="Select a folder" mode="folder"
          placeholder="Folder containing the inventory (e.g. mod folder, game files)" hint="Game install path: " />
        <div class="text-xs text-muted">Game install path: {{ gameInstallPath || "(not set)" }}</div>
        <div class="text-xs text-muted">Script root hint: {{ appConstantsHint || "(not set)" }}</div>
        <div>
          <div class="mb-2 text-sm font-medium">Object types</div>
          <div class="flex flex-wrap items-center gap-2">
            <USelect v-model="selectedTypes" :items="supportedTypes" multiple :disabled="typesDisabled"
              class="min-w-[18rem]" />
            <UButton label="All" color="neutral" variant="outline" :disabled="typesDisabled"
              @click="selectedTypes = [...supportedTypes]" />
            <UButton label="None" color="neutral" variant="outline" :disabled="typesDisabled || !selectedTypes.length"
              @click="selectedTypes = []" />
          </div>
          <div class="mt-1 text-xs text-muted">Note: gfx/, gui/, and music/ types are not currently supported.</div>
        </div>

        <div class="flex items-center justify-between gap-2">
          <UButton label="Clear Results" color="error" variant="outline" :disabled="loading" @click="clearAll" />
          <UButton :loading="loading" :disabled="extractDisabled" :label="loading ? 'Cancel' : 'Extract'"
            @click="loading ? cancelExtraction() : doExtract()" />
        </div>
      </div>
    </UCard>

    <UAlert v-if="extractionErrors.length" color="warning" variant="subtle"
      :title="`Errors (${extractionErrors.length})`" description="Extraction returned one or more errors.">
      <template #body>
        <ul class="mt-2 list-disc pl-5 text-sm">
          <li v-for="error in extractionErrors" :key="error">{{ error }}</li>
        </ul>
      </template>
    </UAlert>

    <UCard v-if="savedInventories.length">
      <template #header>
        <div class="flex items-center justify-between">
          <div class="text-base font-medium">Saved inventories</div>
          <UBadge color="neutral" variant="subtle">{{ savedInventories.length }}</UBadge>
        </div>
      </template>

      <div class="inventory-saved-scroll overflow-x-auto">
        <div class="flex min-w-max gap-4 p-3">
          <div v-for="inv in savedInventories" :key="inv.id" class="inventory-card-width">
            <InventoryCard :inv="inv" :active="currentInventoryId === inv.id" @load="loadInventory"
              @rename="openModal('rename', $event)" @delete="handleDelete" />
          </div>
        </div>
      </div>
    </UCard>

    <UCard v-if="hasExtraction">
      <template #header>
        <div class="flex items-center justify-between">
          <div class="text-base font-medium">{{ allItems.length.toLocaleString() }} items found</div>
          <UButton v-if="isCurrentTemp" label="Save Inventory" @click="openModal('save')" />
        </div>
      </template>

      <UTable :data="pagedItems" :columns="itemColumns" @select="selectInventoryRow">
        <template #type-cell="{ row }">
          <span class="font-medium text-highlighted">{{ row.original.type }}</span>
        </template>
        <template #lines-cell="{ row }"> {{ row.original.lineStart }} - {{ row.original.lineEnd }} </template>
      </UTable>

      <div class="mt-3 flex items-center justify-end gap-3">
        <span class="text-xs text-muted">Page {{ currentPage }} / {{ totalPages }}</span>
        <UPagination v-model:page="currentPage" :items-per-page="rowsPerPage" :total="allItems.length" color="neutral"
          active-color="primary" />
      </div>
    </UCard>

    <div v-else class="py-8 text-center text-muted">No inventory. Select a path and types, then Extract.</div>

    <UModal v-model:open="itemDetailsOpen" title="Item Details" :ui="{ content: 'sm:max-w-4xl' }">
      <template #body>
        <InventoryDetails :inventory-id="currentInventoryId" :item-type="selectedRow?.type ?? null"
          :item-key="selectedRow?.key ?? null" :row="selectedRow" :game="currentGame" />
      </template>
    </UModal>

    <InventoryNameDialog v-model="nameModal.open" :mode="nameModal.mode" :initial-name="nameModalInitialName"
      @save="handleNameModalSave" />
  </div>
</template>
