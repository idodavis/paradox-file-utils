<script setup lang="ts">
/**
 * Vue app root that preserves the PMT app identity and top-level flow.
 */
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import ck3Bg from "../src/assets/CK3-All_Under_Heaven.jpg";
import eu5Bg from "../src/assets/EUV-Release.jpg";
import appIcon from "../src/assets/PMT-SquareIcon-Mint.png?url";
import PMTLogo from "./components/PMTLogo.vue";
import HelpDialog from "./components/HelpDialog.vue";
import { provideCurrentGame } from "./composables/appContext";
import {
  CheckForUpdates,
  GetSettings,
  GetVersion,
  SaveSettings,
} from "../bindings/paradox-modding-tools/services/settingsservice";

const route = useRoute();
const router = useRouter();
const DARK_THEMES = new Set(["PMT", "dracula", "luxury", "business", "coffee", "dim"]);

const themeOptions = ["PMT", "retro", "pastel", "dracula", "luxury", "autumn", "business", "coffee", "dim"];
const gameOptions: { label: string; value: "CK3" | "EU5" }[] = [
  { label: "CK3", value: "CK3" },
  { label: "EU5", value: "EU5" },
];

const currentTheme = ref("PMT");
const currentGame = ref<"CK3" | "EU5">("CK3");
const version = ref("...");
const helpOpen = ref(false);
const appSettings = ref<Record<string, string>>({});

const backgroundImage = computed(() => (currentGame.value === "EU5" ? eu5Bg : ck3Bg));
const currentTitle = computed(() => String(route.meta.title ?? "Tools"));
const currentDescription = computed(() => String(route.meta.description ?? ""));
const isHub = computed(() => route.name === "hub");

function setTheme(theme: string): void {
  currentTheme.value = theme;
  document.documentElement.dataset.theme = theme;
  document.documentElement.classList.toggle("dark", DARK_THEMES.has(theme));
}

function updateSettings(key: string, value: string): void {
  appSettings.value = { ...appSettings.value, [key]: value };
}

async function loadSettings(): Promise<void> {
  const settings = (await GetSettings()) ?? {};
  appSettings.value = Object.fromEntries(Object.entries(settings).filter(([, value]) => value !== undefined)) as Record<
    string,
    string
  >;
  setTheme(appSettings.value["_global.theme"] ?? "PMT");
}

async function saveTheme(theme: string): Promise<void> {
  updateSettings("_global.theme", theme);
  await SaveSettings(appSettings.value);
}

async function loadVersion(): Promise<void> {
  try {
    version.value = await GetVersion();
  } catch (error) {
    version.value = error instanceof Error ? error.message : String(error);
  }
}

function gotoHub(): void {
  void router.push({ name: "hub" });
}

async function onThemeChange(theme: string | null): Promise<void> {
  if (!theme || !themeOptions.includes(theme)) return;
  setTheme(theme);
  await saveTheme(theme);
}

function setGame(game: "CK3" | "EU5"): void {
  currentGame.value = game;
  localStorage.setItem("_global.game", game);
}

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
            <UButton
              v-if="!isHub"
              label="Hub"
              icon="i-lucide-arrow-left"
              color="neutral"
              variant="ghost"
              size="sm"
              @click="gotoHub"
            />
            <PMTLogo v-else :icon-height="40" :text-height="50" />
            <template v-if="!isHub">
              <span class="text-muted">/</span>
              <h1 class="truncate text-lg font-semibold">{{ currentTitle }}</h1>
            </template>
          </div>
          <div class="flex items-center gap-2">
            <USelect
              :model-value="currentGame"
              :items="gameOptions"
              value-key="value"
              class="w-24"
              @update:model-value="setGame"
            />

            <USelectMenu
              :model-value="currentTheme"
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

      <template v-if="isHub">
        <main class="relative flex-1 min-h-0 overflow-auto">
          <router-view />
          <div
            class="absolute inset-0 z-0 bg-cover bg-center opacity-55"
            :style="{ backgroundImage: `url(${backgroundImage})` }"
          />
          <div
            class="absolute inset-0 z-0 bg-[radial-gradient(ellipse_at_center,var(--ui-bg-muted)_0%,transparent_75%)]"
          />
        </main>
      </template>
      <main v-else class="flex min-h-0 flex-1 overflow-hidden">
        <router-view />
      </main>

      <footer
        class="shrink-0 z-10 flex items-center justify-between border-t border-default bg-default px-2 text-sm text-default"
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
