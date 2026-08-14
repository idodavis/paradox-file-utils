<script setup lang="ts">
/**
 * Event Graph: search language-model definitions, neighbors, cascade; open in workbench.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Events } from "@wailsio/runtime";
import { GetWorkspace } from "@services/workspaceservice";
import { Workspace } from "@services/internal/repos/models";
import {
  SearchDefinitions,
  GetNeighbors,
  SimulateCascade,
} from "@services/graphservice";
import {
  RebuildWorkspaceModel,
  Cancel,
  GetModelStatus,
} from "@services/languagemodelservice";
import type { GraphDef, CascadeNode, NeighborResult } from "@services/models";
import { openFile } from "../ide/commands";

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
const searchResults = ref<GraphDef[]>([]);
const selected = ref<GraphDef | null>(null);
const neighbors = ref<NeighborResult | null>(null);
const cascadeNodes = ref<CascadeNode[]>([]);
const typeFilter = ref("");
const defCount = ref(0);

let offProgress: (() => void) | null = null;

const typeOptions = [
  { label: "All", value: "" },
  { label: "Events", value: "events" },
  { label: "Decisions", value: "decisions" },
  { label: "Traits", value: "traits" },
  { label: "Scripted Effects", value: "scripted_effects" },
  { label: "Scripted Triggers", value: "scripted_triggers" },
];

function isCancelled(msg: string): boolean {
  return /cancelled/i.test(msg);
}

/** Load workspace + model status. */
async function load(): Promise<void> {
  loading.value = true;
  error.value = "";
  try {
    workspace.value = await GetWorkspace(workspaceId.value);
    const st = await GetModelStatus(workspaceId.value);
    defCount.value = st?.defCount ?? 0;
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

/** Rebuild workspace language model. */
async function rebuild(): Promise<void> {
  indexing.value = true;
  progressLabel.value = "Building language model…";
  progressPercent.value = 0;
  error.value = "";
  try {
    defCount.value = await RebuildWorkspaceModel(workspaceId.value);
    await runSearch();
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    if (!isCancelled(msg)) error.value = msg;
  } finally {
    indexing.value = false;
  }
}

/** Search definitions. */
async function runSearch(): Promise<void> {
  try {
    searchResults.value =
      (await SearchDefinitions(
        workspaceId.value,
        searchQuery.value,
        typeFilter.value,
      )) ?? [];
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

/** Select a definition and load graph context. */
async function selectDef(d: GraphDef): Promise<void> {
  selected.value = d;
  try {
    neighbors.value = await GetNeighbors(workspaceId.value, d.key);
    cascadeNodes.value =
      (await SimulateCascade(workspaceId.value, d.key, 4)) ?? [];
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

/** Open definition in workbench. */
async function openInIde(): Promise<void> {
  if (!selected.value) return;
  await openFile(selected.value.filePath, selected.value.line);
  void router.push({
    name: "workspace-ide",
    params: { id: workspaceId.value },
  });
}

onMounted(() => {
  offProgress = Events.On("langmodel:progress", (e: unknown) => {
    const detail = (e as { data?: { message?: string; percent?: number } })
      ?.data;
    if (detail?.message) progressLabel.value = detail.message;
    if (typeof detail?.percent === "number") {
      progressPercent.value = detail.percent;
    }
  });
  void load();
});

watch(workspaceId, () => void load());
watch([searchQuery, typeFilter], () => void runSearch());

onBeforeUnmount(() => {
  offProgress?.();
  Cancel();
});
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <div
      class="flex shrink-0 items-center justify-between gap-2 border-b border-default bg-muted/50 px-3 py-2"
    >
      <div class="flex items-center gap-2">
        <UButton
          icon="i-lucide-arrow-left"
          variant="ghost"
          size="sm"
          @click="
            router.push({
              name: 'workspace-ide',
              params: { id: workspaceId },
            })
          "
        />
        <span class="font-semibold">Event Graph</span>
        <UBadge variant="subtle" size="xs">{{ defCount }} defs</UBadge>
      </div>
      <div class="flex items-center gap-2">
        <UButton
          :label="defCount ? 'Rebuild model' : 'Build model'"
          icon="i-lucide-refresh-cw"
          size="sm"
          :loading="indexing"
          @click="rebuild"
        />
        <UButton
          v-if="indexing"
          label="Cancel"
          size="sm"
          color="neutral"
          variant="outline"
          @click="Cancel()"
        />
      </div>
    </div>

    <UProgress
      v-if="indexing"
      :model-value="progressPercent"
      class="px-3 pt-2"
    />
    <p v-if="indexing" class="px-3 text-xs text-muted">{{ progressLabel }}</p>
    <UAlert
      v-if="error"
      color="error"
      variant="subtle"
      :description="error"
      class="m-2"
    />

    <div class="grid min-h-0 flex-1 grid-cols-1 gap-2 overflow-hidden p-2 lg:grid-cols-3">
      <UCard class="min-h-0" :ui="{ body: 'flex min-h-0 flex-col gap-2' }">
        <div class="flex gap-2">
          <UInput
            v-model="searchQuery"
            placeholder="Search keys…"
            class="flex-1"
          />
          <USelect
            v-model="typeFilter"
            :items="typeOptions"
            value-key="value"
            class="w-40"
          />
        </div>
        <div class="min-h-0 flex-1 overflow-auto text-sm">
          <button
            v-for="d in searchResults"
            :key="`${d.type}:${d.key}`"
            type="button"
            class="block w-full truncate rounded px-2 py-1 text-left hover:bg-muted"
            :class="{
              'bg-primary/10': selected?.key === d.key,
            }"
            @click="selectDef(d)"
          >
            <span class="font-medium">{{ d.key }}</span>
            <span class="ml-1 text-xs text-muted">{{ d.type }}</span>
          </button>
          <p
            v-if="!searchResults.length && !loading"
            class="px-2 py-4 text-xs text-muted"
          >
            Build the language model, then search.
          </p>
        </div>
      </UCard>

      <UCard class="min-h-0 lg:col-span-2" :ui="{ body: 'overflow-auto space-y-3' }">
        <template v-if="selected">
          <div class="flex items-start justify-between gap-2">
            <div>
              <h2 class="text-lg font-semibold">{{ selected.key }}</h2>
              <p class="text-xs text-muted">
                {{ selected.type }} · {{ selected.filePath }}:{{
                  selected.line
                }}
              </p>
            </div>
            <UButton
              label="Open in IDE"
              icon="i-lucide-file-code"
              size="sm"
              @click="openInIde"
            />
          </div>
          <div>
            <h3 class="mb-1 text-sm font-semibold">Neighbors</h3>
            <ul class="text-sm">
              <li
                v-for="e in neighbors?.outgoing ?? []"
                :key="'o' + e.toKey + e.edgeType"
              >
                → {{ e.toKey }}
                <span class="text-xs text-muted">({{ e.edgeType }})</span>
              </li>
              <li
                v-for="e in neighbors?.incoming ?? []"
                :key="'i' + e.fromKey + e.edgeType"
              >
                ← {{ e.fromKey }}
                <span class="text-xs text-muted">({{ e.edgeType }})</span>
              </li>
            </ul>
          </div>
          <div>
            <h3 class="mb-1 text-sm font-semibold">Cascade</h3>
            <ul class="font-mono text-xs">
              <li v-for="(n, i) in cascadeNodes" :key="i">
                {{ "  ".repeat(n.depth) }}{{ n.key }}
                <span class="text-muted">{{ n.edgeType }}</span>
              </li>
            </ul>
          </div>
        </template>
        <p v-else class="text-sm text-muted">Select a definition.</p>
      </UCard>
    </div>
  </div>
</template>
