<script setup lang="ts">
/**
 * Selected-item chrome for Views pages: empty state, header, body slot.
 */
import { useOpenInIde } from "../composables/useOpenInIde";

const props = defineProps<{
  workspaceId: string;
  title?: string;
  subtitle?: string;
  file?: string;
  rel?: string;
  line?: number;
}>();

defineSlots<{
  origin(): unknown;
  badges(): unknown;
  default(): unknown;
  empty(): unknown;
}>();

const { openInIde } = useOpenInIde();

/** Open the header file in the workspace IDE. */
function open(): void {
  if (props.file) void openInIde(props.workspaceId, props.file, props.line);
}
</script>

<template>
  <aside
    class="flex h-full min-h-0 w-full flex-col overflow-hidden border-l border-default bg-elevated"
  >
    <div
      v-if="!title"
      class="m-2 rounded-md border border-dashed border-muted px-3 py-4 text-xs text-muted"
    >
      <slot name="empty">Select a row.</slot>
    </div>
    <template v-else>
      <div class="shrink-0 space-y-1.5 border-b border-default px-2 py-1.5">
        <div class="truncate text-sm font-semibold text-default">
          {{ title }}
        </div>
        <div v-if="subtitle" class="truncate text-xs text-muted">
          {{ subtitle }}
        </div>
        <div
          v-if="$slots.origin || $slots.badges"
          class="flex flex-wrap items-center gap-1"
        >
          <slot name="origin" />
          <slot name="badges" />
        </div>
        <div
          v-if="file"
          class="truncate text-xs text-muted"
          :title="file"
        >
          {{ rel || file }}:{{ (line ?? 0) + 1 }}
        </div>
        <UButton
          label="Open in IDE"
          icon="i-lucide-file-code"
          size="xs"
          color="neutral"
          variant="ghost"
          :disabled="!file"
          @click="open"
        />
      </div>
      <div v-if="$slots.default" class="min-h-0 flex-1 overflow-auto p-2">
        <slot />
      </div>
    </template>
  </aside>
</template>
