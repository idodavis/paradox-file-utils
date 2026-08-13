/**
 * Helpers for mapping backend file trees onto Nuxt UI UTree `:items`.
 */
import type { TreeItem } from "@nuxt/ui";
import type { DirEntry } from "@services/models";

/** Explorer root kinds. */
export type FileRootType = "game" | "mod" | "staging";

/** One multi-root explorer root. */
export type FileRoot = { label: string; path: string; type: FileRootType };

/** Extra data hung on tree items for context menus / open. */
export type IdeTreeMeta = {
  fullPath: string;
  isDir: boolean;
  rootType: FileRootType;
  rootPath: string;
};

/** Tree item with IDE metadata. */
export type IdeTreeItem = {
  label?: string;
  icon?: string;
  value?: string;
  defaultExpanded?: boolean;
  disabled?: boolean;
  onSelect?: TreeItem["onSelect"];
  onToggle?: TreeItem["onToggle"];
  meta?: IdeTreeMeta;
  children?: IdeTreeItem[];
};

/** Icon for a file name by extension. */
export function iconForFileName(name: string): string {
  const ext = name.includes(".") ? name.split(".").pop()?.toLowerCase() : "";
  switch (ext) {
    case "yml":
    case "yaml":
      return "i-lucide-languages";
    case "gui":
      return "i-lucide-layout-dashboard";
    case "json":
      return "i-lucide-braces";
    case "mod":
    case "info":
      return "i-lucide-file-cog";
    default:
      return "i-lucide-file-text";
  }
}

/** Map DirEntry list into UTree items (lazy children placeholder for dirs). */
export function dirEntriesToTreeItems(
  entries: DirEntry[],
  rootType: FileRootType,
  rootPath: string,
  onFileSelect: (path: string) => void,
): IdeTreeItem[] {
  return entries.map((entry) => {
    const meta: IdeTreeMeta = {
      fullPath: entry.fullPath,
      isDir: entry.isDir,
      rootType,
      rootPath,
    };
    if (entry.isDir) {
      return {
        label: entry.name,
        icon: "i-lucide-folder",
        value: entry.fullPath,
        meta,
        children: [],
        defaultExpanded: false,
      } satisfies IdeTreeItem;
    }
    return {
      label: entry.name,
      icon: iconForFileName(entry.name),
      value: entry.fullPath,
      meta,
      onSelect: () => onFileSelect(entry.fullPath),
    } satisfies IdeTreeItem;
  });
}

/** Build top-level root items for the explorer. */
export function rootsToTreeItems(
  roots: FileRoot[],
  childrenByRoot: Record<string, DirEntry[]>,
  onFileSelect: (path: string) => void,
): IdeTreeItem[] {
  return roots.map((root) => {
    const kids = childrenByRoot[root.path] ?? [];
    return {
      label: root.label,
      icon:
        root.type === "game"
          ? "i-lucide-gamepad-2"
          : root.type === "staging"
            ? "i-lucide-layers"
            : "i-lucide-folder-cog",
      value: root.path,
      defaultExpanded: false,
      meta: {
        fullPath: root.path,
        isDir: true,
        rootType: root.type,
        rootPath: root.path,
      },
      children: dirEntriesToTreeItems(kids, root.type, root.path, onFileSelect),
    } satisfies IdeTreeItem;
  });
}

/** Find a tree item by full path. */
export function findTreeItem(
  items: IdeTreeItem[],
  fullPath: string,
): IdeTreeItem | null {
  for (const item of items) {
    if (item.meta?.fullPath === fullPath || item.value === fullPath) return item;
    if (item.children?.length) {
      const hit = findTreeItem(item.children, fullPath);
      if (hit) return hit;
    }
  }
  return null;
}

/** Patch children of the item matching fullPath. */
export function setTreeChildren(
  items: IdeTreeItem[],
  fullPath: string,
  children: IdeTreeItem[],
): IdeTreeItem[] {
  const next: IdeTreeItem[] = [];
  for (const item of items) {
    if (item.meta?.fullPath === fullPath || item.value === fullPath) {
      next.push({ ...item, children });
      continue;
    }
    if (item.children?.length) {
      next.push({
        ...item,
        children: setTreeChildren(item.children, fullPath, children),
      });
      continue;
    }
    next.push(item);
  }
  return next;
}
