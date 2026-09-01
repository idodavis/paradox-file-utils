/**
 * Wails-backed VS Code file system provider for the embedded workbench.
 */
import { Events } from "@wailsio/runtime";
import {
  FileType,
  FileChangeType,
  FileSystemProviderCapabilities,
  FileSystemProviderError,
  FileSystemProviderErrorCode,
  type IFileSystemProviderWithFileReadWriteCapability,
  type IStat,
  type IWatchOptions,
  type IFileWriteOptions,
  type IFileDeleteOptions,
  type IFileOverwriteOptions,
  type IFileChange,
} from "@codingame/monaco-vscode-files-service-override";
import * as monaco from "monaco-editor";
import {
  StatPath,
  ListDirectory,
  ReadFileBase64,
  WriteFileBase64,
  CreateDir,
  DeletePath,
  RenamePath,
} from "@services/fileservice";

type Uri = monaco.Uri;

/** Role of a multi-root IDE folder (explorer color tags). */
export type IdeRootKind = "game" | "mod" | "staging";

/** Workspace root with optional read-only policy (game install). */
export type IdeRoot = {
  label: string;
  path: string;
  readOnly: boolean;
  kind: IdeRootKind;
  originId?: string;
  color?: string;
  thumbnail?: string;
};

type Disposable = { dispose(): void };

