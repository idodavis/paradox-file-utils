<script setup lang="ts">
/**
 * Vanilla Pierre UnresolvedFile host for git-style conflict resolution.
 */
import { nextTick, onBeforeUnmount, onMounted, useTemplateRef, watch } from "vue";
import {
  UnresolvedFile,
  type FileContents,
  type MergeConflictActionPayload,
} from "@pierre/diffs";
import { editorTheme, pierreThemeOption } from "../composables/editorTheme";

const props = withDefaults(
  defineProps<{
    /** Conflict-marked file contents. */
    file: FileContents;
    /** Optional header label. */
    label?: string;
  }>(),
  {},
);

const emit = defineEmits<{
  /** Fired after each Accept action with the updated file contents. */
  resolve: [contents: string, payload: MergeConflictActionPayload];
  /** Fired when the working contents change (resolve or remount). */
  "update:contents": [contents: string];
}>();

const host = useTemplateRef<HTMLElement>("host");
let viewer: UnresolvedFile | null = null;
let workingContents = props.file.contents;
let resizeObserver: ResizeObserver | null = null;

/** Mount or remount UnresolvedFile once the host has non-zero size. */
async function mountViewer(): Promise<void> {
  await nextTick();
  if (!host.value) return;
  const { clientWidth, clientHeight } = host.value;
  if (clientWidth < 32 || clientHeight < 32) return;

  viewer?.cleanUp();
  host.value.replaceChildren();
  workingContents = props.file.contents;
  emit("update:contents", workingContents);

  viewer = new UnresolvedFile({
    theme: pierreThemeOption(),
    stickyHeader: true,
    mergeConflictActionsType: "default",
    maxContextLines: 40,
    onMergeConflictResolve(file, payload) {
      workingContents = file.contents;
      emit("update:contents", workingContents);
      emit("resolve", workingContents, payload);
    },
  });
  viewer.render({
    file: { ...props.file, contents: workingContents },
    fileContainer: host.value,
  });
}

onMounted(() => {
  void mountViewer();
  if (!host.value) return;
  resizeObserver = new ResizeObserver(() => {
    if (!viewer) void mountViewer();
  });
  resizeObserver.observe(host.value);
});

watch(
  () => [props.file.name, props.file.contents, props.file.lang, editorTheme.value] as const,
  () => {
    void mountViewer();
  },
);

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  resizeObserver = null;
  viewer?.cleanUp();
  viewer = null;
});

defineExpose({
  /** Current working contents after Accept actions. */
  getContents: () => workingContents,
});
</script>

<template>
  <div class="flex h-full min-h-0 w-full min-w-60 flex-1 flex-col">
    <div v-if="label" class="shrink-0 border-b border-default bg-muted/50 px-3 py-2 text-sm font-semibold">
      {{ label }}
    </div>
    <div class="relative min-h-0 min-w-0 flex-1 overflow-hidden">
      <div ref="host" class="absolute inset-0 h-full w-full overflow-auto" />
    </div>
  </div>
</template>
