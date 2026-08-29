<script setup lang="ts">
/**
 * Conflict monitor: GetOverrides with overview filters. FIOS/LIOS is computed in Go.
 */
import { computed, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useRoute } from "vue-router";
import type { SplitterItem } from "@nuxt/ui";
import { GetOverrides } from "@services/languagemodelservice";
import type { OverrideRow } from "@services/internal/model/models";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";
import IssueOverviewPane from "../components/issues/IssueOverviewPane.vue";
import type {
  OverviewChip,
  OverviewNode,
} from "../components/issues/IssueOverviewPane.vue";
import { useWorkspaceStore } from "../stores/workspace";
import { useOpenInIde } from "../composables/useOpenInIde";
import { originHex } from "../ide/rootDecorations";

type Bucket = "conflict" | "overlay";

const route = useRoute();
const ws = useWorkspaceStore();
const { workspaceMods } = storeToRefs(ws);
const { openInIde } = useOpenInIde();

const workspaceId = computed(() => String(route.params.id ?? ""));
const rows = ref<OverrideRow[]>([]);
const query = ref("");
const kindFilter = ref<string | undefined>();
const ruleFilter = ref<string | undefined>();
const modOrigin = ref<string | undefined>();
const bucket = ref<Bucket>("conflict");
const loading = ref(false);
const error = ref("");

const liveMods = computed(() =>
  workspaceMods.value.filter((m) => !m.isBroken && m.path),
);

const bucketItems: { label: string; value: Bucket }[] = [
  { label: "Conflicts", value: "conflict" },
  { label: "Vanilla overlays", value: "overlay" },
];

const ruleItems = [
  { label: "LIOS", value: "LIOS" },
  { label: "FIOS", value: "FIOS" },
];

const modItems = computed(() =>
  liveMods.value.map((m) => ({ label: m.name, origin: m.id, path: m.path })),
);

const kindItems = computed(() => {
  const set = new Set<string>();
  for (const r of rows.value) {
    if (r.kind) set.add(r.kind);
  }
  return [...set].sort().map((k) => ({ label: k, value: k }));
});

const filteredRows = computed((): OverrideRow[] => {
  const q = query.value.trim().toLowerCase();
  return rows.value.filter((r) => {
    const overlay = !!r.overlay;
    if (bucket.value === "overlay" ? !overlay : overlay) return false;
    if (kindFilter.value && r.kind !== kindFilter.value) return false;
    if (ruleFilter.value && r.rule !== ruleFilter.value) return false;
    if (modOrigin.value) {
      const hit = (r.sites ?? []).some((s) => s.origin === modOrigin.value);
      if (!hit) return false;
    }
    if (q && !`${r.kind} ${r.name}`.toLowerCase().includes(q)) return false;
    return true;
  });
});

const chips = computed((): OverviewChip[] => {
  const byKind = new Map<string, number>();
  const base = rows.value.filter((r) => {
    const overlay = !!r.overlay;
    if (bucket.value === "overlay" ? !overlay : overlay) return false;
    if (ruleFilter.value && r.rule !== ruleFilter.value) return false;
    if (modOrigin.value) {
      const hit = (r.sites ?? []).some((s) => s.origin === modOrigin.value);
      if (!hit) return false;
    }
    return true;
  });
  for (const r of base) {
    byKind.set(r.kind, (byKind.get(r.kind) ?? 0) + 1);
  }
  return [...byKind.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, 12)
    .map(([id, count]) => ({
      id,
      label: id,
      count,
      color: "primary" as const,
    }));
});

const treeNodes = computed((): OverviewNode[] => {
  const by = new Map<string, number>();
  for (const r of filteredRows.value) {
    for (const s of r.sites ?? []) {
      if (!s.file || !s.origin) continue;
      by.set(s.origin, (by.get(s.origin) ?? 0) + 1);
    }
  }
  return [...by.entries()]
    .sort((a, b) => b[1] - a[1])
    .map(([origin, count]) => ({
      id: origin,
      label: liveMods.value.find((m) => m.id === origin)?.name ?? origin,
      count,
    }));
});

