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
  SignatureHelp,
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
import { originHexByOriginId } from "./rootDecorations";

const LANGS = ["paradox", "paradox-gui", "paradox-loc", "paradox-info", "paradox-mod"];

function pathOf(doc: vscode.TextDocument): string {
  let p = doc.uri.fsPath || doc.uri.path;
  if (p.startsWith("/") && /^\/[A-Za-z]:/.test(p)) p = p.slice(1);
  return p;
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

/** Wrap LSP documentation as trusted markdown. */
function withDoc(text: string): vscode.MarkdownString {
  const md = new vscode.MarkdownString(text, true);
  md.supportHtml = true;
  return md;
}

/** Kind (muted) and key (bold) on one line, then hint and docs. */
function hoverCardMarkdown(h: HoverResult): string {
  const kind = h.kind ? `<span class="pmt-hover-kind">${escHtml(h.kind)}</span>` : "";
  const key = h.key ? ` **\`${escHtml(h.key)}\`**` : "";
  const parts = [`${kind}${key}`.trim()];
  if (h.hint) parts.push(escHtml(h.hint));
  if (h.docs) parts.push(escHtml(h.docs));
  return parts.join("\n\n");
}

/** Location line from Go origin + rel (1-based line). */
function hoverSiteMarkdown(h: HoverResult): string {
  const primary = hoverSiteLine(
    h.origin || "",
    h.originName || "",
    h.rel ?? "",
    h.path ?? "",
    h.line,
    h.col,
  );
  const vanilla = h.vanillaPath
    ? hoverSiteLine(
        "vanilla",
        h.vanillaOriginName || "",
        h.vanillaRel ?? "",
        h.vanillaPath,
        h.vanillaLine,
        h.vanillaCol,
      )
    : "";
  const tag = h.vanillaPath ? hoverOverrideTag() : "";
  const parts = [primary, vanilla, tag].filter(Boolean);
  return parts.join("<br>");
}

/** Footer chip when a mod def overlays vanilla. */
function hoverOverrideTag(): string {
  const raw = originHexByOriginId("vanilla");
  const hex = /^#[0-9A-Fa-f]{6}$/.test(raw) ? raw : "#5B9A8B";
  return `<span style="color:${hex};">game override</span>`;
}

/** Colored origin + open-file link for one hover site. */
function hoverSiteLine(
  origin: string,
  originName: string,
  rel: string,
  path: string,
  line?: number,
  col?: number,
): string {
  const label = escHtml(originName);
  if (!label && !rel && !path) return "";
  const key = origin || "vanilla";
  const raw = originHexByOriginId(key);
  const hex = /^#[0-9A-Fa-f]{6}$/.test(raw) ? raw : "#5B9A8B";
  const icon = key === "vanilla" || key === "staging"
    ? "$(root-folder)"
    : "$(package)";
  const name = label || escHtml(key === "vanilla" ? "Game" : key);
  const site = `<span style="color:${hex};">${icon} ${name}</span>`;
  const pathBit = hoverOpenLink(rel, path, line, col);
  return pathBit ? `${site} · ${pathBit}` : site;
}

/** Markdown command link that opens the def file. */
function hoverOpenLink(
  rel: string,
  path: string,
  line?: number,
  col?: number,
): string {
  const text = escHtml(rel || path);
  if (!text) return "";
  if (!path) return text;
  const target = vscode.Uri.file(path).toString();
  const args: unknown[] = col
    ? [target, {
        selection: {
          startLineNumber: (line ?? 0) + 1,
          startColumn: (col ?? 0) + 1,
        },
      }]
    : [target];
  return `[${rel || text}](command:vscode.open?${encodeURIComponent(JSON.stringify(args))})`;
}

function escHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

/** Loc/body as a quote; usage as a codeblock. Styles live in explorerLayout.css
 *  — monaco hover sanitizes inline style, so <pre style=…> never wraps. */
function appendHoverQuote(md: vscode.MarkdownString, text: string) {
  const t = text.replace(/^\n+/, "").replace(/\n+$/, "");
  if (!t) return;
  md.appendMarkdown(`<blockquote>${escHtml(t)}</blockquote>\n`);
}

function appendHoverCode(md: vscode.MarkdownString, text: string) {
  const t = text.replace(/^\n+/, "").replace(/\n+$/, "");
  if (!t) return;
  md.appendMarkdown(`<pre>${escHtml(t)}</pre>\n`);
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

/** Map a Go UTF-8 range onto a VS Code range (same-file converts columns). */
function goRangeToVsCode(
  r: { start: { line: number; character: number }; end: { line: number; character: number } },
  fallback?: vscode.TextDocument,
  same?: boolean,
): vscode.Range {
  if (same && fallback) return rangeToVsCode(fallback, r);
  return new vscode.Range(
    r.start.line,
    r.start.character,
    r.end.line,
    r.end.character,
  );
}

/** Map Go locations onto VS Code Locations (references / symbols). */
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
    out.push(new vscode.Location(uri, goRangeToVsCode(l.range, fallback, same)));
  }
  return out;
}

