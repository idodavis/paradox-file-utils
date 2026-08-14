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
} from "@services/languagemodelservice";
import { useWorkspaceStore } from "../stores/workspace";

const LANGS = ["paradox", "paradox-gui", "paradox-loc", "paradox-info"];

function workspaceId(): string {
  return useWorkspaceStore().activeWorkspaceId ?? "";
}

function pathOf(doc: vscode.TextDocument): string {
  return doc.uri.fsPath;
}

function uriToVsCode(uri: string): vscode.Uri {
  if (uri.startsWith("file:")) return vscode.Uri.parse(uri);
  return vscode.Uri.file(uri);
}

function rangeToVsCode(r: {
  start: { line: number; character: number };
  end: { line: number; character: number };
}): vscode.Range {
  return new vscode.Range(
    r.start.line,
    r.start.character,
    r.end.line,
    r.end.character,
  );
}

/** Attach PMT language intelligence to the workbench (idempotent). */
export function registerLanguageClient(): vscode.Disposable {
  const subs: vscode.Disposable[] = [];

  for (const lang of LANGS) {
    subs.push(
      vscode.languages.registerHoverProvider(lang, {
        async provideHover(doc, pos) {
          const id = workspaceId();
          if (!id) return null;
          const h = await Hover(id, pathOf(doc), pos.line, pos.character);
          if (!h?.contents) return null;
          return new vscode.Hover(h.contents);
        },
      }),
    );
    subs.push(
      vscode.languages.registerCompletionItemProvider(
        lang,
        {
          async provideCompletionItems(doc, pos) {
            const id = workspaceId();
            if (!id) return [];
            const items =
              (await Complete(id, pathOf(doc), pos.line, pos.character)) ??
              [];
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
      ),
    );
    subs.push(
      vscode.languages.registerDefinitionProvider(lang, {
        async provideDefinition(doc, pos) {
          const id = workspaceId();
          if (!id) return [];
          const locs =
            (await Definition(id, pathOf(doc), pos.line, pos.character)) ??
            [];
          return locs.map(
            (l) =>
              new vscode.Location(uriToVsCode(l.uri), rangeToVsCode(l.range)),
          );
        },
      }),
    );
    subs.push(
      vscode.languages.registerReferenceProvider(lang, {
        async provideReferences(doc, pos) {
          const id = workspaceId();
          if (!id) return [];
          const locs =
            (await References(id, pathOf(doc), pos.line, pos.character)) ??
            [];
          return locs.map(
            (l) =>
              new vscode.Location(uriToVsCode(l.uri), rangeToVsCode(l.range)),
          );
        },
      }),
    );
    subs.push(
      vscode.languages.registerDocumentSymbolProvider(lang, {
        async provideDocumentSymbols(doc) {
          const syms = (await DocumentSymbols(pathOf(doc))) ?? [];
          return syms.map(
            (s) =>
              new vscode.SymbolInformation(
                s.name,
                vscode.SymbolKind.Object,
                s.containerName ?? "",
                new vscode.Location(
                  uriToVsCode(s.location.uri),
                  rangeToVsCode(s.location.range),
                ),
              ),
          );
        },
      }),
    );
  }

  subs.push(
    vscode.languages.registerWorkspaceSymbolProvider({
      async provideWorkspaceSymbols(query) {
        const id = workspaceId();
        if (!id) return [];
        const syms = (await WorkspaceSymbols(id, query)) ?? [];
        return syms.map(
          (s) =>
            new vscode.SymbolInformation(
              s.name,
              vscode.SymbolKind.Object,
              s.containerName ?? "",
              new vscode.Location(
                uriToVsCode(s.location.uri),
                rangeToVsCode(s.location.range),
              ),
            ),
        );
      },
    }),
  );

  const diag = vscode.languages.createDiagnosticCollection("pmt");
  subs.push(diag);
  const refreshDiags = async (doc: vscode.TextDocument) => {
    if (!LANGS.includes(doc.languageId)) return;
    try {
      const list = (await Diagnose(pathOf(doc))) ?? [];
      diag.set(
        doc.uri,
        list.map((d) => {
          const sev =
            d.severity === 1
              ? vscode.DiagnosticSeverity.Error
              : vscode.DiagnosticSeverity.Warning;
          return new vscode.Diagnostic(rangeToVsCode(d.range), d.message, sev);
        }),
      );
    } catch {
      /* ignore */
    }
  };
  subs.push(
    vscode.workspace.onDidOpenTextDocument((d) => void refreshDiags(d)),
  );
  subs.push(
    vscode.workspace.onDidSaveTextDocument((d) => void refreshDiags(d)),
  );

  return {
    dispose() {
      for (const s of subs) s.dispose();
    },
  };
}
