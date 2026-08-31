<script setup lang="ts">
/**
 * Collapsed-by-default accordion of event refs, grouped by kind from Go.
 */
import { computed } from "vue";
import type { AccordionItem } from "@nuxt/ui";
import type { EventRefInfo, RefKindGroup } from "@services/internal/views/models";

const props = defineProps<{
  refGroups?: RefKindGroup[] | null;
}>();

const emit = defineEmits<{
  open: [file: string, line: number];
  reroot: [id: string];
}>();

const GRAPH_KINDS = new Set(["event", "on_action", "decision"]);

const items = computed((): AccordionItem[] =>
  (props.refGroups ?? []).map((group) => ({
    label: `${group.kind} (${group.refs?.length ?? 0})`,
    value: group.kind,
    refs: group.refs ?? [],
  })),
);

/** Refs payload from an accordion item. */
function asRefs(item: AccordionItem): EventRefInfo[] {
  return (item as AccordionItem & { refs?: EventRefInfo[] }).refs ?? [];
}

function onRef(r: EventRefInfo): void {
  if (GRAPH_KINDS.has(r.kind)) {
    emit("reroot", r.name);
    return;
  }
  if (r.defFile) emit("open", r.defFile, r.defLine ?? 0);
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
        :disabled="!GRAPH_KINDS.has(r.kind) && !r.defFile"
        @click="onRef(r)"
      >
        {{ r.name }}
      </button>
    </template>
  </UAccordion>
</template>
