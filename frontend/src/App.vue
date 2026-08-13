<script setup lang="ts">
/**
 * Vue app root that preserves the PMT app identity and top-level flow.
 */
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import ck3Bg from "@assets/CK3-All_Under_Heaven.jpg";
import eu5Bg from "@assets/EUV-Release.jpg";
import appIcon from "@assets/PMT-SquareIcon-Mint.png?url";
import PMTLogo from "./components/PMTLogo.vue";
import HelpDialog from "./components/HelpDialog.vue";
import { provideCurrentGame, type GameId } from "./composables/appContext";
import { normalizeSettings } from "./composables/settings";
import {
  CheckForUpdates,
  GetSettings,
  GetVersion,
  SaveSettings,
} from "@services/settingsservice";

const route = useRoute();
const router = useRouter();
const DARK_THEMES = new Set(["PMT", "dracula", "luxury", "business", "coffee", "dim"]);

const themeOptions = [
  "PMT",
  "retro",
  "pastel",
  "dracula",
  "luxury",
  "autumn",
  "business",
  "coffee",
  "dim",
];
const gameOptions: { label: string; value: GameId }[] = [
  { label: "CK3", value: "CK3" },
  { label: "EU5", value: "EU5" },
];

const currentTheme = ref("PMT");
const currentGame = ref<GameId>("CK3");
const version = ref("...");
const helpOpen = ref(false);
const appSettings = ref<Record<string, string>>({});

const backgroundImage = computed(() => (currentGame.value === "EU5" ? eu5Bg : ck3Bg));
const currentTitle = computed(() => String(route.meta.title ?? "Tools"));
const currentDescription = computed(() => String(route.meta.description ?? ""));
const isHub = computed(() => route.name === "hub");
const headerItems = computed(() => [
  { label: "Hub", icon: "i-lucide-house", to: { name: "hub" } },
  { label: currentTitle.value },
]);

/** Apply a theme name to the document and local state. */
function setTheme(theme: string): void {
  currentTheme.value = theme;
  document.documentElement.dataset.theme = theme;
  document.documentElement.classList.toggle("dark", DARK_THEMES.has(theme));
}

/** Merge a single settings key into the in-memory settings map. */
function updateSettings(key: string, value: string): void {
  appSettings.value = { ...appSettings.value, [key]: value };
}

/** Load persisted settings and apply the saved theme. */
async function loadSettings(): Promise<void> {
  appSettings.value = normalizeSettings(await GetSettings());
  setTheme(appSettings.value["_global.theme"] ?? "PMT");
}

/** Persist the current theme to backend settings. */
async function saveTheme(theme: string): Promise<void> {
  updateSettings("_global.theme", theme);
  await SaveSettings(appSettings.value);
}

/** Load the app version string from the backend. */
async function loadVersion(): Promise<void> {
  try {
    version.value = await GetVersion();
  } catch (error) {
    version.value = error instanceof Error ? error.message : String(error);
  }
}

/** Persist the selected game id to localStorage. */
function persistGame(game: string | null): void {
  if (game !== "CK3" && game !== "EU5") return;
  localStorage.setItem("_global.game", game);
}

/** Apply and persist a theme selection from USelectMenu. */
async function onThemeChange(theme: string | null): Promise<void> {
  if (!theme || !themeOptions.includes(theme)) return;
  setTheme(theme);
  await saveTheme(theme);
}

/** Ask the backend to check for application updates. */
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

provideCurrentGame(currentGame);

onMounted(async () => {
  const savedGame = localStorage.getItem("_global.game");
  if (savedGame === "CK3" || savedGame === "EU5") {
    currentGame.value = savedGame;
  }
  await Promise.all([loadSettings(), loadVersion()]);
  const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
  if (link) link.href = appIcon;
});
</script>

<template>
  <UApp>
    <div class="pmt-root flex min-h-screen flex-col">
      <header class="z-10 shrink-0 border-b border-default bg-default px-2 py-2 shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 flex-1 items-center gap-2">
            <PMTLogo v-if="isHub" :icon-height="40" :text-height="50" />
            <UBreadcrumb v-else :items="headerItems" />
          </div>
          <div class="flex items-center gap-2">
            <USelect
              v-model="currentGame"
              :items="gameOptions"
              value-key="value"
              class="w-24"
              @update:model-value="persistGame"
            />
            <USelectMenu
              v-model="currentTheme"
              :items="themeOptions"
              class="w-40"
              icon="i-lucide-palette"
              @update:model-value="onThemeChange"
            />
            <UButton
              icon="i-lucide-settings"
              color="neutral"
              variant="ghost"
              @click="router.push({ name: 'settings' })"
            />
          </div>
        </div>
      </header>

      <main v-if="isHub" class="relative min-h-0 flex-1 overflow-auto">
        <router-view />
        <div
          class="absolute inset-0 z-0 bg-cover bg-center opacity-55"
          :style="{ backgroundImage: `url(${backgroundImage})` }"
        />
        <div
          class="absolute inset-0 z-0 bg-[radial-gradient(ellipse_at_center,var(--ui-bg-muted)_0%,transparent_75%)]"
        />
      </main>
      <main v-else class="flex min-h-0 flex-1 overflow-hidden">
        <router-view />
      </main>

      <footer
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

      <HelpDialog v-model="helpOpen" :title="currentTitle" :description="currentDescription" />
    </div>
  </UApp>
</template>
