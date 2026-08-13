/**
 * Pinia store for Workspace IDE tabs (files + compare) and explorer expansion.
 */
import { computed, ref } from "vue";
import { defineStore } from "pinia";

/** One open file buffer. */
export type FileTab = {
  kind: "file";
  /** Stable id (= path). */
  id: string;
  path: string;
  name: string;
  contents: string;
  savedContents: string;
  dirty: boolean;
};

/** A dedicated compare/diff tab (VS Code-style). */
export type CompareTab = {
  kind: "compare";
  id: string;
  leftPath: string;
  rightPath: string;
  leftContents: string;
  rightContents: string;
  name: string;
};

/** Any IDE editor tab. */
export type IdeTab = FileTab | CompareTab;

/** Build a stable compare-tab id from two paths. */
export function compareTabId(leftPath: string, rightPath: string): string {
  return `compare:${leftPath}|${rightPath}`;
}

/** Explorer / editor buffer state for the workspace IDE. */
export const useIdeStore = defineStore("ide", () => {
  const tabs = ref<IdeTab[]>([]);
  const activeTabId = ref<string | null>(null);
  /** Expanded directory full paths, keyed by workspace id. */
  const expandedByWorkspace = ref<Record<string, string[]>>({});

  const activeTab = computed(
    () => tabs.value.find((t) => t.id === activeTabId.value) ?? null,
  );
  const activeFileTab = computed(() => {
    const t = activeTab.value;
    return t?.kind === "file" ? t : null;
  });
  const fileTabs = computed(() =>
    tabs.value.filter((t): t is FileTab => t.kind === "file"),
  );

  /** Load persisted expansion set for a workspace. */
  function loadExpanded(workspaceId: string): Set<string> {
    const key = `ide.expanded.${workspaceId}`;
    try {
      const raw = localStorage.getItem(key);
      const list = raw ? (JSON.parse(raw) as string[]) : [];
      expandedByWorkspace.value = {
        ...expandedByWorkspace.value,
        [workspaceId]: list,
      };
      return new Set(list);
    } catch {
      return new Set();
    }
  }

  /** Persist expansion set for a workspace. */
  function saveExpanded(workspaceId: string, paths: Set<string>): void {
    const list = [...paths];
    expandedByWorkspace.value = {
      ...expandedByWorkspace.value,
      [workspaceId]: list,
    };
    localStorage.setItem(`ide.expanded.${workspaceId}`, JSON.stringify(list));
  }

  /** Open or focus a file tab. */
  function openFileTab(path: string, contents: string): void {
    const existing = tabs.value.find(
      (t) => t.kind === "file" && t.path === path,
    );
    if (existing) {
      activeTabId.value = existing.id;
      return;
    }
    const name = path.split(/[/\\]/).pop() || path;
    const tab: FileTab = {
      kind: "file",
      id: path,
      path,
      name,
      contents,
      savedContents: contents,
      dirty: false,
    };
    tabs.value = [...tabs.value, tab];
    activeTabId.value = tab.id;
  }

  /** Open or focus a compare tab (does not alter other file tabs). */
  function openCompareTab(
    leftPath: string,
    leftContents: string,
    rightPath: string,
    rightContents: string,
  ): void {
    const id = compareTabId(leftPath, rightPath);
    const existing = tabs.value.find((t) => t.id === id);
    if (existing) {
      activeTabId.value = id;
      return;
    }
    const leftName = leftPath.split(/[/\\]/).pop() || leftPath;
    const rightName = rightPath.split(/[/\\]/).pop() || rightPath;
    const tab: CompareTab = {
      kind: "compare",
      id,
      leftPath,
      rightPath,
      leftContents,
      rightContents,
      name: `${leftName} ↔ ${rightName}`,
    };
    tabs.value = [...tabs.value, tab];
    activeTabId.value = id;
  }

  /** Apply live edit contents to a file tab. */
  function patchTabContents(path: string, contents: string): void {
    tabs.value = tabs.value.map((t) =>
      t.kind === "file" && t.path === path
        ? { ...t, contents, dirty: contents !== t.savedContents }
        : t,
    );
  }

  /** Mark a file tab as saved. */
  function markSaved(path: string): void {
    tabs.value = tabs.value.map((t) =>
      t.kind === "file" && t.path === path
        ? { ...t, savedContents: t.contents, dirty: false }
        : t,
    );
  }

  /** Close one tab by id. */
  function closeTab(id: string): void {
    const idx = tabs.value.findIndex((t) => t.id === id);
    if (idx < 0) return;
    const next = tabs.value.filter((t) => t.id !== id);
    tabs.value = next;
    if (activeTabId.value === id) {
      activeTabId.value = next[Math.max(0, idx - 1)]?.id ?? null;
    }
  }

  /** Close every tab except the given id. */
  function closeOthers(id: string): void {
    tabs.value = tabs.value.filter((t) => t.id === id);
    activeTabId.value = id;
  }

  /** Close every tab (caller must handle dirty prompts first). */
  function closeAll(): void {
    tabs.value = [];
    activeTabId.value = null;
  }

  /** Dirty file tabs only. */
  function dirtyFileTabs(): FileTab[] {
    return fileTabs.value.filter((t) => t.dirty);
  }

  /** Retarget file tabs after rename/move. */
  function retargetPath(from: string, to: string): void {
    tabs.value = tabs.value.map((t) => {
      if (t.kind !== "file") return t;
      if (t.path === from) {
        const name = to.split(/[/\\]/).pop() || to;
        return { ...t, id: to, path: to, name };
      }
      if (t.path.startsWith(from + "/") || t.path.startsWith(from + "\\")) {
        const path = to + t.path.slice(from.length);
        const name = path.split(/[/\\]/).pop() || path;
        return { ...t, id: path, path, name };
      }
      return t;
    });
    if (activeTabId.value === from) activeTabId.value = to;
    else if (
      activeTabId.value?.startsWith(from + "/") ||
      activeTabId.value?.startsWith(from + "\\")
    ) {
      activeTabId.value = to + activeTabId.value.slice(from.length);
    }
  }

  /** Drop tabs whose paths were deleted. */
  function dropDeleted(path: string): void {
    const gone = (p: string) =>
      p === path || p.startsWith(path + "/") || p.startsWith(path + "\\");
    const next = tabs.value.filter((t) => {
      if (t.kind === "file") return !gone(t.path);
      return !gone(t.leftPath) && !gone(t.rightPath);
    });
    const activeGone =
      activeTabId.value != null &&
      !next.some((t) => t.id === activeTabId.value);
    tabs.value = next;
    if (activeGone) activeTabId.value = next[0]?.id ?? null;
  }

  /** Reset IDE session (e.g. workspace switch). */
  function resetSession(): void {
    tabs.value = [];
    activeTabId.value = null;
  }

  return {
    tabs,
    activeTabId,
    activeTab,
    activeFileTab,
    fileTabs,
    loadExpanded,
    saveExpanded,
    openFileTab,
    openCompareTab,
    patchTabContents,
    markSaved,
    closeTab,
    closeOthers,
    closeAll,
    dirtyFileTabs,
    retargetPath,
    dropDeleted,
    resetSession,
  };
});