const columns = [
  { accessorKey: "kind", header: "Kind" },
  { accessorKey: "name", header: "Name" },
  { accessorKey: "rule", header: "Rule" },
  { id: "winner", header: "Winner" },
  { id: "sites", header: "Sites" },
];

const splitItems: SplitterItem[] = [
  {
    slot: "overview",
    minSize: 16,
    defaultSize: 22,
    class: "min-h-0 min-w-0 overflow-hidden border-r border-default",
  },
  { slot: "table", minSize: 40, defaultSize: 78, class: "min-h-0 min-w-0 overflow-hidden" },
];

const splitUi = {
  handle:
    "data-[orientation=horizontal]:w-px bg-border transition-colors " +
    "data-[state=hover]:bg-primary data-[state=drag]:bg-primary",
};

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

/** Display name for a site origin. */
function originLabel(origin: string): string {
  if (!origin) return "vanilla";
  return liveMods.value.find((m) => m.id === origin)?.name ?? origin;
}

function onChip(id: string | undefined): void {
  kindFilter.value = id;
}

function onNode(id: string | undefined): void {
  modOrigin.value = id;
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

    <div
      class="flex shrink-0 flex-wrap items-center gap-2 border-b border-default
        px-2 py-1"
    >
      <USelect
        v-model="bucket"
        :items="bucketItems"
        value-key="value"
        size="xs"
        class="w-44"
      />
      <USelectMenu
        v-model="kindFilter"
        :items="kindItems"
        value-key="value"
        placeholder="All kinds"
        size="xs"
        class="w-40"
      />
      <USelectMenu
        v-model="ruleFilter"
        :items="ruleItems"
        value-key="value"
        placeholder="All rules"
        size="xs"
        class="w-32"
      />
      <USelectMenu
        v-model="modOrigin"
        :items="modItems"
        value-key="origin"
        placeholder="All mods"
        size="xs"
        class="w-44"
      >
        <template #item-leading="{ item }">
          <span
            class="size-2 shrink-0 rounded-full"
            :style="{ backgroundColor: originHex({ kind: 'mod', path: item.path }) }"
          />
        </template>
      </USelectMenu>
    </div>

    <USplitter
      id="conflicts-split"
      auto-save-id="pmt-conflicts-split"
      class="min-h-0 w-full flex-1 overflow-hidden"
      :ui="splitUi"
      :items="splitItems"
    >
      <template #overview>
        <IssueOverviewPane
          :chips="chips"
          :nodes="treeNodes"
          :active-chip="kindFilter"
          :active-node="modOrigin"
          tree-label="By mod"
          @select-chip="onChip"
          @select-node="onNode"
        />
      </template>
      <template #table>
        <div class="h-full min-h-0 overflow-auto p-2">
          <UTable
            :data="filteredRows"
            :columns="columns"
            :loading="loading"
            sticky="header"
            empty="No overlapping definitions."
            :get-row-id="(row: OverrideRow) => `${row.kind}:${row.name}`"
          >
            <template #rule-cell="{ row }">
              <UBadge
                :label="row.original.rule"
                :color="row.original.rule === 'FIOS' ? 'warning' : 'info'"
                variant="subtle"
                size="xs"
              />
            </template>
            <template #winner-cell="{ row }">
              <UBadge
                :label="originLabel(row.original.winner)"
                :color="row.original.winner ? 'primary' : 'neutral'"
                variant="subtle"
                size="xs"
              />
            </template>
            <template #sites-cell="{ row }">
              <div class="flex flex-wrap gap-1">
                <UButton
                  v-for="(site, i) in row.original.sites ?? []"
                  :key="i"
                  :label="`${originLabel(site.origin)}:${site.line}`"
                  size="xs"
                  :color="site.origin ? 'primary' : 'neutral'"
                  variant="subtle"
                  @click="open(site.file, site.line)"
                />
              </div>
            </template>
          </UTable>
        </div>
      </template>
    </USplitter>
  </div>
</template>
