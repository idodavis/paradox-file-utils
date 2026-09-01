/**
 * Shared language-health query: one fetch per workspace, not per mounted strip.
 */
import { computed, reactive, toValue, type MaybeRefOrGetter } from "vue";
import { useQuery, useQueryCache } from "@pinia/colada";
import { useEventListener } from "@vueuse/core";
import { Events } from "@wailsio/runtime";
import {
  EnsureSession,
  GetLanguageHealth,
  RebuildInstallSemantics,
} from "@services/sessionservice";

type ScanState = {
  scanning: boolean;
  pct: number;
  msg: string;
};

const scanByInstall = new Map<string, ScanState>();
let scanListening = false;

function scanState(id: string): ScanState {
  let s = scanByInstall.get(id);
  if (!s) {
    s = reactive({ scanning: false, pct: 0, msg: "" });
    scanByInstall.set(id, s);
  }
  return s;
}

function ensureScanListener(): void {
  if (scanListening) return;
  scanListening = true;
  Events.On("lang:scan-progress", (ev: { data?: unknown }) => {
    const data = ev.data;
    if (!data || typeof data !== "object") return;
    const row = data as { installId?: string; pct?: number; msg?: string };
    if (!row.installId) return;
    const s = scanState(row.installId);
    if (typeof row.pct === "number") s.pct = row.pct;
    if (typeof row.msg === "string") s.msg = row.msg;
  });
}

/** Health payload, rescan action, and shared scan progress for a workspace. */
export function useLanguageHealth(workspaceId: MaybeRefOrGetter<string>) {
  ensureScanListener();
  const queryCache = useQueryCache();
  const id = computed(() => toValue(workspaceId));
  const { data: health, refetch } = useQuery({
    key: () => ["session", "health", id.value],
    query: async () => {
      const wsId = id.value;
      if (!wsId) return null;
      return GetLanguageHealth(wsId);
    },
    enabled: () => !!id.value,
    refetchOnWindowFocus: false,
  });
  const scan = computed(() =>
    scanState(health.value?.installId || id.value || "_"),
  );

  useEventListener(window, "focus", () => {
    if (id.value) void refetch();
  });

  const lastScanned = computed(() => {
    const raw = health.value?.scannedAt;
    if (!raw) return "";
    const ms = Date.parse(raw);
    if (Number.isNaN(ms)) return raw;
    return new Date(ms).toLocaleString();
  });

  /** Auto-rescan install semantics then reindex; bust session queries. */
  async function rescan(): Promise<void> {
    const wsId = id.value;
    const installId = health.value?.installId;
    if (!wsId || !installId || scan.value.scanning) return;
    scan.value.scanning = true;
    scan.value.pct = 0;
    scan.value.msg = "starting";
    try {
      await RebuildInstallSemantics(installId);
      await EnsureSession(wsId);
      await queryCache.invalidateQueries({ key: ["session"] });
    } finally {
      scan.value.scanning = false;
      scan.value.msg = "";
      scan.value.pct = 0;
    }
  }

  return {
    health,
    scanning: computed(() => scan.value.scanning),
    scanPct: computed(() => scan.value.pct),
    scanMsg: computed(() => scan.value.msg),
    lastScanned,
    rescan,
  };
}
