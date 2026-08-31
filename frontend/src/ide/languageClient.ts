/**
 * Register workbench language providers that call Go IdeService.
 */
import * as vscode from "vscode";
import { useDebounceFn } from "@vueuse/core";
import { Events } from "@wailsio/runtime";
import {
  Diagnose,
  Hover,
  Complete,
  Definition,
  References,
  DocumentSymbols,
  WorkspaceSymbols,
  DidOpen,
  DidChange,
  DidClose,
  DidSave,
  Rename,
  FormatDocument,
  FoldingRanges,
  CodeActions,
} from "@services/ideservice";
import type { WorkspaceEdit as LspWorkspaceEdit, Location as LspLocation, HoverResult } from "@services/internal/lsp/models";

const LANGS = ["paradox", "paradox-gui", "paradox-loc", "paradox-info", "paradox-mod"];

function pathOf(doc: vscode.TextDocument): string {
  return doc.uri.fsPath;
}

/** Map editor position to UTF-8 byte column for the Go LSP bridge. */
function positionUtf8(
  doc: vscode.TextDocument,
  pos: vscode.Position,
): { line: number; character: number } {
  const lineText = doc.lineAt(pos.line).text;
  let utf16 = 0;
  let byteCol = 0;
  for (let i = 0; i < lineText.length && utf16 < pos.character; ) {
    const cp = lineText.codePointAt(i)!;
    utf16 += cp > 0xffff ? 2 : 1;
    byteCol += cp <= 0x7f ? 1 : cp <= 0x7ff ? 2 : cp <= 0xffff ? 3 : 4;
    i += cp > 0xffff ? 2 : 1;
  }
  return { line: pos.line, character: byteCol };
}

/** Map a UTF-8 byte column on a line to a UTF-16 code-unit column (Monaco). */
function columnUtf16(lineText: string, byteCol: number): number {
  let utf16 = 0;
  let byte = 0;
  for (let i = 0; i < lineText.length && byte < byteCol; ) {
    const cp = lineText.codePointAt(i)!;
    byte += cp <= 0x7f ? 1 : cp <= 0x7ff ? 2 : cp <= 0xffff ? 3 : 4;
    utf16 += cp > 0xffff ? 2 : 1;
    i += cp > 0xffff ? 2 : 1;
  }
  return utf16;
}

function uriToVsCode(uri: string): vscode.Uri {
  if (uri.startsWith("file:")) return vscode.Uri.parse(uri);
  return vscode.Uri.file(uri);
}

function rangeToVsCode(
  doc: vscode.TextDocument,
  r: {
    start: { line: number; character: number };
    end: { line: number; character: number };
  },
): vscode.Range {
  const startLine = safeLine(doc, r.start.line);
  const endLine = safeLine(doc, r.end.line);
  return new vscode.Range(
    r.start.line,
    columnUtf16(startLine, r.start.character),
    r.end.line,
    columnUtf16(endLine, r.end.character),
  );
}

function safeLine(doc: vscode.TextDocument, line: number): string {
  if (line < 0 || line >= doc.lineCount) return "";
  return doc.lineAt(line).text;
}

/** Plaintext hover card: Go contents plus origin · rel:line. */
function hoverText(h: HoverResult): string {
  const site = hoverSiteLine(h);
  const body = h.contents ?? "";
  return site ? `${body}\n\n${site}` : body;
}

/** Location line from Go origin + rel (1-based line). */
function hoverSiteLine(h: HoverResult): string {
  const origin = h.origin ?? "";
  const rel = h.rel ?? "";
  if (!origin && !rel) return "";
  const label = origin || "vanilla";
  if (!rel) return label;
  return `${label} · ${rel}:${(h.line ?? 0) + 1}`;
}

function applyWorkspaceEdit(
  doc: vscode.TextDocument | undefined,
  edit: LspWorkspaceEdit,
): vscode.WorkspaceEdit {
  const we = new vscode.WorkspaceEdit();
  for (const uri of edit.create ?? []) {
    we.createFile(uriToVsCode(uri), { ignoreIfExists: true });
  }
  for (const [uri, edits] of Object.entries(edit.changes ?? {})) {
    const vsUri = uriToVsCode(uri);
    const sameDoc = doc && vsUri.fsPath === doc.uri.fsPath ? doc : undefined;
    for (const e of edits ?? []) {
      const range = sameDoc
        ? rangeToVsCode(sameDoc, e.range)
        : new vscode.Range(
            e.range.start.line,
            e.range.start.character,
            e.range.end.line,
            e.range.end.character,
          );
      we.replace(vsUri, range, e.newText);
    }
  }
  return we;
}

