<script setup lang="ts">
/**
 * Hub page that mirrors the current PMT card-based entry experience.
 */
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import ck3Bg from "@assets/CK3-All_Under_Heaven.jpg";
import eu5Bg from "@assets/EUV-Release.jpg";
import { OpenURL } from "@services/browserservice";
import { GetLatestPatchNotes } from "@services/steamservice";
import type { LatestPatchNotes } from "@services/models";
import { useCurrentGame } from "../composables/appContext";

const router = useRouter();
const currentGame = useCurrentGame();

const tools = [
  {
    key: "modding-docs",
    title: "Modding Docs",
    description: "Browse script help files (.info / readme.txt) and modding wiki for CK3 and EU5.",
  },
  {
    key: "compare-tool",
    title: "File Compare",
    description: "Compare two file sets or directories and view diffs side-by-side.",
  },
  {
    key: "merge-tool",
    title: "Script Merger",
    description: "Merge Paradox script files with configurable options and conflict review.",
  },
  {
    key: "inventory",
    title: "Inventory Explorer",
    description: "Extract and explore game objects from script files with filtering and references.",
  },
] as const;

const latestPatchNotes = ref<Record<string, LatestPatchNotes>>({});
const patchNotesDialogOpen = ref(false);
const backgroundImage = computed(() => (currentGame.value === "EU5" ? eu5Bg : ck3Bg));
const currentPatchNotes = computed(() => latestPatchNotes.value[currentGame.value]);

/** Navigate to a tool page by route name. */
function openTool(routeName: string): void {
  void router.push({ name: routeName });
}

/** Open the patch notes modal when notes are available. */
async function openPatchNotes(): Promise<void> {
  if (currentPatchNotes.value) patchNotesDialogOpen.value = true;
}

/** Open the current patch notes URL in the system browser. */
async function openPatchNotesUrl(): Promise<void> {
  if (currentPatchNotes.value?.url) await OpenURL(currentPatchNotes.value.url);
}

onMounted(async () => {
  latestPatchNotes.value.CK3 = await GetLatestPatchNotes("CK3");
  latestPatchNotes.value.EU5 = await GetLatestPatchNotes("EU5");
});
</script>

<template>
  <div class="relative z-10 mx-auto flex min-h-0 flex-1 flex-col gap-6 p-4 lg:max-w-7xl lg:flex-row lg:items-center">
    <div class="flex-1">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div v-for="tool in tools" :key="tool.key">
          <UCard variant="subtle" class="h-full shadow-lg bg-elevated/85">
            <template #header>
              <div class="text-base font-semibold">{{ tool.title }}</div>
            </template>

            <div class="text-sm text-muted">{{ tool.description }}</div>

            <template #footer>
              <div class="flex justify-end">
                <UButton label="Open" color="primary" variant="outline" size="sm" @click="openTool(tool.key)" />
              </div>
            </template>
          </UCard>
        </div>
      </div>
    </div>

    <div class="w-full cursor-pointer lg:w-auto" @click="openPatchNotes">
      <UCard class="w-full shadow-xl bg-elevated/85 transition-colors hover:bg-elevated lg:max-w-100">
        <img :src="backgroundImage" alt="Game Wallpaper" class="aspect-4/3 w-full object-cover opacity-85" />
        <template #header>
          <div class="mb-2 inline-flex rounded-full bg-secondary px-3 py-1 text-sm font-medium text-secondary-content">
            Latest Patch Notes
          </div>
          <div class="text-base font-semibold">{{ currentPatchNotes?.title ?? "Loading latest patch notes..." }}</div>
        </template>

        <div v-if="currentPatchNotes" class="line-clamp-6 text-sm text-muted">
          {{ currentPatchNotes.contents }}
        </div>
        <UEmpty v-else icon="i-lucide-newspaper" title="Loading latest patch notes..." />

        <template #footer>
          <div class="flex justify-end">
            <UButton label="Open" color="secondary" variant="outline" size="sm" @click.stop="openPatchNotesUrl" />
          </div>
        </template>
      </UCard>
    </div>

    <UModal v-model:open="patchNotesDialogOpen"
      :ui="{ content: 'sm:max-w-[96vw] bg-default flex flex-col overflow-auto' }">
      <template #body>
        <div v-if="currentPatchNotes" class="space-y-4">
          <div class="flex items-center justify-between border-b border-default pb-2">
            <h3 class="truncate font-bold text-accent">{{ currentPatchNotes.title }}</h3>
          </div>
          <div class="prose prose-invert max-w-none" v-html="currentPatchNotes.contents" />
          <div class="flex justify-end gap-2">
            <UButton label="Open on SteamDB" color="secondary" variant="outline" @click="openPatchNotesUrl" />
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
