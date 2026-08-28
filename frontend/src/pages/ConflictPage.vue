<script setup lang="ts">
/**
 * Conflict monitor: GetOverrides only. FIOS/LIOS is computed in Go.
 */
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { GetOverrides } from "@services/languagemodelservice";
import type { OverrideRow } from "@services/internal/model/models";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";
import { useWorkspaceStore } from "../stores/workspace";
import { useOpenInIde } from "../composables/useOpenInIde";

const route = useRoute();
const ws = useWorkspaceStore();
const { openInIde } = useOpenInIde();

const workspaceId = computed(() => String(route.params.id ?? ""));
const rows = ref<OverrideRow[]>([]);
const query = ref("");
const loading = ref(false);
const error = ref("");

const columns = [
  { accessorKey: "kind", header: "Kind" },
  { accessorKey: "name", header: "Name" },
  { accessorKey: "rule", header: "Rule" },
  { id: "winner", header: "Winner" },
  { id: "sites", header: "Sites" },
];

/** Fetch override rows from Go. */
async function load(): Promise<void> {
  if (!workspaceId.value) return;
  loading.value = true;
  error.value = "";
  try {
    const ready = await ws.ensureReady();
    if (!ready) {
      error.value = "Language session is not live. Use Rescan in the toolbar.";
      rows.value = [];
      return;
    }
    rows.value = (await GetOverrides(workspaceId.value)) ?? [];
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
    rows.value = [];
  } finally {
    loading.value = false;
  }
}

/** Open a definition site in the workspace IDE. */
function open(file: string, line: number): void {
  void openInIde(workspaceId.value, file, line);
}

watch(workspaceId, load, { immediate: true });
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <WorkspaceToolBar
      :workspace-id="workspaceId"
      title="Conflicts"
      active="conflicts"
    >
      <template #trailing>
        <UInput
          v-model="query"
          icon="i-lucide-search"
          placeholder="Filter"
          size="xs"
          class="w-48"
        />
        <LanguageHealthStrip
          v-if="workspaceId"
          :workspace-id="workspaceId"
        />
      </template>
    </WorkspaceToolBar>

    <UAlert
      v-if="error"
      color="error"
      variant="subtle"
      :description="error"
      class="m-2"
    />

    <div class="min-h-0 flex-1 overflow-auto p-2">
      <UTable
        :data="rows"
        :columns="columns"
        :loading="loading"
        :global-filter="query"
        sticky="header"
        empty="No overlapping definitions."
        :get-row-id="(row: OverrideRow) => `${row.kind}:${row.name}`"
      >
        <template #winner-cell="{ row }">
          <span class="text-sm text-default">
            {{ row.original.winner || "vanilla" }}
          </span>
        </template>
        <template #sites-cell="{ row }">
          <div class="flex flex-col items-start gap-0.5">
            <UButton
              v-for="(site, i) in row.original.sites ?? []"
              :key="i"
              :label="`${site.origin || 'vanilla'}:${site.line}`"
              size="xs"
              color="neutral"
              variant="ghost"
              @click="open(site.file, site.line)"
            />
          </div>
        </template>
      </UTable>
    </div>
  </div>
</template>
