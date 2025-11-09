package main

import (
	"fmt"
	"os"
	"path/filepath"

	"paradox-file-utils/internal/config"
	"paradox-file-utils/internal/core"
	"paradox-file-utils/internal/diff"
)

var (
	SYNC_OUTPUT_DIR = "sync-output"
	DIFF_OUTPUT_DIR = "sync-output/diffs"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./vanilla_synchronizer <config.yaml>")
		return
	}

	cfg, err := config.LoadConfig(os.Args[1])
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// Ensure output directories exist
	if err := os.MkdirAll(SYNC_OUTPUT_DIR, 0o755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		return
	}
	if err := os.MkdirAll(DIFF_OUTPUT_DIR, 0o755); err != nil {
		fmt.Printf("Error creating diff directory: %v\n", err)
		return
	}

	for i, file := range cfg.Files {
		relPath := file.RelativePath

		// Build file paths
		fileAPath := filepath.Join(cfg.GameRoot, relPath)
		fileBPath := filepath.Join(cfg.ModRoot, relPath)

		// Convert config to merge options
		mergeOptions := cfg.ToMergeOptions(i)

		// Get merged content
		mergedContent, mergeResult, err := core.GetMergedContent(fileAPath, fileBPath, mergeOptions)
		if err != nil {
			fmt.Printf("Error merging file %s: %v\n", relPath, err)
			continue
		}

		// Write merged file
		outPath := filepath.Join(SYNC_OUTPUT_DIR, filepath.Base(relPath))
		if err := core.WriteMergedFile(outPath, mergedContent); err != nil {
			fmt.Printf("Error writing merged file %s: %v\n", outPath, err)
			continue
		}

		// Generate diff
		diffResult, err := diff.GenerateDiff(fileAPath, outPath)
		if err != nil {
			fmt.Printf("Warning: Could not generate diff: %v\n", err)
		} else {
			// Save diff to file
			diffPath := filepath.Join(DIFF_OUTPUT_DIR, filepath.Base(relPath)+".diff")
			if err := os.WriteFile(diffPath, []byte(diffResult.UnifiedDiff), 0o644); err != nil {
				fmt.Printf("Warning: Could not save diff file: %v\n", err)
			}
			fmt.Printf("Diff saved to: %s\n", diffPath)
		}

		// Print merge summary
		fmt.Printf("Merged file written to: %s\n", outPath)
		if len(mergeResult.EntriesChanged) > 0 {
			fmt.Printf("  Changed entries: %d\n", len(mergeResult.EntriesChanged))
		}
		if len(mergeResult.EntriesAdded) > 0 {
			fmt.Printf("  Added entries: %d\n", len(mergeResult.EntriesAdded))
		}
		if len(mergeResult.EntriesRemoved) > 0 {
			fmt.Printf("  Removed entries: %d\n", len(mergeResult.EntriesRemoved))
		}
	}
}
