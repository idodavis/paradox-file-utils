/**
 * Pinia store for active game, workspace list, and selection.
 */
import { computed, ref, watch } from "vue";
import { defineStore } from "pinia";
import {
  ListWorkspaces,
  GetWorkspace,
  ListWorkspaceMods,
  MarkBrokenPaths,
} from "@services/workspaceservice";
import { Workspace, WorkspaceMod } from "@services/internal/repos/models";

/** Supported game identifiers. */
export type GameId = "ck3" | "eu5" | "vic3";

/** Game options for selector dropdowns. */
export const GAME_OPTIONS: { label: string; value: GameId }[] = [
  { label: "CK3", value: "ck3" },
  { label: "EU5", value: "eu5" },
  { label: "Vic3", value: "vic3" },
];

/** Shared workspace library and active-workspace selection. */
export const useWorkspaceStore = defineStore("workspace", () => {
  const currentGameId = ref<GameId>(
    (localStorage.getItem("workspace.gameId") as GameId) || "ck3",
  );
  const activeWorkspaceId = ref<string | null>(
    localStorage.getItem("workspace.activeId") || null,
  );
  const workspaces = ref<Workspace[]>([]);
  const activeWorkspace = ref<Workspace | null>(null);
  const workspaceMods = ref<WorkspaceMod[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  const hasWorkspaces = computed(() => workspaces.value.length > 0);
  const activeWorkspaceName = computed(
    () => activeWorkspace.value?.name ?? "No workspace",
  );

  /** Refresh all workspaces (every game). */
  async function refresh(): Promise<void> {
    loading.value = true;
    error.value = null;
    try {
      await MarkBrokenPaths();
      workspaces.value = (await ListWorkspaces("")) ?? [];
      if (activeWorkspaceId.value) {
        await loadActiveWorkspace();
      } else if (workspaces.value.length) {
        const match =
          workspaces.value.find((w) => w.gameId === currentGameId.value) ??
          workspaces.value[0];
        if (match?.gameId) currentGameId.value = match.gameId as GameId;
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e);
    } finally {
      loading.value = false;
    }
  }

  /** Load active workspace details and mods. */
  async function loadActiveWorkspace(): Promise<void> {
    if (!activeWorkspaceId.value) {
      activeWorkspace.value = null;
      workspaceMods.value = [];
      return;
    }
    try {
      activeWorkspace.value = await GetWorkspace(activeWorkspaceId.value);
      workspaceMods.value =
        (await ListWorkspaceMods(activeWorkspaceId.value)) ?? [];
    } catch {
      activeWorkspace.value = null;
      workspaceMods.value = [];
    }
  }

  /** Switch to a different workspace by ID. */
  function setActiveWorkspace(id: string | null): void {
    activeWorkspaceId.value = id;
    if (id) {
      localStorage.setItem("workspace.activeId", id);
      const found = workspaces.value.find((w) => w.id === id);
      if (found?.gameId) {
        currentGameId.value = found.gameId as GameId;
        localStorage.setItem("workspace.gameId", found.gameId);
      }
    } else {
      localStorage.removeItem("workspace.activeId");
    }
    void loadActiveWorkspace();
  }

  /** Switch to a different game. */
  function setGame(gameId: GameId): void {
    currentGameId.value = gameId;
    localStorage.setItem("workspace.gameId", gameId);
    activeWorkspaceId.value = null;
    activeWorkspace.value = null;
    workspaceMods.value = [];
    void refresh();
  }

  watch(currentGameId, () => {
    localStorage.setItem("workspace.gameId", currentGameId.value);
  });

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
    refresh,
    loadActiveWorkspace,
    setActiveWorkspace,
    setGame,
  };
});
