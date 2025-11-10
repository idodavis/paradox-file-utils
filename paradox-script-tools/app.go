package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"paradox-file-utils/internal/core"
	"paradox-file-utils/internal/diff"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// OnStartup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
}

// WriteMergedFile writes the merged content to a file
func (a *App) WriteMergedFile(outputPath, content string) error {
	return core.WriteMergedFile(outputPath, content)
}

// SelectDirectory opens a directory selection dialog
func (a *App) SelectDirectory(title string) (string, error) {
	selected, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
	if err != nil {
		return "", err
	}
	return selected, nil
}

// SelectMultipleFiles opens a file selection dialog allowing multiple file selection
func (a *App) SelectMultipleFiles(title, filter string) ([]string, error) {
	selected, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{
				DisplayName: filter,
				Pattern:     filter,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return selected, nil
}

// collectFilesFromPaths collects all .txt files from a mix of files and directories
// Returns a map of relativePath -> fullPath
// For individual files, uses the filename as the relative path
// For directories, uses the relative path from the directory root
func (a *App) collectFilesFromPaths(paths []string) (map[string]string, error) {
	files := make(map[string]string)

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("error accessing path %s: %w", path, err)
		}

		if info.IsDir() {
			// Walk directory and collect all .txt files
			err := filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if fileInfo.IsDir() {
					return nil
				}
				if strings.HasSuffix(strings.ToLower(filePath), ".txt") {
					relPath, err := filepath.Rel(path, filePath)
					if err != nil {
						return err
					}
					// Use directory path + relative path as key to handle multiple directories
					key := filepath.Join(filepath.Base(path), relPath)
					files[key] = filePath
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("error walking directory %s: %w", path, err)
			}
		} else {
			// Individual file - use filename as key
			if strings.HasSuffix(strings.ToLower(path), ".txt") {
				key := filepath.Base(path)
				// If file already exists, append parent dir to make unique
				if _, exists := files[key]; exists {
					parentDir := filepath.Base(filepath.Dir(path))
					key = filepath.Join(parentDir, key)
				}
				files[key] = path
			}
		}
	}

	return files, nil
}

// findMatchingFiles finds files that exist in both sets by matching relative paths
// Returns a map of relativePath -> {fileAPath, fileBPath}
func (a *App) findMatchingFiles(filesA, filesB map[string]string) map[string]struct {
	FileAPath string
	FileBPath string
} {
	matches := make(map[string]struct {
		FileAPath string
		FileBPath string
	})
	matchedA := make(map[string]bool)
	matchedB := make(map[string]bool)

	// First, try to match files by their relative path keys (exact match)
	for keyA, pathA := range filesA {
		if pathB, exists := filesB[keyA]; exists {
			matches[keyA] = struct {
				FileAPath string
				FileBPath string
			}{FileAPath: pathA, FileBPath: pathB}
			matchedA[keyA] = true
			matchedB[keyA] = true
		}
	}

	// Then, try matching by relative path structure (e.g., "events/file.txt")
	// This handles cases where files are in the same subdirectory structure but different roots
	for keyA, pathA := range filesA {
		if matchedA[keyA] {
			continue
		}

		// Extract the relative path structure (everything after the first directory component)
		partsA := strings.Split(keyA, string(filepath.Separator))
		if len(partsA) > 1 {
			relStructA := strings.Join(partsA[1:], string(filepath.Separator))

			for keyB, pathB := range filesB {
				if matchedB[keyB] {
					continue
				}

				partsB := strings.Split(keyB, string(filepath.Separator))
				if len(partsB) > 1 {
					relStructB := strings.Join(partsB[1:], string(filepath.Separator))
					if relStructA == relStructB {
						// Use the more descriptive key
						matchKey := keyA
						if len(keyB) > len(keyA) {
							matchKey = keyB
						}
						matches[matchKey] = struct {
							FileAPath string
							FileBPath string
						}{FileAPath: pathA, FileBPath: pathB}
						matchedA[keyA] = true
						matchedB[keyB] = true
						break
					}
				}
			}
		}
	}

	// Finally, try matching by just the filename (for cases where directory structure differs)
	for keyA, pathA := range filesA {
		if matchedA[keyA] {
			continue
		}

		filenameA := filepath.Base(pathA)
		for keyB, pathB := range filesB {
			if matchedB[keyB] {
				continue
			}

			filenameB := filepath.Base(pathB)
			if filenameA == filenameB {
				// Use the more descriptive key
				matchKey := keyA
				if len(keyB) > len(keyA) {
					matchKey = keyB
				}
				matches[matchKey] = struct {
					FileAPath string
					FileBPath string
				}{FileAPath: pathA, FileBPath: pathB}
				matchedA[keyA] = true
				matchedB[keyB] = true
				break
			}
		}
	}

	return matches
}

