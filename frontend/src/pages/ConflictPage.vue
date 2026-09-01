<script setup lang="ts">
/**
 * Conflict monitor: GetOverrides. FIOS/LIOS is computed in Go.
 * Filters live in UTable column state (overlay, kind, origin, rule).
 */
import { computed, shallowRef, useTemplateRef } from "vue";
import { useRoute } from "vue-router";
import { useQuery } from "@pinia/colada";
import {
  getFacetedUniqueValues,
  type ColumnFiltersState,
  type RowSelectionState,
} from "@tanstack/table-core";
import type { TableColumn } from "@nuxt/ui";
import { GetOverrides } from "@services/viewsservice";
import type { OverrideRow, OverrideSite } from "@services/internal/views/models";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";
import DetailPane from "../components/DetailPane.vue";
import { useOpenInIde } from "../composables/useOpenInIde";
import { useLiveEnabled } from "../composables/useSessionQuery";
import { useWorkspaceStore } from "../stores/workspace";

defineOptions({ name: "ConflictPage" });

const route = useRoute();
const { openInIde } = useOpenInIde();
const ws = useWorkspaceStore();
const workspaceId = computed(() => String(route.params.id ?? ""));
const live = useLiveEnabled(workspaceId);
const originFallback = computed(() =>
  ws.gameName(ws.activeWorkspace?.gameId ?? ws.currentGameId),
);
const vanilla = computed(() => ws.originVanilla);
const scopeItems = [
  { label: "Conflicts", value: "conflicts" },
  { label: "Overrides", value: "overrides" },
];
const scope = computed({
  get: (): string | number => {
    const hit = columnFilters.value.find((f) => f.id === "overlay");
    return hit?.value === true ? "overrides" : "conflicts";
  },
  set: (v: string | number) => {
    const overlay = v === "overrides";
    columnFilters.value = [
      ...columnFilters.value.filter((f) => f.id !== "overlay"),
      { id: "overlay", value: overlay },
    ];
  },
});

/** Mod origin names only (no vanilla / game title); Set-deduped. */
function modOriginNames(row: OverrideRow): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const s of row.sites ?? []) {
    const origin = s.origin || "";
    const name = s.originName || "";
    if (!origin || origin === vanilla.value) continue;
    if (!name || name === originFallback.value) continue;
    if (seen.has(name)) continue;
    seen.add(name);
    out.push(name);
  }
  return out;
}
const globalFilter = shallowRef("");
const columnFilters = shallowRef<ColumnFiltersState>([
  { id: "overlay", value: false },
]);
const selected = shallowRef<OverrideRow | null>(null);
const rowSelection = shallowRef<RowSelectionState>({});
const table = useTemplateRef<{ tableApi?: { getColumn: (id: string) => {
  getFilterValue: () => unknown
  setFilterValue: (v: unknown) => void
  getFacetedUniqueValues: () => Map<unknown, number>
} | undefined } }>("table");

const {
  data: rowsData,
  error: loadError,
  isPending,
} = useQuery({
  key: () => ["session", "overrides", workspaceId.value],
  query: () => GetOverrides(workspaceId.value),
  enabled: () => live.value,
});

const rows = computed(() => rowsData.value ?? []);
const error = computed(() => loadError.value?.message ?? "");

const columns: TableColumn<OverrideRow>[] = [
  { accessorKey: "overlay", header: "Scope", filterFn: "equals" },
  { accessorKey: "kind", header: "Kind", filterFn: "equals" },
  { accessorKey: "name", header: "Name" },
  { accessorKey: "rule", header: "Rule", filterFn: "equals" },
  {
    id: "origin",
    header: "Origin",
    accessorFn: (row) => modOriginNames(row),
    getUniqueValues: (row) => modOriginNames(row),
    filterFn: (row, id, value) => {
      if (value == null || value === "") return true;
      return (row.getValue(id) as string[]).includes(value as string);
    },
  },
  { id: "winner", header: "Winner" },
];

function facetItems(id: string): { label: string; value: string }[] {
  const col = table.value?.tableApi?.getColumn(id);
  return [...(col?.getFacetedUniqueValues() ?? new Map())].map(([v, n]) => ({
    label: `${String(v)} (${n})`,
    value: String(v),
  }));
}
function filterString(c: { getFilterValue: () => unknown }) {
  const v = c.getFilterValue();
  return v == null ? undefined : String(v);
}
function originFacetItems(): { label: string; value: string }[] {
  const counts = new Map<string, number>();
  for (const row of rows.value) {
    for (const name of new Set(modOriginNames(row))) {
      counts.set(name, (counts.get(name) ?? 0) + 1);
    }
  }
  return [...counts].map(([v, n]) => ({ label: `${v} (${n})`, value: v }));
}

/** Open a definition site in the workspace IDE. */
function open(file: string, line: number): void {
  void openInIde(workspaceId.value, file, line);
}

/** Winning definition site for the selected row. */
const winnerSite = computed((): OverrideSite | undefined => {
  const row = selected.value;
  if (!row) return undefined;
  return (row.sites ?? []).find((s) => s.origin === row.winner)
    ?? row.sites?.[0];
});

