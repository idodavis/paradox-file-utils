<script setup lang="ts">
/**
 * IDE-only footer cluster: Problems counts, language, line and column.
 */
import { useIdeStatus } from "../composables/useIdeStatus";

const { language, line, column, errors, warnings, openProblems } =
  useIdeStatus(true);
</script>

<template>
  <div class="flex min-w-0 items-center gap-2 text-xs text-muted">
    <UTooltip text="Problems">
      <span class="inline-flex items-center">
        <UButton
          color="neutral"
          variant="ghost"
          size="xs"
          icon="i-lucide-circle-x"
          :label="String(errors)"
          class="px-1"
          :ui="{ leadingIcon: 'text-error' }"
          @click="openProblems"
        />
        <UButton
          color="neutral"
          variant="ghost"
          size="xs"
          icon="i-lucide-triangle-alert"
          :label="String(warnings)"
          class="px-1"
          :ui="{ leadingIcon: 'text-warning' }"
          @click="openProblems"
        />
      </span>
    </UTooltip>
    <span v-if="language" class="truncate">{{ language }}</span>
    <span v-if="line">Ln {{ line }}, Col {{ column }}</span>
  </div>
</template>
