<script setup lang="ts">
/**
 * Library page: all workspaces grouped by game, hero for active workspace.
 */
import { computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useWorkspaceStore, GAME_OPTIONS } from "../stores/workspace";
import { Workspace } from "@services/internal/repos/models";
import GameIcon from "../components/GameIcon.vue";
import LanguageHealthStrip from "../components/LanguageHealthStrip.vue";

const router = useRouter();
const ctx = useWorkspaceStore();

const showWizardPrompt = computed(() => !ctx.loading && !ctx.hasWorkspaces);

/** Workspaces grouped by game for sectioned library browsing. */
const sections = computed(() =>
  GAME_OPTIONS.map((g) => ({
    gameId: g.value,
    label: g.label,
    workspaces: ctx.workspaces.filter((ws) => ws.gameId === g.value),
  })).filter((s) => s.workspaces.length > 0),
);

/** Parse tags JSON safely. */
function parseTags(ws: Workspace): string[] {
  try {
    return JSON.parse(ws.tags || "[]") as string[];
  } catch {
    return [];
  }
}

/** Open a workspace in the IDE view. */
function openWorkspace(ws: Workspace): void {
  ctx.setActiveWorkspace(ws.id);
  void router.push({ name: "workspace-ide", params: { id: ws.id } });
}

/** Navigate to wizard to create new workspace. */
function createWorkspace(): void {
  void router.push({ name: "wizard" });
}

onMounted(() => {
  void ctx.refresh();
});
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-auto p-4">
    <div class="mb-4 flex items-center justify-between gap-3">
      <div>
        <h1 class="text-xl font-bold">Workspace Library</h1>
        <p class="text-sm text-muted">Manage your modding workspaces</p>
      </div>
      <UButton
        v-if="!showWizardPrompt && !ctx.loading"
        label="New Workspace"
        icon="i-lucide-plus"
        @click="createWorkspace"
      />
    </div>

    <UAlert v-if="ctx.error" color="error" variant="subtle" :description="ctx.error" class="mb-4" />

    <div v-if="ctx.loading" class="flex flex-1 items-center justify-center">
      <UButton loading variant="ghost" label="Loading workspaces..." />
    </div>

    <div v-else-if="showWizardPrompt" class="flex flex-1 items-center justify-center">
      <UEmpty
        icon="i-lucide-folder-plus"
        title="No workspaces yet"
        description="Create your first workspace to get started."
      >
        <template #actions>
          <UButton label="Create Workspace" icon="i-lucide-plus" @click="createWorkspace" />
        </template>
      </UEmpty>
    </div>

    <template v-else>
      <div
        v-if="ctx.activeWorkspace"
        class="mb-4 cursor-pointer"
        @click="openWorkspace(ctx.activeWorkspace)"
      >
        <UCard class="transition-shadow hover:shadow-md">
          <template #header>
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <UBadge color="primary" variant="subtle">Active</UBadge>
                <span class="font-semibold">{{ ctx.activeWorkspace.name }}</span>
              </div>
              <UBadge
                color="neutral"
                variant="outline"
                class="inline-flex items-center gap-1"
              >
                <GameIcon :game-id="ctx.activeWorkspace.gameId" />
                {{ ctx.activeWorkspace.gameId.toUpperCase() }}
              </UBadge>
            </div>
          </template>
          <div class="text-sm text-muted">
            <p>{{ ctx.workspaceMods.length }} mod(s) attached</p>
            <LanguageHealthStrip
              class="mt-2"
              :workspace-id="ctx.activeWorkspace.id"
            />
            <div v-if="parseTags(ctx.activeWorkspace).length" class="mt-1 flex flex-wrap gap-1">
              <UBadge
                v-for="tag in parseTags(ctx.activeWorkspace)"
                :key="tag"
                color="secondary"
                variant="subtle"
                size="xs"
              >
                {{ tag }}
              </UBadge>
            </div>
          </div>
        </UCard>
      </div>

      <div v-for="section in sections" :key="section.gameId" class="mb-6">
        <h2
          class="mb-2 flex items-center gap-1.5 text-sm font-semibold tracking-wide text-muted uppercase"
        >
          <GameIcon :game-id="section.gameId" size="md" />
          {{ section.label }}
        </h2>
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          <div
            v-for="ws in section.workspaces"
            :key="ws.id"
            class="cursor-pointer"
            @click="openWorkspace(ws)"
          >
            <UCard
              class="h-full transition-shadow hover:shadow-md"
              :class="{ 'ring-2 ring-primary': ws.id === ctx.activeWorkspaceId }"
            >
              <template #header>
                <div class="flex items-center justify-between gap-2">
                  <span class="truncate font-medium">{{ ws.name }}</span>
                  <UBadge
                    color="neutral"
                    variant="outline"
                    size="xs"
                    class="inline-flex items-center gap-1"
                  >
                    <GameIcon :game-id="ws.gameId" />
                    {{ ws.gameId.toUpperCase() }}
                  </UBadge>
                </div>
              </template>
              <div class="space-y-1 text-xs text-muted">
                <p v-if="ws.installId">Install configured</p>
                <p v-else class="text-warning">No install set</p>
                <div v-if="parseTags(ws).length" class="flex flex-wrap gap-1">
                  <UBadge
                    v-for="tag in parseTags(ws)"
                    :key="tag"
                    color="secondary"
                    variant="subtle"
                    size="xs"
                  >
                    {{ tag }}
                  </UBadge>
                </div>
              </div>
            </UCard>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