// MergeMultipleFileSets merges multiple file sets (directories and/or files)
// Only merges files that exist in both sets
// optionsMap is a map that will be converted to core.MergeOptions
func (a *App) MergeMultipleFileSets(pathsA, pathsB []string, outputDir string, optionsMap map[string]interface{}) ([]FileMergeResult, error) {
	var results []FileMergeResult

	// Convert optionsMap to core.MergeOptions
	options := a.mapToMergeOptions(optionsMap)

	// Collect files from both sets
	filesA, err := a.collectFilesFromPaths(pathsA)
	if err != nil {
		return nil, fmt.Errorf("error collecting files from set A: %w", err)
	}

	filesB, err := a.collectFilesFromPaths(pathsB)
	if err != nil {
		return nil, fmt.Errorf("error collecting files from set B: %w", err)
	}

	// Find matching files
	matches := a.findMatchingFiles(filesA, filesB)

	// Merge matching files
	for relPath, match := range matches {
		mergedContent, mergeResult, err := core.GetMergedContent(match.FileAPath, match.FileBPath, options)
		if err != nil {
			results = append(results, FileMergeResult{
				FilePath: relPath,
				Error:    err.Error(),
			})
			continue
		}

		// Write merged file
		outPath := filepath.Join(outputDir, relPath)
		if err := core.WriteMergedFile(outPath, mergedContent); err != nil {
			results = append(results, FileMergeResult{
				FilePath: relPath,
				Error:    err.Error(),
			})
			continue
		}

		results = append(results, FileMergeResult{
			FilePath:   relPath,
			FileAPath:  match.FileAPath,
			FileBPath:  match.FileBPath,
			OutputPath: outPath,
			Changed:    len(mergeResult.EntriesChanged),
			Added:      len(mergeResult.EntriesAdded),
			Removed:    len(mergeResult.EntriesRemoved),
		})
	}

	return results, nil
}

// mapToMergeOptions converts a map to core.MergeOptions
func (a *App) mapToMergeOptions(m map[string]interface{}) core.MergeOptions {
	options := core.MergeOptions{
		AddAdditionalEntries: true,
		EntryPlacement:       "bottom",
		KeyList:              []string{},
		CustomCommentPrefix:  "",
	}

	// Handle AddAdditionalEntries
	if aae, ok := m["AddAdditionalEntries"].(bool); ok {
		options.AddAdditionalEntries = aae
	}

	// Handle EntryPlacement
	if ep, ok := m["EntryPlacement"].(string); ok {
		options.EntryPlacement = ep
	}

	// Handle KeyList
	if kl, ok := m["KeyList"].([]interface{}); ok {
		for _, k := range kl {
			if key, ok := k.(string); ok {
				options.KeyList = append(options.KeyList, key)
			}
		}
	}

	// Handle CustomCommentPrefix
	if ccp, ok := m["CustomCommentPrefix"].(string); ok {
		options.CustomCommentPrefix = ccp
	}

	return options
}

// GetDiff gets the diff for a merged file
func (a *App) GetDiff(fileAPath, fileBPath string) (string, error) {
	diffResult, err := diff.GenerateDiff(fileAPath, fileBPath)
	if err != nil {
		return "", err
	}

	return diffResult, nil
}

// CompareDirectories compares two game directories and returns directory comparison
func (a *App) CompareDirectories(dirA, dirB string, fileExtensions []string) (*diff.DirectoryComparison, error) {
	return diff.CompareDirectories(dirA, dirB, fileExtensions)
}

// GenCompToolReport generates a text report of directory comparison
func (a *App) GenerateCompToolReport(comparison *diff.DirectoryComparison) string {
	return diff.GenerateCompToolReport(comparison)
}

// FileMergeResult represents the result of merging a single file
type FileMergeResult struct {
	FilePath   string
	FileAPath  string
	FileBPath  string
	OutputPath string
	Changed    int
	Added      int
	Removed    int
	Error      string
}
