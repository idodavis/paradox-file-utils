<script setup lang="ts">
/**
 * Compact install/cache/index health strip; state lives in useLanguageHealth.
 */
import { useLanguageHealth } from "../composables/useLanguageHealth";

const props = defineProps<{
  workspaceId: string;
}>();

const {
  health,
  scanning,
  scanPct,
  scanMsg,
  lastScanned,
  rescan,
} = useLanguageHealth(() => props.workspaceId);
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
    <span>{{
      health.scriptDocsEffects
        ? `${health.scriptDocsEffects} script_docs`
        : "no script_docs"
    }}</span>
    <span v-if="health.indexReady">{{ health.defCount }} defs</span>
    <span v-else>index pending</span>
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
    <span v-if="!health.scriptDocsEffects && health.dumpHint" class="opacity-80">
      {{ health.dumpHint }}
    </span>
  </div>
</template>
