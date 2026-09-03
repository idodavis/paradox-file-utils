<script setup lang="ts">
/**
 * Heuristic impact report: wiki Modding bullets vs the live mod harvest.
 */
import { computed, shallowRef, watch } from "vue";
import { storeToRefs } from "pinia";
import { useRoute, useRouter } from "vue-router";
import { useMutation, useQuery } from "@pinia/colada";
import type { TableColumn } from "@nuxt/ui";
import { ListGameInstalls } from "@services/workspaceservice";
import { ModAffected, Patches } from "@services/wikiservice";
import type { AffectedRow } from "@services/internal/wiki/models";
import { useWorkspaceStore } from "../../stores/workspace";
import { useLiveEnabled } from "../../composables/useLiveEnabled";
import { useOpenInIde } from "../../composables/useOpenInIde";

const route = useRoute();
const router = useRouter();
const ws = useWorkspaceStore();
const { openInIde } = useOpenInIde();
const { activeWorkspace: workspace, workspaceMods: mods } = storeToRefs(ws);

const workspaceId = computed(() => String(route.params.id ?? ""));
const gameId = computed(() => workspace.value?.gameId ?? "");
const live = useLiveEnabled(workspaceId);

const modId = shallowRef("");
const from = shallowRef("");
const to = shallowRef("");

const { data: list } = useQuery({
  key: () => ["wiki-patches", gameId.value],
  query: () => Patches(gameId.value),
  enabled: () => !!gameId.value,
});

const { data: installs } = useQuery({
  key: () => ["patcher-installs", workspaceId.value, gameId.value],
  query: async () => (await ListGameInstalls(gameId.value)) ?? [],
  enabled: () => !!workspaceId.value && !!gameId.value,
});

const versions = computed(() => {
  const seen = new Set<string>();
  const out: { label: string; value: string }[] = [];
  for (const p of list.value?.pages ?? []) {
    const v = p.version || p.title;
    if (!v || seen.has(v)) continue;
    seen.add(v);
    out.push({ label: v, value: v });
  }
  return out;
});

const liveMods = computed(() =>
  mods.value.filter((m) => !m.isBroken && m.path),
);

watch(
  [liveMods, versions, installs, workspace],
  () => {
    if (!modId.value && liveMods.value[0]) {
      modId.value = liveMods.value[0].id;
    }
    const instVer =
      installs.value?.find((i) => i.id === workspace.value?.installId)
        ?.version ?? "";
    if (!to.value) {
      to.value =
        versions.value.find((v) => v.value === instVer)?.value ??
        versions.value[0]?.value ??
        "";
    }
    if (!from.value) {
      from.value =
        versions.value.find((v) => v.value !== to.value)?.value ?? "";
    }
  },
  { immediate: true },
);

const {
  mutateAsync: runCheck,
  data: report,
  isLoading: checking,
  error: checkError,
} = useMutation({
  mutation: () => ModAffected(workspaceId.value, modId.value, from.value, to.value),
});

const rows = computed(() => report.value?.likely ?? []);
const canCheck = computed(
  () => live.value && !!modId.value && !!from.value && !!to.value,
);

const emptyWhy = computed(() => {
  const n = report.value?.notesWithoutMatch ?? 0;
  return (
    `Heuristic only — wiki Modding bullets often miss recent patches. ` +
    `${n} note${n === 1 ? "" : "s"} had no token match. ` +
    `Rescan the install if patch notes look stale.`
  );
});

const columns: TableColumn<AffectedRow>[] = [
  { accessorKey: "rel", header: "File" },
  { accessorKey: "why", header: "Why" },
  { accessorKey: "patchTitle", header: "Patch" },
  { id: "open", header: "" },
];

/** Run the heuristic against the live session. */
function check(): void {
  if (!canCheck.value) return;
  void runCheck();
}

/** Jump to the hit in the workspace IDE. */
function openRow(row: AffectedRow): void {
  void openInIde(workspaceId.value, row.path);
}

/** Switch to Patcher with the same mod and a matching target install. */
function sendToPatcher(): void {
  const install =
    installs.value?.find((i) => i.version === to.value)?.id ??
    workspace.value?.installId ??
    "";
  void router.replace({
    query: {
      ...route.query,
      tab: "patcher",
      mod: modId.value,
      install,
    },
  });
}
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-auto p-3">
    <p class="text-xs text-muted">
      Heuristic only. Wiki modding bullets are incomplete.
    </p>
    <p v-if="!live" class="text-sm text-muted">
      Index is not live yet. Wait for the session, then Check.
    </p>
    <p v-else-if="!versions.length" class="text-sm text-muted">
      No patch notes cached. Rescan the install to fetch wiki pages.
    </p>
    <div class="flex flex-wrap items-end gap-2">
      <UFormField label="Mod" class="min-w-40">
        <USelect
          v-model="modId"
          :items="liveMods.map((m) => ({ label: m.name, value: m.id }))"
          value-key="value"
          placeholder="Mod"
        />
      </UFormField>
      <UFormField label="From" class="w-32">
        <USelect
          v-model="from"
          :items="versions"
          value-key="value"
          placeholder="From"
        />
      </UFormField>
      <UFormField label="To" class="w-32">
        <USelect
          v-model="to"
          :items="versions"
          value-key="value"
          placeholder="To"
        />
      </UFormField>
      <UButton
        label="Check"
        icon="i-lucide-search"
        :disabled="!canCheck"
        :loading="checking"
        @click="check"
      />
      <UButton
        label="Send matching files to Patcher"
        color="neutral"
        variant="ghost"
        :disabled="!modId"
        @click="sendToPatcher"
      />
    </div>
    <UAlert
      v-if="checkError"
      color="error"
      variant="subtle"
      :description="checkError.message"
    />
    <p v-if="report && rows.length" class="text-sm text-muted">
      {{ rows.length }} likely · {{ report.notesWithoutMatch }} notes with no
      match
    </p>
    <UAlert
      v-else-if="report && !rows.length"
      color="neutral"
      variant="subtle"
      title="No likely hits"
      :description="emptyWhy"
    />
    <UTable v-if="rows.length" :data="rows" :columns="columns">
      <template #rel-cell="{ row }">
        <span class="font-mono text-xs">{{ row.original.rel }}</span>
      </template>
      <template #why-cell="{ row }">
        <span class="text-xs">{{ row.original.why }}</span>
      </template>
      <template #open-cell="{ row }">
        <UButton
          label="IDE"
          size="xs"
          variant="ghost"
          @click="openRow(row.original)"
        />
      </template>
    </UTable>
  </div>
</template>
