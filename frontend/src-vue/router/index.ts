/**
 * Vue router for PMT feature page routing.
 */
import { createRouter, createWebHashHistory } from "vue-router";
import HubPage from "../pages/HubPage.vue";
import SettingsPage from "../pages/SettingsPage.vue";
import ModdingDocsPage from "../pages/ModdingDocsPage.vue";
import ComparePage from "../pages/ComparePage.vue";
import InventoryPage from "../pages/InventoryPage.vue";
import MergePage from "../pages/MergePage.vue";

const routes = [
  {
    path: "/",
    redirect: "/hub",
  },
  {
    path: "/hub",
    name: "hub",
    component: HubPage,
    meta: {
      title: "Hub",
      description: "Open the primary PMT tools from the hub.",
    },
  },
  {
    path: "/modding-docs",
    name: "modding-docs",
    component: ModdingDocsPage,
    meta: {
      title: "Modding Docs",
      description: "Browse script help files and wiki content for CK3 and EU5.",
    },
  },
  {
    path: "/compare-tool",
    name: "compare-tool",
    component: ComparePage,
    meta: {
      title: "File Compare",
      description: "Compare two file sets or directories side by side.",
    },
  },
  {
    path: "/merge-tool",
    name: "merge-tool",
    component: MergePage,
    meta: {
      title: "Script Merger",
      description: "Merge script files with presets, preview, results, and manual conflict resolution.",
    },
  },
  {
    path: "/inventory",
    name: "inventory",
    component: InventoryPage,
    meta: {
      title: "Inventory Explorer",
      description: "Extract and browse game objects from script files with saved inventories and row details.",
    },
  },
  {
    path: "/settings",
    name: "settings",
    component: SettingsPage,
    meta: {
      title: "Settings",
      description: "Adjust game directories and reset app data.",
    },
  },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
