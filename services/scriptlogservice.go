// Package services provides backend services for the Paradox Modding Tools application.
package services

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"time"

	"paradox-modding-tools/services/internal/repos"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ScriptLogService handles game error.log imports and analysis.
type ScriptLogService struct {
	DB   *sqlx.DB
	repo *repos.ScriptLogRepository
}

// ScriptLogSummary holds parsed log statistics.
type ScriptLogSummary struct {
	ErrorCount   int `json:"errorCount"`
	WarningCount int `json:"warningCount"`
	TotalLines   int `json:"totalLines"`
}

func (s *ScriptLogService) getRepo() *repos.ScriptLogRepository {
	if s.repo == nil {
		s.repo = repos.NewScriptLogRepository(s.DB)
	}
	return s.repo
}

// ImportScriptLog reads a game script log (error.log), extracts summary, stores in DB.
func (s *ScriptLogService) ImportScriptLog(workspaceID, path string) (*repos.ScriptLogImport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read log: %w", err)
	}

	summary := s.analyzeLog(string(data))
	summaryJSON, _ := json.Marshal(summary)

	imp := &repos.ScriptLogImport{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		Path:        path,
		ImportedAt:  time.Now().UTC().Format(time.RFC3339),
		Summary:     string(summaryJSON),
	}
	if err := s.getRepo().Insert(imp); err != nil {
		return nil, fmt.Errorf("insert import: %w", err)
	}
	return imp, nil
}

// analyzeLog counts errors and warnings using simple regex patterns.
func (s *ScriptLogService) analyzeLog(content string) ScriptLogSummary {
	errorRe := regexp.MustCompile(`(?i)\[error\]|\berror\b:`)
	warningRe := regexp.MustCompile(`(?i)\[warn(ing)?\]|\bwarn(ing)?\b:`)
	lineRe := regexp.MustCompile(`\r?\n`)

	lines := lineRe.Split(content, -1)

	return ScriptLogSummary{
		ErrorCount:   len(errorRe.FindAllString(content, -1)),
		WarningCount: len(warningRe.FindAllString(content, -1)),
		TotalLines:   len(lines),
	}
}

// ListScriptLogImports returns all imports for a workspace.
func (s *ScriptLogService) ListScriptLogImports(workspaceID string) ([]repos.ScriptLogImport, error) {
	out, err := s.getRepo().List(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list imports: %w", err)
	}
	return out, nil
}

// GetScriptLogImport returns a specific import by ID.
func (s *ScriptLogService) GetScriptLogImport(importID string) (*repos.ScriptLogImport, error) {
	imp, err := s.getRepo().Get(importID)
	if err != nil {
		return nil, fmt.Errorf("get import: %w", err)
	}
	return imp, nil
}
