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
    path: "/workspace/:id/conflicts",
    name: "conflicts",
    component: ConflictPage,
    meta: PAGE_CATALOG.conflicts,
  },
  {
    path: "/workspace/:id/loc",
    name: "loc-coverage",
    component: LocCoveragePage,
    meta: PAGE_CATALOG["loc-coverage"],
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
    path: "/tools/merge",
    name: "tools-merge",
    component: ToolsMergePage,
    meta: PAGE_CATALOG["tools-merge"],
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
