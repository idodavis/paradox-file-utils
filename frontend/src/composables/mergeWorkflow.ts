/**
 * Shared merge workflow state and actions for the Vue frontend.
 */
import { computed, inject, provide, ref } from "vue";
import { GetSettings, GetMergePresets, SaveMergePreset, DeleteMergePreset } from "@services/settingsservice";
import { GetUserDownloadsDir, GetGameScriptRoot, ReadFileContent, WriteWithBOM, SaveFile } from "@services/fileservice";
import {
  MergePreview,
  Merge,
  GenerateMergeReport,
  ValidateMergedFiles,
} from "@services/mergeservice";
import type { FileMergeResult, MergePreset, MergerOptions, PreviewItem } from "@services/models";
import { useCurrentGame } from "./appContext";
import { buildConflictMarkedFile } from "./textMerge";

/** Manual merge file state built entirely on the frontend. */
export type ManualMergeFile = {
  task: PreviewItem;
  contentA: string;
  contentB: string;
  markedContent: string;
  conflictCount: number;
  identical: boolean;
};

const mergeWorkflowKey = Symbol("merge-workflow");

/** Create isolated merge workflow state and actions for the merge page. */
export function createMergeWorkflow() {
  const toast = useToast();
  const currentGame = useCurrentGame();

  const config = ref({
    addAdditionalEntries: true,
    manualConflictResolution: false,
    useKeyList: false,
    customKeys: "",
    matchByFilenameOnly: false,
    includePathPattern: "",
    excludePathPattern: "",
    outputFileSuffix: "",
  });
  const settings = ref<Record<string, string | undefined>>({});
  const pathA = ref("");
  const pathB = ref("");
  const modPath = ref("");
  const filePairs = ref<{ pathA: string; pathB: string; outputName: string }[]>([]);
  const outputDir = ref("");
  const activeTab = ref<string | number>("vanilla");
  const previewItems = ref<PreviewItem[]>([]);
  const selectedRelPaths = ref<Record<string, boolean>>({});
  const mergeResults = ref<FileMergeResult[]>([]);
  const savingReport = ref(false);
  const validating = ref(false);
  const loadingPreview = ref(false);
  const runningMerge = ref(false);
  const errorMsg = ref("");
  const validationErrors = ref<{ path: string; line: number; error: string }[]>([]);
  const presets = ref<{ name: string; options: MergerOptions }[]>([]);
  const currentPresetName = ref("");
  const presetNameToSave = ref("");
  const defaultOutputDir = ref("");
  const currentManualFile = ref<ManualMergeFile | null>(null);
  const manualMergeQueue = ref<PreviewItem[]>([]);
  const diffSide = ref<"A" | "B">("A");
  const showResultDialog = ref(false);
  const resultDialogIndex = ref<number | null>(null);
  const resultFileAContent = ref("");
  const resultFileBContent = ref("");
  const resultMergedContent = ref("");
  const manualQueueTotal = ref(0);
  const manualQueueCurrent = ref(0);

  const labels = computed(() => {
    switch (activeTab.value) {
      case "vanilla":
        return { a: "Vanilla", b: "Mod" };
      case "dirs":
        return { a: "Dir A", b: "Dir B" };
      case "pairs":
        return { a: "File A", b: "File B" };
      default:
        return { a: "File A", b: "File B" };
    }
  });
  const canRun = computed(() => {
    const installPath = settings.value["ck3.install_path"] ?? settings.value["eu5.install_path"] ?? "";
    return {
      vanilla: !!installPath.trim() && !!modPath.value.trim() && !!outputDir.value.trim(),
      dirs: !!pathA.value.trim() && !!pathB.value.trim() && !!outputDir.value.trim(),
      pairs: filePairs.value.some((pair) => pair.pathA && pair.pathB) && !!outputDir.value.trim(),
    };
  });
  const mergeOptions = computed<MergerOptions>(() => ({
    addAdditionalEntries: config.value.addAdditionalEntries,
    manualConflictResolution: config.value.manualConflictResolution,
    keyList: config.value.useKeyList
      ? config.value.customKeys
          .split(/\r?\n/)
          .map((line) => line.trim())
          .filter(Boolean)
      : [],
    matchByFilenameOnly: config.value.matchByFilenameOnly,
    includePathPattern: config.value.includePathPattern,
    excludePathPattern: config.value.excludePathPattern,
    outputFileSuffix: config.value.outputFileSuffix,
    outputDir: outputDir.value,
  }));
  const summary = computed(() => ({
    files: mergeResults.value.length,
    added: mergeResults.value.reduce((sum, item) => sum + (item.added ?? 0), 0),
    changed: mergeResults.value.reduce((sum, item) => sum + (item.changed ?? 0), 0),
  }));
  const conflicts = computed(() => mergeResults.value.filter((item) => (item.resolvedConflicts?.length ?? 0) > 0));
  const selectedResult = computed(() =>
    resultDialogIndex.value === null ? null : (mergeResults.value[resultDialogIndex.value] ?? null),
  );
  const showAdditionsTab = computed(() => config.value.addAdditionalEntries);
  const activeTabOptions = [
    { label: "Vanilla vs mod", value: "vanilla", slot: "vanilla" },
    { label: "Two Directories", value: "dirs", slot: "dirs" },
    { label: "File Pairs", value: "pairs", slot: "pairs" },
  ];
  const resolutionModeOptions = [
    { label: "Auto", value: false },
    { label: "Manual", value: true },
  ];
  const previewColumns = [
    { id: "selected", header: "Use" },
    { accessorKey: "relPath", header: "Relative path" },
    { accessorKey: "outputPath", header: "Output" },
    { id: "wouldOverwrite", header: "Overwrite" },
  ];
  const resultColumns = [
    { id: "outputPath", header: "Saved to" },
    { accessorKey: "changed", header: "Changed" },
    { accessorKey: "added", header: "Added" },
  ];
  const diffSideOptions = computed(() => [
    { label: `${labels.value.a} ↔ Merged`, value: "A" as const },
    { label: `${labels.value.b} ↔ Merged`, value: "B" as const },
  ]);

  function pairsToTasks(pairs: { pathA: string; pathB: string; outputName: string }[]): PreviewItem[] {
    const base = outputDir.value.replace(/\/?$/, "");
    return pairs
      .filter((pair) => pair.pathA && pair.pathB)
      .map((pair) => {
        const baseName = pair.outputName?.trim() || pair.pathA.split(/[/\\]/).pop() || "output";
        const relPath = baseName.replace(/\.[^.]+$/, "") + ".txt";
        return {
          relPath,
          pathA: pair.pathA,
          pathB: pair.pathB,
          outputPath: `${base}/${relPath}`,
          wouldOverwrite: false,
        };
      });
  }

  function resetMergeState(): void {
    mergeResults.value = [];
    errorMsg.value = "";
    previewItems.value = [];
    selectedRelPaths.value = {};
    validationErrors.value = [];
    manualQueueTotal.value = 0;
    manualQueueCurrent.value = 0;
  }

  function setSelectedAll(value: boolean): void {
    selectedRelPaths.value = Object.fromEntries(previewItems.value.map((item) => [item.relPath, value]));
  }

  async function loadSettings(): Promise<void> {
    settings.value = (await GetSettings()) ?? {};
  }

  async function refreshPresets(): Promise<void> {
    try {
      const savedPresets = (await GetMergePresets()) ?? [];
      presets.value = savedPresets.map((preset: MergePreset) => ({ name: preset.name, options: preset.options }));
      const last = localStorage.getItem("last_merge_preset");
      const found = last ? presets.value.find((preset) => preset.name === last) : null;
      if (found) applyPreset(found);
      else if (!outputDir.value) outputDir.value = defaultOutputDir.value;
    } catch {
      presets.value = [];
    }
  }

  function applyPreset(preset: { name: string; options: MergerOptions }): void {
    const options = preset.options;
    config.value.addAdditionalEntries = options.addAdditionalEntries ?? true;
    config.value.manualConflictResolution = options.manualConflictResolution ?? false;
    config.value.useKeyList = (options.keyList?.length ?? 0) > 0;
    config.value.customKeys = (options.keyList ?? []).join("\n");
    config.value.matchByFilenameOnly = options.matchByFilenameOnly ?? false;
    config.value.includePathPattern = options.includePathPattern ?? "";
    config.value.excludePathPattern = options.excludePathPattern ?? "";
    config.value.outputFileSuffix = options.outputFileSuffix ?? "";
    outputDir.value = options.outputDir?.trim() || defaultOutputDir.value || "";
    localStorage.setItem("last_merge_preset", preset.name);
    currentPresetName.value = preset.name;
  }

  function applyPresetByName(name: string | null): void {
    const preset = presets.value.find((item) => item.name === name);
    if (preset) applyPreset(preset);
    else if (!outputDir.value) outputDir.value = defaultOutputDir.value;
  }

  async function saveCurrentPreset(): Promise<void> {
    if (!presetNameToSave.value.trim()) return;
    const name = presetNameToSave.value.trim();
    try {
      await SaveMergePreset(name, mergeOptions.value);
      toast.add({ title: "Preset saved", color: "success" });
      localStorage.setItem("last_merge_preset", name);
      currentPresetName.value = name;
      presetNameToSave.value = "";
      await refreshPresets();
    } catch (error) {
      errorMsg.value = error instanceof Error ? error.message : String(error);
    }
  }

  async function deletePreset(name: string): Promise<void> {
    try {
      await DeleteMergePreset(name);
      if (currentPresetName.value === name) currentPresetName.value = "";
      toast.add({ title: "Preset deleted", color: "success" });
      await refreshPresets();
    } catch (error) {
      errorMsg.value = error instanceof Error ? error.message : String(error);
    }
  }

  async function runPreview(mode: "vanilla" | "dirs"): Promise<void> {
    if (!canRun.value[mode]) return;
    loadingPreview.value = true;
    resetMergeState();
    try {
      const installPath = (settings.value["ck3.install_path"] ?? settings.value["eu5.install_path"] ?? "").trim();
      const pathOne = mode === "vanilla" ? await GetGameScriptRoot(currentGame.value, installPath) : pathA.value;
      const pathTwo = mode === "vanilla" ? modPath.value : pathB.value;
      const items = (await MergePreview(pathOne, pathTwo, outputDir.value, mergeOptions.value)) ?? [];
      previewItems.value = items;
      selectedRelPaths.value = Object.fromEntries(items.map((item) => [item.relPath, true]));
      if (!items.length) errorMsg.value = "No matching files found.";
    } catch (error) {
      errorMsg.value = error instanceof Error ? error.message : String(error);
    } finally {
      loadingPreview.value = false;
    }
  }

  async function runMergeWithTasks(tasks: PreviewItem[]): Promise<void> {
    if (!tasks.length) {
      errorMsg.value = "No files selected.";
      return;
    }
    if (config.value.manualConflictResolution) {
      mergeResults.value = [];
      errorMsg.value = "";
      manualMergeQueue.value = tasks;
      manualQueueTotal.value = tasks.length;
      manualQueueCurrent.value = 1;
      runningMerge.value = true;
      await processManualQueue();
      return;
    }
    runningMerge.value = true;
    mergeResults.value = [];
    errorMsg.value = "";
    try {
      const results = (await Merge(tasks, mergeOptions.value)) ?? [];
      mergeResults.value = results;
    } catch (error) {
      errorMsg.value = error instanceof Error ? error.message : String(error);
    } finally {
      runningMerge.value = false;
    }
  }

  async function runDirMerge(mode: "vanilla" | "dirs"): Promise<void> {
    if (!canRun.value[mode]) return;
    const selected = previewItems.value.filter((item) => selectedRelPaths.value[item.relPath] !== false);
    await runMergeWithTasks(selected);
  }

  async function runPairMerge(): Promise<void> {
    if (!canRun.value.pairs) return;
    await runMergeWithTasks(pairsToTasks(filePairs.value));
  }

  async function processManualQueue(): Promise<void> {
    if (manualMergeQueue.value.length === 0) {
      runningMerge.value = false;
      manualQueueCurrent.value = 0;
      return;
    }
    const task = manualMergeQueue.value[0];
    try {
      const [contentA, contentB] = await Promise.all([
        ReadFileContent(task.pathA),
        ReadFileContent(task.pathB),
      ]);
      const marked = buildConflictMarkedFile(contentA, contentB, {
        fileName: task.relPath,
        labelA: labels.value.a,
        labelB: labels.value.b,
      });
      if (marked.identical) {
        await WriteWithBOM(task.outputPath, contentA);
        mergeResults.value.push({
          filePath: task.relPath,
          fileAPath: task.pathA,
          fileBPath: task.pathB,
          outputPath: task.outputPath,
          changed: 0,
          added: 0,
          resolvedConflicts: [],
        });
        advanceManualQueue();
        return;
      }
      currentManualFile.value = {
        task,
        contentA,
        contentB,
        markedContent: marked.content,
        conflictCount: marked.conflictCount,
        identical: marked.identical,
      };
    } catch (error) {
      errorMsg.value = error instanceof Error ? error.message : String(error);
      manualMergeQueue.value.shift();
      manualQueueCurrent.value += 1;
      await processManualQueue();
    }
  }

  function advanceManualQueue(): void {
    currentManualFile.value = null;
    manualMergeQueue.value.shift();
    if (manualMergeQueue.value.length > 0) manualQueueCurrent.value += 1;
    void processManualQueue();
  }

  async function saveManual(payload: { content: string; stats: { changed: number; added: number } }): Promise<void> {
    if (!currentManualFile.value) return;
    const task = currentManualFile.value.task;
    try {
      await WriteWithBOM(task.outputPath, payload.content);
      mergeResults.value.push({
        filePath: task.relPath,
        fileAPath: task.pathA,
        fileBPath: task.pathB,
        outputPath: task.outputPath,
        changed: payload.stats.changed,
        added: payload.stats.added,
        resolvedConflicts: [{ key: "Manual", usedSide: "Manual", reason: "User selection" }],
      });
      advanceManualQueue();
    } catch (error) {
      errorMsg.value = `Failed to save manual merge: ${String(error)}`;
    }
  }

  async function autoMergeCurrentFile(): Promise<void> {
    if (!currentManualFile.value) return;
    const { task } = currentManualFile.value;
    try {
      const results = await Merge([task], mergeOptions.value);
      if (results?.length) mergeResults.value.push(...results);
    } catch (error) {
      errorMsg.value = `Failed to auto-merge ${task.relPath}: ${String(error)}`;
    } finally {
      advanceManualQueue();
    }
  }

  function skipFile(): void {
    advanceManualQueue();
  }

  function cancelManualMerge(): void {
    currentManualFile.value = null;
    manualMergeQueue.value = [];
    runningMerge.value = false;
    manualQueueTotal.value = 0;
    manualQueueCurrent.value = 0;
  }

  function truncatePath(path: string): string {
    if (!path) return "";
    const parts = path.split(/[/\\]/);
    return parts.length > 2 ? `.../${parts.slice(-2).join("/")}` : (parts.pop() ?? path);
  }

  async function saveReport(): Promise<void> {
    savingReport.value = true;
    try {
      const markdown = await GenerateMergeReport(
        mergeResults.value,
        summary.value.added,
        summary.value.changed,
        0,
        labels.value.a,
        labels.value.b,
      );
      const path = await SaveFile("Save merge report", "merge_report.md", markdown, "md");
      if (path) toast.add({ title: "Report saved", color: "success" });
    } catch (error) {
      errorMsg.value = error instanceof Error ? error.message : String(error);
    } finally {
      savingReport.value = false;
    }
  }

  async function runValidation(): Promise<void> {
    validating.value = true;
    validationErrors.value = [];
    try {
      const outputs = mergeResults.value.map((item) => item.outputPath).filter(Boolean);
      const errors = (await ValidateMergedFiles(outputs)) ?? [];
      validationErrors.value = errors;
      toast.add({
        title: errors.length ? `${errors.length} errors` : "All valid",
        color: errors.length ? "warning" : "success",
      });
    } catch (error) {
      errorMsg.value = error instanceof Error ? error.message : String(error);
    } finally {
      validating.value = false;
    }
  }

  async function openResult(index: number): Promise<void> {
    resultDialogIndex.value = index;
    showResultDialog.value = true;
    const result = mergeResults.value[index];
    if (!result) return;
    const [fileAContent, fileBContent, mergedContent] = await Promise.all([
      ReadFileContent(result.fileAPath),
      ReadFileContent(result.fileBPath),
      ReadFileContent(result.outputPath),
    ]);
    resultFileAContent.value = fileAContent;
    resultFileBContent.value = fileBContent;
    resultMergedContent.value = mergedContent;
  }

  async function initialize(): Promise<void> {
    defaultOutputDir.value = (await GetUserDownloadsDir()) ?? "";
    await loadSettings();
    await refreshPresets();
    if (!outputDir.value) outputDir.value = defaultOutputDir.value;
  }

  async function resetForGameChange(): Promise<void> {
    await loadSettings();
    await refreshPresets();
    resetMergeState();
  }

  return {
    currentGame,
    config,
    settings,
    pathA,
    pathB,
    modPath,
    filePairs,
    outputDir,
    activeTab,
    previewItems,
    selectedRelPaths,
    mergeResults,
    savingReport,
    validating,
    loadingPreview,
    runningMerge,
    errorMsg,
    validationErrors,
    presets,
    currentPresetName,
    presetNameToSave,
    defaultOutputDir,
    currentManualFile,
    manualMergeQueue,
    diffSide,
    showResultDialog,
    resultDialogIndex,
    resultFileAContent,
    resultFileBContent,
    resultMergedContent,
    manualQueueTotal,
    manualQueueCurrent,
    labels,
    canRun,
    mergeOptions,
    summary,
    conflicts,
    selectedResult,
    showAdditionsTab,
    activeTabOptions,
    resolutionModeOptions,
    previewColumns,
    resultColumns,
    diffSideOptions,
    applyPresetByName,
    saveCurrentPreset,
    deletePreset,
    runPreview,
    runDirMerge,
    runPairMerge,
    setSelectedAll,
    saveManual,
    autoMergeCurrentFile,
    skipFile,
    cancelManualMerge,
    truncatePath,
    saveReport,
    runValidation,
    openResult,
    initialize,
    resetForGameChange,
  };
}

export type MergeWorkflow = ReturnType<typeof createMergeWorkflow>;

/** Provide merge workflow state to descendant components. */
export function provideMergeWorkflow(workflow: MergeWorkflow): void {
  provide(mergeWorkflowKey, workflow);
}

/** Read the provided merge workflow, throwing if missing. */
export function useMergeWorkflow(): MergeWorkflow {
  const workflow = inject<MergeWorkflow>(mergeWorkflowKey);
  if (!workflow) throw new Error("Merge workflow was not provided.");
  return workflow;
}
