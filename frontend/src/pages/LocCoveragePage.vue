<script setup lang="ts">
/**
 * Loc coverage: GetLocCoverage. Filters live in UTable column state.
 */
import { computed, ref, shallowRef, watch, useTemplateRef } from "vue";
import { useRoute } from "vue-router";
import { useQuery } from "@pinia/colada";
import { getFacetedUniqueValues, type ColumnFiltersState, type RowSelectionState } from "@tanstack/table-core";
import type { TableColumn } from "@nuxt/ui";
import { GetLocCoverage } from "@services/viewsservice";
import type { LocIssueRow } from "@services/internal/views/models";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";
import DetailPane from "../components/DetailPane.vue";
import OriginBadge, { PILL_UI } from "../components/OriginBadge.vue";
import OriginSelectMenu from "../components/OriginSelectMenu.vue";
import { useOpenInIde } from "../composables/useOpenInIde";
import { useLiveEnabled } from "../composables/useLiveEnabled";
import { originHex, originHexByOriginId } from "../ide/rootDecorations";
import { useWorkspaceStore } from "../stores/workspace";

defineOptions({ name: "LocCoveragePage" });

const route = useRoute();
const { openInIde } = useOpenInIde();
const ws = useWorkspaceStore();
const workspaceId = computed(() => String(route.params.id ?? ""));
const live = useLiveEnabled(workspaceId);
const lang = shallowRef("");
const columnFilters = shallowRef<ColumnFiltersState>([]);
const selected = shallowRef<LocIssueRow | null>(null);
const rowSelection = shallowRef<RowSelectionState>({});
const originIds = ref<string[]>([]);
const table = useTemplateRef<{
  tableApi?: {
    getColumn: (id: string) =>
      | {
          getFilterValue: () => unknown;
          setFilterValue: (v: unknown) => void;
          getFacetedUniqueValues: () => Map<unknown, number>;
        }
      | undefined;
  };
}>("table");

const liveMods = computed(() => ws.workspaceMods.filter((m) => !m.isBroken && m.path));
const originItems = computed(() =>
  liveMods.value.map((m, i) => ({
    id: m.id,
    label: m.name,
    color: originHex({ kind: "mod", path: m.path, color: m.color, wrapIndex: i }),
    thumbnail: ws.thumbUrls[m.id],
  })),
);

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

const current = computed(() => coverage.value.find((c) => c.language === lang.value) ?? null);

const columns: TableColumn<LocIssueRow>[] = [
  { accessorKey: "kind", header: "Kind", filterFn: "equals" },
  {
    accessorKey: "origin",
    header: "Origin",
    filterFn: (row, _id, value) => {
      if (!Array.isArray(value) || !value.length) return true;
      return value.includes(row.original.origin);
    },
  },
  { accessorKey: "rel", header: "File", filterFn: "includesString" },
  { accessorKey: "key", header: "Key" },
];

function setCol(id: string, value: unknown): void {
  const rest = columnFilters.value.filter((f) => f.id !== id);
  columnFilters.value = value == null || value === "" ? rest : [...rest, { id, value }];
}
function colString(id: string): string | undefined {
  const hit = columnFilters.value.find((f) => f.id === id);
  return hit?.value == null ? undefined : String(hit.value);
}
watch(originIds, (ids) => {
  const narrowed = ids.length > 0 && ids.length < originItems.value.length;
  setCol("origin", narrowed ? ids : undefined);
});

function facetItems(id: string): { label: string; value: string }[] {
  const col = table.value?.tableApi?.getColumn(id);
  return [...(col?.getFacetedUniqueValues() ?? new Map())].map(([v, n]) => ({
    label: `${String(v)} (${n})`,
    value: String(v),
  }));
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
  return `${row.kind}:${row.key}:${row.path}:${row.line}`;
}

/** Select one loc issue for the detail pane. */
function onRowSelect(_e: Event, row: { original: LocIssueRow; id: string }): void {
  selected.value = row.original;
  rowSelection.value = { [row.id]: true };
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <WorkspaceToolBar :workspace-id="workspaceId">
      <template #trailing>
        <LanguageHealthStrip v-if="workspaceId" :workspace-id="workspaceId" />
      </template>
    </WorkspaceToolBar>
    <UAlert v-if="error" color="error" variant="subtle" :description="error" class="m-2" />

    <UDashboardToolbar class="px-2 sm:px-2">
      <template #left>
        <USelect v-model="lang" :items="langItems" value-key="value" size="md" class="w-44" />
        <USelect
          :model-value="colString('kind')"
          :items="facetItems('kind')"
          placeholder="Kind"
          size="md"
          class="w-36"
          @update:model-value="setCol('kind', $event)"
        />
        <OriginSelectMenu v-model="originIds" :items="originItems" />
        <UInput
          :model-value="colString('rel') ?? ''"
          placeholder="File contains"
          size="md"
          class="w-48"
          @update:model-value="setCol('rel', $event || undefined)"
        />
      </template>
    </UDashboardToolbar>

    <UDashboardGroup
      storage="local"
      storage-key="pmt-loc"
      class="relative! inset-auto! min-h-0 min-w-0 flex-1 overflow-hidden"
    >
      <UDashboardPanel id="loc-main" resizable :default-size="75" :min-size="50" :max-size="85" class="min-h-0!">
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
          @update:column-filters="
            (v?: ColumnFiltersState) => {
              if (v) columnFilters = v;
            }
          "
          @select="onRowSelect"
        >
          <template #kind-cell="{ row }">
            <UBadge :color="kindColor(row.original.kind)" variant="subtle" size="xs" :ui="PILL_UI">
              {{ row.original.kind }}
            </UBadge>
          </template>
          <template #origin-cell="{ row }">
            <OriginBadge
              :label="row.original.originName || row.original.origin || ''"
              :hex="originHexByOriginId(row.original.origin ?? '')"
            />
          </template>
        </UTable>
      </UDashboardPanel>
      <UDashboardPanel id="loc-detail" class="min-h-0!">
        <DetailPane
          :workspace-id="workspaceId"
          :title="selected?.key"
          :file="selected?.path"
          :rel="selected?.rel"
          :line="selected?.line"
        >
          <template v-if="selected && (selected.originName || selected.origin)" #origin>
            <OriginBadge
              :label="selected.originName || selected.origin || ''"
              :hex="originHexByOriginId(selected.origin ?? '')"
            />
          </template>
          <template v-if="selected" #badges>
            <UBadge :color="kindColor(selected.kind)" variant="subtle" size="xs" :ui="PILL_UI">
              {{ selected.kind }}
            </UBadge>
          </template>
          <p v-if="selected?.value" class="text-xs text-default">“{{ selected.value }}”</p>
        </DetailPane>
      </UDashboardPanel>
    </UDashboardGroup>
  </div>
</template>
