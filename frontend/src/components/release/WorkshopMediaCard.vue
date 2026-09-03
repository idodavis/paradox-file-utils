<script setup lang="ts">
/**
 * Workshop extra images and YouTube ids: carousel, thumbs, enlarge.
 */
import { computed, shallowRef, useTemplateRef, watch } from "vue";
import UCarousel from "@nuxt/ui/components/Carousel.vue";

/** One carousel slide (blob URL or YouTube thumbnail). */
export type MediaSlide = {
  src: string;
  kind: "image" | "youtube";
  rel: string;
  abs: string;
  id: string;
};

const props = defineProps<{
  items: MediaSlide[];
}>();

const emit = defineEmits<{
  addImage: [];
  addYoutube: [url: string];
  remove: [item: MediaSlide];
}>();

const carousel = useTemplateRef<{
  emblaApi?: { scrollTo: (index: number) => void };
}>("carousel");
const activeIndex = shallowRef(0);
const youtubeUrl = shallowRef("");
const enlarged = shallowRef(false);
const atCap = computed(() => props.items.length >= 10);

const active = computed(() => props.items[activeIndex.value]);

watch(
  () => props.items.length,
  (n) => {
    if (activeIndex.value >= n) {
      activeIndex.value = Math.max(0, n - 1);
    }
  },
);

/** Keep the thumbnail strip in sync with the large slide. */
function onSelect(index: number): void {
  activeIndex.value = index;
}

/** Jump the carousel to a thumbnail. */
function select(index: number): void {
  activeIndex.value = index;
  carousel.value?.emblaApi?.scrollTo(index);
}

/** Arrow: previous extra. */
function onClickPrev(): void {
  select(Math.max(0, activeIndex.value - 1));
}

/** Arrow: next extra. */
function onClickNext(): void {
  select(Math.min(props.items.length - 1, activeIndex.value + 1));
}

/** Open the active (or clicked) slide in a modal. */
function enlarge(index: number): void {
  if (!props.items[index]) return;
  activeIndex.value = index;
  enlarged.value = true;
}

/** Send a YouTube URL/id to the parent and clear the field. */
function addYoutube(): void {
  const url = youtubeUrl.value.trim();
  if (!url) return;
  emit("addYoutube", url);
  youtubeUrl.value = "";
}

/** Delete the slide currently in view. */
function removeActive(): void {
  const item = active.value;
  if (item) emit("remove", item);
}
</script>

<template>
  <UCard>
    <template #header>
      <div class="flex flex-col gap-0.5">
        <span class="font-semibold">Workshop media</span>
        <span class="text-xs text-muted">
          Up to 10 Steam extras. Images must be under 1 MB each (png/jpg/gif/webp).
        </span>
      </div>
    </template>
    <div class="flex flex-col gap-3">
      <div class="grid gap-2 sm:grid-cols-2">
        <UButton
          label="Add image"
          icon="i-lucide-image-plus"
          color="primary"
          variant="soft"
          class="justify-center"
          :disabled="atCap"
          @click="emit('addImage')"
        />
        <form class="w-full" @submit.prevent="addYoutube">
          <UFieldGroup class="w-full">
            <UInput
              v-model="youtubeUrl"
              placeholder="YouTube URL or id"
              class="w-full"
              :disabled="atCap"
            />
            <UButton
              type="submit"
              label="Add YouTube"
              icon="i-lucide-youtube"
              variant="outline"
              :disabled="atCap"
            />
          </UFieldGroup>
        </form>
      </div>
      <div v-if="items.length" class="flex flex-col gap-2">
        <div class="relative overflow-visible px-10">
          <UCarousel
            ref="carousel"
            v-slot="{ item, index }"
            arrows
            :items="items"
            :prev="{ onClick: onClickPrev }"
            :next="{ onClick: onClickNext }"
            class="w-full"
            :ui="{
              prev: 'start-0',
              next: 'end-0',
            }"
            @select="onSelect"
          >
            <button
              type="button"
              class="block w-full"
              @click="enlarge(index)"
            >
              <img
                v-if="item.src"
                :src="item.src"
                alt=""
                class="h-56 w-full rounded-lg object-contain"
              >
              <div
                v-else
                class="flex h-56 items-center justify-center rounded-lg
                  bg-elevated text-sm text-muted"
              >
                {{ item.kind === "youtube" ? "YouTube" : "Image" }}
              </div>
            </button>
          </UCarousel>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <div class="flex flex-wrap gap-1">
            <button
              v-for="(item, index) in items"
              :key="item.rel || item.id || index"
              type="button"
              class="size-11 opacity-25 transition-opacity hover:opacity-100"
              :class="{ 'opacity-100': activeIndex === index }"
              @click="select(index)"
            >
              <img
                v-if="item.src"
                :src="item.src"
                alt=""
                width="44"
                height="44"
                class="size-11 rounded-lg object-cover"
              >
              <div v-else class="size-11 rounded-lg bg-elevated" />
            </button>
          </div>
          <UButton
            label="Delete"
            icon="i-lucide-trash"
            color="neutral"
            variant="outline"
            size="sm"
            class="ms-auto"
            @click="removeActive"
          />
        </div>
      </div>
      <p v-else class="text-sm text-muted">
        Add gallery images and YouTube videos for the Steam Workshop listing.
      </p>
      <UAlert
        color="neutral"
        variant="subtle"
        description="Paradox Mods: extra images/videos must be uploaded on the site yourself."
      />
    </div>
    <UModal
      v-model:open="enlarged"
      :title="active?.kind === 'youtube' ? 'YouTube' : 'Preview'"
    >
      <template #body>
        <iframe
          v-if="active?.kind === 'youtube' && active.id"
          class="aspect-video w-full rounded-lg"
          :src="`https://www.youtube.com/embed/${active.id}`"
          title="YouTube preview"
          allow="encrypted-media; picture-in-picture"
        />
        <img
          v-else-if="active?.src"
          :src="active.src"
          alt=""
          class="max-h-[70vh] w-full rounded-lg object-contain"
        />
      </template>
    </UModal>
  </UCard>
</template>
