<script setup lang="ts">
/**
 * Workspace IDE toolbar; workbench fills App.vue host below this strip.
 */
import { computed } from "vue";
import { useRoute } from "vue-router";
import { useQuery } from "@pinia/colada";
import { useWorkspaceStore } from "../stores/workspace";
import { setWorkbenchRoots } from "../ide/workbenchHost";
import { currentWorkbenchTheme } from "../ide/colorThemes";
import { buildIdeRoots } from "../ide/workspaceFolders";
import { EnsureSession } from "@services/sessionservice";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";

defineOptions({ name: "WorkspaceIdePage" });

const route = useRoute();
const ws = useWorkspaceStore();

const workspaceId = computed(() => String(route.params.id ?? ""));

const { error, isPending } = useQuery({
  key: () => ["ide-boot", workspaceId.value],
  query: async () => {
    await ws.loadActiveWorkspace();
    if (workspaceId.value) await EnsureSession(workspaceId.value);
    const roots = await buildIdeRoots(workspaceId.value);
    if (!roots.length) {
      throw new Error(
        "No game, mod, or staging paths available for this workspace.",
      );
    }
    await setWorkbenchRoots(roots, currentWorkbenchTheme());
    return true;
  },
  enabled: () => !!workspaceId.value,
});
</script>

<template>
  <WorkspaceToolBar
    :workspace-id="workspaceId"
    title="Workspace IDE"
    active="workspace-ide"
  >
    <template #trailing>
      <span v-if="isPending" class="text-xs text-muted">Loading workbench…</span>
      <span v-if="error" class="text-xs text-error">{{ error.message }}</span>
      <LanguageHealthStrip
        v-if="workspaceId"
        :workspace-id="workspaceId"
      />
    </template>
  </WorkspaceToolBar>
</template>
