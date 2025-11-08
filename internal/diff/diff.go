package diff

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffResult represents the result of a diff operation
type DiffResult struct {
	UnifiedDiff string
	RawDiffs    []diffmatchpatch.Diff
	FilePath    string
}

// GenerateDiff generates a diff between two file contents
func GenerateDiff(fileAPath, fileBPath, contentA, contentB string) (*DiffResult, error) {
	dmp := diffmatchpatch.New()

	// Read files if content is not provided
	if contentA == "" {
		data, err := os.ReadFile(fileAPath)
		if err != nil {
			return nil, fmt.Errorf("error reading file A: %w", err)
		}
		contentA = string(data)
	}

	if contentB == "" {
		data, err := os.ReadFile(fileBPath)
		if err != nil {
			return nil, fmt.Errorf("error reading file B: %w", err)
		}
		contentB = string(data)
	}

	// Generate diffs
	diffs := dmp.DiffMain(contentA, contentB, false)

	// Generate unified diff
	unifiedDiff := dmp.DiffPrettyText(diffs)

	return &DiffResult{
		UnifiedDiff: unifiedDiff,
		RawDiffs:    diffs,
		FilePath:    fileAPath,
	}, nil
}

// GenerateUnifiedDiff generates a unified diff format (like git diff)
func GenerateUnifiedDiff(fileAPath, fileBPath, contentA, contentB string) (string, error) {
	dmp := diffmatchpatch.New()

	// Read files if content is not provided
	if contentA == "" {
		data, err := os.ReadFile(fileAPath)
		if err != nil {
			return "", fmt.Errorf("error reading file A: %w", err)
		}
		contentA = string(data)
	}

	if contentB == "" {
		data, err := os.ReadFile(fileBPath)
		if err != nil {
			return "", fmt.Errorf("error reading file B: %w", err)
		}
		contentB = string(data)
	}

	// Generate diffs
	diffs := dmp.DiffMain(contentA, contentB, false)

	// Convert to unified diff format
	unifiedDiff := dmp.DiffPrettyText(diffs)

	return unifiedDiff, nil
}

// TODO: Not Used Yet
// SaveDiffToFile saves a diff to a file
func SaveDiffToFile(diffPath, diffContent string) error {
	// Ensure output directory exists
	dir := filepath.Dir(diffPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("error creating diff directory: %w", err)
	}

	file, err := os.Create(diffPath)
	if err != nil {
		return fmt.Errorf("error creating diff file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(diffContent)
	if err != nil {
		return fmt.Errorf("error writing diff file: %w", err)
	}

	return nil
}
