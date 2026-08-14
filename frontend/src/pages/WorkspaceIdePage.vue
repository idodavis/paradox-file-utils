<script setup lang="ts">
/**
 * Workspace IDE toolbar; workbench fills App.vue host below this strip.
 */
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useWorkspaceStore } from "../stores/workspace";
import { useSettingsStore } from "../stores/settings";
import { setWorkbenchRoots } from "../ide/workbenchHost";
import { buildIdeRoots } from "../ide/workspaceFolders";

const route = useRoute();
const router = useRouter();
const ws = useWorkspaceStore();
const settings = useSettingsStore();

const workspaceId = computed(() => String(route.params.id ?? ""));
const error = ref("");
const loading = ref(false);

/** Load workspace and remount workbench roots (after App initializes host). */
async function boot(): Promise<void> {
  loading.value = true;
  error.value = "";
  try {
    if (workspaceId.value) {
      ws.setActiveWorkspace(workspaceId.value);
      await ws.loadActiveWorkspace();
    }
    const roots = await buildIdeRoots(ws.activeWorkspace, ws.workspaceMods);
    if (!roots.length) {
      error.value =
        "No game, mod, or staging paths available for this workspace.";
      return;
    }
    const theme =
      settings.values["_global.theme"] ||
      document.documentElement.dataset.theme ||
      "PMT";
    // Sets roots immediately (picked up if init is still running), then remounts.
    await setWorkbenchRoots(roots, theme);
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    // HMR / double-init noise — workbench is usable; don't alarm the user.
    if (!msg.includes("already initialized")) {
      error.value = msg;
    }
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void boot();
});

watch(workspaceId, () => {
  void boot();
});
</script>

<template>
  <div
    class="flex shrink-0 flex-wrap items-center gap-2 border-b border-default px-2 py-1.5"
  >
    <UButton
      label="Library"
      icon="i-lucide-library"
      color="neutral"
      variant="ghost"
      size="sm"
      @click="router.push({ name: 'library' })"
    />
    <UButton
      label="Patch Center"
      icon="i-lucide-book-open"
      color="neutral"
      variant="ghost"
      size="sm"
      @click="
        router.push({ name: 'patch-center', params: { id: workspaceId } })
      "
    />
    <UButton
      label="Mod Patcher"
      icon="i-lucide-git-compare"
      color="neutral"
      variant="ghost"
      size="sm"
      @click="router.push({ name: 'patcher', params: { id: workspaceId } })"
    />
    <UButton
      label="Event Graph"
      icon="i-lucide-share-2"
      color="neutral"
      variant="ghost"
      size="sm"
      @click="
        router.push({ name: 'event-graph', params: { id: workspaceId } })
      "
    />
    <span v-if="loading" class="text-xs text-muted">Loading workbench…</span>
    <span v-if="error" class="text-xs text-error">{{ error }}</span>
  </div>
</template>
