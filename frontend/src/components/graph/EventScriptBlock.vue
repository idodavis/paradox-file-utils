<script setup lang="ts">
/**
 * Colorized pretty-script lines plus clickable outbound target chips.
 */
import { ref, watch } from "vue";
import type {
  EventScriptLine,
  EventStepTarget,
} from "@services/internal/views/models";
import { colorizeText } from "../../colorize";

const props = defineProps<{
  lines?: EventScriptLine[] | null;
  targets?: EventStepTarget[] | null;
}>();

const emit = defineEmits<{
  select: [id: string];
}>();

const html = ref("");

watch(
  () => props.lines,
  async (lines) => {
    const text = (lines ?? [])
      .map((l) => `${"  ".repeat(Math.min(l.depth, 8))}${l.text}`)
      .join("\n");
    html.value = text ? await colorizeText(text, "paradox") : "";
  },
  { immediate: true },
);

/** Select a target event. */
function onTarget(t: EventStepTarget): void {
  emit("select", t.name);
}
</script>

<template>
  <div class="space-y-1">
    <pre
      v-if="html"
      class="my-0 overflow-x-auto rounded-md bg-elevated px-2 py-1.5 text-xs leading-relaxed"
      v-html="html"
    />
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
