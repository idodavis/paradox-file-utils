/**
 * Vue router for PMT workspace-centric page routing.
 */
import { createRouter, createWebHashHistory } from "vue-router";
import LibraryPage from "../pages/LibraryPage.vue";
import WizardPage from "../pages/WizardPage.vue";
import WorkspaceIdePage from "../pages/WorkspaceIdePage.vue";
import EventGraphPage from "../pages/EventGraphPage.vue";
import ConflictPage from "../pages/ConflictPage.vue";
import LocCoveragePage from "../pages/LocCoveragePage.vue";
import PatcherPage from "../pages/PatcherPage.vue";
import ToolsMergePage from "../pages/ToolsMergePage.vue";
import WorkspaceSettingsPage from "../pages/WorkspaceSettingsPage.vue";
import { useWorkspaceStore } from "../stores/workspace";

const routes = [
  {
    path: "/",
    redirect: "/library",
  },
  {
    path: "/library",
    name: "library",
    component: LibraryPage,
    meta: {
      title: "Library",
      description: "Browse and manage your modding workspaces.",
    },
  },
  {
    path: "/wizard",
    name: "wizard",
    component: WizardPage,
    meta: {
      title: "Create Workspace",
      description: "Set up a new modding workspace.",
    },
  },
  {
    path: "/workspace/:id",
    name: "workspace-ide",
    components: { ide: WorkspaceIdePage },
    meta: {
      title: "Workspace",
      description: "View and edit files in your workspace.",
      workspaceTool: true,
    },
  },
  {
    path: "/workspace/:id/graph",
    name: "event-graph",
    component: EventGraphPage,
    meta: {
      title: "Event Graph",
      description: "Browse event and on_action links in the workspace.",
      workspaceTool: true,
    },
  },
  {
    path: "/workspace/:id/conflicts",
    name: "conflicts",
    component: ConflictPage,
    meta: {
      title: "Conflicts",
      description: "FIOS/LIOS overlapping definitions.",
      workspaceTool: true,
    },
  },
  {
    path: "/workspace/:id/loc",
    name: "loc-coverage",
    component: LocCoveragePage,
    meta: {
      title: "Loc Coverage",
      description: "Missing, orphaned, and untranslated localization keys.",
      workspaceTool: true,
    },
  },
  {
    path: "/workspace/:id/patcher",
    name: "patcher",
    component: PatcherPage,
    meta: {
      title: "Mod Patcher",
      description: "Update mods between game versions.",
      workspaceTool: true,
    },
  },
  {
    path: "/workspace/:id/settings",
    name: "workspace-settings",
    component: WorkspaceSettingsPage,
    meta: {
      title: "Workspace Settings",
      description: "Configure this workspace.",
      workspaceTool: true,
    },
  },
  {
    path: "/tools/merge",
    name: "tools-merge",
    component: ToolsMergePage,
    meta: {
      title: "Ad-hoc Merge",
      description: "Merge two files or directories without a workspace.",
    },
  },
  {
    path: "/settings",
    name: "settings",
    redirect: () => {
      const id = useWorkspaceStore().activeWorkspaceId;
      if (id) return { name: "workspace-settings", params: { id } };
      return { name: "library" };
    },
  },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

router.beforeEach((to) => {
  if (!to.meta.workspaceTool) return true;
  const id = to.params.id;
  if (typeof id !== "string" || !id) return true;
  const ws = useWorkspaceStore();
  ws.setActiveWorkspace(id);
  return true;
});

export default router;
