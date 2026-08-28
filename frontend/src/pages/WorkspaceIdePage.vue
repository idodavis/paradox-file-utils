<script setup lang="ts">
/**
 * Workspace IDE toolbar; workbench fills App.vue host below this strip.
 */
import { computed, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { useWorkspaceStore } from "../stores/workspace";
import { useSettingsStore } from "../stores/settings";
import { setWorkbenchRoots } from "../ide/workbenchHost";
import { buildIdeRoots } from "../ide/workspaceFolders";
import { EnsureSession } from "@services/languagemodelservice";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";

const route = useRoute();
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
      void EnsureSession(workspaceId.value);
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
    await setWorkbenchRoots(roots, theme);
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
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
  <WorkspaceToolBar
    :workspace-id="workspaceId"
    title="Workspace IDE"
    active="workspace-ide"
  >
    <template #trailing>
      <span v-if="loading" class="text-xs text-muted">Loading workbench…</span>
      <span v-if="error" class="text-xs text-error">{{ error }}</span>
      <LanguageHealthStrip
        v-if="workspaceId"
        :workspace-id="workspaceId"
      />
    </template>
  </WorkspaceToolBar>
</template>
