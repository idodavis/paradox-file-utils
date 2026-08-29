// Package repos provides database repositories for the application.
package repos

import "github.com/jmoiron/sqlx"

// PatchRun represents a mod update campaign from baseline to target version.
type PatchRun struct {
	ID                string `json:"id" db:"id"`
	WorkspaceID       string `json:"workspaceId" db:"workspace_id"`
	ModID             string `json:"modId" db:"mod_id"`
	BaselineVersion   string `json:"baselineVersion" db:"baseline_version"`
	TargetVersion     string `json:"targetVersion" db:"target_version"`
	BaselineInstallID string `json:"baselineInstallId" db:"baseline_install_id"`
	TargetInstallID   string `json:"targetInstallId" db:"target_install_id"`
	Status            string `json:"status" db:"status"`
	StagingDir        string `json:"stagingDir" db:"staging_dir"`
	CreatedAt         string `json:"createdAt" db:"created_at"`
	UpdatedAt         string `json:"updatedAt" db:"updated_at"`
}

// PatchRunFile tracks per-file status within a patch run.
type PatchRunFile struct {
	ID          string `json:"id" db:"id"`
	RunID       string `json:"runId" db:"run_id"`
	RelPath     string `json:"relPath" db:"rel_path"`
	Status      string `json:"status" db:"status"`
	Decision    string `json:"decision" db:"decision"`
	PreviewPath string `json:"previewPath" db:"preview_path"`
	Stats       string `json:"stats" db:"stats"`
}

// PatchRepository handles patch_runs and patch_run_files database operations.
type PatchRepository struct {
	db *sqlx.DB
}

// NewPatchRepository creates a new PatchRepository.
func NewPatchRepository(db *sqlx.DB) *PatchRepository {
	return &PatchRepository{db: db}
}

// InsertRun creates a new patch run.
func (r *PatchRepository) InsertRun(run *PatchRun) error {
	_, err := r.db.Exec(
		`INSERT INTO patch_runs (id, workspace_id, mod_id, baseline_version, target_version, baseline_install_id, target_install_id, status, staging_dir, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.WorkspaceID, run.ModID, run.BaselineVersion, run.TargetVersion,
		nullIfEmpty(run.BaselineInstallID), nullIfEmpty(run.TargetInstallID),
		run.Status, run.StagingDir, run.CreatedAt, run.UpdatedAt,
	)
	return err
}

// GetRun returns a patch run by ID.
func (r *PatchRepository) GetRun(id string) (*PatchRun, error) {
	var run PatchRun
	err := r.db.Get(&run,
		`SELECT id, workspace_id, mod_id, baseline_version, target_version,
		 COALESCE(baseline_install_id, '') as baseline_install_id,
		 COALESCE(target_install_id, '') as target_install_id,
		 status, COALESCE(staging_dir, '') as staging_dir, created_at, updated_at
		 FROM patch_runs WHERE id = ?`,
		id,
	)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// ListRuns returns all patch runs for a workspace.
func (r *PatchRepository) ListRuns(workspaceID string) ([]PatchRun, error) {
	var out []PatchRun
	err := r.db.Select(&out,
		`SELECT id, workspace_id, mod_id, baseline_version, target_version,
		 COALESCE(baseline_install_id, '') as baseline_install_id,
		 COALESCE(target_install_id, '') as target_install_id,
		 status, COALESCE(staging_dir, '') as staging_dir, created_at, updated_at
		 FROM patch_runs WHERE workspace_id = ? ORDER BY created_at DESC`,
		workspaceID,
	)
	return out, err
}

// UpdateRunStatus updates the status and updated_at for a run.
func (r *PatchRepository) UpdateRunStatus(id, status, updatedAt string) error {
	_, err := r.db.Exec(`UPDATE patch_runs SET status = ?, updated_at = ? WHERE id = ?`, status, updatedAt, id)
	return err
}

// ClearFiles deletes all files for a run.
func (r *PatchRepository) ClearFiles(runID string) error {
	_, err := r.db.Exec(`DELETE FROM patch_run_files WHERE run_id = ?`, runID)
	return err
}

// InsertFile adds a file to a patch run.
func (r *PatchRepository) InsertFile(f *PatchRunFile) error {
	_, err := r.db.Exec(
		`INSERT INTO patch_run_files (id, run_id, rel_path, status, decision, preview_path, stats) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.RunID, f.RelPath, f.Status, f.Decision, f.PreviewPath, f.Stats,
	)
	return err
}

// ListFiles returns all files for a patch run.
func (r *PatchRepository) ListFiles(runID string) ([]PatchRunFile, error) {
	var out []PatchRunFile
	err := r.db.Select(&out,
		`SELECT id, run_id, rel_path, status, COALESCE(decision, '') as decision, COALESCE(preview_path, '') as preview_path, COALESCE(stats, '') as stats
		 FROM patch_run_files WHERE run_id = ? ORDER BY rel_path`,
		runID,
	)
	return out, err
}

// SetFileDecision updates the decision for a file.
func (r *PatchRepository) SetFileDecision(fileID, decision string) error {
	_, err := r.db.Exec(`UPDATE patch_run_files SET decision = ? WHERE id = ?`, decision, fileID)
	return err
}

// SetFileStatus updates the status for a file.
func (r *PatchRepository) SetFileStatus(fileID, status string) error {
	_, err := r.db.Exec(`UPDATE patch_run_files SET status = ? WHERE id = ?`, status, fileID)
	return err
}

// GetModPath returns the path for a mod by ID.
func (r *PatchRepository) GetModPath(modID string) (string, error) {
	var path string
	err := r.db.Get(&path, `SELECT path FROM workspace_mods WHERE id = ?`, modID)
	return path, err
}

// GetWorkspaceInstallID returns the install_id for a workspace.
func (r *PatchRepository) GetWorkspaceInstallID(workspaceID string) (string, error) {
	var installID string
	err := r.db.Get(&installID, `SELECT COALESCE(install_id, '') FROM workspaces WHERE id = ?`, workspaceID)
	return installID, err
}

// GetInstallInfo returns path and game_id for an install.
func (r *PatchRepository) GetInstallInfo(installID string) (path, gameID string, err error) {
	var inst struct {
		Path   string `db:"path"`
		GameID string `db:"game_id"`
	}
	err = r.db.Get(&inst, `SELECT path, game_id FROM game_installs WHERE id = ?`, installID)
	return inst.Path, inst.GameID, err
}
