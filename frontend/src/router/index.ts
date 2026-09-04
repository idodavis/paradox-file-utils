/**
 * Vue router for PMT workspace-centric page routing.
 */
import { createRouter, createWebHashHistory } from "vue-router";
import LibraryPage from "../pages/LibraryPage.vue";
import WizardPage from "../pages/WizardPage.vue";
import WorkspaceIdePage from "../pages/WorkspaceIdePage.vue";
import EventGraphPage from "../pages/EventGraphPage.vue";
import HealthPage from "../pages/HealthPage.vue";
import PatcherPage from "../pages/PatcherPage.vue";
import ReleasePage from "../pages/ReleasePage.vue";
import WorkspaceSettingsPage from "../pages/WorkspaceSettingsPage.vue";
import { PAGE_CATALOG } from "../workspaceTools";
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
    meta: PAGE_CATALOG.library,
  },
  {
    path: "/wizard",
    name: "wizard",
    component: WizardPage,
    meta: PAGE_CATALOG.wizard,
  },
  {
    path: "/workspace/:id",
    name: "workspace-ide",
    components: { ide: WorkspaceIdePage },
    meta: PAGE_CATALOG["workspace-ide"],
  },
  {
    path: "/workspace/:id/graph",
    name: "event-graph",
    component: EventGraphPage,
    meta: PAGE_CATALOG["event-graph"],
  },
  {
    path: "/workspace/:id/health",
    name: "health",
    component: HealthPage,
    meta: PAGE_CATALOG.health,
  },
  {
    path: "/workspace/:id/patcher",
    name: "patcher",
    component: PatcherPage,
    meta: PAGE_CATALOG.patcher,
  },
  {
    path: "/workspace/:id/release",
    name: "release",
    component: ReleasePage,
    meta: PAGE_CATALOG.release,
  },
  {
    path: "/workspace/:id/settings",
    name: "workspace-settings",
    component: WorkspaceSettingsPage,
    meta: PAGE_CATALOG["workspace-settings"],
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
  if (!to.meta.workspace) return true;
  const id = to.params.id;
  if (typeof id !== "string" || !id) return true;
  const ws = useWorkspaceStore();
  ws.setActiveWorkspace(id);
  return true;
});

export default router;
