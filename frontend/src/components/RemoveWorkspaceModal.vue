<script setup lang="ts">
/**
 * Confirm removing a workspace from PMT (mod folders on disk stay).
 */
const open = defineModel<boolean>("open", { required: true });

defineProps<{
  name: string;
  loading?: boolean;
}>();

const emit = defineEmits<{
  confirm: [];
}>();
</script>

<template>
  <UModal
    v-model:open="open"
    :title="`Remove “${name}”?`"
    description="This workspace leaves Paradox Modding Tools. Mod folders on disk are not deleted. You can attach them to another workspace later."
  >
    <template #footer="{ close }">
      <UButton label="Cancel" color="neutral" variant="outline" @click="close" />
      <UButton
        label="Remove"
        color="error"
        :loading="loading"
        @click="emit('confirm')"
      />
    </template>
  </UModal>
</template>
