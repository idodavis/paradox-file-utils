// Package services: SemanticsService rebuilds install semantic caches.
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"

	"paradox-modding-tools/services/internal/repos"
	"paradox-modding-tools/services/internal/semantics/scanner"

	"github.com/jmoiron/sqlx"
)

// SemanticsService rebuilds and applies per-install semantic caches.
type SemanticsService struct {
	DB *sqlx.DB
	ws *repos.WorkspaceRepository

	mu     sync.Mutex
	cancel context.CancelFunc
}

// SemanticsStatus reports whether a cached semantic model exists for an install.
type SemanticsStatus struct {
	Present   bool   `json:"present"`
	InstallID string `json:"installId"`
	GameID    string `json:"gameId,omitempty"`
	ScannedAt string `json:"scannedAt,omitempty"`
	Path      string `json:"path,omitempty"`
}

func (s *SemanticsService) getWS() *repos.WorkspaceRepository {
	if s.ws == nil {
		s.ws = repos.NewWorkspaceRepository(s.DB)
	}
	return s.ws
}

func (s *SemanticsService) beginJob() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	return ctx
}

func (s *SemanticsService) clearJob() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancel = nil
}

// CancelSemantics cancels an in-flight RebuildSemantics job.
func (s *SemanticsService) CancelSemantics() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// RebuildSemantics scans a game install and writes/applies the cache.
func (s *SemanticsService) RebuildSemantics(installID string) (*scanner.Cache, error) {
	if installID == "" {
		return nil, fmt.Errorf("install id is required")
	}
	ctx := s.beginJob()
	defer s.clearJob()

	path, gameID, err := s.getWS().GetInstallPath(installID)
	if err != nil {
		return nil, fmt.Errorf("install: %w", err)
	}
	cache, err := scanner.ScanInstallCtx(ctx, gameID, path, func(phase string, done, total int, message string) {
		emitProgress(eventSemanticsProgress, ProgressEvent{
			Job: "semantics", Phase: phase, Done: done, Total: total, Message: message,
		})
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			emitProgress(eventSemanticsProgress, ProgressEvent{
				Job: "semantics", Phase: "cancelled", Message: "Cancelled",
			})
			return nil, nil
		}
		return nil, err
	}
	if err := scanner.SaveCache(installID, cache); err != nil {
		return nil, err
	}
	if err := scanner.ApplyCache(gameID, cache); err != nil {
		return nil, err
	}
	return cache, nil
}

// RebuildWorkspaceSemantics rebuilds for the workspace's linked install.
func (s *SemanticsService) RebuildWorkspaceSemantics(workspaceID string) (*scanner.Cache, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	_, installID, err := s.getWS().GetWorkspaceInfo(workspaceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("workspace not found")
		}
		return nil, fmt.Errorf("get workspace: %w", err)
	}
	if installID == "" {
		return nil, fmt.Errorf("workspace has no game install")
	}
	return s.RebuildSemantics(installID)
}

// LoadSemanticsCache returns the saved cache for an install if present.
func (s *SemanticsService) LoadSemanticsCache(installID string) (*scanner.Cache, error) {
	return scanner.LoadCache(installID)
}

// GetSemanticsStatus returns cache presence for a workspace's linked install.
func (s *SemanticsService) GetSemanticsStatus(workspaceID string) (*SemanticsStatus, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	gameID, installID, err := s.getWS().GetWorkspaceInfo(workspaceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("workspace not found")
		}
		return nil, fmt.Errorf("get workspace: %w", err)
	}
	st := &SemanticsStatus{InstallID: installID, GameID: gameID}
	if installID == "" {
		return st, nil
	}
	path, err := scanner.CachePath(installID)
	if err != nil {
		return st, nil
	}
	st.Path = path
	cache, err := scanner.LoadCache(installID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || os.IsNotExist(err) {
			return st, nil
		}
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			return st, nil
		}
		return nil, err
	}
	st.Present = true
	st.ScannedAt = cache.ScannedAt
	st.GameID = cache.GameID
	return st, nil
}
