<script setup lang="ts">
/**
 * Patch Center: wiki patch notes for the workspace game.
 */
import { computed } from "vue";
import { useRoute } from "vue-router";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import PatchNotesPane from "../components/patchCenter/PatchNotesPane.vue";

defineOptions({ name: "PatcherPage" });

const route = useRoute();
const workspaceId = computed(() => String(route.params.id ?? ""));
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <WorkspaceToolBar :workspace-id="workspaceId" />
    <PatchNotesPane class="min-h-0 flex-1" />
  </div>
</template>
