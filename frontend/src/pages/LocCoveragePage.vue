<script setup lang="ts">
/**
 * Loc coverage: GetLocCoverage. Filters live in UTable column state.
 */
import { computed, shallowRef, watch, useTemplateRef } from "vue";
import { useRoute } from "vue-router";
import { useQuery } from "@pinia/colada";
import {
  getFacetedUniqueValues,
  type ColumnFiltersState,
  type RowSelectionState,
} from "@tanstack/table-core";
import type { TableColumn } from "@nuxt/ui";
import { GetLocCoverage } from "@services/viewsservice";
import type { LocIssueRow } from "@services/internal/views/models";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";
import DetailPane from "../components/DetailPane.vue";
import { useOpenInIde } from "../composables/useOpenInIde";
import { useLiveEnabled } from "../composables/useSessionQuery";

defineOptions({ name: "LocCoveragePage" });

const route = useRoute();
const { openInIde } = useOpenInIde();
const workspaceId = computed(() => String(route.params.id ?? ""));
const live = useLiveEnabled(workspaceId);
const lang = shallowRef("");
const columnFilters = shallowRef<ColumnFiltersState>([]);
const selected = shallowRef<LocIssueRow | null>(null);
const rowSelection = shallowRef<RowSelectionState>({});
const table = useTemplateRef<{ tableApi?: {
  getColumn: (id: string) => {
    getFilterValue: () => unknown
    setFilterValue: (v: unknown) => void
    getFacetedUniqueValues: () => Map<unknown, number>
  } | undefined
} }>("table");

const {
  data: coverageData,
  error: loadError,
  isPending,
} = useQuery({
  key: () => ["session", "loc", workspaceId.value],
  query: () => GetLocCoverage(workspaceId.value),
  enabled: () => live.value,
});

const coverage = computed(() => coverageData.value ?? []);
const error = computed(() => loadError.value?.message ?? "");

watch(coverage, (list) => {
  if (!lang.value && list.length) lang.value = list[0]!.language;
});
watch(lang, () => {
  selected.value = null;
  rowSelection.value = {};
});

const langItems = computed(() =>
  coverage.value.map((c) => ({
    label: `${c.language} (${c.defined})`,
    value: c.language,
  })),
);

const current = computed(
  () => coverage.value.find((c) => c.language === lang.value) ?? null,
);

const columns: TableColumn<LocIssueRow>[] = [
  { accessorKey: "kind", header: "Kind", filterFn: "equals" },
  { accessorKey: "originName", header: "Origin", filterFn: "equals" },
  { accessorKey: "rel", header: "File", filterFn: "includesString" },
  { accessorKey: "key", header: "Key" },
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

/** Open a loc site in the workspace IDE. */
function open(file?: string, line?: number): void {
  if (!file) return;
  void openInIde(workspaceId.value, file, line);
}

function kindColor(kind: string): "error" | "warning" | "info" {
  return kind === "missing" ? "error" : kind === "orphaned" ? "warning" : "info";
}

/** Stable table row id for single selection. */
function locRowId(row: LocIssueRow): string {
  return `${row.kind}:${row.key}:${row.file}:${row.line}`;
}

/** Select one loc issue for the detail pane. */
function onRowSelect(_e: Event, row: { original: LocIssueRow; id: string }): void {
  selected.value = row.original;
  rowSelection.value = { [row.id]: true };
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <WorkspaceToolBar :workspace-id="workspaceId" title="Loc Coverage" active="loc-coverage">
      <template #trailing>
        <LanguageHealthStrip v-if="workspaceId" :workspace-id="workspaceId" />
      </template>
    </WorkspaceToolBar>
    <UAlert v-if="error" color="error" variant="subtle" :description="error" class="m-2" />

    <div class="flex shrink-0 items-center gap-2 border-b border-default px-2 py-1">
      <USelect v-model="lang" :items="langItems" value-key="value" size="xs" class="w-44" />
    </div>

    <UDashboardGroup
      storage="local"
      storage-key="pmt-loc"
      class="relative! inset-auto! min-h-0 min-w-0 flex-1 overflow-hidden"
    >
      <UDashboardPanel
        id="loc-main"
        resizable
        :default-size="75"
        :min-size="50"
        :max-size="85"
        class="min-h-0!"
      >
        <UTable
          ref="table"
          class="h-full"
          :column-filters="columnFilters"
          :row-selection="rowSelection"
          :row-selection-options="{ enableMultiRowSelection: false }"
          :data="current?.issues ?? []"
          :columns="columns"
          :loading="isPending"
          :faceted-options="{ getFacetedUniqueValues: getFacetedUniqueValues() }"
          :get-row-id="locRowId"
          sticky="header"
          empty="No loc issues for this filter."
          @update:column-filters="(v?: ColumnFiltersState) => { if (v) columnFilters = v }"
          @select="onRowSelect"
        >
          <template #kind-header="{ column }">
            <USelect :model-value="filterString(column)" :items="facetItems('kind')"
              placeholder="Kind" size="xs" class="w-36"
              @update:model-value="column.setFilterValue($event)" />
          </template>
          <template #originName-header="{ column }">
            <USelect :model-value="filterString(column)" :items="facetItems('originName')"
              placeholder="Origin" size="xs" class="w-40"
              @update:model-value="column.setFilterValue($event)" />
          </template>
          <template #rel-header="{ column }">
            <UInput :model-value="filterString(column) ?? ''" placeholder="File contains"
              size="xs" class="w-40"
              @update:model-value="column.setFilterValue($event || undefined)" />
          </template>
          <template #kind-cell="{ row }">
            <UBadge :color="kindColor(row.original.kind)" variant="subtle" size="xs">
              {{ row.original.kind }}
            </UBadge>
          </template>
        </UTable>
      </UDashboardPanel>
      <UDashboardPanel id="loc-detail" class="min-h-0!">
        <DetailPane
          :workspace-id="workspaceId"
          :title="selected?.key"
          :file="selected?.file"
          :rel="selected?.rel"
          :line="selected?.line"
        >
          <template v-if="selected" #badges>
            <UBadge :color="kindColor(selected.kind)" variant="subtle" size="xs">
              {{ selected.kind }}
            </UBadge>
            <UBadge
              v-if="selected.originName || selected.origin"
              :label="selected.originName || selected.origin"
              :color="selected.origin ? 'primary' : 'neutral'"
              variant="subtle"
              size="xs"
            />
          </template>
          <p v-if="selected?.value" class="text-xs text-default">
            “{{ selected.value }}”
          </p>
        </DetailPane>
      </UDashboardPanel>
    </UDashboardGroup>
  </div>
</template>
