<script setup lang="ts">
/**
 * Structural HTML/CSS preview of a parsed Paradox .gui widget tree.
 */
import { computed } from "vue";
import type { GuiNode } from "@services/models";

const props = defineProps<{
  roots: GuiNode[];
  selectedName?: string | null;
}>();

const emit = defineEmits<{
  select: [name: string];
}>();

const layoutClass = (n: GuiNode): string => {
  const k = (n.kind || "").toLowerCase();
  if (k === "hbox") return "flex flex-row flex-wrap gap-1";
  if (k === "vbox") return "flex flex-col gap-1";
  return "relative";
};

const boxStyle = (n: GuiNode): Record<string, string> => {
  const s: Record<string, string> = {
    border: "1px solid var(--ui-border)",
    background: "color-mix(in oklab, var(--ui-bg) 80%, transparent)",
    minWidth: "24px",
    minHeight: "20px",
    padding: "4px",
    fontSize: "11px",
  };
  if (n.width) s.width = `${Math.min(n.width, 640)}px`;
  if (n.height) s.height = `${Math.min(n.height, 400)}px`;
  if (props.selectedName && n.name === props.selectedName) {
    s.outline = "2px solid var(--ui-primary)";
  }
  return s;
};

const flatProps = computed(() => {
  const find = (nodes: GuiNode[]): GuiNode | null => {
    for (const n of nodes) {
      if (n.name && n.name === props.selectedName) return n;
      const c = find(n.children ?? []);
      if (c) return c;
    }
    return null;
  };
  return find(props.roots);
});
</script>

<template>
  <div class="flex h-full min-h-0 flex-col gap-2 overflow-hidden p-2">
    <div class="shrink-0 text-xs font-semibold text-muted">GUI Preview</div>
    <div class="min-h-0 flex-1 overflow-auto rounded border border-default bg-muted/30 p-2">
      <template v-if="roots.length">
        <div
          v-for="(n, i) in roots"
          :key="`${n.name ?? n.kind}-${i}`"
          class="mb-2"
        >
          <button
            type="button"
            :class="layoutClass(n)"
            :style="boxStyle(n)"
            @click="n.name && emit('select', n.name)"
          >
            <span class="font-medium text-muted">{{ n.kind }}{{ n.name ? `: ${n.name}` : "" }}</span>
            <span v-if="n.text" class="block truncate">{{ n.text }}</span>
            <div
              v-for="(c, j) in n.children ?? []"
              :key="`${c.name ?? c.kind}-${j}`"
              :class="layoutClass(c)"
              :style="boxStyle(c)"
              @click.stop="c.name && emit('select', c.name)"
            >
              <span class="text-muted">{{ c.kind }}{{ c.name ? `: ${c.name}` : "" }}</span>
              <span v-if="c.text" class="block truncate">{{ c.text }}</span>
              <div
                v-for="(gc, k) in c.children ?? []"
                :key="`${gc.name ?? gc.kind}-${k}`"
                :class="layoutClass(gc)"
                :style="boxStyle(gc)"
                @click.stop="gc.name && emit('select', gc.name)"
              >
                <span class="text-muted">{{ gc.kind }}{{ gc.name ? `: ${gc.name}` : "" }}</span>
              </div>
            </div>
          </button>
        </div>
      </template>
      <UEmpty v-else icon="i-lucide-layout" title="No widgets" description="Open a .gui file to preview." />
    </div>
    <div v-if="flatProps" class="shrink-0 max-h-40 overflow-auto rounded border border-default p-2 text-xs">
      <div class="mb-1 font-semibold">{{ flatProps.name || flatProps.kind }}</div>
      <div v-for="(v, k) in flatProps.props ?? {}" :key="k" class="flex gap-2">
        <span class="text-muted">{{ k }}</span>
        <span class="truncate">{{ v }}</span>
      </div>
    </div>
  </div>
</template>
