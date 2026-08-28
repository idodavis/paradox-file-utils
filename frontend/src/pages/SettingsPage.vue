<script setup lang="ts">
/**
 * Settings page: fonts, data reset, workspace library link.
 */
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { storeToRefs } from "pinia";
import { ResetData } from "@services/settingsservice";
import {
  ListGameInstalls,
  UpdateGameInstall,
} from "@services/workspaceservice";
import { GameInstall } from "@services/internal/repos/models";
import FileSelector from "../components/FileSelector.vue";
import { useSettingsStore } from "../stores/settings";
import { useWorkspaceStore } from "../stores/workspace";

const router = useRouter();
const settings = useSettingsStore();
const ws = useWorkspaceStore();
const { fontScale, loading, saving } = storeToRefs(settings);

const installs = ref<GameInstall[]>([]);
const docsEdits = ref<Record<string, string>>({});

const resetting = ref(false);
const resetOpen = ref(false);
const message = ref("");
const messageKind = ref<"positive" | "negative" | "info">("info");

const messageColor = computed(() => {
  switch (messageKind.value) {
    case "negative":
      return "error";
    case "positive":
      return "success";
    case "info":
      return "info";
    default: {
      const _exhaustive: never = messageKind.value;
      return _exhaustive;
    }
  }
});

/** Set status message. */
function setMessage(text: string, kind: "positive" | "negative" | "info" = "info"): void {
  message.value = text;
  messageKind.value = kind;
}

/** Reload settings. */
async function load(): Promise<void> {
  try {
    await settings.load();
    installs.value = (await ListGameInstalls(ws.currentGameId)) ?? [];
    const next: Record<string, string> = {};
    for (const inst of installs.value) {
      next[inst.id] = inst.docsPath || "";
    }
    docsEdits.value = next;
    setMessage("Settings loaded.", "info");
  } catch (error) {
    setMessage(
      `Failed to load: ${error instanceof Error ? error.message : String(error)}`,
      "negative",
    );
  }
}

/** Save settings. */
async function save(): Promise<void> {
  try {
    await settings.save();
    setMessage("Settings saved.", "positive");
  } catch (error) {
    setMessage(
      `Save failed: ${error instanceof Error ? error.message : String(error)}`,
      "negative",
    );
  }
}

/** Save docs path override for an install. */
async function saveInstall(inst: GameInstall): Promise<void> {
  try {
    await UpdateGameInstall(inst.id, inst.path, docsEdits.value[inst.id] || "");
    setMessage("Install updated.", "positive");
    await load();
  } catch (error) {
    setMessage(
      `Install update failed: ${error instanceof Error ? error.message : String(error)}`,
      "negative",
    );
  }
}

/** Reset all data. */
async function resetData(): Promise<void> {
  resetOpen.value = false;
  resetting.value = true;
  try {
    await ResetData();
    await load();
    setMessage("Data reset complete.", "positive");
  } catch (error) {
    setMessage(
      `Reset failed: ${error instanceof Error ? error.message : String(error)}`,
      "negative",
    );
  } finally {
    resetting.value = false;
  }
}

/** Navigate to library. */
function goToLibrary(): void {
  void router.push({ name: "library" });
}

onMounted(load);
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-auto p-4">
    <div class="mx-auto w-full max-w-xl">
      <UCard>
        <template #header>
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h1 class="text-lg font-semibold">Settings</h1>
              <p class="text-xs text-muted">App configuration and data management</p>
            </div>
            <UButton label="Library" icon="i-lucide-library" color="neutral" variant="outline" size="sm"
              @click="goToLibrary" />
          </div>
        </template>

        <div class="space-y-4">
          <UAlert v-if="message" :color="messageColor" variant="subtle" :description="message" />

          <UCard variant="subtle">
            <template #header>
              <span class="font-medium">Appearance</span>
            </template>
            <div class="space-y-4">
              <UFormField label="UI scale"
                :description="`App chrome at ${fontScale}% (editor font is set in the workbench)`">
                <div class="flex items-center gap-3">
                  <UButton icon="i-lucide-minus" size="xs" color="neutral" variant="outline"
                    :disabled="fontScale <= settings.FONT_SCALE_MIN" @click="settings.setFontScale(fontScale - 5)" />
                  <span class="w-12 text-center text-sm tabular-nums">{{ fontScale }}%</span>
                  <UButton icon="i-lucide-plus" size="xs" color="neutral" variant="outline"
                    :disabled="fontScale >= settings.FONT_SCALE_MAX" @click="settings.setFontScale(fontScale + 5)" />
                </div>
              </UFormField>
            </div>
          </UCard>

          <UCard variant="subtle">
            <template #header>
              <span class="font-medium">Game Installs & Workspaces</span>
            </template>
            <div class="space-y-3">
              <p class="text-sm text-muted">
                Override install and script_docs paths for {{ ws.currentGameId.toUpperCase() }}.
              </p>
              <div v-for="inst in installs" :key="inst.id" class="space-y-2 rounded border border-default p-2">
                <p class="text-sm font-medium">{{ inst.name }} ({{ inst.version || "unknown" }})</p>
                <FileSelector :model-value="inst.path" mode="folder" label="Install path"
                  dialog-title="Select game install folder" @update:model-value="(p: string) => { inst.path = p }" />
                <FileSelector v-model="docsEdits[inst.id]" mode="folder" label="Docs path (empty = detected)"
                  dialog-title="Select script_docs folder" />
                <UButton label="Save paths" size="xs" variant="outline" @click="saveInstall(inst)" />
              </div>
              <p v-if="!installs.length" class="text-sm text-muted">
                No installs yet — add one from the workspace wizard.
              </p>
            </div>
            <template #footer>
              <UButton label="Open Library" icon="i-lucide-library" size="sm" @click="goToLibrary" />
            </template>
          </UCard>

          <hr class="border-default" />

          <div class="flex flex-wrap items-center justify-between gap-2">
            <UButton label="Reset all data" color="error" variant="outline" size="sm" :loading="resetting"
              @click="resetOpen = true" />
            <div class="flex gap-2">
              <UButton label="Reload" color="neutral" variant="outline" size="sm" :loading="loading" @click="load" />
              <UButton label="Save" size="sm" :loading="saving" @click="save" />
            </div>
          </div>
        </div>
      </UCard>
    </div>

    <UModal v-model:open="resetOpen" title="Reset all data?"
      description="This will delete workspaces, indexes, patch cache, and script logs. Settings will be kept.">
      <template #footer="{ close }">
        <UButton label="Cancel" color="neutral" variant="outline" @click="close" />
        <UButton label="Reset" color="error" :loading="resetting" @click="resetData" />
      </template>
    </UModal>
  </div>
</template>
