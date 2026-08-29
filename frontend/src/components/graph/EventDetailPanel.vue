<script setup lang="ts">
/**
 * Event inspector: header, loc, attributes, sections, incoming edges, options, refs.
 */
import { computed } from "vue";
import type { AccordionItem } from "@nuxt/ui";
import type {
  EventDetail,
  EventGraphEdge,
  EventOptionInfo,
} from "@services/internal/graph/models";
import { useOpenInIde } from "../../composables/useOpenInIde";
import EventDetailHeader from "./EventDetailHeader.vue";
import EventScriptBlock from "./EventScriptBlock.vue";
import EventRefsAccordion from "./EventRefsAccordion.vue";

const props = defineProps<{
  workspaceId: string;
  detail: EventDetail | null;
  incoming?: EventGraphEdge[];
}>();

const emit = defineEmits<{ select: [id: string] }>();

const { openInIde } = useOpenInIde();

const GATES = new Set(["trigger", "cancellation_trigger", "on_trigger_fail"]);

const gates = computed(() =>
  (props.detail?.sections ?? []).filter((s) => GATES.has(s.name.toLowerCase())),
);
const effects = computed(() =>
  (props.detail?.sections ?? []).filter((s) => !GATES.has(s.name.toLowerCase())),
);
const optionItems = computed((): AccordionItem[] =>
  (props.detail?.options ?? []).map((option, i) => ({
    label: option.name?.text || option.name?.key || `option ${i + 1}`,
    value: `opt-${i}`,
    option,
  })),
);

function open(path?: string, line?: number): void {
  if (path) void openInIde(props.workspaceId, path, line);
}

function asOption(item: AccordionItem): EventOptionInfo | undefined {
  return (item as AccordionItem & { option?: EventOptionInfo }).option;
}

function sectionColor(name: string): "warning" | "info" | "neutral" {
  switch (name.toLowerCase()) {
    case "trigger":
    case "cancellation_trigger":
    case "on_trigger_fail":
      return "warning";
    case "immediate":
    case "after":
      return "info";
    default:
      return "neutral";
  }
}
</script>

<template>
  <aside class="flex h-full min-h-0 w-full flex-col overflow-hidden bg-default">
    <div
      v-if="!detail"
      class="p-2 text-xs text-muted"
    >
      Click a node to inspect. Double-click to re-root.
    </div>
    <template v-else>
      <EventDetailHeader
        :detail="detail"
        @open="open(detail.file, detail.line)"
      />
      <div class="min-h-0 flex-1 space-y-3 overflow-auto p-2 text-xs">
        <div
          v-if="detail.title"
          class="border-l-2 border-info pl-2"
        >
          <div class="text-muted">Title</div>
          <button
            v-if="detail.title.file"
            type="button"
            class="text-left text-default hover:underline"
            @click="open(detail.title.file, detail.title.line)"
          >
            “{{ detail.title.text || detail.title.key }}”
          </button>
          <div v-else class="text-default">
            “{{ detail.title.text || detail.title.key }}”
          </div>
        </div>
        <div
          v-if="detail.desc"
          class="border-l-2 border-info pl-2"
        >
          <div class="text-muted">Desc</div>
          <button
            v-if="detail.desc.file"
            type="button"
            class="text-left text-default hover:underline"
            @click="open(detail.desc.file, detail.desc.line)"
          >
            “{{ detail.desc.text || detail.desc.key }}”
          </button>
          <div v-else class="text-default">
            “{{ detail.desc.text || detail.desc.key }}”
          </div>
        </div>
        <div
          v-if="detail.flavor"
          class="border-l-2 border-info pl-2"
        >
          <div class="text-muted">Flavor</div>
          <button
            v-if="detail.flavor.file"
            type="button"
            class="text-left text-default hover:underline"
            @click="open(detail.flavor.file, detail.flavor.line)"
          >
            “{{ detail.flavor.text || detail.flavor.key }}”
          </button>
          <div v-else class="text-default">
            “{{ detail.flavor.text || detail.flavor.key }}”
          </div>
        </div>

        <div v-if="detail.fields?.length" class="space-y-0.5">
          <div class="font-medium text-toned">Attributes</div>
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
        </div>

        <div v-if="gates.length" class="space-y-2">
          <div class="font-medium text-toned">When it runs</div>
          <div
            v-for="sec in gates"
            :key="sec.name"
            class="border-l-2 border-warning pl-2"
          >
            <UBadge
              :label="sec.name"
              :color="sectionColor(sec.name)"
              variant="subtle"
              size="xs"
            />
            <EventScriptBlock
              class="mt-1"
              :lines="sec.lines"
              :targets="sec.targets"
              @select="emit('select', $event)"
            />
          </div>
        </div>

        <div v-if="incoming?.length" class="space-y-1">
          <div class="font-medium text-toned">Fired by</div>
          <UButton
            v-for="(e, i) in incoming"
            :key="i"
            :label="`${e.from}${e.via ? ` · ${e.via}` : ''}`"
            size="xs"
            color="neutral"
            variant="subtle"
            @click="emit('select', e.from)"
          />
        </div>

        <div v-if="effects.length" class="space-y-2">
          <div class="font-medium text-toned">What it fires</div>
          <div
            v-for="sec in effects"
            :key="sec.name"
            class="border-l-2 border-info pl-2"
          >
            <UBadge
              :label="sec.name"
              :color="sectionColor(sec.name)"
              variant="subtle"
              size="xs"
            />
            <EventScriptBlock
              class="mt-1"
              :lines="sec.lines"
              :targets="sec.targets"
              @select="emit('select', $event)"
            />
          </div>
        </div>

        <div v-if="optionItems.length" class="space-y-1">
          <div class="font-medium text-toned">Branching options</div>
          <UAccordion
            type="multiple"
            :items="optionItems"
            :ui="{ trigger: 'text-xs', body: 'text-xs' }"
          >
            <template #body="{ item }">
              <div
                v-for="f in asOption(item)?.fields ?? []"
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
                :lines="asOption(item)?.lines"
                :targets="asOption(item)?.targets"
                @select="emit('select', $event)"
              />
              <div
                v-if="asOption(item)?.trigger"
                class="mt-1 border-l-2 border-warning pl-2"
              >
                <div class="text-muted">trigger</div>
                <EventScriptBlock :lines="asOption(item)?.trigger?.lines" />
              </div>
              <div
                v-if="asOption(item)?.aiChance"
                class="mt-1 border-l-2 border-neutral pl-2"
              >
                <div class="text-muted">ai_chance</div>
                <EventScriptBlock :lines="asOption(item)?.aiChance?.lines" />
              </div>
            </template>
          </UAccordion>
        </div>

        <div v-if="detail.refs?.length" class="space-y-1">
          <div class="font-medium text-toned">Refs</div>
          <EventRefsAccordion :refs="detail.refs" @open="open" />
        </div>
      </div>
    </template>
  </aside>
</template>
