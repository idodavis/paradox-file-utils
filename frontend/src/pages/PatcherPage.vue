<script setup lang="ts">
/**
 * Patch Center shell: Patch Notes, Impact Check, and file retarget.
 */
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { TabsItem } from "@nuxt/ui";
import WorkspaceToolBar from "../components/WorkspaceToolBar.vue";
import PatchNotesPane from "../components/patchCenter/PatchNotesPane.vue";
import ImpactCheckPane from "../components/patchCenter/ImpactCheckPane.vue";
import PatcherRetarget from "../components/patchCenter/PatcherRetarget.vue";

defineOptions({ name: "PatcherPage" });

type CenterTab = "notes" | "impact" | "patcher";

const route = useRoute();
const router = useRouter();
const workspaceId = computed(() => String(route.params.id ?? ""));

const items: TabsItem[] = [
  { label: "Patch Notes", value: "notes", icon: "i-lucide-file-text" },
  { label: "Impact Check", value: "impact", icon: "i-lucide-search" },
  { label: "Patcher", value: "patcher", icon: "i-lucide-arrow-left-right" },
];

/** Route query `tab`, default Patch Notes. */
const tab = computed({
  get(): string | number {
    return parseTab(route.query.tab);
  },
  set(next: string | number) {
    void router.replace({ query: { ...route.query, tab: parseTab(next) } });
  },
});

function parseTab(raw: unknown): CenterTab {
  const s = String(raw ?? "");
  switch (s) {
    case "notes":
    case "impact":
    case "patcher":
      return s;
    default:
      return "notes";
  }
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <WorkspaceToolBar :workspace-id="workspaceId" />
    <UTabs
      v-model="tab"
      :items="items"
      :content="false"
      variant="link"
      size="sm"
      class="shrink-0 px-2"
    />
    <PatchNotesPane v-show="tab === 'notes'" class="min-h-0 flex-1" />
    <ImpactCheckPane v-show="tab === 'impact'" class="min-h-0 flex-1" />
    <PatcherRetarget v-show="tab === 'patcher'" class="min-h-0 flex-1" />
  </div>
</template>
