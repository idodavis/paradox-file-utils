<script setup lang="ts">
/**
 * Compact official game icon; hidden when no asset exists for the id.
 */
import { computed } from "vue";
import { gameIconSrc } from "../composables/gameIcons";

const props = withDefaults(
  defineProps<{ gameId: string; size?: "sm" | "md" }>(),
  { size: "sm" },
);

const src = computed(() => gameIconSrc(props.gameId));
const sizeClass = computed(() =>
  props.size === "md" ? "size-5" : "size-4",
);
</script>

<template>
  <img
    v-if="src"
    :src="src"
    :alt="gameId.toUpperCase()"
    :class="['shrink-0 rounded-sm object-contain', sizeClass]"
  />
</template>
