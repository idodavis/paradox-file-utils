// Package repos provides database repositories for the application.
package repos

import "github.com/jmoiron/sqlx"

// ScriptLogImport represents an imported script log analysis.
type ScriptLogImport struct {
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspaceId" db:"workspace_id"`
	Path        string `json:"path" db:"path"`
	ImportedAt  string `json:"importedAt" db:"imported_at"`
	Summary     string `json:"summary" db:"summary"`
}

// ScriptLogRepository handles script_log_imports database operations.
type ScriptLogRepository struct {
	db *sqlx.DB
}

// NewScriptLogRepository creates a new ScriptLogRepository.
func NewScriptLogRepository(db *sqlx.DB) *ScriptLogRepository {
	return &ScriptLogRepository{db: db}
}

// Insert adds a new script log import.
func (r *ScriptLogRepository) Insert(imp *ScriptLogImport) error {
	_, err := r.db.Exec(
		`INSERT INTO script_log_imports (id, workspace_id, path, imported_at, summary) VALUES (?, ?, ?, ?, ?)`,
		imp.ID, imp.WorkspaceID, imp.Path, imp.ImportedAt, imp.Summary,
	)
	return err
}

// List returns all imports for a workspace.
func (r *ScriptLogRepository) List(workspaceID string) ([]ScriptLogImport, error) {
	var out []ScriptLogImport
	err := r.db.Select(&out,
		`SELECT id, workspace_id, path, imported_at, summary FROM script_log_imports WHERE workspace_id = ? ORDER BY imported_at DESC`,
		workspaceID,
	)
	return out, err
}

// Get returns an import by ID.
func (r *ScriptLogRepository) Get(id string) (*ScriptLogImport, error) {
	var imp ScriptLogImport
	err := r.db.Get(&imp, `SELECT id, workspace_id, path, imported_at, summary FROM script_log_imports WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &imp, nil
}
