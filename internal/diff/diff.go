package diff

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aymanbagabas/go-udiff"
)

// DiffResult represents the result of a diff operation
type DiffResult struct {
	UnifiedDiff string
	FilePath    string
}

// DirectoryComparison represents a comparison between two game versions
type DirectoryComparison struct {
	ChangedFiles []ChangedFile
	AddedFiles   []string
	RemovedFiles []string
}

// ChangedFile represents a file that changed between versions
type ChangedFile struct {
	RelativePath string
	FullPathA    string
	FullPathB    string
	Diff         string
	Error        string
}

// GenerateDiff generates a diff between two file contents (like git diff)
func GenerateDiff(fileAPath string, fileBPath string) (*DiffResult, error) {
	aFileContent, err := os.ReadFile(fileAPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file A: %w", err)
	}
	bFileContent, err := os.ReadFile(fileBPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file B: %w", err)
	}

	// Generate diffs
	diff := udiff.Unified(fileAPath, fileBPath, string(aFileContent), string(bFileContent))

	return &DiffResult{
		UnifiedDiff: diff,
		FilePath:    fileAPath,
	}, nil
}

// CompareDirectories compares two game directories and returns a PatchComparison
func CompareDirectories(dirA, dirB string, fileExtensions []string) (*DirectoryComparison, error) {
	if fileExtensions == nil {
		fileExtensions = []string{".txt"}
	}

	comparison := &DirectoryComparison{
		ChangedFiles: []ChangedFile{},
		AddedFiles:   []string{},
		RemovedFiles: []string{},
	}

	// Build file maps
	filesA := make(map[string]string)
	filesB := make(map[string]string)

	// Collect files from directory A
	err := filepath.Walk(dirA, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		// Check if file has one of the extensions we care about
		hasExtension := false
		for _, ext := range fileExtensions {
			if strings.HasSuffix(strings.ToLower(path), ext) {
				hasExtension = true
				break
			}
		}

		if !hasExtension {
			return nil
		}

		relPath, err := filepath.Rel(dirA, path)
		if err != nil {
			return err
		}
		filesA[relPath] = path
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory A: %w", err)
	}

	// Collect files from directory B
	err = filepath.Walk(dirB, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		// Check if file has one of the extensions we care about
		hasExtension := false
		for _, ext := range fileExtensions {
			if strings.HasSuffix(strings.ToLower(path), ext) {
				hasExtension = true
				break
			}
		}

		if !hasExtension {
			return nil
		}

		relPath, err := filepath.Rel(dirB, path)
		if err != nil {
			return err
		}
		filesB[relPath] = path
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory B: %w", err)
	}

	// Find changed files
	for relPath, pathA := range filesA {
		if pathB, exists := filesB[relPath]; exists {
			// File exists in both, check if it changed
			diffResult, err := GenerateDiff(pathA, pathB)
			if err != nil {
				comparison.ChangedFiles = append(comparison.ChangedFiles, ChangedFile{
					RelativePath: relPath,
					FullPathA:    pathA,
					FullPathB:    pathB,
					Error:        err.Error(),
				})
				continue
			}

			// Check if there are actual differences
			hasChanges := diffResult.UnifiedDiff != ""

			if hasChanges {
				comparison.ChangedFiles = append(comparison.ChangedFiles, ChangedFile{
					RelativePath: relPath,
					FullPathA:    pathA,
					FullPathB:    pathB,
					Diff:         diffResult.UnifiedDiff,
				})
			}
		} else {
			// File exists only in A (removed)
			comparison.RemovedFiles = append(comparison.RemovedFiles, relPath)
		}
	}

	// Find added files (exist only in B)
	for relPath := range filesB {
		if _, exists := filesA[relPath]; !exists {
			comparison.AddedFiles = append(comparison.AddedFiles, relPath)
		}
	}

	return comparison, nil
}

// GenerateCompToolReport generates a text report of the directory comparison
func GenerateCompToolReport(comparison *DirectoryComparison) string {
	var report strings.Builder

	report.WriteString("=== Comparison Tool Report ===\n\n")
	report.WriteString(fmt.Sprintf("Changed Files: %d\n", len(comparison.ChangedFiles)))
	report.WriteString(fmt.Sprintf("Added Files: %d\n", len(comparison.AddedFiles)))
	report.WriteString(fmt.Sprintf("Removed Files: %d\n\n", len(comparison.RemovedFiles)))

	if len(comparison.ChangedFiles) > 0 {
		report.WriteString("=== Changed Files ===\n")
		for _, file := range comparison.ChangedFiles {
			report.WriteString(fmt.Sprintf("\n--- %s ---\n", file.RelativePath))
			if file.Error != "" {
				report.WriteString(fmt.Sprintf("Error: %s\n", file.Error))
			} else {
				report.WriteString(file.Diff)
			}
		}
		report.WriteString("\n")
	}

	if len(comparison.AddedFiles) > 0 {
		report.WriteString("=== Added Files ===\n")
		for _, file := range comparison.AddedFiles {
			report.WriteString(fmt.Sprintf("  + %s\n", file))
		}
		report.WriteString("\n")
	}

	if len(comparison.RemovedFiles) > 0 {
		report.WriteString("=== Removed Files ===\n")
		for _, file := range comparison.RemovedFiles {
			report.WriteString(fmt.Sprintf("  - %s\n", file))
		}
		report.WriteString("\n")
	}

	return report.String()
}
