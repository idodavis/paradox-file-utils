<script setup lang="ts">
/**
 * Label plus an inline info icon whose tooltip holds the field description.
 */
defineProps<{
  /** Help copy shown in the tooltip. */
  text: string;
  /** Optional field name rendered immediately before the icon. */
  label?: string;
}>();
</script>

<template>
  <span class="inline-flex items-center gap-1">
    <span v-if="label">{{ label }}</span>
    <UTooltip
      :text="text"
      :delay-duration="150"
      arrow
      :content="{ side: 'bottom', align: 'start', sideOffset: 6 }"
      :ui="{
        content: 'h-auto max-w-64 items-start px-2.5 py-2',
        text: 'whitespace-normal text-pretty leading-snug',
      }"
    >
      <button
        type="button"
        class="inline-flex size-4 shrink-0 items-center justify-center
          text-muted hover:text-default"
        :aria-label="label ? `${label} help` : 'Help'"
      >
        <UIcon name="i-lucide-info" class="size-3.5" />
      </button>
    </UTooltip>
  </span>
</template>
