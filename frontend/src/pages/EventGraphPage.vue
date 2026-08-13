<script setup lang="ts">
/**
 * Event Graph: search index objects, show neighbors/edges, cascade simulation.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Events } from "@wailsio/runtime";
import { GetWorkspace } from "@services/workspaceservice";
import { Workspace, IndexObject } from "@services/internal/repos/models";
import {
  ReindexWorkspace,
  CancelIndex,
  SearchIndex,
  ListIndexObjects,
  SimulateCascade,
  CountIndexObjects,
} from "@services/indexerservice";
import { GetSemanticsStatus } from "@services/semanticsservice";
import { CascadeNode, SemanticsStatus } from "@services/models";

const route = useRoute();
const router = useRouter();

const workspaceId = computed(() => String(route.params.id ?? ""));
const workspace = ref<Workspace | null>(null);
const loading = ref(false);
const indexing = ref(false);
const progressLabel = ref("");
const progressPercent = ref(0);
const error = ref("");
const searchQuery = ref("");
const searchResults = ref<IndexObject[]>([]);
const selectedObject = ref<IndexObject | null>(null);
const cascadeNodes = ref<CascadeNode[]>([]);
const typeFilter = ref("");
const indexCount = ref(0);
const hadIndex = ref(false);
const semantics = ref<SemanticsStatus | null>(null);

let loadGen = 0;
let offProgress: (() => void) | null = null;

const typeOptions = [
  { label: "All", value: "" },
  { label: "Events", value: "events" },
  { label: "Decisions", value: "decisions" },
  { label: "Traits", value: "traits" },
  { label: "Scripted Effects", value: "scripted_effects" },
  { label: "Scripted Triggers", value: "scripted_triggers" },
];

const indexActionLabel = computed(() => (hadIndex.value ? "Reindex" : "Index"));
const progressTitle = computed(() =>
  hadIndex.value ? "Reindexing…" : "Indexing…",
);

/** Soft-cancel helper: cancelled jobs are not failures. */
function isCancelled(msg: string): boolean {
  return /cancelled/i.test(msg);
}

/** Navigate immediately; cancel any in-flight index first. */
function goBack(): void {
  if (indexing.value) void CancelIndex();
  indexing.value = false;
  void router.push({ name: "workspace-ide", params: { id: workspaceId.value } });
}

/** Load workspace + index/semantics status. */
async function loadWorkspace(): Promise<void> {
  const id = workspaceId.value;
  if (!id) return;
  const gen = ++loadGen;
  loading.value = true;
  error.value = "";
  try {
    const [ws, count, sem] = await Promise.all([
      GetWorkspace(id),
      CountIndexObjects(id).catch(() => 0),
      GetSemanticsStatus(id).catch(() => null),
    ]);
    if (gen !== loadGen) return;
    workspace.value = ws;
    indexCount.value = count;
    hadIndex.value = count > 0;
    semantics.value = sem;
  } catch (e) {
    if (gen !== loadGen) return;
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    if (gen === loadGen) loading.value = false;
  }
}

/** Index the workspace (edges included; cancellable; does not block navigation). */
async function reindex(): Promise<void> {
  const id = workspaceId.value;
  if (!id || indexing.value) return;
  indexing.value = true;
  progressLabel.value = "Starting…";
  progressPercent.value = 0;
  error.value = "";
  try {
    indexCount.value = await ReindexWorkspace(id);
    if (workspaceId.value !== id) return;
    hadIndex.value = indexCount.value > 0;
    progressLabel.value = "";
    progressPercent.value = 0;
    await search();
  } catch (e) {
    if (workspaceId.value !== id) return;
    const msg = e instanceof Error ? e.message : String(e);
    if (!isCancelled(msg)) error.value = msg;
    progressLabel.value = "";
    progressPercent.value = 0;
  } finally {
    if (workspaceId.value === id) indexing.value = false;
  }
}

