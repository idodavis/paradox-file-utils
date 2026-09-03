<script setup lang="ts">
/**
 * Event inspector: templates the Go EventDetail payload as header-sized accordions.
 */
import { computed } from "vue";
import type { AccordionItem } from "@nuxt/ui";
import type { EventDetail, EventLocField } from "@services/internal/views/models";
import { useOpenInIde } from "../../composables/useOpenInIde";
import DetailPane from "../DetailPane.vue";
import EventScriptBlock from "./EventScriptBlock.vue";
import EventRefsAccordion from "./EventRefsAccordion.vue";
import OriginBadge, { PILL_UI } from "../OriginBadge.vue";
import { originHexByOriginId } from "../../ide/rootDecorations";

const props = defineProps<{
  workspaceId: string;
  detail: EventDetail | null;
}>();

const emit = defineEmits<{ reroot: [id: string] }>();
const { openInIde } = useOpenInIde();

/** Compact accordion chrome matching DetailPane header density. */
const HEADER_ACCORDION_UI = {
  item: "mb-1 overflow-hidden rounded-md border border-default last:mb-0",
  header: "bg-elevated",
  trigger: "px-2 py-1.5 text-xs font-medium",
  trailingIcon: "size-3.5",
  body: "px-2 text-xs",
};

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
      label: s.name,
      value: `sec-${i}`,
      section: s,
    })),
    ...(d.incoming?.length ? [{ label: "Fired by", value: "fired" }] : []),
    ...(optionItems.value.length
      ? [{ label: "Branching options", value: "opts" }]
      : []),
    ...(d.refGroups?.length ? [{ label: "Refs", value: "refs" }] : []),
  ];
});

const locRows = computed(() => {
  const d = props.detail;
  if (!d) return [];
  return [
    { label: "Title", loc: d.title },
    { label: "Desc", loc: d.desc },
    { label: "Flavor", loc: d.flavor },
  ].filter((row): row is { label: string; loc: EventLocField } => !!row.loc);
});

/** Open a payload path in the workspace IDE. */
function open(path?: string, line?: number): void {
  if (path) void openInIde(props.workspaceId, path, line);
}
</script>

<template>
  <DetailPane
    :workspace-id="workspaceId"
    :title="detail?.id"
    :subtitle="detail?.title?.text"
    :file="detail?.path"
    :rel="detail?.rel"
    :line="detail?.line"
  >
    <template #empty>
      Click a node to inspect. Double-click to re-root.
    </template>
    <template v-if="detail && (detail.originName || detail.origin)" #origin>
      <OriginBadge
        :label="detail.originName || detail.origin || ''"
        :hex="originHexByOriginId(detail.origin ?? '')"
      />
    </template>
    <template v-if="detail" #badges>
      <UBadge
        v-if="detail.kind && detail.kind !== 'event'"
        :label="detail.kind"
        color="neutral"
        variant="subtle"
        size="xs"
        :ui="PILL_UI"
      />
      <UBadge
        v-if="detail.type"
        :label="detail.type"
        color="neutral"
        variant="subtle"
        size="xs"
        :ui="PILL_UI"
      />
      <UBadge
        v-if="detail.hidden"
        label="hidden"
        color="warning"
        variant="subtle"
        size="xs"
        :ui="PILL_UI"
      />
      <UBadge
        v-if="detail.theme"
        :label="detail.theme"
        color="info"
        variant="subtle"
        size="xs"
        :ui="PILL_UI"
      />
    </template>
    <UAccordion
      v-if="detail"
      type="multiple"
      :items="items"
      :ui="HEADER_ACCORDION_UI"
    >
      <template #body="{ item }">
        <template v-if="item.value === 'loc'">
          <div
            v-for="row in locRows"
            :key="row.label"
            class="mb-2 border-l-2 border-info pl-2 last:mb-0"
          >
            <div class="text-muted">{{ row.label }}</div>
            <button
              v-if="row.loc?.path"
              type="button"
              class="text-left text-default hover:underline"
              @click="open(row.loc.path, row.loc.line)"
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
            @click="open(detail.path, f.line)"
          >
            <span class="text-muted">{{ f.key }}</span>
            <span class="text-default">
              = {{ f.quoted ? `"${f.value}"` : f.value }}
            </span>
          </button>
        </template>
        <template v-else-if="item.section">
          <div
            class="border-l-2 pl-2"
            :class="item.section.role === 'gate'
              ? 'border-warning'
              : 'border-info'"
          >
            <EventScriptBlock
              :lines="item.section.lines"
              :targets="item.section.targets"
              @select="emit('reroot', $event)"
            />
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
            :ui="{
              trigger: 'px-2 py-1.5 text-xs font-medium',
              trailingIcon: 'size-3.5',
              body: 'px-2 text-xs',
            }"
          >
            <template #body="{ item: opt }">
              <div
                v-for="f in opt.option?.fields ?? []"
                :key="f.key"
                class="truncate"
              >
                <span class="text-muted">{{ f.key }}</span>
                <span class="text-default">
                  = {{ f.quoted ? `"${f.value}"` : f.value }}
                </span>
              </div>
              <EventScriptBlock
                class="mt-1"
                :lines="opt.option?.lines"
                :targets="opt.option?.targets"
                @select="emit('reroot', $event)"
              />
              <div
                v-if="opt.option?.trigger"
                class="mt-1 border-l-2 border-warning pl-2"
              >
                <div class="text-muted">trigger</div>
                <EventScriptBlock :lines="opt.option.trigger.lines" />
              </div>
              <div
                v-if="opt.option?.aiChance"
                class="mt-1 border-l-2 border-neutral pl-2"
              >
                <div class="text-muted">ai_chance</div>
                <EventScriptBlock :lines="opt.option.aiChance.lines" />
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
  </DetailPane>
</template>
