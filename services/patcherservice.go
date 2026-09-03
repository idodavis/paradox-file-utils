// Package services provides backend services for the Paradox Modding Tools application.
package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// PatcherService manages mod version patching.
type PatcherService struct {
	Store        *Store
	FileService  *FileService
	MergeService *MergeService
}

// PatchRunPreview is the result of previewing a patch run.
type PatchRunPreview struct {
	RunID        string         `json:"runId"`
	Files        []PatchRunFile `json:"files"`
	TotalFiles   int            `json:"totalFiles"`
	SafeCount    int            `json:"safeCount"`
	ReviewCount  int            `json:"reviewCount"`
	SkippedCount int            `json:"skippedCount"`
}

// StartPatchRun creates a new patch run for a mod.
func (p *PatcherService) StartPatchRun(
	workspaceID, modID, targetVersion, targetInstallID string,
) (*PatchRun, error) {
	id, now := uuid.New().String(), nowUTC()
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("user config dir: %w", err)
	}
	stagingDir := filepath.Join(configDir, appConfigDirName, "patch_runs", id, "staging")
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return nil, fmt.Errorf("create staging dir: %w", err)
	}
	run := PatchRun{
		ID: id, WorkspaceID: workspaceID, ModID: modID,
		TargetVersion: targetVersion, TargetInstallID: targetInstallID,
		Status: "pending", StagingDir: stagingDir, CreatedAt: now, UpdatedAt: now,
	}
	err = p.Store.Mutate(func(c *Config) error {
		c.PatchRuns = append(c.PatchRuns, run)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("insert patch run: %w", err)
	}
	return &run, nil
}

// PreviewPatchRun scans mod vs target vanilla, classifies files, writes preview staging.
func (p *PatcherService) PreviewPatchRun(runID string) (*PatchRunPreview, error) {
	run, modPath, err := p.runAndMod(runID)
	if err != nil {
		return nil, err
	}
	targetPath, err := p.resolveTargetPath(run)
	if err != nil {
		return nil, err
	}
	if targetPath == "" {
		return nil, fmt.Errorf("no target install configured")
	}
	modFiles, err := p.FileService.collectFilesFromPath(modPath, []string{".txt"})
	if err != nil {
		return nil, fmt.Errorf("collect mod files: %w", err)
	}
	targetFiles, err := p.FileService.collectFilesFromPath(targetPath, []string{".txt"})
	if err != nil {
		return nil, fmt.Errorf("collect target files: %w", err)
	}
	preview := &PatchRunPreview{RunID: runID}
	for rel, modP := range modFiles {
		f, err := p.previewOne(run.StagingDir, rel, modP, targetFiles[rel], preview)
		if err != nil {
			return nil, err
		}
		preview.Files = append(preview.Files, f)
	}
	preview.TotalFiles = len(preview.Files)
	now := nowUTC()
	return preview, p.Store.Mutate(func(c *Config) error {
		r := findRun(c, runID)
		if r == nil {
			return fmt.Errorf("patch run not found")
		}
		r.Files, r.Status, r.UpdatedAt = preview.Files, "previewed", now
		return nil
	})
}

// SetFileDecision sets the user decision for a patch run file.
func (p *PatcherService) SetFileDecision(fileID, decision string) error {
	return p.Store.Mutate(func(c *Config) error {
		f := findRunFile(c, fileID)
		if f == nil {
			return fmt.Errorf("file not found")
		}
		f.Decision = decision
		return nil
	})
}

// CancelPatchRun drops a run and its staging dir.
func (p *PatcherService) CancelPatchRun(runID string) error {
	var staging string
	err := p.Store.Mutate(func(c *Config) error {
		for i := range c.PatchRuns {
			if c.PatchRuns[i].ID != runID {
				continue
			}
			staging = c.PatchRuns[i].StagingDir
			c.PatchRuns = append(c.PatchRuns[:i], c.PatchRuns[i+1:]...)
			return nil
		}
		return fmt.Errorf("patch run not found")
	})
	if err != nil {
		return err
	}
	if staging != "" {
		_ = os.RemoveAll(staging)
	}
	return nil
}

// ApplyPatchRun applies accepted changes from a patch run to the mod.
func (p *PatcherService) ApplyPatchRun(runID string) error {
	run, modPath, err := p.runAndMod(runID)
	if err != nil {
		return err
	}
	for _, f := range run.Files {
		if f.Decision != "accept" || f.PreviewPath == "" {
			continue
		}
		dst := filepath.Join(modPath, f.RelPath)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("create dir for %s: %w", f.RelPath, err)
		}
		content, err := os.ReadFile(f.PreviewPath)
		if err != nil {
			return fmt.Errorf("read preview %s: %w", f.RelPath, err)
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", f.RelPath, err)
		}
	}
	now := nowUTC()
	return p.Store.Mutate(func(c *Config) error {
		r := findRun(c, runID)
		if r == nil {
			return fmt.Errorf("patch run not found")
		}
		for i := range r.Files {
			if r.Files[i].Decision == "accept" && r.Files[i].PreviewPath != "" {
				r.Files[i].Status = "applied"
			}
		}
		r.Status, r.UpdatedAt = "applied", now
		return nil
	})
}

func (p *PatcherService) previewOne(
	staging, rel, modP, tgt string, prev *PatchRunPreview,
) (PatchRunFile, error) {
	f := PatchRunFile{
		ID: uuid.New().String(), RelPath: rel, ModPath: modP, TargetPath: tgt,
	}
	if tgt == "" {
		f.Status = "mod_only"
		prev.SkippedCount++
		return f, nil
	}
	out := filepath.Join(staging, rel)
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return f, fmt.Errorf("create preview dir: %w", err)
	}
	r := p.MergeService.mergeAndWrite(
		modP, tgt, out, rel, MergerOptions{AddAdditionalEntries: true})
	if r.Error != "" {
		f.Status, f.Stats = "error", PatchRunFileStats{Error: r.Error}
		return f, nil
	}
	f.PreviewPath, f.Stats = r.OutputPath, PatchRunFileStats{
		Changed: r.Changed, Added: r.Added, Conflicts: len(r.ResolvedConflicts)}
	if f.Stats.Conflicts > 0 || f.Stats.Changed > 3 {
		f.Status = "review"
		prev.ReviewCount++
	} else {
		f.Status = "safe"
		prev.SafeCount++
	}
	return f, nil
}

func (p *PatcherService) runAndMod(runID string) (*PatchRun, string, error) {
	var run *PatchRun
	var modPath string
	p.Store.Read(func(c *Config) {
		if r := findRun(c, runID); r != nil {
			cp := *r
			run = &cp
			if m := findMod(c, r.ModID); m != nil {
				modPath = m.Path
			}
		}
	})
	if run == nil {
		return nil, "", fmt.Errorf("patch run not found")
	}
	if modPath == "" {
		return nil, "", fmt.Errorf("mod not found")
	}
	return run, modPath, nil
}

func (p *PatcherService) resolveTargetPath(run *PatchRun) (string, error) {
	installID := run.TargetInstallID
	var root string
	var err error
	p.Store.Read(func(c *Config) {
		if installID == "" {
			if ws := findWorkspace(c, run.WorkspaceID); ws != nil {
				installID = ws.InstallID
			}
		}
		root, err = installScriptRoot(c, installID)
	})
	if err != nil {
		return "", err
	}
	if root == "" {
		return "", fmt.Errorf("no target install configured")
	}
	return root, nil
}
