<script setup lang="ts">
/**
 * Loc coverage: GetLocCoverage / LookupLoc. Filters live in UTable column state.
 */
import { computed, shallowRef, watch, useTemplateRef } from "vue";
import { useRoute } from "vue-router";
import { useQuery } from "@pinia/colada";
import { getFacetedUniqueValues, type ColumnFiltersState } from "@tanstack/table-core";
import type { TableColumn } from "@nuxt/ui";
import { GetLocCoverage, LookupLoc } from "@services/viewsservice";
import type { LocIssueRow } from "@services/internal/views/models";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";
import { useOpenInIde } from "../composables/useOpenInIde";
import { useLiveEnabled } from "../composables/useSessionQuery";

defineOptions({ name: "LocCoveragePage" });

const route = useRoute();
const { openInIde } = useOpenInIde();
const workspaceId = computed(() => String(route.params.id ?? ""));
const live = useLiveEnabled(workspaceId);
const lang = shallowRef("");
const lookupKey = shallowRef("");
const lookupSubmitted = shallowRef("");
const columnFilters = shallowRef<ColumnFiltersState>([]);
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

const { data: lookup } = useQuery({
  key: () => ["session", "loc-lookup", workspaceId.value, lookupSubmitted.value],
  query: () => LookupLoc(workspaceId.value, lookupSubmitted.value),
  enabled: () => live.value && !!lookupSubmitted.value,
});

const coverage = computed(() => coverageData.value ?? []);
const error = computed(() => loadError.value?.message ?? "");

watch(coverage, (list) => {
  if (!lang.value && list.length) lang.value = list[0]!.language;
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
  { accessorKey: "line", header: "Line" },
  { accessorKey: "value", header: "Value" },
  { id: "open", header: "" },
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

/** Resolve one loc key via Go. */
function runLookup(): void {
  lookupSubmitted.value = lookupKey.value.trim();
}

/** Open a loc site in the workspace IDE. */
function open(file?: string, line?: number): void {
  if (!file) return;
  void openInIde(workspaceId.value, file, line);
}

function kindColor(kind: string): "error" | "warning" | "info" {
  return kind === "missing" ? "error" : kind === "orphaned" ? "warning" : "info";
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
      <UInput v-model="lookupKey" placeholder="Lookup key" size="xs" class="w-56" />
      <UButton label="Lookup" size="xs" color="neutral" variant="ghost" @click="runLookup" />
      <template v-if="lookup">
        <span class="font-medium text-default">{{ lookup.key }}</span>
        <UBadge v-if="lookup.originName || lookup.origin"
          :label="lookup.originName || lookup.origin"
          :color="lookup.origin ? 'primary' : 'neutral'" variant="subtle" size="xs" />
        <span class="truncate text-muted">{{ lookup.text }}</span>
        <UButton v-if="lookup.file" label="Open" size="xs" color="neutral" variant="ghost"
          @click="open(lookup.file, lookup.line)" />
      </template>
    </div>

    <div class="min-h-0 flex-1 overflow-auto p-2">
      <UTable
        ref="table"
        :column-filters="columnFilters"
        @update:column-filters="(v?: ColumnFiltersState) => { if (v) columnFilters = v }"
        :data="current?.issues ?? []"
        :columns="columns"
        :loading="isPending"
        :faceted-options="{ getFacetedUniqueValues: getFacetedUniqueValues() }"
        sticky="header"
        empty="No loc issues for this filter."
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
        <template #open-cell="{ row }">
          <UButton v-if="row.original.file" label="Open" size="xs" color="neutral"
            variant="ghost" @click="open(row.original.file, row.original.line)" />
        </template>
      </UTable>
    </div>
  </div>
</template>
