/**
 * Workspace helpers and GameId — Pinia-backed via useWorkspaceStore.
 * Kept for stable import paths used across pages.
 */
export {
  GAME_OPTIONS,
  useWorkspaceStore,
  type GameId,
} from "../stores/workspace";

import { useWorkspaceStore } from "../stores/workspace";

/** Prefer useWorkspaceStore; alias for call-site migration. */
export function useWorkspaceContext() {
  return useWorkspaceStore();
}
