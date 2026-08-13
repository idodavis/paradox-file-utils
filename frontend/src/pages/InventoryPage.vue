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
import { normalizeSettings } from "../composables/settings";

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
  currentGame.value === "CK3"
    ? (settings.value["ck3.install_path"] ?? "")
    : (settings.value["eu5.install_path"] ?? ""),
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

/** Clear the current extraction results and selection. */
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

/** Load backend settings used for install-path hints. */
async function loadSettings(): Promise<void> {
  settings.value = normalizeSettings(await GetSettings());
}

/** Refresh supported types and saved inventories for the current game. */
async function refresh(): Promise<void> {
  const game = currentGame.value;
  const [types, list] = await Promise.all([GetSupportedTypes(game), ListInventoriesForGame(game)]);
  supportedTypes.value = types ?? [];
  savedInventories.value = list ?? [];
  if (currentInventoryId.value && currentInventoryGame.value !== null && currentInventoryGame.value !== game) {
    clearAll();
  }
}

/** Extract inventory items from the selected folder and types. */
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

/** Cancel an in-flight extraction request. */
function cancelExtraction(): void {
  extractionPromise?.cancel?.();
  extractionPromise = null;
  loading.value = false;
}

/** Load a saved inventory into the results table. */
async function loadInventory(inv: InventorySummary): Promise<void> {
  currentInventoryId.value = inv.id;
  currentInventoryGame.value = inv.game;
  hasExtraction.value = true;
  isCurrentTemp.value = false;
  allItems.value = (await GetInventoryItems(inv.id)) ?? [];
}

/** Save or rename an inventory from the name dialog. */
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

/** Delete a saved inventory and clear it if it is currently open. */
async function handleDelete(inv: InventorySummary): Promise<void> {
  await DeleteInventory(inv.id);
  if (currentInventoryId.value === inv.id) clearAll();
  await refresh();
}

/** Open the save/rename dialog for the current or given inventory. */
function openModal(mode: "save" | "rename", inv?: InventorySummary): void {
  nameModal.value = { open: true, mode, invId: currentInventoryId.value ?? inv?.id ?? null };
}

/** Show details for a selected inventory row. */
async function loadSelectedRow(row: InventoryItemRow): Promise<void> {
  selectedRow.value = row;
  itemDetailsOpen.value = true;
}

/** Handle a UTable row select event. */
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
  <div class="flex h-full min-h-0 flex-1 flex-col gap-2 overflow-hidden p-3">
    <div class="flex shrink-0 flex-wrap items-center justify-between gap-2">
      <div>
        <h1 class="text-lg font-semibold">Inventory</h1>
        <p class="text-xs text-muted">Extract and browse script objects</p>
      </div>
      <div class="text-xs text-muted">
        {{ gameInstallPath || "Install path not set" }}
      </div>
    </div>

    <UAccordion
      :items="[{ label: 'Extract', icon: 'i-lucide-folder-search', value: 'extract' }]"
      :default-value="hasExtraction ? undefined : 'extract'"
      :unmount-on-hide="false"
      class="shrink-0"
    >
      <template #body>
        <div class="space-y-2 pb-2">
          <FileSelector
            v-model="file"
            label="Folder to extract"
            dialog-title="Select a folder"
            mode="folder"
            placeholder="Mod or game folder"
          />
          <UFormField label="Object types" help="gfx/, gui/, and music/ types are not supported.">
            <UFieldGroup class="w-full">
              <USelect
                v-model="selectedTypes"
                :items="supportedTypes"
                multiple
                :disabled="typesDisabled"
                class="min-w-64 flex-1"
                size="sm"
              />
              <UButton
                label="All"
                color="neutral"
                variant="outline"
                size="sm"
                :disabled="typesDisabled"
                @click="selectedTypes = [...supportedTypes]"
              />
              <UButton
                label="None"
                color="neutral"
                variant="outline"
                size="sm"
                :disabled="typesDisabled || !selectedTypes.length"
                @click="selectedTypes = []"
              />
            </UFieldGroup>
          </UFormField>
          <div class="flex justify-between gap-2">
            <UButton label="Clear" color="error" variant="ghost" size="sm" :disabled="loading" @click="clearAll" />
            <UButton
              :loading="loading"
              :disabled="extractDisabled"
              :label="loading ? 'Cancel' : 'Extract'"
              size="sm"
              @click="loading ? cancelExtraction() : doExtract()"
            />
          </div>
        </div>
      </template>
    </UAccordion>

    <UAlert
      v-if="extractionErrors.length"
      color="warning"
      variant="subtle"
      class="shrink-0"
      :title="`Errors (${extractionErrors.length})`"
    />

    <div v-if="savedInventories.length" class="shrink-0 overflow-x-auto">
      <div class="flex min-w-max gap-3 p-1">
        <div v-for="inv in savedInventories" :key="inv.id" class="inventory-card-width">
          <InventoryCard
            :inv="inv"
            :active="currentInventoryId === inv.id"
            @load="loadInventory"
            @rename="openModal('rename', $event)"
            @delete="handleDelete"
          />
        </div>
      </div>
    </div>

    <div class="min-h-0 flex-1 overflow-hidden rounded-lg border border-default">
      <template v-if="hasExtraction">
        <div class="flex items-center justify-between border-b border-default px-3 py-2">
          <div class="text-sm font-medium">{{ allItems.length.toLocaleString() }} items</div>
          <UButton v-if="isCurrentTemp" label="Save Inventory" size="sm" @click="openModal('save')" />
        </div>
        <div class="h-[calc(100%-5.5rem)] overflow-auto">
          <UTable :data="pagedItems" :columns="itemColumns" @select="selectInventoryRow">
            <template #type-cell="{ row }">
              <span class="font-medium text-highlighted">{{ row.original.type }}</span>
            </template>
            <template #lines-cell="{ row }">
              {{ row.original.lineStart }} - {{ row.original.lineEnd }}
            </template>
          </UTable>
        </div>
        <div class="flex items-center justify-end gap-3 border-t border-default px-3 py-2">
          <span class="text-xs text-muted">Page {{ currentPage }} / {{ totalPages }}</span>
          <UPagination
            v-model:page="currentPage"
            :items-per-page="rowsPerPage"
            :total="allItems.length"
            color="neutral"
            active-color="primary"
            size="sm"
          />
        </div>
      </template>
      <UEmpty
        v-else
        class="h-full"
        icon="i-lucide-folder-search"
        title="No inventory"
        description="Select a path and types, then Extract."
      />
    </div>

    <UModal v-model:open="itemDetailsOpen" title="Item Details" :ui="{ content: 'sm:max-w-4xl' }">
      <template #body>
        <InventoryDetails
          :inventory-id="currentInventoryId"
          :item-type="selectedRow?.type ?? null"
          :item-key="selectedRow?.key ?? null"
          :row="selectedRow"
          :game="currentGame"
        />
      </template>
    </UModal>

    <InventoryNameDialog
      v-model="nameModal.open"
      :mode="nameModal.mode"
      :initial-name="nameModalInitialName"
      @save="handleNameModalSave"
    />
  </div>
</template>
