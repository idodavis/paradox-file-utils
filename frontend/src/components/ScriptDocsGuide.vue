<script setup lang="ts">
/**
 * Walkthrough for generating script_docs, plus what PMT can and cannot do
 * without it. The game writes these files only from its own console, so this is
 * the one part of the language engine the user has to do by hand — once per
 * game version, since PMT archives its own copy afterwards.
 */
import { computed } from "vue";
import type { ScriptDocsHealth } from "@services/models";

const open = defineModel<boolean>("open", { required: true });

const props = defineProps<{
  schema: ScriptDocsHealth;
  gameName?: string;
}>();

const steps = computed(() => [
  {
    title: `Add the ${props.schema.launchOption} launch option`,
    body: "In Steam: right-click the game, Properties, Launch Options. In the Paradox launcher there is no equivalent, so use Steam.",
  },
  {
    title: "Start the game and open a save or a new game",
    body: "The console is only available in-game, not from the main menu.",
  },
  {
    title: "Open the console with ` or ~",
    body: "The key depends on your keyboard layout; both are worth trying.",
  },
  {
    title: `Run ${props.schema.command}`,
    body: "It runs instantly and prints nothing. That is expected.",
  },
  {
    title: "Come back and rescan",
    body: props.schema.folder
      ? `The files land in ${props.schema.folder}. PMT copies them into its own cache, so you can delete them afterwards.`
      : "PMT copies the files into its own cache, so you can delete them afterwards.",
  },
]);

const gained = [
  "Scope-aware completion — only the effects and triggers valid in the block you are editing",
  "Typed hover — what an object is to the engine, and what a token runs against",
  "Wrong-scope diagnostics",
  "The citable prefix table (culture:, title:, c:, …)",
];

const kept = [
  "Go to definition, find references, and rename",
  "Workspace Health: conflicts, load order, dangling references",
  "Localization coverage",
  "The Event Graph",
];
</script>

<template>
  <UModal
    v-model:open="open"
    title="Generate script_docs"
    :description="`${gameName || 'The game'} can print its own scripting documentation. PMT reads it as the type system — it is never guessed.`"
    :ui="{ content: 'max-w-2xl' }"
  >
    <template #body>
      <div class="space-y-5 text-sm">
        <UAlert
          v-if="schema.source === 'archive'"
          color="info"
          variant="subtle"
          icon="i-lucide-archive"
          title="Using PMT's archived copy"
          description="Your generated files are gone, but PMT kept a copy, so everything still works. Regenerate only after a game update."
        />
        <UAlert
          v-else-if="schema.stale"
          color="warning"
          variant="subtle"
          icon="i-lucide-clock"
          title="These files predate the last game update"
          :description="`Written ${new Date(schema.readAt || '').toLocaleDateString()}. The engine API may have changed since; regenerating takes a minute.`"
        />

        <ol class="space-y-3">
          <li v-for="(s, i) in steps" :key="s.title" class="flex gap-3">
            <span
              class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-elevated text-xs font-medium"
            >
              {{ i + 1 }}
            </span>
            <div class="min-w-0">
              <p class="font-medium text-highlighted">{{ s.title }}</p>
              <p class="text-muted break-words">{{ s.body }}</p>
            </div>
          </li>
        </ol>

        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <p class="mb-1 font-medium text-highlighted">With script_docs</p>
            <ul class="space-y-1 text-muted">
              <li v-for="g in gained" :key="g" class="flex gap-2">
                <UIcon name="i-lucide-check" class="mt-1 size-3.5 shrink-0 text-success" />
                <span>{{ g }}</span>
              </li>
            </ul>
          </div>
          <div>
            <p class="mb-1 font-medium text-highlighted">Works without it</p>
            <ul class="space-y-1 text-muted">
              <li v-for="k in kept" :key="k" class="flex gap-2">
                <UIcon name="i-lucide-dot" class="mt-1 size-3.5 shrink-0 text-muted" />
                <span>{{ k }}</span>
              </li>
            </ul>
          </div>
        </div>

        <p v-if="schema.source !== 'missing'" class="text-xs text-muted">
          Currently loaded: {{ schema.effects.toLocaleString() }} effects,
          {{ schema.triggers.toLocaleString() }} triggers, {{ schema.scopeTypes }} scope types,
          {{ schema.prefixes }} prefixes.
        </p>
      </div>
    </template>
    <template #footer="{ close }">
      <UButton label="Close" color="neutral" variant="outline" @click="close" />
    </template>
  </UModal>
</template>
