<script setup lang="ts">
/**
 * Compact install/cache/index health strip for Library and IDE, with scan progress.
 */
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { Events } from "@wailsio/runtime";
import {
  EnsureSession,
  GetLanguageHealth,
  LaunchGameDebug,
  RebuildWorkspaceSemantics,
} from "@services/languagemodelservice";
import type { LanguageHealth } from "@services/models";

const props = defineProps<{
  workspaceId: string;
}>();

const health = ref<LanguageHealth | null>(null);
const scanning = ref(false);
const launching = ref(false);
const scanPct = ref(0);
const scanMsg = ref("");

const lastScanned = computed(() => {
  const raw = health.value?.scannedAt;
  if (!raw) return "";
  const ms = Date.parse(raw);
  if (Number.isNaN(ms)) return raw;
  return new Date(ms).toLocaleString();
});

/** Load health for the current workspace. */
async function load(): Promise<void> {
  if (!props.workspaceId) {
    health.value = null;
    return;
  }
  try {
    health.value = (await GetLanguageHealth(props.workspaceId)) ?? null;
  } catch {
    health.value = null;
  }
}

/** Auto-rescan install semantics then reindex. */
async function rescan(): Promise<void> {
  if (!props.workspaceId || scanning.value) return;
  scanning.value = true;
  scanPct.value = 0;
  scanMsg.value = "starting";
  try {
    await RebuildWorkspaceSemantics(props.workspaceId);
    await EnsureSession(props.workspaceId);
    await load();
  } finally {
    scanning.value = false;
    scanMsg.value = "";
    scanPct.value = 0;
  }
}

async function launchDebug(): Promise<void> {
  if (!props.workspaceId || launching.value) return;
  launching.value = true;
  try {
    await LaunchGameDebug(props.workspaceId);
    await load();
  } finally {
    launching.value = false;
  }
}

function onFocus(): void {
  if (!props.workspaceId) return;
  void load();
}

/** Apply a scan-progress Wails event if it is for this workspace. */
function onScanProgress(ev: { data?: unknown }): void {
  const data = ev.data;
  if (!data || typeof data !== "object") return;
  const row = data as { workspaceId?: string; pct?: number; msg?: string };
  if (row.workspaceId && row.workspaceId !== props.workspaceId) return;
  if (typeof row.pct === "number") scanPct.value = row.pct;
  if (typeof row.msg === "string") scanMsg.value = row.msg;
}

let offScan: (() => void) | undefined;

onMounted(() => {
  void load();
  window.addEventListener("focus", onFocus);
  offScan = Events.On("lang:scan-progress", onScanProgress);
});
onUnmounted(() => {
  window.removeEventListener("focus", onFocus);
  offScan?.();
});
watch(
  () => props.workspaceId,
  () => void load(),
);
</script>

<template>
  <div
    v-if="health"
    class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted"
  >
    <span :class="health.installOk ? 'text-success' : 'text-error'">
      {{ health.installOk ? "Install OK" : "Install missing" }}
    </span>
    <span v-if="health.gameVersion">
      {{ health.gameVersion
      }}<template v-if="health.cacheStale"> (cache stale)</template>
    </span>
    <span>{{ health.docsPresent ? "script_docs found" : "no dumps" }}</span>
    <span v-if="health.indexReady">{{ health.defCount }} defs</span>
    <span v-else>index pending</span>
    <UButton
      label="Launch debug"
      size="xs"
      color="neutral"
      variant="ghost"
      :loading="launching"
      @click.stop="launchDebug"
    />
    <UButton
      label="Rescan"
      size="xs"
      color="neutral"
      variant="ghost"
      :loading="scanning"
      @click.stop="rescan"
    />
    <span v-if="!scanning && lastScanned">Scanned {{ lastScanned }}</span>
    <div
      v-if="scanning"
      class="flex min-w-40 max-w-64 flex-1 items-center gap-2"
    >
      <UProgress
        :model-value="scanPct"
        :max="100"
        size="xs"
        class="min-w-24 flex-1"
      />
      <span class="shrink-0 text-xs text-muted">{{ scanMsg }}</span>
    </div>
    <span v-if="!health.docsPresent && health.dumpHint" class="opacity-80">
      {{ health.dumpHint }}
    </span>
  </div>
</template>
