<script setup lang="ts">
/**
 * Conflict monitor: GetOverrides. FIOS/LIOS is computed in Go.
 * Filters live in UTable column state (overlay, kind, origin, rule).
 */
import { computed, shallowRef, useTemplateRef } from "vue";
import { useRoute } from "vue-router";
import { useQuery } from "@pinia/colada";
import { getFacetedUniqueValues, type ColumnFiltersState } from "@tanstack/table-core";
import type { TableColumn } from "@nuxt/ui";
import { GetOverrides } from "@services/viewsservice";
import type { OverrideRow } from "@services/internal/views/models";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";
import { useOpenInIde } from "../composables/useOpenInIde";
import { useLiveEnabled } from "../composables/useSessionQuery";

defineOptions({ name: "ConflictPage" });

const route = useRoute();
const { openInIde } = useOpenInIde();
const workspaceId = computed(() => String(route.params.id ?? ""));
const live = useLiveEnabled(workspaceId);
const globalFilter = shallowRef("");
const columnFilters = shallowRef<ColumnFiltersState>([
  { id: "overlay", value: false },
]);
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

const overlayItems = [
  { label: "Conflicts", value: false },
  { label: "Vanilla overlays", value: true },
];

const columns: TableColumn<OverrideRow>[] = [
  { accessorKey: "overlay", header: "Scope", filterFn: "equals" },
  { accessorKey: "kind", header: "Kind", filterFn: "equals" },
  { accessorKey: "name", header: "Name" },
  { accessorKey: "rule", header: "Rule", filterFn: "equals" },
  {
    id: "origin",
    header: "Origin",
    accessorFn: (row) =>
      (row.sites ?? []).map((s) => s.originName || "Vanilla"),
    getUniqueValues: (row) =>
      (row.sites ?? []).map((s) => s.originName || "Vanilla"),
    filterFn: (row, id, value) => {
      if (value == null || value === "") return true;
      return (row.getValue(id) as string[]).includes(value as string);
    },
  },
  { id: "winner", header: "Winner" },
  { id: "sites", header: "Sites" },
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
function overlayFilter(c: { getFilterValue: () => unknown }) {
  return c.getFilterValue() === true;
}

/** Open a definition site in the workspace IDE. */
function open(file: string, line: number): void {
  void openInIde(workspaceId.value, file, line);
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

    <div class="min-h-0 flex-1 overflow-auto p-2">
      <UTable
        ref="table"
        :column-filters="columnFilters"
        :global-filter="globalFilter"
        :data="rows"
        :columns="columns"
        :loading="isPending"
        :faceted-options="{ getFacetedUniqueValues: getFacetedUniqueValues() }"
        sticky="header"
        empty="No overlapping definitions across workspace mods."
        :get-row-id="(row: OverrideRow) => `${row.kind}:${row.name}`"
        @update:column-filters="(v?: ColumnFiltersState) => { if (v) columnFilters = v }"
        @update:global-filter="(v?: string) => { globalFilter = v ?? '' }"
      >
        <template #overlay-header="{ column }">
          <USelect :model-value="overlayFilter(column)" :items="overlayItems"
            value-key="value" size="xs" class="w-44"
            @update:model-value="column.setFilterValue($event)" />
        </template>
        <template #kind-header="{ column }">
          <USelect :model-value="filterString(column)" :items="facetItems('kind')"
            placeholder="Kind" size="xs" class="w-36"
            @update:model-value="column.setFilterValue($event)" />
        </template>
        <template #origin-header="{ column }">
          <USelect :model-value="filterString(column)" :items="facetItems('origin')"
            placeholder="Origin" size="xs" class="w-40"
            @update:model-value="column.setFilterValue($event)" />
        </template>
        <template #rule-header="{ column }">
          <USelect :model-value="filterString(column)" :items="facetItems('rule')"
            placeholder="Rule" size="xs" class="w-28"
            @update:model-value="column.setFilterValue($event)" />
        </template>
        <template #overlay-cell="{ row }">
          <UBadge :label="row.original.overlay ? 'overlay' : 'conflict'"
            :color="row.original.overlay ? 'neutral' : 'warning'"
            variant="subtle" size="xs" />
        </template>
        <template #rule-cell="{ row }">
          <UBadge :label="row.original.rule"
            :color="row.original.rule === 'FIOS' ? 'warning' : 'info'"
            variant="subtle" size="xs" />
        </template>
        <template #winner-cell="{ row }">
          <UBadge :label="row.original.winnerName || 'Vanilla'"
            :color="row.original.winner ? 'primary' : 'neutral'"
            variant="subtle" size="xs" />
        </template>
        <template #sites-cell="{ row }">
          <div class="flex flex-wrap gap-1">
            <UButton
              v-for="(site, i) in row.original.sites ?? []"
              :key="i"
              :label="`${site.originName || 'Vanilla'}:${site.line}`"
              size="xs"
              :color="site.origin ? 'primary' : 'neutral'"
              variant="subtle"
              @click="open(site.file, site.line)"
            />
          </div>
        </template>
      </UTable>
    </div>
  </div>
</template>
