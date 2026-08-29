<script setup lang="ts">
/**
 * Loc coverage page: GetLocCoverage / LookupLoc with overview filters.
 */
import { computed, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useRoute } from "vue-router";
import type { SplitterItem } from "@nuxt/ui";
import { GetLocCoverage, LookupLoc } from "@services/languagemodelservice";
import type {
  LocCoverage,
  LocIssue,
  LocLookup,
} from "@services/internal/graph/models";
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

/** Coverage bucket names from the Go payload. */
type LocKind = "missing" | "orphaned" | "untranslated";

/** One coverage finding with its bucket name for the table. */
interface LocRow extends LocIssue {
  kind: LocKind;
}

const route = useRoute();
const ws = useWorkspaceStore();
const { workspaceMods } = storeToRefs(ws);
const { openInIde } = useOpenInIde();

const workspaceId = computed(() => String(route.params.id ?? ""));
const coverage = ref<LocCoverage[]>([]);
const lang = ref<string>("");
const lookupKey = ref("");
const lookup = ref<LocLookup | null>(null);
const query = ref("");
const modOrigin = ref<string | undefined>();
const kindFilter = ref<LocKind | undefined>();
const fileFilter = ref<string | undefined>();
const loading = ref(false);
const error = ref("");

const liveMods = computed(() =>
  workspaceMods.value.filter((m) => !m.isBroken && m.path),
);

const langItems = computed(() =>
  coverage.value.map((c) => ({
    label: `${c.language} (${c.defined})`,
    value: c.language,
  })),
);

const kindItems: { label: string; value: LocKind }[] = [
  { label: "Missing", value: "missing" },
  { label: "Orphaned", value: "orphaned" },
  { label: "Untranslated", value: "untranslated" },
];

const modItems = computed(() =>
  liveMods.value.map((m) => ({ label: m.name, origin: m.id, path: m.path })),
);

const current = computed(
  () => coverage.value.find((c) => c.language === lang.value) ?? null,
);

const issueRows = computed((): LocRow[] => {
  const c = current.value;
  if (!c) return [];
  return [
    ...(c.missing ?? []).map((i) => ({ ...i, kind: "missing" as const })),
    ...(c.orphaned ?? []).map((i) => ({ ...i, kind: "orphaned" as const })),
    ...(c.untranslated ?? []).map(
      (i) => ({ ...i, kind: "untranslated" as const }),
    ),
  ];
});

const filteredRows = computed((): LocRow[] => {
  const q = query.value.trim().toLowerCase();
  return issueRows.value.filter((r) => {
    if (kindFilter.value && r.kind !== kindFilter.value) return false;
    if (modOrigin.value && r.origin !== modOrigin.value) return false;
    if (fileFilter.value && r.file !== fileFilter.value) return false;
    if (q && !r.key.toLowerCase().includes(q)) return false;
    return true;
  });
});

const chips = computed((): OverviewChip[] => {
  const base = issueRows.value.filter((r) => {
    if (modOrigin.value && r.origin !== modOrigin.value) return false;
    if (fileFilter.value && r.file !== fileFilter.value) return false;
    const q = query.value.trim().toLowerCase();
    if (q && !r.key.toLowerCase().includes(q)) return false;
    return true;
  });
  const byKind = { missing: 0, orphaned: 0, untranslated: 0 };
  for (const r of base) byKind[r.kind]++;
  return [
    { id: "missing", label: "Missing", count: byKind.missing, color: "error" },
    { id: "orphaned", label: "Orphaned", count: byKind.orphaned, color: "warning" },
    {
      id: "untranslated",
      label: "Untranslated",
      count: byKind.untranslated,
      color: "info",
    },
  ];
});

const treeNodes = computed((): OverviewNode[] => {
  const by = new Map<string, OverviewNode>();
  for (const r of filteredRows.value) {
    if (!r.file) continue;
    const cur = by.get(r.file);
    if (cur) cur.count++;
    else by.set(r.file, { id: r.file, label: r.rel || r.file, count: 1 });
  }
  return [...by.values()].sort((a, b) => b.count - a.count).slice(0, 80);
});

