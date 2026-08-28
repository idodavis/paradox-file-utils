<script setup lang="ts">
/**
 * Loc coverage page: GetLocCoverage / LookupLoc only.
 */
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { GetLocCoverage, LookupLoc } from "@services/languagemodelservice";
import type {
  LocCoverage,
  LocIssue,
  LocLookup,
} from "@services/internal/graph/models";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";
import { useWorkspaceStore } from "../stores/workspace";
import { useOpenInIde } from "../composables/useOpenInIde";

/** Coverage bucket names from the Go payload. */
type LocKind = "missing" | "orphaned" | "untranslated";

/** One coverage finding with its bucket name for the table. */
interface LocRow extends LocIssue {
  kind: LocKind;
}

const route = useRoute();
const ws = useWorkspaceStore();
const { openInIde } = useOpenInIde();

const workspaceId = computed(() => String(route.params.id ?? ""));
const coverage = ref<LocCoverage[]>([]);
const lang = ref<string>("");
const lookupKey = ref("");
const lookup = ref<LocLookup | null>(null);
const query = ref("");
const loading = ref(false);
const error = ref("");

const langItems = computed(() =>
  coverage.value.map((c) => ({
    label: `${c.language} (${c.defined})`,
    value: c.language,
  })),
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

const columns = [
  { accessorKey: "kind", header: "Kind" },
  { accessorKey: "key", header: "Key" },
  { accessorKey: "file", header: "File" },
  { accessorKey: "line", header: "Line" },
  { accessorKey: "value", header: "Value" },
  { id: "open", header: "" },
];

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
        class="w-48"
      />
      <UInput
        v-model="query"
        icon="i-lucide-search"
        placeholder="Filter keys"
        size="xs"
        class="w-48"
      />
    </div>

    <div class="min-h-0 flex-1 overflow-auto p-2">
      <UTable
        :data="issueRows"
        :columns="columns"
        :loading="loading"
        :global-filter="query"
        sticky="header"
        empty="No loc issues for this language."
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
  </div>
</template>
