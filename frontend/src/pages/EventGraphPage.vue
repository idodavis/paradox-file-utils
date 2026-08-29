<script setup lang="ts">
/**
 * Event graph page: GetEventGraph + GetEventDetail. Layout is dagre in GraphCanvas.
 */
import { computed, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useRoute } from "vue-router";
import type { SplitterItem } from "@nuxt/ui";
import {
  GetEventDetail,
  GetEventGraph,
} from "@services/languagemodelservice";
import type {
  EventDetail,
  EventGraph,
  EventGraphParams,
  SuggestionItem,
} from "@services/internal/graph/models";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";
import GraphCanvas from "../components/graph/GraphCanvas.vue";
import EventDetailPanel from "../components/graph/EventDetailPanel.vue";
import { useWorkspaceStore } from "../stores/workspace";
import { originHex } from "../ide/rootDecorations";
import type { GraphLayoutMode } from "../composables/useGraphLayout";

const route = useRoute();
const ws = useWorkspaceStore();
const { workspaceMods } = storeToRefs(ws);

const workspaceId = computed(() => String(route.params.id ?? ""));
const graph = ref<EventGraph | null>(null);
const detail = ref<EventDetail | null>(null);
const selectedId = ref("");
const root = ref<string | undefined>();
const namespace = ref<string | undefined>();
const modRoot = ref<string | undefined>();
const layout = ref<GraphLayoutMode>("lr");
const loading = ref(false);
const error = ref("");
let detailGen = 0;

const layoutItems = [
  { label: "After", value: "lr", icon: "i-lucide-arrow-right" },
  { label: "Tree", value: "tree", icon: "i-lucide-git-fork" },
];

const liveMods = computed(() =>
  workspaceMods.value.filter((m) => !m.isBroken && m.path),
);

const modItems = computed(() =>
  liveMods.value.map((m) => ({ label: m.name, path: m.path })),
);

interface PickerItem {
  label: string;
  id?: string;
  origin?: string;
  type?: "label" | "separator" | "item";
  disabled?: boolean;
}

/** Explorer-matching origin color for a picker row. */
function itemOriginHex(origin: string | undefined): string {
  if (!origin) return originHex({ kind: "game", path: "" });
  const path = liveMods.value.find((m) => m.id === origin)?.path ?? origin;
  return originHex({ kind: "mod", path });
}

/** Group picker rows as Vanilla / per-mod, matching the library dropdown. */
function groupByOrigin(items: SuggestionItem[]): PickerItem[][] {
  const groups: PickerItem[][] = [];
  const toItem = (it: SuggestionItem): PickerItem => ({
    label: it.id,
    id: it.id,
    origin: it.origin,
  });
  const vanilla = items.filter((i) => !i.origin);
  if (vanilla.length) {
    groups.push([
      { type: "label", label: "Vanilla", disabled: true },
      ...vanilla.map(toItem),
    ]);
  }
  for (const m of liveMods.value) {
    const list = items.filter((i) => i.origin === m.id);
    if (!list.length) continue;
    groups.push([
      { type: "label", label: m.name, disabled: true },
      ...list.map(toItem),
    ]);
  }
  return groups;
}

const filteredIds = computed(() => {
  const ids = graph.value?.suggestions?.ids ?? [];
  const ns = namespace.value;
  if (!ns) return ids;
  return ids.filter((i) => i.id.startsWith(`${ns}.`));
});

const rootItems = computed(() => groupByOrigin(filteredIds.value));
const nsItems = computed(() =>
  groupByOrigin(graph.value?.suggestions?.namespaces ?? []),
);

const incoming = computed(() =>
  (graph.value?.edges ?? []).filter((e) => e.to === selectedId.value),
);

/** Graph | inspector; detail pane stays a fixed-px inspector. */
const splitItems: SplitterItem[] = [
  { slot: "canvas", minSize: 40, defaultSize: 72, class: "min-h-0 min-w-0 overflow-hidden" },
  {
    slot: "detail",
    sizeUnit: "px",
    minSize: 240,
    maxSize: 560,
    defaultSize: 320,
    class: "min-h-0 min-w-0 overflow-hidden",
  },
];

const splitUi = {
  handle:
    "data-[orientation=horizontal]:w-px bg-border transition-colors " +
    "data-[state=hover]:bg-primary data-[state=drag]:bg-primary",
};

const graphParams = computed((): EventGraphParams => {
  const p: EventGraphParams = {};
  if (root.value) p.root = root.value;
  if (namespace.value) p.namespace = namespace.value;
  if (modRoot.value) p.modRoot = modRoot.value;
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
    if (
      namespace.value &&
      root.value &&
      !root.value.startsWith(`${namespace.value}.`)
    ) {
      root.value = undefined;
    }
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

/** Clear root, namespace, and mod focus. */
function clearFilters(): void {
  root.value = undefined;
  namespace.value = undefined;
  modRoot.value = undefined;
}

watch(namespace, (ns) => {
  if (ns && root.value && !root.value.startsWith(`${ns}.`)) {
    root.value = undefined;
  }
});

watch([workspaceId, root, namespace, modRoot], loadGraph, { immediate: true });
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
        :items="rootItems"
        value-key="id"
        placeholder="Root event"
        size="xs"
        class="w-56"
        :loading="loading"
        :virtualize="filteredIds.length > 400"
      >
        <template #item-leading="{ item }">
          <span
            v-if="item.id"
            class="size-2 shrink-0 rounded-full"
            :style="{ backgroundColor: itemOriginHex(item.origin) }"
          />
        </template>
      </USelectMenu>
      <USelectMenu
        v-model="namespace"
        :items="nsItems"
        value-key="id"
        placeholder="Namespace"
        size="xs"
        class="w-44"
      >
        <template #item-leading="{ item }">
          <span
            v-if="item.id"
            class="size-2 shrink-0 rounded-full"
            :style="{ backgroundColor: itemOriginHex(item.origin) }"
          />
        </template>
      </USelectMenu>
      <USelectMenu
        v-model="modRoot"
        :items="modItems"
        value-key="path"
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
      <UButton
        label="All"
        size="xs"
        color="neutral"
        variant="ghost"
        :disabled="!root && !namespace && !modRoot"
        @click="clearFilters"
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

    <USplitter
      id="event-graph-split"
      auto-save-id="pmt-event-graph-split"
      class="min-h-0 w-full flex-1 overflow-hidden"
      :ui="splitUi"
      :items="splitItems"
    >
      <template #canvas>
        <GraphCanvas
          class="h-full min-h-0 w-full"
          :nodes="graph?.nodes ?? []"
          :edges="graph?.edges ?? []"
          :layout="layout"
          :selected-id="selectedId"
          @select="onSelect"
          @reroot="onReroot"
        />
      </template>
      <template #detail>
        <EventDetailPanel
          :workspace-id="workspaceId"
          :detail="detail"
          :incoming="incoming"
          @select="onSelect"
        />
      </template>
    </USplitter>
  </div>
</template>
