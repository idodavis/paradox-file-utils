<script setup lang="ts">
/**
 * Patch Center: wiki versions list, patch HTML, script log import.
 */
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import FileSelector from "../components/FileSelector.vue";
import { GetWorkspace } from "@services/workspaceservice";
import { Workspace, WikiPatch, ScriptLogImport } from "@services/internal/repos/models";
import { ListVersions, GetPatchModdingSection } from "@services/wikiservice";
import { PatchVersion } from "@services/models";
import { ImportScriptLog, ListScriptLogImports } from "@services/scriptlogservice";
import { OpenURL } from "@services/browserservice";

type ScriptLogSummary = { errorCount?: number; warningCount?: number };

const route = useRoute();
const router = useRouter();

const workspaceId = computed(() => route.params.id as string);
const workspace = ref<Workspace | null>(null);
const versions = ref<PatchVersion[]>([]);
const selectedVersion = ref<string | null>(null);
const patchContent = ref<WikiPatch | null>(null);
const scriptLogs = ref<ScriptLogImport[]>([]);
const logPath = ref("");
const loading = ref(false);
const error = ref("");
const wikiBlocked = computed(() => /Cloudflare|wiki API blocked/i.test(error.value));

/** Load workspace and wiki versions. */
async function loadData(): Promise<void> {
  loading.value = true;
  error.value = "";
  try {
    workspace.value = await GetWorkspace(workspaceId.value);
    if (!workspace.value) throw new Error("Workspace not found");
    versions.value = (await ListVersions(workspace.value.gameId)) ?? [];
    scriptLogs.value = (await ListScriptLogImports(workspaceId.value)) ?? [];
    if (versions.value.length && !selectedVersion.value) {
      selectedVersion.value = versions.value[0].version;
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

/** Load patch content for selected version. */
async function loadPatch(): Promise<void> {
  if (!selectedVersion.value || !workspace.value) return;
  loading.value = true;
  error.value = "";
  try {
    patchContent.value = await GetPatchModdingSection(workspace.value.gameId, selectedVersion.value);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
    patchContent.value = null;
  } finally {
    loading.value = false;
  }
}

/** Import a script log file. */
async function importLog(): Promise<void> {
  if (!logPath.value) return;
  loading.value = true;
  try {
    await ImportScriptLog(workspaceId.value, logPath.value);
    scriptLogs.value = (await ListScriptLogImports(workspaceId.value)) ?? [];
    logPath.value = "";
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
}

/** Parse log summary JSON. */
function parseLogSummary(log: ScriptLogImport): ScriptLogSummary | null {
  try {
    return JSON.parse(log.summary) as ScriptLogSummary;
  } catch {
    return null;
  }
}

/** Open the selected patch page in the system browser. */
async function openWiki(): Promise<void> {
  const url = patchContent.value?.sourceUrl
    ?? versions.value.find((v) => v.version === selectedVersion.value)?.url
    ?? "";
  if (!url) return;
  try {
    await OpenURL(url);
  } catch {
    /* ignore */
  }
}

watch(selectedVersion, loadPatch);
watch(workspaceId, loadData, { immediate: true });

onMounted(loadData);
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden">
    <div class="flex shrink-0 items-center justify-between gap-2 border-b border-default bg-muted/50 px-3 py-2">
      <div class="flex items-center gap-2">
        <UButton
          icon="i-lucide-arrow-left"
          variant="ghost"
          size="sm"
          @click="router.push({ name: 'workspace-ide', params: { id: workspaceId } })"
        />
        <span class="font-semibold">Patch Center</span>
        <UBadge v-if="workspace" color="neutral" variant="outline" size="xs">
          {{ workspace.gameId.toUpperCase() }}
        </UBadge>
      </div>
      <UButton
        label="Open Patcher"
        icon="i-lucide-git-merge"
        size="sm"
        @click="router.push({ name: 'patcher', params: { id: workspaceId } })"
      />
    </div>

    <UAlert v-if="error" color="error" variant="subtle" :description="error" class="m-2">
      <template v-if="wikiBlocked" #actions>
        <UButton
          size="xs"
          label="Open wiki in browser"
          icon="i-lucide-external-link"
          variant="outline"
          @click="openWiki"
        />
      </template>
    </UAlert>

    <div class="flex min-h-0 flex-1 gap-3 overflow-hidden p-3">
      <div class="flex w-56 shrink-0 flex-col gap-3 overflow-auto">
        <UCard :ui="{ body: 'p-2' }">
          <template #header>
            <span class="text-sm font-semibold">Patch Versions</span>
          </template>
          <div class="max-h-48 space-y-1 overflow-auto">
            <button
              v-for="v in versions"
              :key="v.version"
              class="w-full rounded px-2 py-1 text-left text-sm hover:bg-muted"
              :class="{ 'bg-primary/10 text-primary': selectedVersion === v.version }"
              @click="selectedVersion = v.version"
            >
              {{ v.version }}
            </button>
          </div>
        </UCard>

        <UCard :ui="{ body: 'p-2' }">
          <template #header>
            <span class="text-sm font-semibold">Script Logs</span>
          </template>
          <div class="space-y-2">
            <FileSelector
              v-model="logPath"
              mode="file"
              label=""
              dialog-title="Select error.log"
              file-filter="*.log; *.txt"
              placeholder="Select error.log"
            />
            <UButton
              label="Import"
              icon="i-lucide-upload"
              size="xs"
              :disabled="!logPath"
              :loading="loading"
              @click="importLog"
            />
            <div v-for="log in scriptLogs" :key="log.id" class="rounded border border-default p-1 text-xs">
              <p class="truncate text-muted">{{ log.path.split(/[/\\]/).pop() }}</p>
              <template v-if="parseLogSummary(log)">
                <span class="text-error">{{ parseLogSummary(log)?.errorCount ?? 0 }} errors</span>
                <span class="mx-1 text-muted">|</span>
                <span class="text-warning">{{ parseLogSummary(log)?.warningCount ?? 0 }} warnings</span>
              </template>
            </div>
          </div>
        </UCard>
      </div>

      <UCard class="min-h-0 min-w-0 flex-1" :ui="{ body: 'overflow-auto' }">
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-semibold">Patch {{ selectedVersion ?? "..." }} - Modding Changes</span>
            <a
              v-if="patchContent?.sourceUrl"
              :href="patchContent.sourceUrl"
              target="_blank"
              class="text-sm text-primary hover:underline"
            >
              View on Wiki
            </a>
          </div>
        </template>
        <div v-if="loading" class="flex items-center justify-center py-8">
          <UButton loading variant="ghost" label="Loading..." />
        </div>
        <div
          v-else-if="patchContent?.htmlContent"
          class="prose prose-sm max-w-none dark:prose-invert"
          v-html="patchContent.htmlContent"
        />
        <UEmpty
          v-else
          icon="i-lucide-file-text"
          title="No modding section"
          description="This patch may not have documented modding changes."
        />
      </UCard>
    </div>
  </div>
</template>
