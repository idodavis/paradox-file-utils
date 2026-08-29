// Package services provides backend services for the Paradox Modding Tools application.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"paradox-modding-tools/services/internal/game"
	"paradox-modding-tools/services/internal/repos"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PatcherService manages mod version patching/update campaigns.
type PatcherService struct {
	DB           *sqlx.DB
	FileService  *FileService
	MergeService *MergeService
	repo         *repos.PatchRepository
}

// PatchRunFileStats holds merge statistics for a file.
type PatchRunFileStats struct {
	Changed   int      `json:"changed"`
	Added     int      `json:"added"`
	Conflicts int      `json:"conflicts"`
	Keys      []string `json:"keys,omitempty"`
}

// PatchRunPreview is the result of previewing a patch run.
type PatchRunPreview struct {
	RunID        string               `json:"runId"`
	Files        []repos.PatchRunFile `json:"files"`
	TotalFiles   int                  `json:"totalFiles"`
	SafeCount    int                  `json:"safeCount"`
	ReviewCount  int                  `json:"reviewCount"`
	SkippedCount int                  `json:"skippedCount"`
}

func (p *PatcherService) getRepo() *repos.PatchRepository {
	if p.repo == nil {
		p.repo = repos.NewPatchRepository(p.DB)
	}
	return p.repo
}

// StartPatchRun creates a new patch run for a mod.
func (p *PatcherService) StartPatchRun(workspaceID, modID, baselineVersion, targetVersion, baselineInstallID, targetInstallID string) (*repos.PatchRun, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("user config dir: %w", err)
	}
	stagingDir := filepath.Join(configDir, appConfigDirName, "patch_runs", id, "staging")
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, fmt.Errorf("create staging dir: %w", err)
	}

	run := &repos.PatchRun{
		ID:                id,
		WorkspaceID:       workspaceID,
		ModID:             modID,
		BaselineVersion:   baselineVersion,
		TargetVersion:     targetVersion,
		BaselineInstallID: baselineInstallID,
		TargetInstallID:   targetInstallID,
		Status:            "pending",
		StagingDir:        stagingDir,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := p.getRepo().InsertRun(run); err != nil {
		return nil, fmt.Errorf("insert patch run: %w", err)
	}
	return run, nil
}

// PreviewPatchRun scans mod vs target vanilla, classifies files, writes preview staging.
func (p *PatcherService) PreviewPatchRun(runID string) (*PatchRunPreview, error) {
	repo := p.getRepo()
	run, err := repo.GetRun(runID)
	if err != nil {
		return nil, fmt.Errorf("get patch run: %w", err)
	}

	modPath, err := repo.GetModPath(run.ModID)
	if err != nil {
		return nil, fmt.Errorf("get mod: %w", err)
	}

	targetPath, err := p.resolveTargetPath(repo, run)
	if err != nil {
		return nil, err
	}
	if targetPath == "" {
		return nil, fmt.Errorf("no target install configured")
	}

	modFiles, err := p.FileService.CollectFilesFromPath(modPath, FileCollectorFilter{Extensions: []string{".txt"}})
	if err != nil {
		return nil, fmt.Errorf("collect mod files: %w", err)
	}

	targetFiles, err := p.FileService.CollectFilesFromPath(targetPath, FileCollectorFilter{Extensions: []string{".txt"}})
	if err != nil {
		return nil, fmt.Errorf("collect target files: %w", err)
	}

	if err := repo.ClearFiles(runID); err != nil {
		return nil, fmt.Errorf("clear old files: %w", err)
	}

	preview := &PatchRunPreview{RunID: runID}

	for relPath, modFilePath := range modFiles {
		file := repos.PatchRunFile{
			ID:      uuid.New().String(),
			RunID:   runID,
			RelPath: relPath,
			Status:  "pending",
		}

		targetFilePath, hasTarget := targetFiles[relPath]
		if !hasTarget {
			file.Status = "mod_only"
			preview.SkippedCount++
		} else {
			previewPath := filepath.Join(run.StagingDir, relPath)
			if err := os.MkdirAll(filepath.Dir(previewPath), 0o755); err != nil {
				return nil, fmt.Errorf("create preview dir: %w", err)
			}

			items, err := p.MergeService.MergePreview(context.Background(), modFilePath, targetFilePath, run.StagingDir, MergerOptions{
				AddAdditionalEntries: true,
			})
			if err != nil {
				file.Status = "error"
				file.Stats = fmt.Sprintf(`{"error": %q}`, err.Error())
			} else if len(items) > 0 {
				results, err := p.MergeService.Merge(context.Background(), items, MergerOptions{
					AddAdditionalEntries: true,
					OutputDir:            run.StagingDir,
				})
				if err != nil {
					file.Status = "error"
				} else if len(results) > 0 {
					r := results[0]
					file.PreviewPath = r.OutputPath
					stats := PatchRunFileStats{
						Changed:   r.Changed,
						Added:     r.Added,
						Conflicts: len(r.ResolvedConflicts),
					}
					if stats.Conflicts > 0 || stats.Changed > 3 {
						file.Status = "review"
						preview.ReviewCount++
					} else {
						file.Status = "safe"
						preview.SafeCount++
					}
					statsJSON, _ := json.Marshal(stats)
					file.Stats = string(statsJSON)
				}
			}
		}

		if err := repo.InsertFile(&file); err != nil {
			return nil, fmt.Errorf("insert file: %w", err)
		}

		preview.Files = append(preview.Files, file)
		preview.TotalFiles++
	}

	_ = repo.UpdateRunStatus(runID, "previewed", time.Now().UTC().Format(time.RFC3339))

	return preview, nil
}

