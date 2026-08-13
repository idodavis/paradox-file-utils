<script setup lang="ts">
/**
 * Resizable split pane matching the legacy Svelte interaction model.
 */
import { computed, ref } from "vue";

const props = withDefaults(
  defineProps<{
    secondOpen?: boolean;
    defaultSecondSize?: number;
    fixedSide?: "first" | "second";
    orientation?: "horizontal" | "vertical";
    class?: string;
  }>(),
  {
    secondOpen: true,
    defaultSecondSize: 600,
    fixedSide: "second",
    orientation: "horizontal",
    class: "",
  },
);

const fixedSize = ref<number>();
const dragStart = ref<{ pos: number; startSize: number } | null>(null);

const isHorizontal = computed(() => props.orientation === "horizontal");
const currentSize = computed(() => fixedSize.value ?? props.defaultSecondSize);

function onResizePointerDown(event: PointerEvent): void {
  event.preventDefault();
  dragStart.value = {
    pos: isHorizontal.value ? event.clientX : event.clientY,
    startSize: currentSize.value,
  };
  (event.target as HTMLElement).setPointerCapture?.(event.pointerId);
}

function onResizePointerMove(event: PointerEvent): void {
  if (!dragStart.value) return;
  const current = isHorizontal.value ? event.clientX : event.clientY;
  const maxDim = isHorizontal.value ? window.innerWidth : window.innerHeight;
  const delta = props.fixedSide === "first" ? current - dragStart.value.pos : dragStart.value.pos - current;
  const maxSize = maxDim * 0.8;
  fixedSize.value = Math.max(200, Math.min(maxSize, dragStart.value.startSize + delta));
}

function onResizePointerUp(event: PointerEvent): void {
  if (dragStart.value) {
    (event.target as HTMLElement).releasePointerCapture?.(event.pointerId);
  }
  dragStart.value = null;
}

const handleClass = computed(() =>
  isHorizontal.value
    ? "flex w-2 shrink-0 cursor-col-resize touch-none select-none items-center justify-center bg-accented transition-colors hover:bg-primary/30"
    : "flex h-2 shrink-0 cursor-row-resize touch-none select-none items-center justify-center bg-accented transition-colors hover:bg-primary/30",
);
const handlePip = computed(() =>
  isHorizontal.value ? "h-8 w-0.5 rounded-full bg-default/25" : "h-0.5 w-8 rounded-full bg-default/25",
);
const fixedStyle = computed(() =>
  isHorizontal.value
    ? `width: ${currentSize.value}px; max-width: calc(100% - 200px)`
    : `height: ${currentSize.value}px; max-height: calc(100% - 200px)`,
);
</script>

<template>
  <div
    :class="[
      isHorizontal ? 'flex' : 'flex flex-col',
      'min-h-0 h-full w-full overflow-hidden rounded-lg border border-default',
      props.class,
    ]"
  >
    <template v-if="props.fixedSide === 'first'">
      <div
        :style="fixedStyle"
        :class="['shrink-0 min-h-0 overflow-hidden flex flex-col', isHorizontal ? '' : 'min-w-0']"
      >
        <slot name="first" />
      </div>

      <template v-if="props.secondOpen">
        <div
          :class="handleClass"
          role="separator"
          aria-label="Drag to resize"
          :aria-orientation="isHorizontal ? 'vertical' : 'horizontal'"
          @pointerdown="onResizePointerDown"
          @pointermove="onResizePointerMove"
          @pointerup="onResizePointerUp"
        >
          <div :class="handlePip" />
        </div>

        <div class="min-h-0 min-w-0 flex-1 overflow-hidden">
          <slot name="second" />
        </div>
      </template>
    </template>

    <template v-else>
      <div class="min-h-0 min-w-0 flex-1 overflow-hidden">
        <slot name="first" />
      </div>

      <template v-if="props.secondOpen">
        <div
          :class="handleClass"
          role="separator"
          aria-label="Drag to resize"
          :aria-orientation="isHorizontal ? 'vertical' : 'horizontal'"
          @pointerdown="onResizePointerDown"
          @pointermove="onResizePointerMove"
          @pointerup="onResizePointerUp"
        >
          <div :class="handlePip" />
        </div>

        <div
          :style="fixedStyle"
          :class="['shrink-0 min-h-0 overflow-hidden flex flex-col', isHorizontal ? '' : 'min-w-0']"
        >
          <slot name="second" />
        </div>
      </template>
    </template>
  </div>
</template>