/** Map Go locations onto VS Code URIs without opening every target file. */
function locationsFromGo(
  locs: LspLocation[],
  fallback?: vscode.TextDocument,
): vscode.Location[] {
  const out: vscode.Location[] = [];
  for (const l of locs) {
    const uri = uriToVsCode(l.uri);
    const same =
      fallback &&
      uri.fsPath.replace(/\\/g, "/").toLowerCase() ===
        fallback.uri.fsPath.replace(/\\/g, "/").toLowerCase();
    out.push(
      new vscode.Location(
        uri,
        same && fallback
          ? rangeToVsCode(fallback, l.range)
          : new vscode.Range(
              l.range.start.line,
              l.range.start.character,
              l.range.end.line,
              l.range.end.character,
            ),
      ),
    );
  }
  return out;
}

/** Register the same provider factory for every PMT language id. */
function registerLangProviders(
  subs: vscode.Disposable[],
  factory: (lang: string) => vscode.Disposable,
): void {
  for (const lang of LANGS) subs.push(factory(lang));
}

/** Attach PMT language intelligence to the workbench (idempotent). */
export function registerLanguageClient(
  getWorkspaceId: () => string,
): vscode.Disposable {
  const subs: vscode.Disposable[] = [];
  const diag = vscode.languages.createDiagnosticCollection("pmt");
  subs.push(diag);

  const dirty = new Map<string, vscode.TextDocument>();
  const flushChanges = useDebounceFn(() => {
    const id = getWorkspaceId();
    const docs = [...dirty.values()];
    dirty.clear();
    if (!id) return;
    for (const doc of docs) {
      void DidChange(id, pathOf(doc), doc.getText()).then(() => refreshDiags(doc));
    }
  }, 200);
  const syncOpen = (doc: vscode.TextDocument) => {
    const id = getWorkspaceId();
    if (!id || !LANGS.includes(doc.languageId)) return;
    void DidOpen(id, pathOf(doc), doc.getText());
  };

  const refreshDiags = async (doc: vscode.TextDocument) => {
    const id = getWorkspaceId();
    if (!id || !LANGS.includes(doc.languageId)) return;
    try {
      const list = (await Diagnose(id, pathOf(doc))) ?? [];
      diag.set(
        doc.uri,
        list.map((d) => {
          const sev =
            d.severity === 1
              ? vscode.DiagnosticSeverity.Error
              : vscode.DiagnosticSeverity.Warning;
          return new vscode.Diagnostic(rangeToVsCode(doc, d.range), d.message, sev);
        }),
      );
    } catch {
      /* ignore */
    }
  };

  const onChange = (doc: vscode.TextDocument) => {
    const id = getWorkspaceId();
    if (!id || !LANGS.includes(doc.languageId)) return;
    dirty.set(doc.uri.toString(), doc);
    void flushChanges();
  };

  registerLangProviders(subs, (lang) =>
    vscode.languages.registerHoverProvider(lang, {
      async provideHover(doc, pos) {
        const id = getWorkspaceId();
        if (!id) return null;
        const p = positionUtf8(doc, pos);
        const h = await Hover(id, pathOf(doc), p.line, p.character);
        if (!h?.contents) return null;
        return new vscode.Hover(hoverText(h));
      },
    }),
  );
  registerLangProviders(subs, (lang) =>
    vscode.languages.registerCompletionItemProvider(
      lang,
      {
        async provideCompletionItems(doc, pos, token) {
          const id = getWorkspaceId();
          if (!id || token?.isCancellationRequested) return undefined;
          const p = positionUtf8(doc, pos);
          const items =
            (await Complete(id, pathOf(doc), p.line, p.character)) ?? [];
          if (token?.isCancellationRequested) return undefined;
          return items.map((it) => {
            const c = new vscode.CompletionItem(
              it.label,
              vscode.CompletionItemKind.Value,
            );
            c.detail = it.detail;
            return c;
          });
        },
      },
      ".",
      "[",
      ":",
    ),
  );
  registerLangProviders(subs, (lang) =>
    vscode.languages.registerDefinitionProvider(lang, {
      async provideDefinition(doc, pos) {
        const id = getWorkspaceId();
        if (!id) return [];
        const p = positionUtf8(doc, pos);
        const locs =
          (await Definition(id, pathOf(doc), p.line, p.character)) ?? [];
        return locationsFromGo(locs, doc);
      },
    }),
  );
  registerLangProviders(subs, (lang) =>
    vscode.languages.registerReferenceProvider(lang, {
      async provideReferences(doc, pos) {
        const id = getWorkspaceId();
        if (!id) return [];
        const p = positionUtf8(doc, pos);
        const locs =
          (await References(id, pathOf(doc), p.line, p.character)) ?? [];
        return locationsFromGo(locs, doc);
      },
    }),
  );
  registerLangProviders(subs, (lang) =>
    vscode.languages.registerDocumentSymbolProvider(lang, {
      async provideDocumentSymbols(doc) {
        const id = getWorkspaceId();
        if (!id) return [];
        const syms = (await DocumentSymbols(id, pathOf(doc))) ?? [];
        return syms.map(
          (s) =>
            new vscode.SymbolInformation(
              s.name,
              vscode.SymbolKind.Object,
              s.containerName ?? "",
              new vscode.Location(
                uriToVsCode(s.location.uri),
                rangeToVsCode(doc, s.location.range),
              ),
            ),
        );
      },
    }),
  );
  registerLangProviders(subs, (lang) =>
    vscode.languages.registerRenameProvider(lang, {
      async provideRenameEdits(doc, pos, newName) {
        const id = getWorkspaceId();
        if (!id) return null;
        const p = positionUtf8(doc, pos);
        const edit = await Rename(
          id,
          pathOf(doc),
          p.line,
          p.character,
          newName,
        );
        if (!edit?.changes) return null;
        return applyWorkspaceEdit(doc, edit);
      },
    }),
  );
  registerLangProviders(subs, (lang) =>
    vscode.languages.registerDocumentFormattingEditProvider(lang, {
      async provideDocumentFormattingEdits(doc) {
        const id = getWorkspaceId();
        if (!id) return [];
        const edits = (await FormatDocument(id, pathOf(doc))) ?? [];
        return edits.map(
          (e) => new vscode.TextEdit(rangeToVsCode(doc, e.range), e.newText),
        );
      },
    }),
  );
  registerLangProviders(subs, (lang) =>
    vscode.languages.registerFoldingRangeProvider(lang, {
      async provideFoldingRanges(doc) {
        const id = getWorkspaceId();
        if (!id) return [];
        const ranges = (await FoldingRanges(id, pathOf(doc))) ?? [];
        return ranges.map(
          (r) =>
            new vscode.FoldingRange(
              r.startLine,
              r.endLine,
              vscode.FoldingRangeKind.Region,
            ),
        );
      },
    }),
  );
  registerLangProviders(subs, (lang) =>
    vscode.languages.registerCodeActionsProvider(lang, {
      async provideCodeActions(doc) {
        const id = getWorkspaceId();
        if (!id) return [];
        const actions = (await CodeActions(id, pathOf(doc))) ?? [];
        return actions.map((a) => {
          const ca = new vscode.CodeAction(
            a.title,
            vscode.CodeActionKind.QuickFix,
          );
          if (a.edit) ca.edit = applyWorkspaceEdit(doc, a.edit);
          return ca;
        });
      },
    }),
  );

  subs.push(
    vscode.languages.registerWorkspaceSymbolProvider({
      async provideWorkspaceSymbols(query) {
        const id = getWorkspaceId();
        if (!id) return [];
        const syms = (await WorkspaceSymbols(id, query)) ?? [];
        const out: vscode.SymbolInformation[] = [];
        for (const s of syms) {
          const r = s.location.range;
          out.push(
            new vscode.SymbolInformation(
              s.name,
              vscode.SymbolKind.Object,
              s.containerName ?? "",
              new vscode.Location(
                uriToVsCode(s.location.uri),
                new vscode.Range(
                  r.start.line,
                  r.start.character,
                  r.end.line,
                  r.end.character,
                ),
              ),
            ),
          );
        }
        return out;
      },
    }),
  );

  subs.push(
    vscode.workspace.onDidOpenTextDocument((d) => {
      syncOpen(d);
      void refreshDiags(d);
    }),
  );
  subs.push(
    vscode.workspace.onDidChangeTextDocument((e) => onChange(e.document)),
  );
  subs.push(
    vscode.workspace.onDidSaveTextDocument((d) => {
      const id = getWorkspaceId();
      if (id && LANGS.includes(d.languageId)) {
        void DidSave(id, pathOf(d)).then(() => refreshDiags(d));
        return;
      }
      void refreshDiags(d);
    }),
  );
  subs.push(
    vscode.workspace.onDidCloseTextDocument((d) => {
      const id = getWorkspaceId();
      if (id && LANGS.includes(d.languageId)) {
        void DidClose(id, pathOf(d));
      }
      diag.delete(d.uri);
    }),
  );

  for (const doc of vscode.workspace.textDocuments) {
    syncOpen(doc);
    void refreshDiags(doc);
  }

  Events.On("lang:cache-updated", () => {
    for (const doc of vscode.workspace.textDocuments) {
      void refreshDiags(doc);
    }
  });

  return {
    dispose() {
      for (const s of subs) s.dispose();
    },
  };
}
