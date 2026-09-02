<script setup lang="ts">
/**
 * Event graph page: GetEventGraph + GetEventDetail. Layout is dagre in GraphCanvas.
 */
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { useQuery } from "@pinia/colada";
import { useLocalStorage } from "@vueuse/core";
import {
  GetEventDetail, GetEventGraph,
} from "@services/viewsservice";
import type { EventGraphParams } from "@services/internal/views/models";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";
import GraphCanvas from "../components/graph/GraphCanvas.vue";
import EventDetailPanel from "../components/graph/EventDetailPanel.vue";
import GameIcon from "../components/GameIcon.vue";
import { useLiveEnabled } from "../composables/useSessionQuery";
import type { GraphLayoutMode } from "../composables/useGraphLayout";
import { originHex } from "../ide/rootDecorations";
import { useWorkspaceStore } from "../stores/workspace";

defineOptions({ name: "EventGraphPage" });

const MENU_UI = { content: "min-w-96 w-[420px]", viewport: "max-h-96" };

const route = useRoute();
const ws = useWorkspaceStore();
const vanilla = computed(() => ws.originVanilla);

const liveMods = computed(() =>
  ws.workspaceMods.filter((m) => !m.isBroken && m.path),
);
const originItems = computed(() => [
  ...liveMods.value.map((m) => ({ label: m.name, id: m.id, path: m.path })),
  {
    label: ws.gameName(ws.activeWorkspace?.gameId ?? ws.currentGameId),
    id: vanilla.value,
  },
]);
const origins = ref<string[]>([]);
watch(
  liveMods,
  (mods) => {
    const ids = [...mods.map((m) => m.id), vanilla.value];
    const keep = origins.value.filter((id) => ids.includes(id));
    origins.value = keep.length ? keep : ids;
  },
  { immediate: true },
);
const originsNarrowed = computed(
  () =>
    origins.value.length > 0 &&
    origins.value.length < originItems.value.length,
);
const originColors = computed((): Record<string, string> =>
  Object.fromEntries([
    [vanilla.value, originHex({
      kind: "game",
      path: "",
      color: ws.activeWorkspace?.gameColor,
    })],
    ...liveMods.value.map((m, i) => [
      m.id,
      originHex({
        kind: "mod",
        path: m.path,
        color: m.color,
        wrapIndex: i,
      }),
    ]),
  ]),
);

/** Explorer-matching swatch for a picker or payload origin. */
function itemOriginHex(origin: string | undefined): string {
  const key = !origin || origin === vanilla.value ? vanilla.value : origin;
  return originColors.value[key] ?? originHex({ kind: "mod", path: origin ?? "" });
}

const workspaceId = computed(() => String(route.params.id ?? ""));
const live = useLiveEnabled(workspaceId);
const selectedId = ref("");
const root = ref<string | undefined>();
const namespace = ref<string | undefined>();
const expand = ref<string[]>([]);
const layout = useLocalStorage<GraphLayoutMode>("pmt.graph.layout", "lr");
const graphCanvas = ref<{ recenter: () => Promise<void> } | null>(null);

const layoutItems = [
  { label: "After", value: "lr", icon: "i-lucide-arrow-right" },
  { label: "Tree", value: "tree", icon: "i-lucide-git-fork" },
];

const graphParams = computed((): EventGraphParams => {
  const p: EventGraphParams = {};
  if (root.value) p.root = root.value;
  if (namespace.value) p.namespace = namespace.value;
  if (originsNarrowed.value) p.origins = origins.value;
  if (expand.value.length) p.expand = expand.value;
  return p;
});

const {
  data: graph,
  error: graphError,
  isPending: graphPending,
} = useQuery({
  key: () => ["session", "graph", workspaceId.value, graphParams.value],
  query: () => GetEventGraph(workspaceId.value, graphParams.value),
  enabled: () => live.value,
  placeholderData: (prev) => prev,
});

const { data: detailData } = useQuery({
  key: () => ["session", "event-detail", workspaceId.value, selectedId.value],
  query: () => GetEventDetail(workspaceId.value, selectedId.value),
  enabled: () => live.value && !!selectedId.value,
});

const detail = computed(() => (selectedId.value ? detailData.value : null));
const error = computed(() => graphError.value?.message ?? "");
const loading = computed(() => graphPending.value && !graph.value);

/** Click: select; more-stubs append their parent to Expand. */
function onSelect(id: string): void {
  if (id.startsWith("more-in:") || id.startsWith("more:")) {
    const parent = id.startsWith("more-in:") ? id.slice(8) : id.slice(5);
    if (parent && !expand.value.includes(parent)) {
      expand.value = [...expand.value, parent];
    }
    return;
  }
  selectedId.value = id;
}

/** Double-click or event chip: re-root; align namespace so load keeps this id. */
function onReroot(id: string): void {
  if (id.startsWith("more:") || id.startsWith("more-in:")) {
    onSelect(id);
    return;
  }
  const dot = id.indexOf(".");
  const ns = dot > 0 ? id.slice(0, dot) : "";
  if (ns && namespace.value && namespace.value !== ns) {
    namespace.value = ns;
  }
  root.value = id;
  selectedId.value = id;
}

function onRecenter(): void { void graphCanvas.value?.recenter(); }
function onShowMore(): void {
  const id = selectedId.value;
  if (!id || id.startsWith("more:")) return;
  if (!expand.value.includes(id)) expand.value = [...expand.value, id];
}
function setLayout(v: string): void {
  if (v === "lr" || v === "tree") layout.value = v;
}
function onNamespace(v: string | null | undefined): void {
  namespace.value = v ?? undefined;
}
function setRoot(v: string | null | undefined): void { if (v) root.value = v; }
function clearRoot(): void {
  root.value = undefined;
  namespace.value = undefined;
  selectedId.value = "";
}

