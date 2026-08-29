<script setup lang="ts">
/**
 * Collapsed-by-default accordion of event refs, grouped by kind.
 */
import { computed } from "vue";
import type { AccordionItem } from "@nuxt/ui";
import type { EventRefInfo } from "@services/internal/graph/models";

const props = defineProps<{
  refs: EventRefInfo[];
}>();

const emit = defineEmits<{
  open: [file: string, line: number];
}>();

interface KindGroup {
  kind: string;
  refs: EventRefInfo[];
}

const groups = computed((): KindGroup[] => {
  const by = new Map<string, EventRefInfo[]>();
  for (const r of props.refs) {
    const list = by.get(r.kind) ?? [];
    list.push(r);
    by.set(r.kind, list);
  }
  return [...by.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([kind, refs]) => ({ kind, refs }));
});

const items = computed((): AccordionItem[] =>
  groups.value.map((g) => ({
    label: `${g.kind} (${g.refs.length})`,
    value: g.kind,
    refs: g.refs,
  })),
);

/** Refs payload from an accordion item. */
function asRefs(item: AccordionItem): EventRefInfo[] {
  return (item as AccordionItem & { refs?: EventRefInfo[] }).refs ?? [];
}
</script>

<template>
  <UAccordion
    v-if="items.length"
    type="multiple"
    :items="items"
    :ui="{ trigger: 'text-xs', body: 'text-xs' }"
  >
    <template #body="{ item }">
      <button
        v-for="(r, i) in asRefs(item)"
        :key="i"
        type="button"
        class="block w-full truncate text-left text-xs text-default
          hover:underline disabled:text-muted disabled:no-underline"
        :disabled="!r.defFile"
        @click="r.defFile && emit('open', r.defFile, r.defLine ?? 0)"
      >
        {{ r.name }}
      </button>
    </template>
  </UAccordion>
</template>
