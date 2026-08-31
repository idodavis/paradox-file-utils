/**
 * Pinia Colada helper: enable session queries only after a live language session.
 */
import { shallowRef, toValue, watch, type MaybeRefOrGetter } from "vue";
import { useWorkspaceStore } from "../stores/workspace";

/**
 * True after `ensureReady()` for the given workspace id.
 * Use as `enabled` so session queries do not fire against a dead session.
 */
export function useLiveEnabled(workspaceId: MaybeRefOrGetter<string>) {
  const live = shallowRef(false);
  watch(
    () => toValue(workspaceId),
    async (id) => {
      live.value = false;
      if (!id) return;
      live.value = await useWorkspaceStore().ensureReady();
    },
    { immediate: true },
  );
  return live;
}
