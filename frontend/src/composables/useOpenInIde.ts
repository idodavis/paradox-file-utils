/**
 * Navigate to workspace IDE then open a file once the workbench is ready.
 */
import { nextTick, type MaybeRefOrGetter, toValue } from "vue";
import { useRouter } from "vue-router";
import { openFile } from "../ide/commands";
import { whenWorkbenchReady } from "../ide/workbenchHost";

/** Open a file in the workbench after routing to the IDE route. */
export function useOpenInIde() {
  const router = useRouter();

  async function openInIde(
    workspaceId: MaybeRefOrGetter<string>,
    path: string,
    line?: number,
  ): Promise<void> {
    const id = toValue(workspaceId);
    if (!id || !path) return;
    await router.push({ name: "workspace-ide", params: { id } });
    await nextTick();
    await whenWorkbenchReady();
    await openFile(path, line);
  }

  return { openInIde };
}