function fsPath(resource: Uri): string {
  let p = resource.fsPath || resource.path;
  if (p.startsWith("/") && /^\/[A-Za-z]:/.test(p)) {
    p = p.slice(1);
  }
  return p.replace(/\//g, "\\");
}

/** Decode standard base64 to bytes without a per-char loop. */
function bytesFromB64(b64: string): Uint8Array {
  const bin = atob(b64);
  return Uint8Array.from(bin, (c) => c.charCodeAt(0));
}

/** Encode bytes as standard base64 in chunks (avoids call-stack limits). */
function b64FromBytes(data: Uint8Array): string {
  const chunk = 0x8000;
  const parts: string[] = [];
  for (let i = 0; i < data.length; i += chunk) {
    parts.push(String.fromCharCode(...data.subarray(i, i + chunk)));
  }
  return btoa(parts.join(""));
}

function normFs(p: string): string {
  return p.replace(/\//g, "\\").toLowerCase().replace(/\\+$/, "");
}

function underRoot(filePath: string, root: string): boolean {
  const a = normFs(filePath);
  const b = normFs(root);
  return a === b || a.startsWith(b + "\\");
}

type ModRootDeletedFn = (originId: string) => void | Promise<void>;

let modRootDeleted: ModRootDeletedFn | undefined;

/** Called after a successful disk delete of a mounted mod root. */
export function setModRootDeletedHook(fn: ModRootDeletedFn | undefined): void {
  modRootDeleted = fn;
}

/** Minimal event emitter matching VS Code Event shape. */
class SimpleEmitter<T> {
  private listeners = new Set<(e: T) => void>();
  readonly event = (listener: (e: T) => void): Disposable => {
    this.listeners.add(listener);
    return {
      dispose: () => {
        this.listeners.delete(listener);
      },
    };
  };
  fire(e: T): void {
    for (const l of this.listeners) l(e);
  }
}

/** File system provider bridging workbench URIs to Go FileService. */
export class WailsFileSystemProvider
  implements IFileSystemProviderWithFileReadWriteCapability
{
  readonly capabilities =
    FileSystemProviderCapabilities.FileReadWrite |
    FileSystemProviderCapabilities.PathCaseSensitive;

  private readonly _onDidChangeCapabilities = new SimpleEmitter<void>();
  readonly onDidChangeCapabilities = this._onDidChangeCapabilities.event;

  private readonly _onDidChangeFile = new SimpleEmitter<readonly IFileChange[]>();
  readonly onDidChangeFile = this._onDidChangeFile.event;

  private roots: IdeRoot[] = [];

  constructor() {
    Events.On("fs:changed", (ev: { data?: unknown }) => {
      this.onExternalChange(ev.data);
    });
  }

  /** Apply a session-watcher event to the workbench file tree. */
  private onExternalChange(data: unknown): void {
    if (!data || typeof data !== "object") return;
    const row = data as { path?: string; deleted?: boolean };
    if (!row.path) return;
    this._onDidChangeFile.fire([
      {
        resource: monaco.Uri.file(row.path),
        type: row.deleted ? FileChangeType.DELETED : FileChangeType.UPDATED,
      },
    ]);
  }

  /** Update multi-root mount points and read-only game roots. */
  setRoots(roots: IdeRoot[]): void {
    this.roots = roots.map((r) => ({
      ...r,
      path: r.path.replace(/\//g, "\\"),
    }));
  }

  /** True when the path is under a read-only (game) root. */
  isReadOnly(filePath: string): boolean {
    return this.roots.some((r) => r.readOnly && underRoot(filePath, r.path));
  }

  /** True when path is under a mounted IDE root (or no roots yet). */
  private manages(path: string): boolean {
    if (!this.roots.length) return false;
    return this.roots.some((r) => underRoot(path, r.path));
  }

  private assertWritable(path: string): void {
    for (const r of this.roots) {
      if (r.readOnly && underRoot(path, r.path)) {
        throw FileSystemProviderError.create(
          "Game install files are read-only",
          FileSystemProviderErrorCode.NoPermissions,
        );
      }
    }
  }

  private notFound(): never {
    throw FileSystemProviderError.create(
      "File not found",
      FileSystemProviderErrorCode.FileNotFound,
    );
  }

  watch(_resource: Uri, _opts: IWatchOptions): Disposable {
    return { dispose() {} };
  }

  async stat(resource: Uri): Promise<IStat> {
    const path = fsPath(resource);
    // Let memory/other overlays own non-root paths (e.g. /pmt.code-workspace).
    if (!this.manages(path)) this.notFound();
    const st = await StatPath(path);
    if (!st.exists) this.notFound();
    return {
      type: st.isDir ? FileType.Directory : FileType.File,
      ctime: st.ctimeMs,
      mtime: st.mtimeMs,
      size: st.size,
    };
  }

  async mkdir(resource: Uri): Promise<void> {
    const path = fsPath(resource);
    if (!this.manages(path)) this.notFound();
    this.assertWritable(path);
    await CreateDir(path);
  }

  async readdir(resource: Uri): Promise<[string, FileType][]> {
    const path = fsPath(resource);
    if (!this.manages(path)) this.notFound();
    const entries = (await ListDirectory(path)) ?? [];
    return entries.map((e) => [
      e.name,
      e.isDir ? FileType.Directory : FileType.File,
    ]);
  }

  /** Origin id when path is exactly a kind:mod root, not a child. */
  private modRootOrigin(path: string): string | undefined {
    const key = normFs(path);
    return this.roots.find(
      (r) => r.kind === "mod" && r.originId && normFs(r.path) === key,
    )?.originId;
  }

  async delete(resource: Uri, _opts: IFileDeleteOptions): Promise<void> {
    const path = fsPath(resource);
    if (!this.manages(path)) this.notFound();
    this.assertWritable(path);
    const originId = this.modRootOrigin(path);
    await DeletePath(path);
    if (originId && modRootDeleted) await modRootDeleted(originId);
  }

  async rename(from: Uri, to: Uri, _opts: IFileOverwriteOptions): Promise<void> {
    const src = fsPath(from);
    const dst = fsPath(to);
    if (!this.manages(src) || !this.manages(dst)) this.notFound();
    this.assertWritable(src);
    this.assertWritable(dst);
    await RenamePath(src, dst);
  }

  async readFile(resource: Uri): Promise<Uint8Array> {
    const path = fsPath(resource);
    if (!this.manages(path)) this.notFound();
    const file = await ReadFileBase64(path);
    if (!file.exists) this.notFound();
    return bytesFromB64(file.b64 ?? "");
  }

  async writeFile(
    resource: Uri,
    content: Uint8Array,
    _opts: IFileWriteOptions,
  ): Promise<void> {
    const path = fsPath(resource);
    if (!this.manages(path)) this.notFound();
    this.assertWritable(path);
    await WriteFileBase64(path, b64FromBytes(content));
  }
}