const columns = [
  { accessorKey: "kind", header: "Kind" },
  { accessorKey: "key", header: "Key" },
  { id: "file", header: "File" },
  { accessorKey: "line", header: "Line" },
  { accessorKey: "value", header: "Value" },
  { id: "open", header: "" },
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

/** Load per-language coverage from Go. */
async function load(): Promise<void> {
  if (!workspaceId.value) return;
  loading.value = true;
  error.value = "";
  try {
    const ready = await ws.ensureReady();
    if (!ready) {
      error.value = "Language session is not live. Use Rescan in the toolbar.";
      coverage.value = [];
      return;
    }
    coverage.value = (await GetLocCoverage(workspaceId.value)) ?? [];
    if (!lang.value && coverage.value.length) {
      lang.value = coverage.value[0]!.language;
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
    coverage.value = [];
  } finally {
    loading.value = false;
  }
}

/** Resolve one loc key via Go. */
async function runLookup(): Promise<void> {
  const key = lookupKey.value.trim();
  if (!key || !workspaceId.value) {
    lookup.value = null;
    return;
  }
  try {
    lookup.value = await LookupLoc(workspaceId.value, key);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
    lookup.value = null;
  }
}

/** Open a loc site in the workspace IDE. */
function open(file?: string, line?: number): void {
  if (!file) return;
  void openInIde(workspaceId.value, file, line);
}

function kindColor(
  kind: LocKind,
): "error" | "warning" | "info" {
  switch (kind) {
    case "missing":
      return "error";
    case "orphaned":
      return "warning";
    case "untranslated":
      return "info";
    default: {
      const _never: never = kind;
      return _never;
    }
  }
}

function onChip(id: string | undefined): void {
  kindFilter.value = id as LocKind | undefined;
}

watch(workspaceId, load, { immediate: true });
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <WorkspaceToolBar
      :workspace-id="workspaceId"
      title="Loc Coverage"
      active="loc-coverage"
    >
      <template #trailing>
        <UInput
          v-model="lookupKey"
          placeholder="Lookup key"
          size="xs"
          class="w-56"
        />
        <UButton
          label="Lookup"
          size="xs"
          color="neutral"
          variant="ghost"
          @click="runLookup"
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
      v-if="lookup"
      class="flex shrink-0 items-center gap-2 border-b border-default px-2 py-1
        text-sm"
    >
      <span class="font-medium text-default">{{ lookup.key }}</span>
      <UBadge
        v-if="lookup.origin"
        :label="lookup.origin"
        :color="lookup.origin === 'vanilla' ? 'neutral' : 'primary'"
        variant="subtle"
        size="xs"
      />
      <span class="truncate text-muted">{{ lookup.text }}</span>
      <UButton
        v-if="lookup.file"
        label="Open"
        size="xs"
        color="neutral"
        variant="ghost"
        @click="open(lookup.file, lookup.line)"
      />
    </div>

    <div
      class="flex shrink-0 flex-wrap items-center gap-2 border-b border-default
        px-2 py-1"
    >
      <USelect
        v-model="lang"
        :items="langItems"
        value-key="value"
        size="xs"
        class="w-44"
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
      <USelectMenu
        v-model="kindFilter"
        :items="kindItems"
        value-key="value"
        placeholder="All kinds"
        size="xs"
        class="w-40"
      />
      <UInput
        v-model="query"
        icon="i-lucide-search"
        placeholder="Filter keys"
        size="xs"
        class="w-44"
      />
    </div>

    <USplitter
      id="loc-coverage-split"
      auto-save-id="pmt-loc-coverage-split"
      class="min-h-0 w-full flex-1 overflow-hidden"
      :ui="splitUi"
      :items="splitItems"
    >
      <template #overview>
        <IssueOverviewPane
          :chips="chips"
          :nodes="treeNodes"
          :active-chip="kindFilter"
          :active-node="fileFilter"
          @select-chip="onChip"
          @select-node="fileFilter = $event"
        />
      </template>
      <template #table>
        <div class="h-full min-h-0 overflow-auto p-2">
          <UTable
            :data="filteredRows"
            :columns="columns"
            :loading="loading"
            sticky="header"
            empty="No loc issues for this filter."
          >
            <template #kind-cell="{ row }">
              <UBadge
                :color="kindColor(row.original.kind)"
                variant="subtle"
                size="xs"
              >
                {{ row.original.kind }}
              </UBadge>
            </template>
            <template #file-cell="{ row }">
              <span
                class="truncate text-xs text-muted"
                :title="row.original.file"
              >
                {{ row.original.rel || row.original.file }}
              </span>
            </template>
            <template #open-cell="{ row }">
              <UButton
                v-if="row.original.file"
                label="Open"
                size="xs"
                color="neutral"
                variant="ghost"
                @click="open(row.original.file, row.original.line)"
              />
            </template>
          </UTable>
        </div>
      </template>
    </USplitter>
  </div>
</template>