/** Map Go locations onto LocationLinks so Peek can show the assignment block. */
function locationLinksFromGo(
  locs: LspLocation[],
  fallback?: vscode.TextDocument,
): vscode.LocationLink[] {
  const out: vscode.LocationLink[] = [];
  for (const l of locs) {
    const uri = uriToVsCode(l.uri);
    const same =
      fallback &&
      uri.fsPath.replace(/\\/g, "/").toLowerCase() ===
        fallback.uri.fsPath.replace(/\\/g, "/").toLowerCase();
    const sel = goRangeToVsCode(l.range, fallback, same);
    out.push({
      targetUri: uri,
      targetRange: l.targetRange
        ? goRangeToVsCode(l.targetRange, fallback, same)
        : sel,
      targetSelectionRange: sel,
    });
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
        if (!h?.kind && !h?.key) return null;
        const md = new vscode.MarkdownString("", true);
        md.supportHtml = true;
        md.supportThemeIcons = true;
        md.isTrusted = { enabledCommands: ["vscode.open"] };
        md.appendMarkdown(hoverCardMarkdown(h));
        if (h.body) appendHoverQuote(md, h.body);
        if (h.usage) appendHoverCode(md, h.usage);
        const site = hoverSiteMarkdown(h);
        if (site) md.appendMarkdown(`\n\n---\n\n${site}`);
        return new vscode.Hover(md);
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
          dirty.delete(doc.uri.toString());
          await DidChange(id, pathOf(doc), doc.getText());
          if (token?.isCancellationRequested) return undefined;
          const p = positionUtf8(doc, pos);
          const items =
            (await Complete(id, pathOf(doc), p.line, p.character)) ?? [];
          if (token?.isCancellationRequested) return undefined;
          return items.map((it, i) => {
            const kind =
              it.kind === "property"
                ? vscode.CompletionItemKind.Property
                : vscode.CompletionItemKind.Value;
            const c = new vscode.CompletionItem(it.label, kind);
            c.detail = it.detail;
            c.sortText = String(i).padStart(4, "0");
            if (it.documentation) {
              c.documentation = withDoc(it.documentation);
            }
            if (it.insertText) {
              c.insertText = it.insertText.includes("$")
                ? new vscode.SnippetString(it.insertText)
                : it.insertText;
            }
            if (it.range) {
              c.range = rangeToVsCode(doc, it.range);
            }
            return c;
          });
        },
      },
      ".",
      "[",
      ":",
      "{",
    ),
  );
  registerLangProviders(subs, (lang) =>
    vscode.languages.registerSignatureHelpProvider(
      lang,
      {
        async provideSignatureHelp(doc, pos, token) {
          const id = getWorkspaceId();
          if (!id || token?.isCancellationRequested) return undefined;
          const p = positionUtf8(doc, pos);
          const h = await SignatureHelp(id, pathOf(doc), p.line, p.character);
          if (!h?.label || token?.isCancellationRequested) return undefined;
          const info = new vscode.SignatureInformation(h.label, h.documentation);
          for (const pName of h.parameters ?? []) {
            info.parameters.push(new vscode.ParameterInformation(pName));
          }
          info.activeParameter = h.activeParameter ?? 0;
          return {
            signatures: [info],
            activeSignature: 0,
            activeParameter: h.activeParameter ?? 0,
          };
        },
      },
      "(",
      "{",
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
        return locationLinksFromGo(locs, doc);
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