watch(namespace, (ns) => {
  if (ns && root.value && !root.value.startsWith(`${ns}.`)) {
    root.value = undefined;
  }
  expand.value = [];
});
watch(root, () => { expand.value = []; });
watch(workspaceId, () => {
  root.value = undefined;
  namespace.value = undefined;
  selectedId.value = "";
  expand.value = [];
});
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
          size="md"
          class="w-28"
          @update:model-value="setLayout"
        />
        <LanguageHealthStrip
          v-if="workspaceId"
          :workspace-id="workspaceId"
        />
      </template>
    </WorkspaceToolBar>

    <UDashboardToolbar class="px-2 sm:px-2">
      <template #left>
        <USelectMenu
          :model-value="root"
          :items="graph?.suggestions?.ids ?? []"
          label-key="id" value-key="id" placeholder="Root event"
          size="md" class="min-w-64 flex-1" :ui="MENU_UI" :loading="loading"
          :virtualize="(graph?.suggestions?.ids?.length ?? 0) > 400"
          @update:model-value="setRoot"
        >
          <template #item-leading="{ item }">
            <img
              v-if="item.origin && ws.thumbUrls[item.origin]"
              :src="ws.thumbUrls[item.origin]"
              alt=""
              class="size-4 rounded-sm object-cover"
            />
            <GameIcon
              v-else-if="!item.origin || item.origin === vanilla"
              :game-id="ws.activeWorkspace?.gameId ?? ws.currentGameId"
            />
            <span v-else class="size-2 shrink-0 rounded-full"
              :style="{ backgroundColor: itemOriginHex(item.origin) }" />
          </template>
        </USelectMenu>
        <USelectMenu
          :model-value="namespace"
          :items="graph?.suggestions?.namespaces ?? []"
          label-key="id" value-key="id" placeholder="Namespace"
          size="md" class="w-56" :ui="MENU_UI" clear
          @update:model-value="onNamespace"
        >
          <template #item-leading="{ item }">
            <span v-if="item.id" class="size-2 shrink-0 rounded-full"
              :style="{ backgroundColor: itemOriginHex(item.origin) }" />
          </template>
        </USelectMenu>
        <USelectMenu
          v-model="origins" :items="originItems" value-key="id" multiple
          placeholder="Origins" size="md" class="w-72" :ui="MENU_UI"
        >
          <template #item-leading="{ item }">
            <img
              v-if="item.id !== vanilla && ws.thumbUrls[item.id]"
              :src="ws.thumbUrls[item.id]"
              alt=""
              class="size-4 rounded-sm object-cover"
            />
            <GameIcon
              v-else-if="item.id === vanilla"
              :game-id="ws.activeWorkspace?.gameId ?? ws.currentGameId"
            />
            <span
              v-else
              class="size-2 shrink-0 rounded-full"
              :style="{ backgroundColor: itemOriginHex(item.id) }"
            />
          </template>
        </USelectMenu>
        <UButton
          label="Clear"
          icon="i-lucide-x"
          size="md"
          color="neutral"
          variant="outline"
          :disabled="!root && !namespace"
          @click="clearRoot"
        />
        <UButton
          label="Recenter"
          icon="i-lucide-locate-fixed"
          size="md"
          color="neutral"
          variant="soft"
          :disabled="!root"
          @click="onRecenter"
        />
        <UButton
          v-if="graph?.truncated"
          label="Show more"
          size="md"
          color="neutral"
          variant="ghost"
          :disabled="!selectedId || selectedId.startsWith('more:') || selectedId.startsWith('more-in:')"
          @click="onShowMore"
        />
        <span
          v-if="root && graph"
          class="text-xs text-muted"
        >
          {{ graph.nodes?.length ?? 0 }} nodes
        </span>
        <UBadge
          v-if="graph?.truncated"
          color="warning"
          variant="subtle"
          size="md"
        >
          truncated
        </UBadge>
        <span v-if="graph?.truncated" class="text-xs text-muted">
          Double-click a node at the edge to walk further.
        </span>
        <span v-if="!root && graph?.emptyReason" class="text-xs text-muted">
          {{ graph.emptyReason }}
        </span>
      </template>
    </UDashboardToolbar>

    <UAlert
      v-if="error"
      color="error"
      variant="subtle"
      :description="error"
      class="m-2"
    />

    <UDashboardGroup
      storage="local"
      storage-key="pmt-event-graph"
      class="relative! inset-auto! min-h-0 min-w-0 flex-1 overflow-hidden"
    >
      <UDashboardPanel
        id="event-graph-main"
        resizable
        :default-size="75"
        :min-size="50"
        :max-size="85"
        class="min-h-0!"
      >
        <GraphCanvas
          ref="graphCanvas"
          class="h-full min-h-0 w-full"
          :nodes="graph?.nodes ?? []"
          :edges="graph?.edges ?? []"
          :layout="layout"
          :selected-id="selectedId"
          :root-id="root"
          @select="onSelect"
          @reroot="onReroot"
          @clear-selection="selectedId = ''"
        />
      </UDashboardPanel>
      <UDashboardPanel id="event-graph-detail" class="min-h-0!">
        <EventDetailPanel
          :workspace-id="workspaceId"
          :detail="detail ?? null"
          @reroot="onReroot"
        />
      </UDashboardPanel>
    </UDashboardGroup>
  </div>
</template>
