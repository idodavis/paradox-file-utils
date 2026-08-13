<script setup lang="ts">
/**
 * Vue app root with library-centric shell navigation.
 */
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { DropdownMenuItem } from "@nuxt/ui";
import appIcon from "@assets/PMT-SquareIcon-Mint.png?url";
import PMTLogo from "./components/PMTLogo.vue";
import HelpDialog from "./components/HelpDialog.vue";
import {
  createWorkspaceContext,
  provideWorkspaceContext,
  GAME_OPTIONS,
} from "./composables/workspaceContext";
import { normalizeSettings } from "./composables/settings";
import {
  THEME_SWATCHES,
  themeMenuItems,
  type ThemeMenuItem,
  type ThemeSwatchColors,
} from "./composables/themeSwatches";
import {
  CheckForUpdates,
  GetSettings,
  GetVersion,
  SaveSettings,
} from "@services/settingsservice";

const route = useRoute();
const router = useRouter();

const DARK_THEMES = new Set(["PMT", "dracula", "luxury", "business", "coffee", "dim"]);
const themeNames = ["PMT", "retro", "pastel", "dracula", "luxury", "autumn", "business", "coffee", "dim"] as const;
const themeItems = themeMenuItems(themeNames);

const wsContext = createWorkspaceContext();
provideWorkspaceContext(wsContext);

const currentTheme = ref("PMT");
const currentThemeSwatches = computed(
  (): ThemeSwatchColors => THEME_SWATCHES[currentTheme.value] ?? THEME_SWATCHES.PMT,
);
const version = ref("...");
const helpOpen = ref(false);
const appSettings = ref<Record<string, string>>({});

const currentTitle = computed(() => String(route.meta.title ?? "Tools"));
const currentDescription = computed(() => String(route.meta.description ?? ""));
const isLibrary = computed(() => route.name === "library");

const headerItems = computed(() => [
  { label: "Library", icon: "i-lucide-library", to: { name: "library" } },
  { label: currentTitle.value },
]);

const workspaceDropdownItems = computed<DropdownMenuItem[][]>(() => {
  const items: DropdownMenuItem[][] = [];
  const all = wsContext.workspaces.value;
  const activeId = wsContext.activeWorkspaceId.value;
  const activeGame = wsContext.activeWorkspace.value?.gameId
    ?? wsContext.currentGameId.value;

  const toItem = (ws: (typeof all)[number]): DropdownMenuItem => ({
    label: ws.name,
    icon: ws.id === activeId ? "i-lucide-check" : "i-lucide-folder",
    onSelect: () => {
      wsContext.setActiveWorkspace(ws.id);
      void router.push({ name: "workspace-ide", params: { id: ws.id } });
    },
  });

  const sameGame = all.filter((ws) => ws.gameId === activeGame);
  const otherGames = GAME_OPTIONS
    .map((g) => g.value)
    .filter((id) => id !== activeGame);

  if (sameGame.length) {
    items.push(sameGame.slice(0, 8).map(toItem));
  }
  for (const gameId of otherGames) {
    const group = all.filter((ws) => ws.gameId === gameId);
    if (!group.length) continue;
    items.push([
      { label: gameId.toUpperCase(), type: "label" as const, disabled: true },
      ...group.slice(0, 5).map(toItem),
    ]);
  }
  items.push([
    { label: "New Workspace", icon: "i-lucide-plus", onSelect: () => router.push({ name: "wizard" }) },
    { label: "Library", icon: "i-lucide-library", onSelect: () => router.push({ name: "library" }) },
    { label: "Ad-hoc Merge", icon: "i-lucide-git-merge", onSelect: () => router.push({ name: "tools-merge" }) },
  ]);
  return items;
});

/** Apply a theme to the document. */
function setTheme(theme: string): void {
  currentTheme.value = theme;
  document.documentElement.dataset.theme = theme;
  document.documentElement.classList.toggle("dark", DARK_THEMES.has(theme));
}

/** Merge a settings key. */
function updateSettings(key: string, value: string): void {
  appSettings.value = { ...appSettings.value, [key]: value };
}

/** Load settings and apply saved theme. */
async function loadSettings(): Promise<void> {
  appSettings.value = normalizeSettings(await GetSettings());
  setTheme(appSettings.value["_global.theme"] ?? "PMT");
}

