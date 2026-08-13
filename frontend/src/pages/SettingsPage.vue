<script setup lang="ts">
/**
 * Vue port of the PMT settings page.
 */
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import FileSelector from "../components/FileSelector.vue";
import { GetSettings, SaveSettings } from "@services/settingsservice";
import { ResetData } from "@services/dbservice";
import { normalizeSettings } from "../composables/settings";

const router = useRouter();
const loading = ref(false);
const saving = ref(false);
const resetting = ref(false);
const resetOpen = ref(false);
const message = ref("");
const messageKind = ref<"positive" | "negative" | "info">("info");
const settings = ref<Record<string, string>>({});
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

const ck3InstallPath = computed({
  get: () => settings.value["ck3.install_path"] ?? "",
  set: (value: string) => {
    settings.value["ck3.install_path"] = value;
  },
});

const eu5InstallPath = computed({
  get: () => settings.value["eu5.install_path"] ?? "",
  set: (value: string) => {
    settings.value["eu5.install_path"] = value;
  },
});

/** Set the status alert text and color. */
function setMessage(text: string, kind: "positive" | "negative" | "info" = "info"): void {
  message.value = text;
  messageKind.value = kind;
}

/** Reload settings from the backend. */
async function load(): Promise<void> {
  loading.value = true;
  try {
    settings.value = normalizeSettings(await GetSettings());
    setMessage("Settings loaded.", "info");
  } catch (error) {
    setMessage(`Failed to load settings: ${error instanceof Error ? error.message : String(error)}`, "negative");
  } finally {
    loading.value = false;
  }
}

/** Persist the current settings map. */
async function save(): Promise<void> {
  saving.value = true;
  try {
    await SaveSettings(settings.value);
    setMessage("Settings saved.", "positive");
  } catch (error) {
    setMessage(`Save failed: ${error instanceof Error ? error.message : String(error)}`, "negative");
  } finally {
    saving.value = false;
  }
}

/** Reset inventories and caches after the confirm modal is accepted. */
async function resetData(): Promise<void> {
  resetOpen.value = false;
  resetting.value = true;
  try {
    await ResetData();
    await load();
    setMessage("Data reset complete.", "positive");
  } catch (error) {
    setMessage(`Reset failed: ${error instanceof Error ? error.message : String(error)}`, "negative");
  } finally {
    resetting.value = false;
  }
}

/** Navigate back to the hub. */
function backToHub(): void {
  void router.push({ name: "hub" });
}

onMounted(load);
</script>

<template>
  <div class="p-3">
    <UCard>
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h1 class="text-lg font-semibold">Settings</h1>
            <p class="text-xs text-muted">Game install directories used by Docs, Compare, and Merge</p>
          </div>
          <UButton label="Back" icon="i-lucide-arrow-left" color="neutral" variant="outline" size="sm" @click="backToHub" />
        </div>
      </template>

      <div class="space-y-4">
        <UAlert v-if="message" :color="messageColor" variant="subtle" :description="message" />
        <FileSelector
          v-model="ck3InstallPath"
          mode="folder"
          label="CK3 install directory"
          dialog-title="Select CK3 game directory"
          placeholder="Select CK3 game directory"
        />
        <FileSelector
          v-model="eu5InstallPath"
          mode="folder"
          label="EU5 install directory"
          dialog-title="Select EU5 game directory"
          placeholder="Select EU5 game directory"
        />
        <div class="flex flex-wrap items-center justify-between gap-2">
          <UButton
            label="Reset all data"
            color="error"
            variant="outline"
            size="sm"
            :loading="resetting"
            @click="resetOpen = true"
          />
          <div class="flex gap-2">
            <UButton label="Reload" color="neutral" variant="outline" size="sm" :loading="loading" @click="load" />
            <UButton label="Save Settings" size="sm" :loading="saving" @click="save" />
          </div>
        </div>
      </div>
    </UCard>

    <UModal
      v-model:open="resetOpen"
      title="Reset all data?"
      description="This will delete inventories, doc cache, and patch notes. Game install paths and constants will be kept."
    >
      <template #footer="{ close }">
        <UButton label="Cancel" color="neutral" variant="outline" @click="close" />
        <UButton label="Reset" color="error" :loading="resetting" @click="resetData" />
      </template>
    </UModal>
  </div>
</template>
