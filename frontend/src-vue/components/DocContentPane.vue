<script setup lang="ts">
/**
 * Document content viewer used by the Modding Docs page.
 */
import EditorView from "./EditorView.vue";
import LangThemeSelect from "./LangThemeSelect.vue";

defineProps<{
  name: string;
  content: string;
  placeholder: string;
}>();
</script>

<template>
  <UCard class="h-full" :ui="{ root: 'h-full flex flex-col', body: 'flex-1 min-h-0 p-0' }">
    <div
      class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden border border-default bg-[var(--color-dark-input)]"
    >
      <div class="flex items-center justify-between gap-2 border-b border-default bg-accented px-3 py-2">
        <span class="min-w-0 truncate font-semibold text-accent">{{ name || "Select a file" }}</span>
        <LangThemeSelect />
      </div>
      <div class="relative flex min-h-0 flex-1 flex-col overflow-hidden">
        <EditorView
          v-if="content"
          :content="content"
          :label="name"
          :placeholder="placeholder"
          :read-only="true"
          lang="plaintext"
        />
        <div v-else class="flex flex-1 items-center justify-center p-4 text-muted">
          <p>{{ placeholder }}</p>
        </div>
      </div>
    </div>
  </UCard>
</template>
