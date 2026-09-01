// ideservice.go is the Wails RPC shell for Monaco LSP. It holds a pointer to
// SessionService and delegates to package lsp.

package services

import (
	"fmt"
	"path/filepath"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/lsp"
	"paradox-modding-tools/services/internal/session"
)

// IdeService exposes editor Did* and LSP methods over Wails.
type IdeService struct {
	Session *SessionService
}

// DidOpen syncs an editor buffer into the session.
func (s *IdeService) DidOpen(workspaceID, path, text string) error {
	return withSessionDo(s.Session, workspaceID, func(sess *session.Session) {
		sess.DidOpen(path, text)
	})
}

// DidChange syncs buffer text into the session.
func (s *IdeService) DidChange(workspaceID, path, text string) error {
	return withSessionDo(s.Session, workspaceID, func(sess *session.Session) {
		sess.DidChange(path, text)
	})
}

// DidClose drops an editor buffer overlay.
func (s *IdeService) DidClose(workspaceID, path string) (err error) {
	defer recoverErr(&err)
	if live := s.Session.pool().Get(workspaceID); live != nil {
		live.DidClose(path)
	}
	return nil
}

// DidSave reindexes path from its buffer or disk.
func (s *IdeService) DidSave(workspaceID, path string) error {
	return withSessionDo(s.Session, workspaceID, func(sess *session.Session) {
		sess.DidSave(path)
	})
}

// Diagnose returns diagnostics for a file.
func (s *IdeService) Diagnose(workspaceID, path string) ([]lsp.Diagnostic, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) []lsp.Diagnostic {
		return lsp.Diagnose(sess, path)
	})
}

// Hover returns hover info at a 0-based UTF-8 position.
func (s *IdeService) Hover(workspaceID, path string, line, character int) (*lsp.HoverResult, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) *lsp.HoverResult {
		return lsp.Hover(sess, path, line, character)
	})
}

// Complete returns completion items at a 0-based UTF-8 position.
func (s *IdeService) Complete(workspaceID, path string, line, character int) ([]lsp.CompletionItem, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) []lsp.CompletionItem {
		return lsp.Complete(sess, path, line, character)
	})
}

// Definition returns go-to-definition locations.
func (s *IdeService) Definition(workspaceID, path string, line, character int) ([]lsp.Location, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) []lsp.Location {
		return lsp.Definition(sess, path, line, character)
	})
}

// References returns reference locations.
func (s *IdeService) References(workspaceID, path string, line, character int) ([]lsp.Location, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) []lsp.Location {
		return lsp.References(sess, path, line, character)
	})
}

// Rename returns a workspace edit for the identifier at pos.
func (s *IdeService) Rename(workspaceID, path string, line, character int, newName string) (*lsp.WorkspaceEdit, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) *lsp.WorkspaceEdit {
		return lsp.Rename(sess, path, line, character, newName)
	})
}

// FormatDocument returns conservative format edits.
func (s *IdeService) FormatDocument(workspaceID, path string) ([]lsp.TextEdit, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) []lsp.TextEdit {
		return lsp.FormatDocument(sess, path)
	})
}

// FoldingRanges returns foldable block ranges.
func (s *IdeService) FoldingRanges(workspaceID, path string) ([]lsp.FoldingRange, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) []lsp.FoldingRange {
		return lsp.FoldingRanges(sess, path)
	})
}

// CodeActions returns quick-fixes for the file.
func (s *IdeService) CodeActions(workspaceID, path string) ([]lsp.CodeAction, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) []lsp.CodeAction {
		return lsp.CodeActions(sess, path)
	})
}

// DocumentSymbols returns top-level symbols in a file.
func (s *IdeService) DocumentSymbols(workspaceID, path string) ([]lsp.SymbolInformation, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) []lsp.SymbolInformation {
		return lsp.DocumentSymbols(sess, path)
	})
}

// WorkspaceSymbols searches definitions in the session.
func (s *IdeService) WorkspaceSymbols(workspaceID, query string) ([]lsp.SymbolInformation, error) {
	return withSession(s.Session, workspaceID, func(sess *session.Session) []lsp.SymbolInformation {
		return lsp.WorkspaceSymbols(sess, query)
	})
}

func (w *WorkspaceService) getScriptRoot(installID string) (string, error) {
	var path, gameID string
	w.Store.Read(func(c *Config) {
		if inst := findInstall(c, installID); inst != nil {
			path, gameID = inst.Path, inst.GameID
		}
	})
	if path == "" {
		return "", fmt.Errorf("install not found")
	}
	info := game.Get(gameID)
	if info == nil {
		return "", fmt.Errorf("unknown game: %s", gameID)
	}
	return filepath.Join(path, info.ScriptRoot), nil
}

// IdeRoot is one folder in the workspace IDE multi-root set.
type IdeRoot struct {
	Label    string `json:"label"`
	Path     string `json:"path"`
	ReadOnly bool   `json:"readOnly"`
	Kind     string `json:"kind"`
	OriginId   string `json:"originId,omitempty"`
	Color      string `json:"color,omitempty"`
	Thumbnail  string `json:"thumbnail,omitempty"`
}

// gameFilesLabel is the explorer folder name for the install (registry Name).
func gameFilesLabel(gameID string) string {
	if g := game.Get(gameID); g != nil && g.Name != "" {
		return g.Name
	}
	return "Game"
}

// GetIdeRoots returns game / mod / staging folders for the workspace IDE.
func (w *WorkspaceService) GetIdeRoots(workspaceID string) ([]IdeRoot, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	ws, err := w.GetWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}
	var roots []IdeRoot
	mods := append([]WorkspaceMod(nil), ws.Mods...)
	sortWorkspaceMods(mods)
	for _, mod := range mods {
		if mod.IsBroken || mod.Path == "" {
			continue
		}
		label := mod.Name
		if label == "" {
			label = "Mod"
		}
		roots = append(roots, IdeRoot{
			Label: label, Path: mod.Path, Kind: "mod",
			OriginId: mod.ID, Color: mod.Color, Thumbnail: mod.Thumbnail,
		})
	}
	if ws.StagingDir != "" {
		roots = append(roots, IdeRoot{
			Label: "Staging", Path: ws.StagingDir, Kind: "staging", OriginId: "staging",
		})
	}
	if root, e := w.getScriptRoot(ws.InstallID); e == nil && root != "" {
		roots = append(roots, IdeRoot{
			Label: gameFilesLabel(ws.GameID), Path: root,
			ReadOnly: true, Kind: "game", OriginId: game.OriginVanilla,
		})
	}
	return roots, nil
}
