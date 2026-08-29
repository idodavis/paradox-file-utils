<script setup lang="ts">
/**
 * Shared toolbar for workspace tool routes (IDE, graph, conflicts, loc, patcher).
 */
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { NavigationMenuItem } from "@nuxt/ui";

const props = defineProps<{
  workspaceId: string;
  title: string;
  active?: string;
}>();

const route = useRoute();
const router = useRouter();

const id = computed(() => props.workspaceId || String(route.params.id ?? ""));

const tools: { label: string; icon: string; name: string }[] = [
  { label: "IDE", icon: "i-lucide-code", name: "workspace-ide" },
  { label: "Event Graph", icon: "i-lucide-git-fork", name: "event-graph" },
  { label: "Conflicts", icon: "i-lucide-layers", name: "conflicts" },
  { label: "Loc Coverage", icon: "i-lucide-languages", name: "loc-coverage" },
  { label: "Mod Patcher", icon: "i-lucide-git-compare", name: "patcher" },
];

/** Tool routes as navigation-menu items with active-state highlighting. */
const toolItems = computed<NavigationMenuItem[]>(() =>
  tools.map((t) => ({
    label: t.label,
    icon: t.icon,
    active: props.active ? props.active === t.name : route.name === t.name,
    to: { name: t.name, params: { id: id.value } },
  })),
);
</script>

<template>
  <div
    class="flex shrink-0 flex-wrap items-center gap-2 border-b border-default
      bg-muted/50 px-2 py-1.5"
  >
    <UButton
      icon="i-lucide-arrow-left"
      label="Library"
      color="neutral"
      variant="ghost"
      size="sm"
      @click="router.push({ name: 'library' })"
    />
    <span class="font-semibold text-default">{{ title }}</span>
    <UNavigationMenu
      :items="toolItems"
      variant="pill"
      highlight
      class="flex-wrap"
    />
    <div class="ml-auto flex items-center gap-2">
      <slot name="trailing" />
    </div>
  </div>
</template>
