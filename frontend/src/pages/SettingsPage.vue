<script setup lang="ts">
/**
 * Vue port of the PMT settings page.
 */
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import FolderSelector from "../components/FolderSelector.vue";
import { GetSettings, SaveSettings } from "@services/settingsservice";
import { ResetData } from "@services/dbservice";

const router = useRouter();
const loading = ref(false);
const saving = ref(false);
const resetting = ref(false);
const message = ref("");
const messageKind = ref<"positive" | "negative" | "info">("info");
const settings = ref<Record<string, string>>({});
const messageColor = computed(() => {
  if (messageKind.value === "negative") return "error";
  if (messageKind.value === "positive") return "success";
  return "info";
});

function normalizeSettings(input: { [_ in string]?: string } | null): Record<string, string> {
  return Object.fromEntries(Object.entries(input ?? {}).filter(([, value]) => value !== undefined)) as Record<
    string,
    string
  >;
}

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

function setMessage(text: string, kind: "positive" | "negative" | "info" = "info"): void {
  message.value = text;
  messageKind.value = kind;
}

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

async function resetData(): Promise<void> {
  if (
    !confirm(
      "Reset all data? This will delete inventories, doc cache, and patch notes. Game install paths and constants will be kept.",
    )
  ) {
    return;
  }

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

function backToHub(): void {
  void router.push({ name: "hub" });
}

onMounted(load);
</script>

<template>
  <div class="p-4">
    <UCard>
      <template #header>
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div class="text-xs font-semibold uppercase tracking-wide text-primary">Application Settings</div>
            <div class="text-2xl font-bold">Game Install Directories</div>
            <div class="mt-2 text-sm text-muted">
              Set the top-level install path for each game. These are used by Modding Docs, Compare, and Merge.
            </div>
          </div>
          <UButton label="Back" icon="i-lucide-arrow-left" color="neutral" variant="outline" @click="backToHub" />
        </div>
      </template>

      <div class="space-y-6">
        <UAlert v-if="message" :color="messageColor" variant="subtle" :description="message" />

        <FolderSelector v-model="ck3InstallPath" label="CK3 install directory" dialog-title="Select CK3 game directory"
          placeholder="Select CK3 game directory"
          hint="Steam example: C:\\Program Files (x86)\\Steam\\steamapps\\common\\Crusader Kings III" />

        <FolderSelector v-model="eu5InstallPath" label="EU5 install directory" dialog-title="Select EU5 game directory"
          placeholder="Select EU5 game directory"
          hint="Steam example: C:\\Program Files (x86)\\Steam\\steamapps\\common\\Europa Universalis V" />

        <div class="flex flex-wrap items-center justify-between gap-3">
          <UButton label="Reset all data" color="error" variant="outline" :loading="resetting" @click="resetData" />
          <div class="flex gap-2">
            <UButton label="Reload" color="neutral" variant="outline" :loading="loading" @click="load" />
            <UButton label="Save Settings" :loading="saving" @click="save" />
          </div>
        </div>
      </div>
    </UCard>
  </div>
</template>
