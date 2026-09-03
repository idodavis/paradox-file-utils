<script lang="ts">
/**
 * Origin color picker shared by game, staging, and mod settings.
 */
export const ORIGIN_COLOR_DESC =
  "Used for IDE explorer roots, origin chips on Conflicts / Loc coverage / Event graph, origin filter menus, and hover origin labels. Empty uses the palette (game teal, staging amber, mods by load-order).";
</script>

<script setup lang="ts">
import { computed } from "vue";
import { originHex } from "../ide/rootDecorations";

const color = defineModel<string>({ required: true });

const props = withDefaults(
  defineProps<{
    kind: "game" | "staging" | "mod";
    wrapIndex?: number;
    label?: string;
  }>(),
  { wrapIndex: 0, label: "Origin color" },
);

/** Resolved #rrggbb for the swatch (palette when the stored color is empty). */
const hex = computed(() =>
  originHex({
    kind: props.kind,
    path: "",
    color: color.value,
    wrapIndex: props.wrapIndex,
  }).toLowerCase(),
);

/** Persist a native color-input value. */
function onPick(ev: Event): void {
  color.value = (ev.target as HTMLInputElement).value;
}

function reset(): void {
  color.value = "";
}

</script>

<template>
  <UFormField :label="label" :description="ORIGIN_COLOR_DESC">
    <div class="flex items-center gap-2">
      <span
        class="relative size-8 shrink-0 overflow-hidden rounded border border-default"
        :style="{ backgroundColor: hex }"
      >
        <input
          type="color"
          :key="`${kind}-${hex}`"
          :value="hex"
          class="absolute inset-0 size-full cursor-pointer opacity-0"
          @input="onPick"
        >
      </span>
      <span class="font-mono text-xs text-muted tabular-nums">{{ hex }}</span>
      <UButton
        v-if="color"
        label="Reset color"
        size="xs"
        variant="ghost"
        @click="reset"
      />
    </div>
  </UFormField>
</template>
