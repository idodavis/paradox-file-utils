/**
 * Helpers for mapping backend file trees onto Nuxt UI UTree `:items`.
 */
import type { TreeNode } from "@services/models";

/** A UTree item with optional select handler and nested children. */
export type AppTreeItem = {
  label: string;
  icon?: string;
  defaultExpanded?: boolean;
  onSelect?: () => void;
  children?: AppTreeItem[];
};

/** Convert TreeNode records into UTree `:items`. */
export function toTreeItems(
  nodes: TreeNode[],
  onFileSelect: (node: TreeNode) => void,
  expandAll = true,
): AppTreeItem[] {
  return nodes.map((node) => {
    const children = node.children?.length
      ? toTreeItems(node.children, onFileSelect, expandAll)
      : undefined;
    return {
      label: node.name,
      icon: children?.length ? "i-lucide-folder" : "i-lucide-file-text",
      defaultExpanded: expandAll,
      onSelect: children?.length ? undefined : () => onFileSelect(node),
      children,
    };
  });
}
