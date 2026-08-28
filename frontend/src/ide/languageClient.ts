/**
 * Register workbench language providers that call Go LanguageModelService.
 */
import * as vscode from "vscode";
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
  SemanticTokens,
  InlayHints,
  SignatureHelp,
  CodeActions,
} from "@services/languagemodelservice";
import type { WorkspaceEdit as LspWorkspaceEdit } from "@services/internal/lsp/models";

const LANGS = ["paradox", "paradox-gui", "paradox-loc", "paradox-info", "paradox-mod"];

const TOKEN_TYPES: string[] = [
  "comment",
  "string",
  "number",
  "property",
  "keyword",
  "variable",
  "operator",
  "type",
  "parameter",
];

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

async function openDoc(uri: string): Promise<vscode.TextDocument | undefined> {
  try {
    return await vscode.workspace.openTextDocument(uriToVsCode(uri));
  } catch {
    return undefined;
  }
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

  const changeTimers = new Map<string, ReturnType<typeof setTimeout>>();
  const tokenTimers = new Map<string, ReturnType<typeof setTimeout>>();
  const tokenAbort = new Map<string, AbortController>();
  const syncOpen = (doc: vscode.TextDocument) => {
    const id = getWorkspaceId();
    if (!id || !LANGS.includes(doc.languageId)) return;
    void DidOpen(id, pathOf(doc), doc.getText(), doc.languageId, doc.version);
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
    const key = doc.uri.toString();
    const prev = changeTimers.get(key);
    if (prev) clearTimeout(prev);
    changeTimers.set(
      key,
      setTimeout(() => {
        changeTimers.delete(key);
        void DidChange(
          id,
          pathOf(doc),
          doc.getText(),
          doc.languageId,
          doc.version,
        ).then(() => refreshDiags(doc));
      }, 200),
    );
  };

  registerLangProviders(subs, (lang) =>
    vscode.languages.registerHoverProvider(lang, {
      async provideHover(doc, pos) {
        const id = getWorkspaceId();
        if (!id) return null;
        const p = positionUtf8(doc, pos);
        const h = await Hover(id, pathOf(doc), p.line, p.character);
        if (!h?.contents) return null;
        return new vscode.Hover(h.contents);
      },
    }),
  );
  registerLangProviders(subs, (lang) =>
    vscode.languages.registerCompletionItemProvider(
      lang,
      {
        async provideCompletionItems(doc, pos) {
          const id = getWorkspaceId();
          if (!id) return [];
          const p = positionUtf8(doc, pos);
          const items =
            (await Complete(id, pathOf(doc), p.line, p.character)) ?? [];
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
      " ",
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
        const out: vscode.Location[] = [];
        for (const l of locs) {
          const target = await openDoc(l.uri);
          out.push(
            new vscode.Location(uriToVsCode(l.uri), rangeToVsCode(target ?? doc, l.range)),
          );
        }
        return out;
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
        const out: vscode.Location[] = [];
        for (const l of locs) {
          const target = await openDoc(l.uri);
          out.push(
            new vscode.Location(uriToVsCode(l.uri), rangeToVsCode(target ?? doc, l.range)),
          );
        }
        return out;
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
    vscode.languages.registerSignatureHelpProvider(
      lang,
      {
        async provideSignatureHelp(doc, pos) {
          const id = getWorkspaceId();
          if (!id) return null;
          const p = positionUtf8(doc, pos);
          const h = await SignatureHelp(
            id,
            pathOf(doc),
            p.line,
            p.character,
          );
          if (!h?.label) return null;
          const info = new vscode.SignatureInformation(
            h.label,
            h.documentation,
          );
          const help = new vscode.SignatureHelp();
          help.signatures = [info];
          help.activeSignature = 0;
          help.activeParameter = 0;
          return help;
        },
      },
      " ",
      "=",
    ),
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
    vscode.languages.registerDocumentSemanticTokensProvider(
      lang,
      {
        provideDocumentSemanticTokens(doc, token) {
          const key = doc.uri.toString();
          const prev = tokenTimers.get(key);
          if (prev) clearTimeout(prev);
          tokenAbort.get(key)?.abort();
          const ac = new AbortController();
          tokenAbort.set(key, ac);

          return new Promise<vscode.SemanticTokens>((resolve) => {
            const timer = setTimeout(async () => {
              tokenTimers.delete(key);
              const empty = () => new vscode.SemanticTokensBuilder().build();
              if (token.isCancellationRequested || ac.signal.aborted) {
                resolve(empty());
                return;
              }
              const id = getWorkspaceId();
              const builder = new vscode.SemanticTokensBuilder();
              if (!id) {
                resolve(builder.build());
                return;
              }
              try {
                const spans = (await SemanticTokens(id, pathOf(doc))) ?? [];
                if (token.isCancellationRequested || ac.signal.aborted) {
                  resolve(empty());
                  return;
                }
                for (const sp of spans) {
                  const t = TOKEN_TYPES.indexOf(sp.type);
                  if (t < 0) continue;
                  const lineText = safeLine(doc, sp.line);
                  const start16 = columnUtf16(lineText, sp.startCol);
                  const end16 = columnUtf16(lineText, sp.startCol + sp.length);
                  builder.push(sp.line, start16, end16 - start16, t, 0);
                }
                resolve(builder.build());
              } catch {
                resolve(empty());
              }
            }, 200);
            tokenTimers.set(key, timer);
            token.onCancellationRequested(() => {
              clearTimeout(timer);
              tokenTimers.delete(key);
              ac.abort();
              resolve(new vscode.SemanticTokensBuilder().build());
            });
          });
        },
      },
      new vscode.SemanticTokensLegend(TOKEN_TYPES),
    ),
  );
  registerLangProviders(subs, (lang) =>
    vscode.languages.registerInlayHintsProvider(lang, {
      async provideInlayHints(doc) {
        const id = getWorkspaceId();
        if (!id) return [];
        const hints = (await InlayHints(id, pathOf(doc))) ?? [];
        return hints.map((h) => {
          const hint = new vscode.InlayHint(
            new vscode.Position(
              h.line,
              columnUtf16(safeLine(doc, h.line), h.character),
            ),
            h.label,
            h.kind === 1
              ? vscode.InlayHintKind.Type
              : vscode.InlayHintKind.Parameter,
          );
          return hint;
        });
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
          const target = await openDoc(s.location.uri);
          const range = target
            ? rangeToVsCode(target, s.location.range)
            : new vscode.Range(
                s.location.range.start.line,
                s.location.range.start.character,
                s.location.range.end.line,
                s.location.range.end.character,
              );
          out.push(
            new vscode.SymbolInformation(
              s.name,
              vscode.SymbolKind.Object,
              s.containerName ?? "",
              new vscode.Location(uriToVsCode(s.location.uri), range),
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

  return {
    dispose() {
      for (const t of changeTimers.values()) clearTimeout(t);
      for (const t of tokenTimers.values()) clearTimeout(t);
      for (const ac of tokenAbort.values()) ac.abort();
      for (const s of subs) s.dispose();
    },
  };
}