/** Persist the theme. */
async function saveTheme(theme: string): Promise<void> {
  updateSettings("_global.theme", theme);
  await SaveSettings(appSettings.value);
}

/** Load version string. */
async function loadVersion(): Promise<void> {
  try {
    version.value = await GetVersion();
  } catch (error) {
    version.value = error instanceof Error ? error.message : String(error);
  }
}

/** Apply and persist a theme. */
async function onThemeChange(theme: string | null): Promise<void> {
  if (!theme || !(theme in THEME_SWATCHES)) return;
  setTheme(theme);
  await saveTheme(theme);
}

/** Swatch colors for a SelectMenu theme item. */
function itemSwatches(item: ThemeMenuItem | string | undefined): ThemeSwatchColors {
  if (!item) return THEME_SWATCHES.PMT;
  if (typeof item === "string") return THEME_SWATCHES[item] ?? THEME_SWATCHES.PMT;
  return item.swatches;
}

/** Check for app updates. */
async function checkForUpdates(): Promise<void> {
  await CheckForUpdates();
}

watch(
  currentTheme,
  (value) => {
    document.documentElement.dataset.theme = value;
  },
  { immediate: true },
);

onMounted(async () => {
  await Promise.all([loadSettings(), loadVersion(), wsContext.refresh()]);
  const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
  if (link) link.href = appIcon;
});
</script>

<template>
  <UApp>
    <div class="pmt-root flex h-screen min-h-0 flex-col overflow-hidden">
      <header class="z-10 shrink-0 border-b border-default bg-default px-2 py-2 shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 flex-1 items-center gap-2">
            <PMTLogo v-if="isLibrary" :icon-height="36" :text-height="44" />
            <UBreadcrumb v-else :items="headerItems" />
          </div>
          <div class="flex items-center gap-2">
            <UDropdownMenu :items="workspaceDropdownItems">
              <UButton :label="wsContext.activeWorkspaceName.value" trailing-icon="i-lucide-chevron-down"
                color="neutral" variant="outline" class="max-w-48 truncate" />
            </UDropdownMenu>
            <USelectMenu v-model="currentTheme" :items="themeItems" value-key="value" :ui="{
              content: 'min-w-44',
              base: 'w-auto gap-1.5 ps-2 pe-2',
              leading: 'static inset-auto',
              trailing: 'static inset-auto',
              value: 'hidden',
            }" @update:model-value="onThemeChange">
              <template #leading>
                <span class="inline-flex shrink-0 overflow-hidden rounded-sm border border-default" aria-hidden="true">
                  <span v-for="(color, i) in currentThemeSwatches" :key="i" class="size-3.5"
                    :style="{ backgroundColor: color }" />
                </span>
              </template>
              <template #default>
                <span class="sr-only">{{ currentTheme }}</span>
              </template>
              <template #item-leading="{ item }">
                <span class="inline-flex shrink-0 overflow-hidden rounded-sm border border-default" aria-hidden="true">
                  <span v-for="(color, i) in itemSwatches(item as ThemeMenuItem)" :key="i" class="size-3.5"
                    :style="{ backgroundColor: color }" />
                </span>
              </template>
            </USelectMenu>
            <UButton icon="i-lucide-settings" color="neutral" variant="ghost"
              @click="router.push({ name: 'settings' })" />
          </div>
        </div>
      </header>

      <main class="flex min-h-0 flex-1 flex-col overflow-hidden">
        <div class="min-h-0 min-w-0 flex-1 overflow-hidden">
          <router-view />
        </div>
      </main>

      <footer
        class="z-10 flex shrink-0 items-center justify-between border-t border-default bg-default px-2 text-sm text-default">
        <PMTLogo :icon-height="25" :text-height="30" />
        <div class="flex items-center gap-2">
          <UButton :label="`Help for ${currentTitle}`" icon="i-lucide-circle-help" color="neutral" variant="ghost"
            size="sm" @click="helpOpen = true" />
          <UButton :label="version" icon="i-lucide-refresh-cw" color="neutral" variant="ghost" size="sm"
            @click="checkForUpdates" />
        </div>
      </footer>

      <HelpDialog v-model="helpOpen" :title="currentTitle" :description="currentDescription" />
    </div>
  </UApp>
</template>
