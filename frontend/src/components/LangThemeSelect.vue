<script setup lang="ts">
/**
 * Global code language and theme selector for Monaco-backed views.
 */
import { codeLanguage, codeTheme, CODE_LANGUAGES, CODE_THEME_LABELS, CODE_THEMES } from "../stores/monaco";

const themeOptions = CODE_THEMES.map((value) => ({ label: CODE_THEME_LABELS[value], value }));
const languageOptions = CODE_LANGUAGES.map((value) => ({ label: value, value }));
</script>

<template>
  <div class="flex items-center gap-2">
    <USelect
      v-model="codeLanguage"
      :items="languageOptions"
      value-key="value"
      class="min-w-[9rem]"
      aria-label="Code language"
    />
    <USelect
      v-model="codeTheme"
      :items="themeOptions"
      value-key="value"
      class="min-w-[12rem]"
      aria-label="Code theme"
    />
  </div>
</template>
