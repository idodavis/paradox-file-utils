<script setup lang="ts">
/**
 * Read-only inspector for one Go EventDetail (including simSteps).
 */
import { computed } from "vue";
import type {
  EventDetail,
  EventScriptLine,
} from "@services/internal/graph/models";
import { useOpenInIde } from "../../composables/useOpenInIde";

const props = defineProps<{
  workspaceId: string;
  detail: EventDetail | null;
}>();

const { openInIde } = useOpenInIde();

const locBits = computed(() => {
  const d = props.detail;
  if (!d) return [];
  return [
    { label: "Title", loc: d.title },
    { label: "Desc", loc: d.desc },
    { label: "Flavor", loc: d.flavor },
  ].filter((x) => x.loc);
});

const simItems = computed(() =>
  (props.detail?.simSteps ?? []).map((step, i) => ({
    label: step.title || step.kind,
    value: `sim-${i}`,
    step,
  })),
);

/** Open a path from this detail in the workspace IDE. */
function open(path?: string, line?: number): void {
  if (!path) return;
  void openInIde(props.workspaceId, path, line);
}

/** Open the event's defining file. */
function openDef(): void {
  const d = props.detail;
  if (!d?.file) return;
  open(d.file, d.line);
}

/** Indent a rendered script line from Go (display only). */
function linePad(line: EventScriptLine): string {
  return `${Math.min(line.depth, 8) * 0.75}rem`;
}
</script>

<template>
  <aside
    class="flex h-full min-h-0 w-80 shrink-0 flex-col overflow-hidden border-l
      border-default bg-default"
  >
    <div
      v-if="!detail"
      class="p-3 text-sm text-muted"
    >
      Click a node to inspect. Double-click to re-root.
    </div>
    <template v-else>
      <div class="shrink-0 space-y-1 border-b border-default px-3 py-2">
        <div class="truncate font-semibold text-default">{{ detail.id }}</div>
        <div class="flex flex-wrap items-center gap-1 text-xs text-muted">
          <UBadge v-if="detail.type" color="neutral" variant="subtle" size="xs">
            {{ detail.type }}
          </UBadge>
          <UBadge v-if="detail.hidden" color="warning" variant="subtle" size="xs">
            hidden
          </UBadge>
          <span v-if="detail.theme">{{ detail.theme }}</span>
        </div>
        <UButton
          label="Open in IDE"
          icon="i-lucide-file-code"
          size="xs"
          color="neutral"
          variant="ghost"
          :disabled="!detail.file"
          @click="openDef"
        />
      </div>
      <div class="min-h-0 flex-1 space-y-3 overflow-auto p-3">
        <div v-if="locBits.length" class="space-y-1 text-sm">
          <div v-for="bit in locBits" :key="bit.label">
            <div class="text-xs text-muted">{{ bit.label }}</div>
            <button
              v-if="bit.loc?.file"
              type="button"
              class="text-left text-default hover:underline"
              @click="open(bit.loc.file, bit.loc.line)"
            >
              {{ bit.loc.text || bit.loc.key }}
            </button>
            <div v-else class="text-default">
              {{ bit.loc?.text || bit.loc?.key }}
            </div>
          </div>
        </div>
        <div v-if="simItems.length" class="space-y-2">
          <div class="text-xs font-medium text-muted">Sim order</div>
          <div
            v-for="item in simItems"
            :key="item.value"
            class="space-y-1 border-b border-default pb-2 last:border-0"
          >
            <div class="text-sm font-medium text-default">{{ item.label }}</div>
            <p v-if="item.step.subtitle" class="text-xs text-muted">
              {{ item.step.subtitle }}
            </p>
            <p v-if="item.step.note" class="text-xs text-muted">
              {{ item.step.note }}
            </p>
            <div
              v-for="(line, i) in item.step.lines ?? []"
              :key="i"
              class="font-mono text-xs text-default"
              :style="{ paddingLeft: linePad(line) }"
            >
              {{ line.text }}
            </div>
            <div
              v-for="(t, i) in item.step.targets ?? []"
              :key="`t-${i}`"
              class="text-xs text-muted"
            >
              {{ t.via }} → {{ t.name }}
            </div>
          </div>
        </div>
        <div v-if="detail.options?.length" class="space-y-1">
          <div class="text-xs font-medium text-muted">Options</div>
          <div
            v-for="(opt, i) in detail.options"
            :key="i"
            class="text-sm text-default"
          >
            {{ opt.name?.text || opt.name?.key || `option ${i + 1}` }}
          </div>
        </div>
        <div v-if="detail.refs?.length" class="space-y-1">
          <div class="text-xs font-medium text-muted">Refs</div>
          <button
            v-for="(r, i) in detail.refs"
            :key="i"
            type="button"
            class="block text-left text-xs text-default hover:underline"
            :disabled="!r.defFile"
            @click="open(r.defFile, r.defLine)"
          >
            {{ r.kind }} {{ r.name }}
          </button>
        </div>
      </div>
    </template>
  </aside>
</template>