/** Select one override for the detail pane. */
function onRowSelect(
  _e: Event,
  row: { original: OverrideRow; id: string },
): void {
  selected.value = row.original;
  rowSelection.value = { [row.id]: true };
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <WorkspaceToolBar :workspace-id="workspaceId" title="Conflicts" active="conflicts">
      <template #trailing>
        <UInput v-model="globalFilter" icon="i-lucide-search" placeholder="Filter"
          size="xs" class="w-48" />
        <LanguageHealthStrip v-if="workspaceId" :workspace-id="workspaceId" />
      </template>
    </WorkspaceToolBar>
    <UAlert v-if="error" color="error" variant="subtle" :description="error" class="m-2" />

    <UDashboardGroup
      storage="local"
      storage-key="pmt-conflicts"
      class="relative! inset-auto! min-h-0 min-w-0 flex-1 overflow-hidden"
    >
      <UDashboardPanel
        id="conflicts-main"
        resizable
        :default-size="75"
        :min-size="50"
        :max-size="85"
        class="min-h-0!"
      >
        <div class="flex shrink-0 items-center gap-3 border-b border-default px-2 py-1">
          <UTabs
            v-model="scope"
            :items="scopeItems"
            :content="false"
            variant="pill"
            size="sm"
          />
        </div>
        <UTable
          ref="table"
          class="h-full"
          :column-filters="columnFilters"
          :global-filter="globalFilter"
          :row-selection="rowSelection"
          :row-selection-options="{ enableMultiRowSelection: false }"
          :data="rows"
          :columns="columns"
          :loading="isPending"
          :faceted-options="{ getFacetedUniqueValues: getFacetedUniqueValues() }"
          sticky="header"
          empty="No overlapping definitions across workspace mods."
          :get-row-id="(row: OverrideRow) => `${row.kind}:${row.name}`"
          @update:column-filters="(v?: ColumnFiltersState) => { if (v) columnFilters = v }"
          @update:global-filter="(v?: string) => { globalFilter = v ?? '' }"
          @select="onRowSelect"
        >
          <template #overlay-header>
            <span class="text-xs text-muted">Scope</span>
          </template>
          <template #kind-header="{ column }">
            <USelect :model-value="filterString(column)" :items="facetItems('kind')"
              placeholder="Kind" size="xs" class="w-36"
              @update:model-value="column.setFilterValue($event)" />
          </template>
          <template #origin-header="{ column }">
            <USelect :model-value="filterString(column)" :items="originFacetItems()"
              placeholder="Origin" size="xs" class="w-40"
              @update:model-value="column.setFilterValue($event)" />
          </template>
          <template #rule-header="{ column }">
            <USelect :model-value="filterString(column)" :items="facetItems('rule')"
              placeholder="Rule" size="xs" class="w-28"
              @update:model-value="column.setFilterValue($event)" />
          </template>
          <template #overlay-cell="{ row }">
            <UBadge :label="row.original.overlay ? 'override' : 'conflict'"
              :color="row.original.overlay ? 'neutral' : 'warning'"
              variant="subtle" size="xs" />
          </template>
          <template #rule-cell="{ row }">
            <UBadge :label="row.original.rule"
              :color="row.original.rule === 'FIOS' ? 'warning' : 'info'"
              variant="subtle" size="xs" />
          </template>
          <template #winner-cell="{ row }">
            <UBadge :label="row.original.winnerName || originFallback"
              :color="row.original.winner ? 'primary' : 'neutral'"
              variant="subtle" size="xs" />
          </template>
        </UTable>
      </UDashboardPanel>
      <UDashboardPanel id="conflicts-detail" class="min-h-0!">
        <DetailPane
          :workspace-id="workspaceId"
          :title="selected?.name"
          :file="winnerSite?.file"
          :rel="winnerSite?.rel"
          :line="winnerSite?.line"
        >
          <template v-if="selected" #badges>
            <UBadge :label="selected.overlay ? 'override' : 'conflict'"
              :color="selected.overlay ? 'neutral' : 'warning'"
              variant="subtle" size="xs" />
            <UBadge :label="selected.rule"
              :color="selected.rule === 'FIOS' ? 'warning' : 'info'"
              variant="subtle" size="xs" />
            <UBadge :label="selected.winnerName || originFallback"
              :color="selected.winner ? 'primary' : 'neutral'"
              variant="subtle" size="xs" />
          </template>
          <div class="flex flex-col gap-1">
            <UButton
              v-for="(site, i) in selected?.sites ?? []"
              :key="i"
              :label="`${site.originName || originFallback} · ${site.rel || site.file}:${site.line + 1}`"
              size="xs"
              :color="site.origin === selected?.winner ? 'primary' : 'neutral'"
              variant="subtle"
              class="justify-start"
              @click="open(site.file, site.line)"
            />
          </div>
        </DetailPane>
      </UDashboardPanel>
    </UDashboardGroup>
  </div>
</template>
