<script setup lang="ts">
/**
 * Indented script lines plus clickable outbound target chips.
 */
import type {
  EventScriptLine,
  EventStepTarget,
} from "@services/internal/graph/models";

const props = defineProps<{
  lines?: EventScriptLine[] | null;
  targets?: EventStepTarget[] | null;
}>();

const emit = defineEmits<{
  select: [id: string];
}>();

/** Indent a rendered script line from Go (display only). */
function linePad(line: EventScriptLine): string {
  return `${Math.min(line.depth, 8) * 0.75}rem`;
}

/** Select a target event. */
function onTarget(t: EventStepTarget): void {
  emit("select", t.name);
}
</script>

<template>
  <div class="space-y-1">
    <div
      v-for="(line, i) in props.lines ?? []"
      :key="i"
      class="font-mono text-xs text-default"
      :style="{ paddingLeft: linePad(line) }"
    >
      {{ line.text }}
    </div>
    <div
      v-if="(props.targets ?? []).length"
      class="flex flex-wrap gap-1 pt-0.5"
    >
      <UButton
        v-for="(t, i) in props.targets ?? []"
        :key="`${t.name}-${i}`"
        :label="t.name"
        size="xs"
        color="primary"
        variant="subtle"
        @click="onTarget(t)"
      />
    </div>
  </div>
</template>