/** Cancel in-flight reindex without treating it as an error. */
function cancelReindex(): void {
  void CancelIndex();
  indexing.value = false;
  progressLabel.value = "";
  progressPercent.value = 0;
}

/** Search for objects. */
async function search(): Promise<void> {
  const id = workspaceId.value;
  if (!id) return;
  if (!searchQuery.value.trim()) {
    try {
      searchResults.value = (await ListIndexObjects(id, typeFilter.value)) ?? [];
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    }
    return;
  }
  loading.value = true;
  try {
    searchResults.value = (await SearchIndex(id, searchQuery.value)) ?? [];
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

/** Select an object and simulate cascade. */
async function selectObject(obj: IndexObject): Promise<void> {
  const id = workspaceId.value;
  if (!id) return;
  selectedObject.value = obj;
  loading.value = true;
  try {
    cascadeNodes.value = (await SimulateCascade(id, obj.objKey, 5)) ?? [];
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
    cascadeNodes.value = [];
  } finally {
    loading.value = false;
  }
}

watch(searchQuery, () => {
  if (!searchQuery.value) void search();
});
watch(typeFilter, () => {
  void search();
});
watch(workspaceId, loadWorkspace, { immediate: true });

onMounted(() => {
  offProgress = Events.On("index:progress", (ev) => {
    const raw = ev as unknown as { data?: Record<string, unknown> } & Record<string, unknown>;
    const d = raw.data ?? raw;
    if (!d || !indexing.value) return;
    if (d.phase === "cancelled") {
      indexing.value = false;
      progressLabel.value = "";
      progressPercent.value = 0;
      return;
    }
    const pct = typeof d.percent === "number" ? d.percent : undefined;
    progressPercent.value = pct ?? progressPercent.value;
    const pctText = pct != null ? ` ${Math.round(pct)}%` : "";
    progressLabel.value = `${String(d.message ?? d.phase ?? "Indexing")}${pctText}`;
  });
});

onBeforeUnmount(() => {
  loadGen++;
  offProgress?.();
  if (indexing.value) void CancelIndex();
});
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <div class="flex shrink-0 items-center justify-between gap-2 border-b border-default bg-muted/50 px-3 py-2">
      <div class="flex items-center gap-2">
        <UTooltip text="Back to workspace IDE"><UButton icon="i-lucide-arrow-left" variant="ghost" size="sm" @click="goBack"  /></UTooltip>
        <span class="font-semibold">Event Graph</span>
        <UBadge v-if="indexCount" color="neutral" variant="outline" size="xs">
          {{ indexCount }} objects
        </UBadge>
        <UBadge
          v-if="semantics"
          :color="semantics.present ? 'success' : 'warning'"
          variant="subtle"
          size="xs"
        >
          {{ semantics.present ? `Semantics ${semantics.scannedAt?.slice(0, 10) ?? "ready"}` : "No semantics" }}
        </UBadge>
      </div>
      <div class="flex items-center gap-2">
        <UTooltip v-if="indexing" text="Cancel indexing">
          <UButton
            label="Cancel"
            icon="i-lucide-x"
            color="neutral"
            variant="outline"
            size="sm"
            @click="cancelReindex"
          />
        </UTooltip>
        <UTooltip :text="`${indexActionLabel} workspace scripts and localization`">
          <UButton
            :label="indexActionLabel"
            icon="i-lucide-refresh-cw"
            variant="outline"
            size="sm"
            :loading="indexing"
            :disabled="indexing"
            @click="reindex"
          />
        </UTooltip>
      </div>
    </div>

    <UAlert v-if="error" color="error" variant="subtle" :description="error" class="m-2" />
    <UAlert
      v-else-if="semantics && !semantics.present"
      color="warning"
      variant="subtle"
      class="m-2"
      title="Semantics cache missing"
      description="Rebuild semantics from the Workspace IDE for better type classification."
    />

    <div
      v-if="indexing"
      class="flex flex-col items-center justify-center gap-2 border-b border-default bg-muted/30 px-6 py-6"
    >
      <div class="text-sm font-medium">{{ progressTitle }}</div>
      <div class="w-full max-w-md text-center text-xs text-muted">
        {{ progressLabel || "Starting…" }} · you can navigate away
      </div>
      <UProgress
        class="w-full max-w-md"
        :model-value="progressPercent > 0 ? Math.min(100, progressPercent) : null"
      />
    </div>

    <div class="flex min-h-0 flex-1 gap-3 overflow-hidden p-3">
      <div class="flex w-80 shrink-0 flex-col gap-3 overflow-hidden">
        <UCard :ui="{ body: 'p-2 space-y-2' }">
          <UInput v-model="searchQuery" placeholder="Search objects..." icon="i-lucide-search" />
          <USelect
            v-model="typeFilter"
            :items="typeOptions"
            value-key="value"
            placeholder="Filter by type"
          />
          <UButton label="Search" size="sm" :loading="loading" :disabled="indexing" @click="search" />
        </UCard>

        <UCard class="min-h-0 flex-1" :ui="{ body: 'overflow-auto p-0' }">
          <template #header>
            <span class="text-sm font-semibold">Results ({{ searchResults.length }})</span>
          </template>
          <div v-if="!searchResults.length" class="p-4 text-center text-sm text-muted">
            No results. Click {{ indexActionLabel }} to build the index.
          </div>
          <button
            v-for="obj in searchResults"
            :key="obj.id"
            class="flex w-full flex-col border-b border-default px-3 py-2 text-left hover:bg-muted/50"
            :class="{ 'bg-primary/10': selectedObject?.id === obj.id }"
            @click="selectObject(obj)"
          >
            <span class="font-medium">{{ obj.objKey }}</span>
            <span class="text-xs text-muted">
              {{ obj.objType }} - {{ obj.filePath.split(/[/\\]/).pop() }}
            </span>
          </button>
        </UCard>
      </div>

      <UCard class="min-h-0 min-w-0 flex-1" :ui="{ body: 'overflow-auto' }">
        <template #header>
          <span class="font-semibold">
            {{ selectedObject ? `Cascade: ${selectedObject.objKey}` : "Select an object" }}
          </span>
        </template>
        <div v-if="!selectedObject" class="py-8 text-center text-muted">
          Select an object from the search results to view its cascade graph.
        </div>
        <div v-else class="space-y-2">
          <div class="rounded border border-default p-2">
            <div class="font-semibold">{{ selectedObject.objKey }}</div>
            <div class="text-sm text-muted">
              Type: {{ selectedObject.objType }} | Line: {{ selectedObject.line }}
            </div>
            <div class="text-xs text-muted">{{ selectedObject.filePath }}</div>
            <div v-if="selectedObject.summary" class="mt-1 text-xs">{{ selectedObject.summary }}</div>
          </div>

          <div v-if="cascadeNodes.length" class="space-y-1">
            <div class="text-sm font-medium">Cascade ({{ cascadeNodes.length }} nodes)</div>
            <div
              v-for="node in cascadeNodes"
              :key="node.key"
              class="flex items-center gap-2 rounded border border-default px-2 py-1"
              :style="{ marginLeft: `${node.depth * 16}px` }"
            >
              <UBadge :color="node.depth === 0 ? 'primary' : 'neutral'" variant="subtle" size="xs">
                {{ node.depth }}
              </UBadge>
              <span class="flex-1 truncate font-mono text-sm">{{ node.key }}</span>
              <UBadge v-if="node.edgeType" color="secondary" variant="outline" size="xs">
                {{ node.edgeType }}
              </UBadge>
              <span v-if="node.children?.length" class="text-xs text-muted">
                → {{ node.children.length }}
              </span>
            </div>
          </div>
          <UEmpty
            v-else-if="selectedObject"
            icon="i-lucide-git-branch"
            title="No cascade"
            description="This object has no outgoing edges."
          />
        </div>
      </UCard>
    </div>
  </div>
</template>
