<script setup lang="ts">
/**
 * PMT brand mark used in the Vue app.
 */
import logoIcon from "../../src/assets/PMT-Logo-Icon.svg?url";
import logoText from "../../src/assets/PMT-Logo-Text.svg?url";

withDefaults(
  defineProps<{
    iconHeight?: number;
    textHeight?: number;
  }>(),
  {
    iconHeight: 30,
    textHeight: 40,
  },
);
</script>

<template>
  <div class="inline-flex items-center">
    <span
      class="inline-block"
      :style="{
        width: `${iconHeight * 2}px`,
        height: `${iconHeight}px`,
        backgroundColor: 'var(--p, #dec572)',
        maskImage: `url(${logoIcon})`,
        maskRepeat: 'no-repeat',
        maskPosition: 'center',
        maskSize: 'contain',
      }"
    />
    <span
      class="inline-block"
      :style="{
        width: `${textHeight * 3}px`,
        height: `${textHeight}px`,
        backgroundColor: 'var(--s, #5ad7b2)',
        maskImage: `url(${logoText})`,
        maskRepeat: 'no-repeat',
        maskPosition: '0% center',
        maskSize: 'contain',
        marginLeft: `${-0.9 * iconHeight}px`,
      }"
    />
  </div>
</template>
