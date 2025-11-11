package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"paradox-file-utils/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file-or-directory> [<file-or-directory> ...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Tests parser against one or more files or directories.\n")
		os.Exit(1)
	}

	var filesToTest []string

	// Collect all files to test
	for _, arg := range os.Args[1:] {
		info, err := os.Stat(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing %s: %v\n", arg, err)
			continue
		}

		if info.IsDir() {
			// Walk directory and collect all .txt files
			err := filepath.Walk(arg, func(filePath string, fileInfo os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if fileInfo.IsDir() {
					return nil
				}
				if strings.HasSuffix(strings.ToLower(filePath), ".txt") {
					filesToTest = append(filesToTest, filePath)
				}
				return nil
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error walking directory %s: %v\n", arg, err)
			}
		} else {
			// Individual file
			if strings.HasSuffix(strings.ToLower(arg), ".txt") {
				filesToTest = append(filesToTest, arg)
			}
		}
	}

	if len(filesToTest) == 0 {
		fmt.Fprintf(os.Stderr, "No .txt files found to test.\n")
		os.Exit(1)
	}

	// Test each file
	successCount := 0
	failureCount := 0

	for _, filePath := range filesToTest {
		_, err := parser.ParseFile(filePath)
		if err != nil {
			fmt.Printf("FAIL: %s - %v\n", filePath, err)
			failureCount++
		} else {
			fmt.Printf("OK:   %s\n", filePath)
			successCount++
		}
	}

	// Print summary
	fmt.Printf("\nSummary: %d passed, %d failed, %d total\n", successCount, failureCount, len(filesToTest))

	if failureCount > 0 {
		os.Exit(1)
	}
}
