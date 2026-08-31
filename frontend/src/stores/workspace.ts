/**
 * Pinia store for active game, workspace list, and selection.
 */
import { computed, ref } from "vue";
import { defineStore } from "pinia";
import { useLocalStorage } from "@vueuse/core";
import {
  ListWorkspaces,
  GetWorkspace,
  ListWorkspaceMods,
  MarkBrokenPaths,
} from "@services/workspaceservice";
import {
  EnsureSession,
  GetModelStatus,
} from "@services/sessionservice";
import { Workspace, WorkspaceMod } from "@services/models";

/** Supported game identifiers. */
export type GameId = "ck3" | "eu5" | "vic3";

/** Game options for selector dropdowns. */
export const GAME_OPTIONS: { label: string; value: GameId }[] = [
  { label: "CK3", value: "ck3" },
  { label: "EU5", value: "eu5" },
  { label: "Vic3", value: "vic3" },
];

/** Workspace default loc language choices (empty/english is the inherit lang). */
export const LOC_LANG_ITEMS: { label: string; value: string }[] = [
  { label: "English", value: "english" },
  { label: "French", value: "french" },
  { label: "German", value: "german" },
  { label: "Spanish", value: "spanish" },
  { label: "Russian", value: "russian" },
  { label: "Korean", value: "korean" },
  { label: "Simplified Chinese", value: "simp_chinese" },
  { label: "Polish", value: "polish" },
  { label: "Turkish", value: "turkish" },
  { label: "Brazilian Portuguese", value: "braz_por" },
  { label: "Japanese", value: "japanese" },
];

/** Shared workspace library and active-workspace selection. */
export const useWorkspaceStore = defineStore("workspace", () => {
  const currentGameId = useLocalStorage<GameId>("workspace.gameId", "ck3");
  const activeWorkspaceId = useLocalStorage("workspace.activeId", "");
  const workspaces = ref<Workspace[]>([]);
  const activeWorkspace = ref<Workspace | null>(null);
  const workspaceMods = ref<WorkspaceMod[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  let loadToken = 0;
  let inflight: Promise<void> | null = null;
  let inflightId = "";

  const hasWorkspaces = computed(() => workspaces.value.length > 0);
  const activeWorkspaceName = computed(
    () => activeWorkspace.value?.name ?? "No workspace",
  );
  const workspacesByGame = computed(() =>
    GAME_OPTIONS.map((g) => ({
      gameId: g.value,
      label: g.label,
      workspaces: workspaces.value.filter((w) => w.gameId === g.value),
    })).filter((s) => s.workspaces.length > 0),
  );

  /** Refresh all workspaces (every game). */
  async function refresh(): Promise<void> {
    loading.value = true;
    error.value = null;
    try {
      await MarkBrokenPaths();
      workspaces.value = (await ListWorkspaces("")) ?? [];
      const id = activeWorkspaceId.value;
      const listed = id
        ? workspaces.value.find((w) => w.id === id)
        : undefined;
      if (listed) {
        activeWorkspace.value = listed;
        workspaceMods.value = listed.mods ?? [];
        if (listed.gameId) currentGameId.value = listed.gameId as GameId;
      } else {
        if (id) activeWorkspaceId.value = "";
        activeWorkspace.value = null;
        workspaceMods.value = [];
        if (workspaces.value.length) {
          const match =
            workspaces.value.find((w) => w.gameId === currentGameId.value) ??
            workspaces.value[0];
          if (match?.gameId) currentGameId.value = match.gameId as GameId;
        }
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  /** Load active workspace details and mods (in-flight deduped per id). */
  async function loadActiveWorkspace(): Promise<void> {
    const id = activeWorkspaceId.value;
    if (!id) {
      activeWorkspace.value = null;
      workspaceMods.value = [];
      return;
    }
    if (inflight && inflightId === id) return inflight;
    const token = ++loadToken;
    inflightId = id;
    inflight = (async () => {
      const listed = workspaces.value.find((w) => w.id === id);
      if (listed) {
        if (token !== loadToken) return;
        activeWorkspace.value = listed;
        workspaceMods.value = listed.mods ?? [];
        return;
      }
      try {
        const [ws, mods] = await Promise.all([
          GetWorkspace(id),
          ListWorkspaceMods(id),
        ]);
        if (token !== loadToken) return;
        activeWorkspace.value = ws;
        workspaceMods.value = mods ?? [];
      } catch {
        if (token !== loadToken) return;
        if (activeWorkspaceId.value === id) activeWorkspaceId.value = "";
        activeWorkspace.value = null;
        workspaceMods.value = [];
      }
    })().finally(() => {
      if (token === loadToken) inflight = null;
    });
    return inflight;
  }

  /** Switch to a different workspace by ID. No-op when already selected. */
  function setActiveWorkspace(id: string | null): void {
    const next = id ?? "";
    if (next === activeWorkspaceId.value) {
      if (next && !activeWorkspace.value) void loadActiveWorkspace();
      return;
    }
    activeWorkspaceId.value = next;
    if (next) {
      const found = workspaces.value.find((w) => w.id === next);
      if (found?.gameId) currentGameId.value = found.gameId as GameId;
    }
    void loadActiveWorkspace();
  }

  /** True when this workspace has a live language session (not merely defs on disk). */
  async function ensureReady(): Promise<boolean> {
    const id = activeWorkspaceId.value;
    if (!id) return false;
    try {
      await EnsureSession(id);
      const st = await GetModelStatus(id);
      return Boolean(st?.live);
    } catch {
      return false;
    }
  }

  return {
    currentGameId,
    activeWorkspaceId,
    workspaces,
    activeWorkspace,
    workspaceMods,
    loading,
    error,
    hasWorkspaces,
    activeWorkspaceName,
    workspacesByGame,
    refresh,
    loadActiveWorkspace,
    setActiveWorkspace,
    ensureReady,
  };
});
