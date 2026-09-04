// Package services provides backend services for the Paradox Modding Tools application.
package services

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// FileService provides filesystem helpers for the IDE workbench.
type FileService struct{}

// DirEntry is one immediate child of a directory (for lazy file trees).
type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
}

// CreateDir creates a directory (parents created as needed).
func (f *FileService) CreateDir(fullPath string) error {
	fullPath = filepath.Clean(filepath.FromSlash(fullPath))
	if fullPath == "" || fullPath == "." {
		return fmt.Errorf("path is required")
	}
	return os.MkdirAll(fullPath, 0o755)
}

// RenamePath renames or moves a file or directory.
func (f *FileService) RenamePath(oldPath, newPath string) error {
	oldPath = filepath.Clean(filepath.FromSlash(oldPath))
	newPath = filepath.Clean(filepath.FromSlash(newPath))
	if oldPath == "" || newPath == "" {
		return fmt.Errorf("path is required")
	}
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	return os.Rename(oldPath, newPath)
}

// DeletePath removes a file or directory recursively.
func (f *FileService) DeletePath(fullPath string) error {
	fullPath = filepath.Clean(filepath.FromSlash(fullPath))
	if fullPath == "" || fullPath == "." {
		return fmt.Errorf("path is required")
	}
	return os.RemoveAll(fullPath)
}

// PathStat describes a filesystem node for the workbench FS bridge.
type PathStat struct {
	Exists  bool  `json:"exists"`
	IsDir   bool  `json:"isDir"`
	Size    int64 `json:"size"`
	MtimeMs int64 `json:"mtimeMs"`
	CtimeMs int64 `json:"ctimeMs"`
}

// StatPath returns metadata for a path (Exists=false when missing, no error).
func (f *FileService) StatPath(fullPath string) (PathStat, error) {
	fullPath = filepath.Clean(filepath.FromSlash(fullPath))
	info, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PathStat{Exists: false}, nil
		}
		return PathStat{}, err
	}
	mtime := info.ModTime().UnixMilli()
	return PathStat{
		Exists:  true,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		MtimeMs: mtime,
		CtimeMs: mtime,
	}, nil
}

// FileBytes is a binary file payload for the IDE FS bridge. Missing files are
// Exists=false with no error so the workbench can skip optional paths quietly.
type FileBytes struct {
	Exists bool   `json:"exists"`
	B64    string `json:"b64,omitempty"`
}

// ReadFileBase64 reads raw file bytes as standard base64, or Exists=false if missing.
func (f *FileService) ReadFileBase64(fullPath string) (FileBytes, error) {
	fullPath = filepath.Clean(filepath.FromSlash(fullPath))
	data, err := os.ReadFile(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return FileBytes{Exists: false}, nil
		}
		return FileBytes{}, fmt.Errorf("read file: %w", err)
	}
	return FileBytes{Exists: true, B64: base64.StdEncoding.EncodeToString(data)}, nil
}

// WriteFileBase64 writes raw bytes from standard base64.
func (f *FileService) WriteFileBase64(fullPath, b64 string) error {
	fullPath = filepath.Clean(filepath.FromSlash(fullPath))
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return os.WriteFile(fullPath, data, 0o644)
}

// ListDirectory lists immediate children of dirPath (non-recursive).
// Shows all non-dot files and directories (binary included; editor handles view).
func (f *FileService) ListDirectory(dirPath string) ([]DirEntry, error) {
	dirPath = filepath.Clean(filepath.FromSlash(dirPath))
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("list directory: %w", err)
	}
	out := make([]DirEntry, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		out = append(out, DirEntry{
			Name: name, IsDir: e.IsDir(),
		})
	}
	slices.SortFunc(out, func(a, b DirEntry) int {
		if a.IsDir != b.IsDir {
			if a.IsDir {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out, nil
}
