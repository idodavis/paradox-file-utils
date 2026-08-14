package services

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const utf8BOM = "\uFEFF"

// FileService provides directory/file selection dialogs and filesystem helpers for merge/IDE.
type FileService struct{}

// GetUserDownloadsDir returns the user's Downloads directory (e.g. ~/Downloads).
func (f *FileService) GetUserDownloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Downloads", "PMT-Merge"), nil
}

func (f *FileService) SelectDirectory(title string) (string, error) {
	app := application.Get()
	dialog := app.Dialog.OpenFile()
	dialog.SetTitle(title)
	dialog.CanChooseDirectories(true)
	dialog.CanChooseFiles(false)
	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return "", nil
	}
	return path, err
}

func (f *FileService) SelectSingleFile(title, filter string) (string, error) {
	app := application.Get()
	dialog := app.Dialog.OpenFile()
	dialog.SetTitle(title)
	dialog.CanChooseFiles(true)
	dialog.CanChooseDirectories(false)
	if filter != "" {
		dialog.AddFilter(filter, filter)
	}
	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return "", nil
	}
	return path, err
}

// WriteWithBOM writes content to outputPath as UTF-8 with BOM. Creates parent directories as needed.
func (f *FileService) WriteWithBOM(outputPath, content string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	if !strings.HasPrefix(content, utf8BOM) {
		content = utf8BOM + content
	}
	return os.WriteFile(outputPath, []byte(content), 0o644)
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

type FileCollectorFilter struct {
	Extensions  []string // e.g. [".txt", ".lua"]
	FileNames   []string // e.g. ["readme.txt", "mod.lua"]
	Regex       string   // e.g. "^(readme|mod)\.txt$"
	IncludePath string   // regex on rel path, e.g. "events/" - include only if matches
	ExcludePath string   // regex on rel path, e.g. "common/" - exclude if matches
}

// CollectFilesFromPath collects all .txt files from a mix of files and directories
// Returns a map of relativePath -> fullPath
func (f *FileService) CollectFilesFromPath(inputPath string, filter FileCollectorFilter) (map[string]string, error) {
	files := make(map[string]string)

	walkErr := filepath.WalkDir(inputPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if len(filter.Extensions) > 0 && !slices.Contains(filter.Extensions, filepath.Ext(path)) {
			return nil
		}
		if len(filter.FileNames) > 0 && !slices.Contains(filter.FileNames, filepath.Base(path)) {
			return nil
		}
		rel, err := filepath.Rel(inputPath, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		if filter.Regex != "" && !regexp.MustCompile(filter.Regex).MatchString(path) {
			return nil
		}
		if filter.IncludePath != "" {
			if re, err := regexp.Compile(filter.IncludePath); err == nil && !re.MatchString(relSlash) {
				return nil
			}
		}
		if filter.ExcludePath != "" {
			if re, err := regexp.Compile(filter.ExcludePath); err == nil && re.MatchString(relSlash) {
				return nil
			}
		}
		files[relSlash] = path
		return nil
	})
	if walkErr != nil && walkErr != fs.SkipAll {
		return nil, fmt.Errorf("Tree walk error in %s: %w", inputPath, walkErr)
	}

	return files, nil
}

// FileMatch represents a matched path pair
type PathMatch struct {
	PathA string `json:"pathA"`
	PathB string `json:"pathB"`
}

// FindMatchingPaths finds paths that exist in both sets. When matchByFilenameOnly is true,
// matches only by filename (e.g. for zz_mod_file.txt where paths differ).
func (f *FileService) FindMatchingPaths(filesA, filesB map[string]string, matchByFilenameOnly bool) (map[string]PathMatch, error) {
	if matchByFilenameOnly {
		return f.findMatchingByFilenameOnly(filesA, filesB), nil
	}
	return f.findMatchingByPath(filesA, filesB), nil
}

// CollectAndMatchPaths collects files from both paths and returns matching pairs.
func (f *FileService) CollectAndMatchPaths(pathA, pathB string, filter FileCollectorFilter, matchByFilenameOnly bool) (map[string]PathMatch, error) {
	filesA, err := f.CollectFilesFromPath(pathA, filter)
	if err != nil {
		return nil, err
	}
	filesB, err := f.CollectFilesFromPath(pathB, filter)
	if err != nil {
		return nil, err
	}
	return f.FindMatchingPaths(filesA, filesB, matchByFilenameOnly)
}

func (f *FileService) findMatchingByFilenameOnly(filesA, filesB map[string]string) map[string]PathMatch {
	matches := make(map[string]PathMatch)
	matchedB := make(map[string]bool)
	filenameToB := make(map[string][]string)
	for k, p := range filesB {
		base := filepath.Base(p)
		filenameToB[base] = append(filenameToB[base], k)
	}
	for keyA, pathA := range filesA {
		base := filepath.Base(pathA)
		for _, keyB := range filenameToB[base] {
			if matchedB[keyB] {
				continue
			}
			pathB := filesB[keyB]
			matchKey := keyA
			if len(keyB) > len(keyA) {
				matchKey = keyB
			}
			matches[matchKey] = PathMatch{PathA: pathA, PathB: pathB}
			matchedB[keyB] = true
			break
		}
	}
	return matches
}

func (f *FileService) findMatchingByPath(filesA, filesB map[string]string) map[string]PathMatch {
	matches := make(map[string]PathMatch)
	matchedA := make(map[string]bool)
	matchedB := make(map[string]bool)

	for keyA, pathA := range filesA {
		if pathB, exists := filesB[keyA]; exists {
			matches[keyA] = PathMatch{PathA: pathA, PathB: pathB}
			matchedA[keyA] = true
			matchedB[keyA] = true
		}
	}

	for keyA, pathA := range filesA {
		if matchedA[keyA] {
			continue
		}
		partsA := strings.Split(keyA, string(filepath.Separator))
		if len(partsA) <= 1 {
			continue
		}
		relStructA := strings.Join(partsA[1:], string(filepath.Separator))
		for keyB, pathB := range filesB {
			if matchedB[keyB] {
				continue
			}
			partsB := strings.Split(keyB, string(filepath.Separator))
			if len(partsB) > 1 {
				relStructB := strings.Join(partsB[1:], string(filepath.Separator))
				if relStructA == relStructB {
					matchKey := keyA
					if len(keyB) > len(keyA) {
						matchKey = keyB
					}
					matches[matchKey] = PathMatch{PathA: pathA, PathB: pathB}
					matchedA[keyA] = true
					matchedB[keyB] = true
					break
				}
			}
		}
	}

	for keyA, pathA := range filesA {
		if matchedA[keyA] {
			continue
		}
		filenameA := filepath.Base(pathA)
		for keyB, pathB := range filesB {
			if matchedB[keyB] {
				continue
			}
			if filepath.Base(pathB) == filenameA {
				matchKey := keyA
				if len(keyB) > len(keyA) {
					matchKey = keyB
				}
				matches[matchKey] = PathMatch{PathA: pathA, PathB: pathB}
				matchedA[keyA] = true
				matchedB[keyB] = true
				break
			}
		}
	}
	return matches
}

// DirEntry is one immediate child of a directory (for lazy file trees).
type DirEntry struct {
	Name     string `json:"name"`
	RelPath  string `json:"relPath"`
	FullPath string `json:"fullPath"`
	IsDir    bool   `json:"isDir"`
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
		if os.IsNotExist(err) {
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

// ReadFileBase64 reads raw file bytes as standard base64 (binary-safe for the IDE bridge).
func (f *FileService) ReadFileBase64(fullPath string) (string, error) {
	fullPath = filepath.Clean(filepath.FromSlash(fullPath))
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return encodeBase64(data), nil
}

// WriteFileBase64 writes raw bytes from standard base64.
func (f *FileService) WriteFileBase64(fullPath, b64 string) error {
	fullPath = filepath.Clean(filepath.FromSlash(fullPath))
	data, err := decodeBase64(b64)
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
		full := filepath.Join(dirPath, name)
		out = append(out, DirEntry{
			Name: name, RelPath: name, FullPath: full, IsDir: e.IsDir(),
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

