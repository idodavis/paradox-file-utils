// Package repos provides database repositories for the application.
package repos

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// IndexObject represents an indexed definition from scripts.
type IndexObject struct {
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspaceId" db:"workspace_id"`
	ObjType     string `json:"objType" db:"obj_type"`
	ObjKey      string `json:"objKey" db:"obj_key"`
	FilePath    string `json:"filePath" db:"file_path"`
	Line        int    `json:"line" db:"line"`
	Summary     string `json:"summary" db:"summary"`
}

// IndexEdge represents a relationship between indexed objects.
type IndexEdge struct {
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspaceId" db:"workspace_id"`
	FromKey     string `json:"fromKey" db:"from_key"`
	ToKey       string `json:"toKey" db:"to_key"`
	EdgeType    string `json:"edgeType" db:"edge_type"`
}

// IndexRepository handles index_objects and index_edges database operations.
type IndexRepository struct {
	db *sqlx.DB
}

// NewIndexRepository creates a new IndexRepository.
func NewIndexRepository(db *sqlx.DB) *IndexRepository {
	return &IndexRepository{db: db}
}

// ClearObjects deletes all objects for a workspace.
func (r *IndexRepository) ClearObjects(workspaceID string) error {
	_, err := r.db.Exec(`DELETE FROM index_objects WHERE workspace_id = ?`, workspaceID)
	return err
}

// ClearEdges deletes all edges for a workspace.
func (r *IndexRepository) ClearEdges(workspaceID string) error {
	_, err := r.db.Exec(`DELETE FROM index_edges WHERE workspace_id = ?`, workspaceID)
	return err
}

// UpsertObject inserts or updates an index object.
func (r *IndexRepository) UpsertObject(obj *IndexObject) error {
	_, err := r.db.Exec(
		`INSERT INTO index_objects (id, workspace_id, obj_type, obj_key, file_path, line, summary) VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(workspace_id, obj_type, obj_key) DO UPDATE SET
		 file_path=excluded.file_path, line=excluded.line, summary=excluded.summary`,
		obj.ID, obj.WorkspaceID, obj.ObjType, obj.ObjKey, obj.FilePath, obj.Line, obj.Summary,
	)
	return err
}

// UpsertObjects inserts or updates many index objects in one transaction.
func (r *IndexRepository) UpsertObjects(objs []IndexObject) error {
	if len(objs) == 0 {
		return nil
	}
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Preparex(
		`INSERT INTO index_objects (id, workspace_id, obj_type, obj_key, file_path, line, summary)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(workspace_id, obj_type, obj_key) DO UPDATE SET
		 file_path=excluded.file_path, line=excluded.line, summary=excluded.summary`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i := range objs {
		o := &objs[i]
		if _, err := stmt.Exec(
			o.ID, o.WorkspaceID, o.ObjType, o.ObjKey, o.FilePath, o.Line, o.Summary,
		); err != nil {
			return fmt.Errorf("upsert object: %w", err)
		}
	}
	return tx.Commit()
}

// CountObjects returns how many indexed objects exist for a workspace.
func (r *IndexRepository) CountObjects(workspaceID string) (int, error) {
	var n int
	err := r.db.Get(&n, `SELECT COUNT(*) FROM index_objects WHERE workspace_id = ?`, workspaceID)
	return n, err
}

// InsertEdge adds an edge between objects.
func (r *IndexRepository) InsertEdge(edge *IndexEdge) error {
	_, err := r.db.Exec(
		`INSERT INTO index_edges (id, workspace_id, from_key, to_key, edge_type) VALUES (?, ?, ?, ?, ?)`,
		edge.ID, edge.WorkspaceID, edge.FromKey, edge.ToKey, edge.EdgeType,
	)
	return err
}

// Search searches indexed objects by key or summary pattern.
func (r *IndexRepository) Search(workspaceID, query string) ([]IndexObject, error) {
	var out []IndexObject
	pattern := "%" + query + "%"
	err := r.db.Select(&out,
		`SELECT id, workspace_id, obj_type, obj_key, file_path, line, COALESCE(summary, '') as summary
		 FROM index_objects WHERE workspace_id = ? AND (obj_key LIKE ? OR summary LIKE ?) ORDER BY obj_type, obj_key LIMIT 100`,
		workspaceID, pattern, pattern,
	)
	return out, err
}

// ListObjects returns all objects for a workspace, optionally filtered by type.
func (r *IndexRepository) ListObjects(workspaceID, typeFilter string) ([]IndexObject, error) {
	var out []IndexObject
	query := `SELECT id, workspace_id, obj_type, obj_key, file_path, line, COALESCE(summary, '') as summary FROM index_objects WHERE workspace_id = ?`
	args := []interface{}{workspaceID}
	if typeFilter != "" {
		query += ` AND obj_type = ?`
		args = append(args, typeFilter)
	}
	query += ` ORDER BY obj_type, obj_key`
	err := r.db.Select(&out, query, args...)
	return out, err
}

// ListEventObjects returns all event objects for a workspace.
func (r *IndexRepository) ListEventObjects(workspaceID string) ([]IndexObject, error) {
	var out []IndexObject
	err := r.db.Select(&out, `SELECT id, obj_type, obj_key, file_path FROM index_objects WHERE workspace_id = ? AND obj_type = 'events'`, workspaceID)
	return out, err
}

// ListEdges returns all edges for a workspace.
func (r *IndexRepository) ListEdges(workspaceID string) ([]IndexEdge, error) {
	var out []IndexEdge
	err := r.db.Select(&out, `SELECT from_key, to_key, edge_type FROM index_edges WHERE workspace_id = ?`, workspaceID)
	return out, err
}

// GetWorkspaceInfo returns game_id and install_id for a workspace.
func (r *IndexRepository) GetWorkspaceInfo(workspaceID string) (gameID, installID string, err error) {
	var ws struct {
		GameID    string `db:"game_id"`
		InstallID string `db:"install_id"`
	}
	err = r.db.Get(&ws, `SELECT game_id, COALESCE(install_id, '') as install_id FROM workspaces WHERE id = ?`, workspaceID)
	return ws.GameID, ws.InstallID, err
}

// GetInstallPath returns the path and game_id for an install.
func (r *IndexRepository) GetInstallPath(installID string) (path, gameID string, err error) {
	var inst struct {
		Path   string `db:"path"`
		GameID string `db:"game_id"`
	}
	err = r.db.Get(&inst, `SELECT path, game_id FROM game_installs WHERE id = ?`, installID)
	return inst.Path, inst.GameID, err
}

// ListModPaths returns all mod paths for a workspace.
func (r *IndexRepository) ListModPaths(workspaceID string) ([]string, error) {
	var paths []string
	err := r.db.Select(&paths, `SELECT path FROM workspace_mods WHERE workspace_id = ?`, workspaceID)
	return paths, err
}
