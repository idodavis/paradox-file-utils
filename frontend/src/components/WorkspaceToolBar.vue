<script setup lang="ts">
/**
 * Shared toolbar for workspace tool routes (IDE, graph, conflicts, loc, patcher).
 */
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { NavigationMenuItem } from "@nuxt/ui";
import { useSettingsStore } from "../stores/settings";
import { WORKSPACE_TOOLS } from "../workspaceTools";

const props = defineProps<{
  workspaceId: string;
  title: string;
  active?: string;
}>();

const route = useRoute();
const router = useRouter();
const settings = useSettingsStore();

const id = computed(() => props.workspaceId || String(route.params.id ?? ""));

/** History back unless the previous entry is the create-workspace wizard. */
function goBack(): void {
  const back = window.history.state?.back;
  if (!back || String(back).includes("/wizard")) {
    void router.replace({ name: "library" });
    return;
  }
  router.back();
}

/** Tool routes as navigation-menu items with active-state highlighting. */
const toolItems = computed<NavigationMenuItem[]>(() => {
  const visible = settings.visibleTools;
  let shown = WORKSPACE_TOOLS.filter((t) => visible.includes(t.name));
  if (!shown.length) {
    shown = WORKSPACE_TOOLS.filter((t) => t.name === "workspace-ide");
  }
  return shown.map((t) => ({
    label: t.label,
    icon: t.icon,
    active: props.active ? props.active === t.name : route.name === t.name,
    to: { name: t.name, params: { id: id.value } },
  }));
});
</script>

<template>
  <UDashboardToolbar class="px-2 sm:px-2">
    <template #left>
      <UButton
        icon="i-lucide-arrow-left"
        label="Back"
        color="neutral"
        variant="ghost"
        size="sm"
        @click="goBack"
      />
      <span class="font-semibold text-default">{{ title }}</span>
      <UNavigationMenu
        :items="toolItems"
        variant="pill"
        highlight
        class="flex-wrap"
      />
    </template>
    <template #right>
      <slot name="trailing" />
    </template>
  </UDashboardToolbar>
</template>
