<script setup lang="ts">
/**
 * Event inspector: header, loc, attributes, sections, incoming edges, options, refs.
 */
import { computed } from "vue";
import type { AccordionItem } from "@nuxt/ui";
import type {
  EventDetail,
  EventOptionInfo,
  EventSectionInfo,
} from "@services/internal/views/models";
import { useOpenInIde } from "../../composables/useOpenInIde";
import EventScriptBlock from "./EventScriptBlock.vue";
import EventRefsAccordion from "./EventRefsAccordion.vue";

const props = defineProps<{
  workspaceId: string;
  detail: EventDetail | null;
}>();

const emit = defineEmits<{ reroot: [id: string] }>();
const { openInIde } = useOpenInIde();

const optionItems = computed((): AccordionItem[] =>
  (props.detail?.options ?? []).map((option, i) => ({
    label: option.name?.text || option.name?.key || `option ${i + 1}`,
    value: `opt-${i}`,
    option,
  })),
);

const items = computed((): AccordionItem[] => {
  const d = props.detail;
  if (!d) return [];
  return [
    ...(d.title || d.desc || d.flavor
      ? [{ label: "Localization", value: "loc" }]
      : []),
    ...(d.fields?.length ? [{ label: "Attributes", value: "attrs" }] : []),
    ...(d.sections ?? []).map((s, i) => ({
      label: s.name, value: `sec-${i}`, section: s,
    })),
    ...(d.incoming?.length ? [{ label: "Fired by", value: "fired" }] : []),
    ...(optionItems.value.length
      ? [{ label: "Branching options", value: "opts" }]
      : []),
    ...(d.refGroups?.length ? [{ label: "Refs", value: "refs" }] : []),
  ];
});

function open(path?: string, line?: number): void {
  if (path) void openInIde(props.workspaceId, path, line);
}

function asOption(item: AccordionItem): EventOptionInfo | undefined {
  return (item as AccordionItem & { option?: EventOptionInfo }).option;
}
function asSection(item: AccordionItem): EventSectionInfo | undefined {
  return (item as AccordionItem & { section?: EventSectionInfo }).section;
}
</script>

