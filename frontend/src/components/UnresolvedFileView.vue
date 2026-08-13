<script setup lang="ts">
/**
 * Vanilla Pierre UnresolvedFile host for git-style conflict resolution.
 */
import { onBeforeUnmount, onMounted, useTemplateRef, watch } from "vue";
import {
  UnresolvedFile,
  type FileContents,
  type MergeConflictActionPayload,
} from "@pierre/diffs";

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

/** Mount or remount UnresolvedFile for the current conflict-marked file. */
function mountViewer(): void {
  if (!host.value) return;
  viewer?.cleanUp();
  host.value.replaceChildren();
  workingContents = props.file.contents;
  emit("update:contents", workingContents);

  viewer = new UnresolvedFile({
    theme: { dark: "pierre-dark", light: "pierre-light" },
    stickyHeader: true,
    mergeConflictActionsType: "default",
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

onMounted(mountViewer);

watch(
  () => [props.file.name, props.file.contents, props.file.lang] as const,
  () => {
    mountViewer();
  },
);

onBeforeUnmount(() => {
  viewer?.cleanUp();
  viewer = null;
});

defineExpose({
  /** Current working contents after Accept actions. */
  getContents: () => workingContents,
});
</script>

<template>
  <div class="flex h-full min-h-0 flex-1 flex-col">
    <div v-if="label" class="shrink-0 border-b border-default bg-muted/50 px-3 py-2 text-sm font-semibold">
      {{ label }}
    </div>
    <div class="relative min-h-0 flex-1 overflow-hidden">
      <div ref="host" class="absolute inset-0 overflow-auto" />
    </div>
  </div>
</template>
