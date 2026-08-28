<script setup lang="ts">
/**
 * Event graph page: GetEventGraph + GetEventDetail. Layout is dagre in GraphCanvas.
 */
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import {
  GetEventDetail,
  GetEventGraph,
} from "@services/languagemodelservice";
import type {
  EventDetail,
  EventGraph,
  EventGraphParams,
} from "@services/internal/graph/models";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";
import GraphCanvas from "../components/graph/GraphCanvas.vue";
import EventDetailPanel from "../components/graph/EventDetailPanel.vue";
import { useWorkspaceStore } from "../stores/workspace";
import type { GraphLayoutMode } from "../composables/useGraphLayout";

const route = useRoute();
const ws = useWorkspaceStore();

const workspaceId = computed(() => String(route.params.id ?? ""));
const graph = ref<EventGraph | null>(null);
const detail = ref<EventDetail | null>(null);
const selectedId = ref("");
const root = ref<string | undefined>();
const namespace = ref<string | undefined>();
const layout = ref<GraphLayoutMode>("lr");
const loading = ref(false);
const error = ref("");
let detailGen = 0;

const layoutItems = [
  { label: "After", value: "lr", icon: "i-lucide-arrow-right" },
  { label: "Tree", value: "tree", icon: "i-lucide-git-fork" },
];

const idItems = computed(() => graph.value?.suggestions?.ids ?? []);
const nsItems = computed(() => graph.value?.suggestions?.namespaces ?? []);

const graphParams = computed((): EventGraphParams => {
  const p: EventGraphParams = {};
  if (root.value) p.root = root.value;
  if (namespace.value) p.namespace = namespace.value;
  return p;
});

/** Load the Go event graph for the current query. */
async function loadGraph(): Promise<void> {
  if (!workspaceId.value) return;
  loading.value = true;
  error.value = "";
  try {
    const ready = await ws.ensureReady();
    if (!ready) {
      error.value = "Language session is not live. Use Rescan in the toolbar.";
      graph.value = null;
      return;
    }
    graph.value = await GetEventGraph(workspaceId.value, graphParams.value);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
    graph.value = null;
  } finally {
    loading.value = false;
  }
}

/** Load inspector payload for the selected event id. */
async function loadDetail(id: string): Promise<void> {
  const n = ++detailGen;
  if (!id || !workspaceId.value) {
    detail.value = null;
    return;
  }
  try {
    const d = await GetEventDetail(workspaceId.value, id);
    if (n === detailGen) detail.value = d;
  } catch {
    if (n === detailGen) detail.value = null;
  }
}

/** Click: select and fetch detail. */
function onSelect(id: string): void {
  selectedId.value = id;
  void loadDetail(id);
}

/** Double-click: re-root the Go query on this id. */
function onReroot(id: string): void {
  root.value = id;
  selectedId.value = id;
  void loadDetail(id);
}

/** Apply a dagre layout preset from the toolbar select. */
function setLayout(v: string): void {
  if (v === "lr" || v === "tree") layout.value = v;
}

watch([workspaceId, root, namespace], loadGraph, { immediate: true });
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <WorkspaceToolBar
      :workspace-id="workspaceId"
      title="Event Graph"
      active="event-graph"
    >
      <template #trailing>
        <USelect
          :model-value="layout"
          :items="layoutItems"
          value-key="value"
          size="xs"
          class="w-28"
          @update:model-value="setLayout"
        />
        <LanguageHealthStrip
          v-if="workspaceId"
          :workspace-id="workspaceId"
        />
      </template>
    </WorkspaceToolBar>

    <div
      class="flex shrink-0 flex-wrap items-center gap-2 border-b border-default
        px-2 py-1"
    >
      <USelectMenu
        v-model="root"
        :items="idItems"
        placeholder="Root event"
        size="xs"
        class="w-56"
        :loading="loading"
      />
      <USelectMenu
        v-model="namespace"
        :items="nsItems"
        placeholder="Namespace"
        size="xs"
        class="w-44"
      />
      <UButton
        label="All"
        size="xs"
        color="neutral"
        variant="ghost"
        :disabled="!root && !namespace"
        @click="root = undefined; namespace = undefined"
      />
      <UBadge
        v-if="graph?.truncated"
        color="warning"
        variant="subtle"
        size="xs"
      >
        truncated
      </UBadge>
      <span v-if="graph?.emptyReason" class="text-xs text-muted">
        {{ graph.emptyReason }}
      </span>
    </div>

    <UAlert
      v-if="error"
      color="error"
      variant="subtle"
      :description="error"
      class="m-2"
    />

    <div class="flex min-h-0 flex-1 overflow-hidden">
      <GraphCanvas
        class="min-h-0 min-w-0 flex-1"
        :nodes="graph?.nodes ?? []"
        :edges="graph?.edges ?? []"
        :layout="layout"
        :selected-id="selectedId"
        @select="onSelect"
        @reroot="onReroot"
      />
      <EventDetailPanel :workspace-id="workspaceId" :detail="detail" />
    </div>
  </div>
</template>
