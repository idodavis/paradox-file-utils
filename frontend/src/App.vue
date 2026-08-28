<script setup lang="ts">
/**
 * Vue app root with library-centric shell and workbench singleton host.
 */
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { DropdownMenuItem } from "@nuxt/ui";
import appIcon from "@assets/PMT-SquareIcon-Mint.png?url";
import PMTLogo from "./components/PMTLogo.vue";
import GameIcon from "./components/GameIcon.vue";
import { GAME_OPTIONS, useWorkspaceStore } from "./stores/workspace";
import { useSettingsStore } from "./stores/settings";
import {
  THEME_SWATCHES,
  themeMenuItems,
  type ThemeMenuItem,
  type ThemeSwatchColors,
} from "./composables/themeSwatches";
import { CheckForUpdates, GetVersion } from "@services/settingsservice";
import { isWorkbenchReady } from "./ide/workbenchHost";
import IdeWorkbenchLayout from "./components/IdeWorkbenchLayout.vue";
import { applyWorkbenchTheme } from "./ide/themeBridge";
import {
  isDarkTheme,
  normalizeThemeName,
  PMT_THEME_NAMES,
} from "./ide/appThemes";
import { useIdeShellStore } from "./stores/ideShell";

const route = useRoute();
const router = useRouter();
const ws = useWorkspaceStore();
const settings = useSettingsStore();
const ideShell = useIdeShellStore();

const themeNames = PMT_THEME_NAMES;
const themeItems = themeMenuItems(themeNames);

const currentTheme = ref("PMT");
const currentThemeSwatches = computed(
  (): ThemeSwatchColors =>
    THEME_SWATCHES[currentTheme.value] ?? THEME_SWATCHES.PMT,
);
const version = ref("...");
const helpOpen = ref(false);

const currentTitle = computed(() => String(route.meta.title ?? "Tools"));
const currentDescription = computed(
  () => String(route.meta.description ?? ""),
);
const isLibrary = computed(() => route.name === "library");
const showWorkbench = computed(
  () => route.name === "workspace-ide" || ideShell.mergeReview,
);

const headerItems = computed(() => [
  { label: "Library", icon: "i-lucide-library", to: { name: "library" } },
  { label: currentTitle.value },
]);

/** Game id attached to a workspace dropdown group label. */
function menuGameId(item: unknown): string {
  if (!item || typeof item !== "object" || !("gameId" in item)) return "";
  const id = (item as { gameId?: unknown }).gameId;
  return typeof id === "string" ? id : "";
}

const workspaceDropdownItems = computed<DropdownMenuItem[][]>(() => {
  const items: DropdownMenuItem[][] = [];
  const all = ws.workspaces;
  const activeId = ws.activeWorkspaceId;
  const activeGame = ws.activeWorkspace?.gameId ?? ws.currentGameId;

  const toItem = (workspace: (typeof all)[number]): DropdownMenuItem => ({
    label: workspace.name,
    icon: workspace.id === activeId ? "i-lucide-check" : "i-lucide-folder",
    onSelect: () => {
      ws.setActiveWorkspace(workspace.id);
      void router.push({
        name: "workspace-ide",
        params: { id: workspace.id },
      });
    },
  });

  /** Game group heading with official icon when one ships. */
  const gameLabel = (gameId: string): DropdownMenuItem => ({
    label: gameId.toUpperCase(),
    type: "label" as const,
    disabled: true,
    slot: "game-group",
    gameId,
  });

  const sameGame = all.filter((w) => w.gameId === activeGame);
  const otherGames = GAME_OPTIONS.map((g) => g.value).filter(
    (id) => id !== activeGame,
  );

  if (sameGame.length) {
    items.push([gameLabel(activeGame), ...sameGame.slice(0, 8).map(toItem)]);
  }
  for (const gameId of otherGames) {
    const group = all.filter((w) => w.gameId === gameId);
    if (!group.length) continue;
    items.push([gameLabel(gameId), ...group.slice(0, 5).map(toItem)]);
  }
  items.push([
    {
      label: "New Workspace",
      icon: "i-lucide-plus",
      onSelect: () => router.push({ name: "wizard" }),
    },
    {
      label: "Library",
      icon: "i-lucide-library",
      onSelect: () => router.push({ name: "library" }),
    },
    {
      label: "Ad-hoc Merge",
      icon: "i-lucide-git-merge",
      onSelect: () => router.push({ name: "tools-merge" }),
    },
  ]);
  return items;
});

/** Apply a theme to the document. */
function setTheme(theme: string): void {
  const name = normalizeThemeName(theme);
  currentTheme.value = name;
  document.documentElement.dataset.theme = name;
  document.documentElement.classList.toggle("dark", isDarkTheme(name));
}

/** Persist the theme via settings store. */
async function saveTheme(theme: string): Promise<void> {
  await settings.set("_global.theme", theme);
}

/** Load version string. */
async function loadVersion(): Promise<void> {
  try {
    version.value = await GetVersion();
  } catch (error) {
    version.value = error instanceof Error ? error.message : String(error);
  }
}

