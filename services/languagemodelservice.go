// Package services: LanguageModelService rebuilds and queries workspace language models.
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/langmodel"
	"paradox-modding-tools/services/internal/lsp"
	"paradox-modding-tools/services/internal/repos"
	"paradox-modding-tools/services/internal/semantics/scanner"

	"github.com/jmoiron/sqlx"
)

// LanguageModelService builds JSON workspace models and serves LSP-style queries.
type LanguageModelService struct {
	DB *sqlx.DB
	ws *repos.WorkspaceRepository

	mu     sync.Mutex
	cancel context.CancelFunc
}

// ModelStatus reports whether a cached language model exists.
type ModelStatus = langmodel.Status

func (s *LanguageModelService) getWS() *repos.WorkspaceRepository {
	if s.ws == nil {
		s.ws = repos.NewWorkspaceRepository(s.DB)
	}
	return s.ws
}

// beginJob cancels any prior job and returns a child of parent (Wails call ctx).
func (s *LanguageModelService) beginJob(parent context.Context) context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	return ctx
}

func (s *LanguageModelService) clearJob() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancel = nil
}

// Cancel cancels an in-flight RebuildWorkspaceModel job.
func (s *LanguageModelService) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// RebuildWorkspaceModel walks install/mod/staging roots and writes the JSON model.
// ctx is the Wails binding call context (cancelled when the frontend aborts the call).
func (s *LanguageModelService) RebuildWorkspaceModel(
	ctx context.Context, workspaceID string,
) (int, error) {
	if workspaceID == "" {
		return 0, fmt.Errorf("workspace id is required")
	}
	jobCtx := s.beginJob(ctx)
	defer s.clearJob()

	wsRepo := s.getWS()
	gameID, installID, err := wsRepo.GetWorkspaceInfo(workspaceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("workspace not found")
		}
		return 0, fmt.Errorf("get workspace: %w", err)
	}

	emitProgress(eventLangModelProgress, ProgressEvent{
		Job: "langmodel", Phase: "roots", WorkspaceID: workspaceID,
		Message: "Resolving roots…",
	})

	var roots []string
	cachePath := ""
	if installID != "" {
		instPath, instGameID, err := wsRepo.GetInstallPath(installID)
		if err == nil {
			if gameID == "" {
				gameID = instGameID
			}
			info := game.Get(gameID)
			if info != nil {
				roots = append(roots, filepath.Join(instPath, info.ScriptRoot))
			}
			if p, err := scanner.CachePath(installID); err == nil {
				if _, err := os.Stat(p); err == nil {
					cachePath = p
				}
			}
		}
	}

	if mods, err := wsRepo.ListModPaths(workspaceID); err == nil {
		roots = append(roots, mods...)
	}
	if ws, err := wsRepo.GetWorkspace(workspaceID); err == nil && ws.StagingDir != "" {
		if st, err := os.Stat(ws.StagingDir); err == nil && st.IsDir() {
			roots = append(roots, ws.StagingDir)
		}
	}

	if err := jobCtx.Err(); err != nil {
		return s.cancelled(workspaceID)
	}

	m, err := langmodel.Build(
		jobCtx, workspaceID, cachePath, roots,
		func(phase string, done, total int, message string) {
			emitProgress(eventLangModelProgress, ProgressEvent{
				Job: "langmodel", Phase: phase, Done: done, Total: total,
				WorkspaceID: workspaceID, Message: message,
			})
		},
	)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(jobCtx.Err(), context.Canceled) {
			return s.cancelled(workspaceID)
		}
		return 0, err
	}
	if err := langmodel.Save(m); err != nil {
		return 0, fmt.Errorf("save model: %w", err)
	}
	n := len(m.Definitions)
	emitProgress(eventLangModelProgress, ProgressEvent{
		Job: "langmodel", Phase: "done", Done: n, Total: n, WorkspaceID: workspaceID,
		Message: fmt.Sprintf("Built %d definitions", n), Percent: 100,
	})
	return n, nil
}

func (s *LanguageModelService) cancelled(workspaceID string) (int, error) {
	emitProgress(eventLangModelProgress, ProgressEvent{
		Job: "langmodel", Phase: "cancelled", WorkspaceID: workspaceID,
		Message: "Cancelled",
	})
	return 0, fmt.Errorf("cancelled")
}

// GetModelStatus returns cache presence for a workspace.
func (s *LanguageModelService) GetModelStatus(workspaceID string) (*ModelStatus, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	return langmodel.GetStatus(workspaceID)
}

func (s *LanguageModelService) loadModel(workspaceID string) (*langmodel.Model, error) {
	m, err := langmodel.Load(workspaceID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		if _, statErr := os.Stat(mustModelPath(workspaceID)); os.IsNotExist(statErr) {
			return nil, nil
		}
		return nil, err
	}
	return m, nil
}

func mustModelPath(workspaceID string) string {
	p, err := langmodel.ModelPath(workspaceID)
	if err != nil {
		return ""
	}
	return p
}

// Diagnose returns parse diagnostics for a script file.
func (s *LanguageModelService) Diagnose(path string) []lsp.Diagnostic {
	return lsp.Diagnose(path)
}

// Hover returns hover info at a 0-based LSP position using the workspace model.
func (s *LanguageModelService) Hover(
	workspaceID, path string, line, character int,
) (*lsp.HoverResult, error) {
	m, err := s.loadModel(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.Hover(m, path, line, character), nil
}

// Complete returns completion items at a 0-based position.
func (s *LanguageModelService) Complete(
	workspaceID, path string, line, character int,
) ([]lsp.CompletionItem, error) {
	m, err := s.loadModel(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.Complete(m, path, line, character), nil
}

// Definition returns go-to-definition locations.
func (s *LanguageModelService) Definition(
	workspaceID, path string, line, character int,
) ([]lsp.Location, error) {
	m, err := s.loadModel(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.Definition(m, path, line, character), nil
}

// References returns reference locations.
func (s *LanguageModelService) References(
	workspaceID, path string, line, character int,
) ([]lsp.Location, error) {
	m, err := s.loadModel(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.References(m, path, line, character), nil
}

// DocumentSymbols returns top-level symbols in a file.
func (s *LanguageModelService) DocumentSymbols(path string) []lsp.SymbolInformation {
	return lsp.DocumentSymbols(path)
}

// WorkspaceSymbols searches definitions in the workspace model.
func (s *LanguageModelService) WorkspaceSymbols(
	workspaceID, query string,
) ([]lsp.SymbolInformation, error) {
	m, err := s.loadModel(workspaceID)
	if err != nil {
		return nil, err
	}
	return lsp.WorkspaceSymbols(m, query), nil
}
