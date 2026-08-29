<script setup lang="ts">
/**
 * Event inspector header: id, loc title, badges, relative path, Open in IDE.
 */
import type { EventDetail } from "@services/internal/graph/models";

defineProps<{ detail: EventDetail }>();

const emit = defineEmits<{ open: [] }>();
</script>

<template>
  <div class="shrink-0 space-y-1.5 border-b border-default px-2 py-1.5">
    <div class="truncate text-sm font-semibold text-default">
      {{ detail.id }}
    </div>
    <div
      v-if="detail.title?.text"
      class="truncate text-xs text-muted"
    >
      {{ detail.title.text }}
    </div>
    <div class="flex flex-wrap items-center gap-1">
      <UBadge
        v-if="detail.type"
        :label="detail.type"
        color="neutral"
        variant="subtle"
        size="xs"
      />
      <UBadge
        v-if="detail.hidden"
        label="hidden"
        color="warning"
        variant="subtle"
        size="xs"
      />
      <UBadge
        v-if="detail.theme"
        :label="detail.theme"
        color="info"
        variant="subtle"
        size="xs"
      />
      <UBadge
        :label="detail.origin ? 'mod' : 'vanilla'"
        :color="detail.origin ? 'primary' : 'neutral'"
        variant="subtle"
        size="xs"
      />
    </div>
    <button
      v-if="detail.file"
      type="button"
      class="block w-full truncate text-left text-xs text-muted
        hover:text-default hover:underline"
      :title="detail.file"
      @click="emit('open')"
    >
      {{ detail.rel || detail.file }}:{{ detail.line + 1 }}
    </button>
    <UButton
      label="Open in IDE"
      icon="i-lucide-file-code"
      size="xs"
      color="neutral"
      variant="ghost"
      :disabled="!detail.file"
      @click="emit('open')"
    />
  </div>
</template>