/** Apply and persist a theme; sync workbench colors when ready. */
async function onThemeChange(theme: string | null): Promise<void> {
  if (!theme || !(theme in THEME_SWATCHES)) return;
  setTheme(theme);
  await saveTheme(theme);
  if (isWorkbenchReady()) {
    await applyWorkbenchTheme(theme);
  }
}

/** Swatch colors for a SelectMenu theme item. */
function itemSwatches(
  item: ThemeMenuItem | string | undefined,
): ThemeSwatchColors {
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
  await Promise.all([settings.load(), loadVersion(), ws.refresh()]);
  setTheme(normalizeThemeName(settings.values["_global.theme"]));
  const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
  if (link) link.href = appIcon;
});
</script>

<template>
  <UApp>
    <div class="pmt-root flex h-screen min-h-0 flex-col overflow-hidden">
      <header
        class="z-10 shrink-0 border-b border-default bg-default px-2 py-2 shadow-sm"
      >
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 flex-1 items-center gap-2">
            <PMTLogo v-if="isLibrary" :icon-height="36" :text-height="44" />
            <UBreadcrumb v-else :items="headerItems" />
          </div>
          <div class="flex items-center gap-2">
            <UDropdownMenu :items="workspaceDropdownItems">
              <template #game-group-leading="{ item }">
                <GameIcon :game-id="menuGameId(item)" />
              </template>
              <UButton
                :label="ws.activeWorkspaceName"
                trailing-icon="i-lucide-chevron-down"
                color="neutral"
                variant="outline"
                class="max-w-48 truncate"
              >
                <template
                  v-if="ws.activeWorkspace?.gameId"
                  #leading
                >
                  <GameIcon :game-id="ws.activeWorkspace.gameId" />
                </template>
              </UButton>
            </UDropdownMenu>
            <USelectMenu
              v-model="currentTheme"
              :items="themeItems"
              value-key="value"
              :ui="{
                content: 'min-w-44',
                base: 'w-auto gap-1.5 ps-2 pe-2',
                leading: 'static inset-auto',
                trailing: 'static inset-auto',
                value: 'hidden',
              }"
              @update:model-value="onThemeChange"
            >
              <template #leading>
                <span
                  class="inline-flex shrink-0 overflow-hidden rounded-sm border border-default"
                  aria-hidden="true"
                >
                  <span
                    v-for="(color, i) in currentThemeSwatches"
                    :key="i"
                    class="size-3.5"
                    :style="{ backgroundColor: color }"
                  />
                </span>
              </template>
              <template #default>
                <span class="sr-only">{{ currentTheme }}</span>
              </template>
              <template #item-leading="{ item }">
                <span
                  class="inline-flex shrink-0 overflow-hidden rounded-sm border border-default"
                  aria-hidden="true"
                >
                  <span
                    v-for="(color, i) in itemSwatches(item as ThemeMenuItem)"
                    :key="i"
                    class="size-3.5"
                    :style="{ backgroundColor: color }"
                  />
                </span>
              </template>
            </USelectMenu>
            <UButton
              icon="i-lucide-settings"
              color="neutral"
              variant="ghost"
              @click="router.push({ name: 'settings' })"
            />
          </div>
        </div>
      </header>

      <main class="relative flex min-h-0 flex-1 flex-col overflow-hidden">
        <div
          v-show="!showWorkbench"
          class="min-h-0 min-w-0 flex-1 overflow-hidden"
        >
          <router-view />
        </div>
        <IdeWorkbenchLayout
          v-if="showWorkbench"
          :visible="showWorkbench"
          :theme="currentTheme"
        >
          <template #toolbar>
            <div
              v-if="route.name === 'workspace-ide'"
              class="shrink-0 border-b border-default"
            >
              <router-view />
            </div>
            <div
              v-else-if="ideShell.mergeReview"
              class="flex shrink-0 items-center gap-2 border-b border-default px-2 py-1"
            >
              <UButton
                label="Back to Patcher"
                icon="i-lucide-arrow-left"
                size="sm"
                color="neutral"
                variant="ghost"
                @click="ideShell.endMergeReview()"
              />
              <span class="text-xs text-muted">Reviewing diffs in workbench</span>
            </div>
          </template>
        </IdeWorkbenchLayout>
      </main>

      <footer
        v-show="!showWorkbench"
        class="z-10 flex shrink-0 items-center justify-between border-t border-default bg-default px-2 text-sm text-default"
      >
        <PMTLogo :icon-height="25" :text-height="30" />
        <div class="flex items-center gap-2">
          <UButton
            :label="`Help for ${currentTitle}`"
            icon="i-lucide-circle-help"
            color="neutral"
            variant="ghost"
            size="sm"
            @click="helpOpen = true"
          />
          <UButton
            :label="version"
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="ghost"
            size="sm"
            @click="checkForUpdates"
          />
        </div>
      </footer>

      <UModal
        v-model:open="helpOpen"
        :title="currentTitle"
        :description="currentDescription"
        :ui="{ content: 'sm:max-w-2xl' }"
      >
        <template #footer="{ close }">
          <UButton
            label="Close"
            color="neutral"
            variant="outline"
            icon="i-lucide-x"
            @click="close"
          />
        </template>
      </UModal>
    </div>
  </UApp>
</template>