<template>
  <aside class="flex h-full min-h-0 w-full flex-col overflow-hidden bg-default">
    <div v-if="!detail" class="p-2 text-xs text-muted">
      Click a node to inspect. Double-click to re-root.
    </div>
    <template v-else>
      <div class="shrink-0 space-y-1.5 border-b border-default px-2 py-1.5">
        <div class="truncate text-sm font-semibold text-default">
          {{ detail.id }}
        </div>
        <div v-if="detail.title?.text" class="truncate text-xs text-muted">
          {{ detail.title.text }}
        </div>
        <div class="flex flex-wrap items-center gap-1">
          <UBadge
            v-if="detail.type"
            :label="detail.type"
            color="neutral"
            variant="subtle"
            size="xs"
          />
          <UBadge
            v-if="detail.hidden"
            label="hidden"
            color="warning"
            variant="subtle"
            size="xs"
          />
          <UBadge
            v-if="detail.theme"
            :label="detail.theme"
            color="info"
            variant="subtle"
            size="xs"
          />
          <UBadge
            :label="detail.origin ? 'mod' : 'vanilla'"
            :color="detail.origin ? 'primary' : 'neutral'"
            variant="subtle"
            size="xs"
          />
        </div>
        <button
          v-if="detail.file"
          type="button"
          class="block w-full truncate text-left text-xs text-muted
            hover:text-default hover:underline"
          :title="detail.file"
          @click="open(detail.file, detail.line)"
        >
          {{ detail.rel || detail.file }}:{{ detail.line + 1 }}
        </button>
        <UButton label="Open in IDE" icon="i-lucide-file-code" size="xs"
          color="neutral" variant="ghost" :disabled="!detail.file"
          @click="open(detail.file, detail.line)" />
      </div>
      <div class="min-h-0 flex-1 overflow-auto p-2">
        <UAccordion
          type="multiple"
          :items="items"
          :ui="{
            item: 'mb-1 overflow-hidden rounded-md border border-default last:mb-0',
            header: 'bg-elevated',
            trigger: 'px-2 py-1.5 text-xs font-medium',
            body: 'px-2 text-xs',
          }"
        >
          <template #body="{ item }">
            <template v-if="item.value === 'loc'">
              <div
                v-for="row in [
                  { label: 'Title', loc: detail.title },
                  { label: 'Desc', loc: detail.desc },
                  { label: 'Flavor', loc: detail.flavor },
                ]"
                v-show="row.loc"
                :key="row.label"
                class="mb-2 border-l-2 border-info pl-2 last:mb-0"
              >
                <div class="text-muted">{{ row.label }}</div>
                <button
                  v-if="row.loc?.file"
                  type="button"
                  class="text-left text-default hover:underline"
                  @click="open(row.loc.file, row.loc.line)"
                >
                  “{{ row.loc?.text || row.loc?.key }}”
                </button>
                <div v-else class="text-default">
                  “{{ row.loc?.text || row.loc?.key }}”
                </div>
              </div>
            </template>
            <template v-else-if="item.value === 'attrs'">
              <button
                v-for="f in detail.fields"
                :key="f.key"
                type="button"
                class="block w-full truncate text-left hover:underline"
                @click="open(detail.file, f.line)"
              >
                <span class="text-muted">{{ f.key }}</span>
                <span class="text-default">
                  = {{ f.quoted ? `"${f.value}"` : f.value }}
                </span>
              </button>
            </template>
            <template v-else-if="asSection(item)">
              <div class="border-l-2 pl-2"
                :class="asSection(item)?.role === 'gate' ? 'border-warning' : 'border-info'">
                <UBadge :label="asSection(item)?.name"
                  :color="asSection(item)?.role === 'gate' ? 'warning' : 'info'"
                  variant="subtle" size="xs" />
                <EventScriptBlock class="mt-1" :lines="asSection(item)?.lines"
                  :targets="asSection(item)?.targets" @select="emit('reroot', $event)" />
              </div>
            </template>
            <template v-else-if="item.value === 'fired'">
              <UButton
                v-for="(e, i) in detail.incoming ?? []"
                :key="i"
                :label="`${e.from}${e.via ? ` · ${e.via}` : ''}`"
                size="xs"
                color="neutral"
                variant="subtle"
                class="mb-1"
                @click="emit('reroot', e.from)"
              />
            </template>
            <template v-else-if="item.value === 'opts'">
              <UAccordion
                type="multiple"
                :items="optionItems"
                :ui="{ trigger: 'text-xs', body: 'text-xs' }"
              >
                <template #body="{ item: opt }">
                  <div v-for="f in asOption(opt)?.fields ?? []" :key="f.key" class="truncate">
                    <span class="text-muted">{{ f.key }}</span>
                    <span class="text-default">
                      = {{ f.quoted ? `"${f.value}"` : f.value }}
                    </span>
                  </div>
                  <EventScriptBlock
                    class="mt-1"
                    :lines="asOption(opt)?.lines"
                    :targets="asOption(opt)?.targets"
                    @select="emit('reroot', $event)"
                  />
                  <div v-if="asOption(opt)?.trigger" class="mt-1 border-l-2 border-warning pl-2">
                    <div class="text-muted">trigger</div>
                    <EventScriptBlock :lines="asOption(opt)?.trigger?.lines" />
                  </div>
                  <div v-if="asOption(opt)?.aiChance" class="mt-1 border-l-2 border-neutral pl-2">
                    <div class="text-muted">ai_chance</div>
                    <EventScriptBlock :lines="asOption(opt)?.aiChance?.lines" />
                  </div>
                </template>
              </UAccordion>
            </template>
            <template v-else-if="item.value === 'refs'">
              <EventRefsAccordion
                :ref-groups="detail.refGroups ?? []"
                @open="open"
                @reroot="emit('reroot', $event)"
              />
            </template>
          </template>
        </UAccordion>
      </div>
    </template>
  </aside>
</template>
