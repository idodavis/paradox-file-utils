<script setup lang="ts">
/**
 * Compact install/cache/index health strip for Library and IDE.
 */
import { onMounted, onUnmounted, ref, watch } from "vue";
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
  try {
    await RebuildWorkspaceSemantics(props.workspaceId);
    await EnsureSession(props.workspaceId);
    await load();
  } finally {
    scanning.value = false;
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

onMounted(() => {
  void load();
  window.addEventListener("focus", onFocus);
});
onUnmounted(() => {
  window.removeEventListener("focus", onFocus);
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
    <span v-if="!health.docsPresent && health.dumpHint" class="opacity-80">
      {{ health.dumpHint }}
    </span>
  </div>
</template>