// resolveTargetPath determines the target install path for a patch run.
func (p *PatcherService) resolveTargetPath(repo *repos.PatchRepository, run *repos.PatchRun) (string, error) {
	installID := run.TargetInstallID
	if installID == "" {
		var err error
		installID, err = repo.GetWorkspaceInstallID(run.WorkspaceID)
		if err != nil {
			return "", fmt.Errorf("get workspace: %w", err)
		}
	}
	if installID == "" {
		return "", nil
	}

	instPath, gameID, err := repo.GetInstallInfo(installID)
	if err != nil {
		return "", fmt.Errorf("get target install: %w", err)
	}

	info := game.Get(gameID)
	if info == nil {
		return "", fmt.Errorf("unknown game %s", gameID)
	}
	return filepath.Join(instPath, info.ScriptRoot), nil
}

// ListPatchRuns returns all patch runs for a workspace.
func (p *PatcherService) ListPatchRuns(workspaceID string) ([]repos.PatchRun, error) {
	out, err := p.getRepo().ListRuns(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list patch runs: %w", err)
	}
	return out, nil
}

// GetPatchRun returns a patch run by ID.
func (p *PatcherService) GetPatchRun(runID string) (*repos.PatchRun, error) {
	run, err := p.getRepo().GetRun(runID)
	if err != nil {
		return nil, fmt.Errorf("get patch run: %w", err)
	}
	return run, nil
}

// GetPatchRunFiles returns all files for a patch run.
func (p *PatcherService) GetPatchRunFiles(runID string) ([]repos.PatchRunFile, error) {
	out, err := p.getRepo().ListFiles(runID)
	if err != nil {
		return nil, fmt.Errorf("get patch run files: %w", err)
	}
	return out, nil
}

// SetFileDecision sets the user decision for a patch run file.
func (p *PatcherService) SetFileDecision(fileID, decision string) error {
	if err := p.getRepo().SetFileDecision(fileID, decision); err != nil {
		return fmt.Errorf("set decision: %w", err)
	}
	return nil
}

// ApplyPatchRun applies accepted changes from a patch run to the mod.
func (p *PatcherService) ApplyPatchRun(runID string) error {
	repo := p.getRepo()
	run, err := repo.GetRun(runID)
	if err != nil {
		return fmt.Errorf("get patch run: %w", err)
	}

	modPath, err := repo.GetModPath(run.ModID)
	if err != nil {
		return fmt.Errorf("get mod: %w", err)
	}

	files, err := repo.ListFiles(runID)
	if err != nil {
		return fmt.Errorf("get patch run files: %w", err)
	}

	for _, f := range files {
		if f.Decision != "accept" || f.PreviewPath == "" {
			continue
		}
		targetPath := filepath.Join(modPath, f.RelPath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("create dir for %s: %w", f.RelPath, err)
		}
		content, err := os.ReadFile(f.PreviewPath)
		if err != nil {
			return fmt.Errorf("read preview %s: %w", f.RelPath, err)
		}
		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", f.RelPath, err)
		}
		_ = repo.SetFileStatus(f.ID, "applied")
	}

	_ = repo.UpdateRunStatus(runID, "applied", time.Now().UTC().Format(time.RFC3339))
	return nil
}
