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
import { GetIdeRoots } from "@services/workspaceservice";
import { EnsureSession } from "@services/sessionservice";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";
import { guideOpen, setGuideOpen } from "../composables/useGuidePrefs";

defineOptions({ name: "WorkspaceIdePage" });

const route = useRoute();
const ws = useWorkspaceStore();

const workspaceId = computed(() => String(route.params.id ?? ""));

const { error, isPending } = useQuery({
  key: () => ["ide-boot", workspaceId.value],
  query: async () => {
    const id = String(route.params.id ?? "");
    if (!id) return true;
    await ws.loadActiveWorkspace();
    // Mount folders even when the session fails (stale loc sidecar, etc.).
    // Re-throw after setWorkbenchRoots so the strip still shows the error.
    let sessionErr: unknown;
    try {
      await EnsureSession(id);
    } catch (e) {
      sessionErr = e;
    }
    const rec = await GetIdeRoots(id);
    const roots = rec?.roots ?? [];
    if (!roots.length) {
      throw new Error("No game, mod, or staging paths available for this workspace.");
    }
    await setWorkbenchRoots(roots, currentWorkbenchTheme(), {
      workspaceId: id,
      persistTabs: !rec?.resetIdeOnOpen,
      openFiles: rec?.ideOpenFiles ?? [],
      activeFile: rec?.ideActiveFile ?? "",
    });
    if (sessionErr) throw sessionErr;
    return true;
  },
  enabled: () => !!workspaceId.value,
});
</script>

<template>
  <WorkspaceToolBar :workspace-id="workspaceId">
    <template #trailing>
      <UButton
        icon="i-lucide-book-open"
        label="Guide"
        size="xs"
        color="neutral"
        :variant="guideOpen ? 'soft' : 'ghost'"
        @click="setGuideOpen(!guideOpen)"
      />
      <span v-if="isPending" class="text-xs text-muted">Loading workbench…</span>
      <span v-if="error" class="text-xs text-error">{{ error.message }}</span>
      <LanguageHealthStrip v-if="workspaceId" :workspace-id="workspaceId" />
    </template>
  </WorkspaceToolBar>
</template>
