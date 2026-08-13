/**
 * Vue router for PMT workspace-centric page routing.
 */
import { createRouter, createWebHashHistory } from "vue-router";
import LibraryPage from "../pages/LibraryPage.vue";
import WizardPage from "../pages/WizardPage.vue";
import WorkspaceIdePage from "../pages/WorkspaceIdePage.vue";
import PatchCenterPage from "../pages/PatchCenterPage.vue";
import PatcherPage from "../pages/PatcherPage.vue";
import EventGraphPage from "../pages/EventGraphPage.vue";
import ToolsMergePage from "../pages/ToolsMergePage.vue";
import SettingsPage from "../pages/SettingsPage.vue";

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
    component: WorkspaceIdePage,
    meta: {
      title: "Workspace",
      description: "View and edit files in your workspace.",
    },
  },
  {
    path: "/workspace/:id/patch",
    name: "patch-center",
    component: PatchCenterPage,
    meta: {
      title: "Patch Center",
      description: "Browse wiki patch notes and import script logs.",
    },
  },
  {
    path: "/workspace/:id/patcher",
    name: "patcher",
    component: PatcherPage,
    meta: {
      title: "Mod Patcher",
      description: "Update mods between game versions.",
    },
  },
  {
    path: "/workspace/:id/graph",
    name: "event-graph",
    component: EventGraphPage,
    meta: {
      title: "Event Graph",
      description: "Explore script object relationships and cascade effects.",
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
    component: SettingsPage,
    meta: {
      title: "Settings",
      description: "App configuration and data management.",
    },
  },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
