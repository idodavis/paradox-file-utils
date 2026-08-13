// Package loc parses Paradox localization YAML (l_*: KEY:N "value").
package loc

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

var (
	langHeaderRE = regexp.MustCompile(`^l_([a-zA-Z_]+)\s*:`)
	entryRE      = regexp.MustCompile(`^\s*([^\s:#][^:]*?)\s*:\s*(\d+)?\s*"(.*)"\s*$`)
)

// Entry is one localization key/value.
type Entry struct {
	Key      string
	Version  string
	Value    string
	Language string
	FilePath string
	Line     int
}

// ParseFile reads a Paradox loc .yml/.yaml file.
func ParseFile(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data, path)
}

// Parse parses loc file bytes.
func Parse(data []byte, path string) ([]Entry, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	sc := bufio.NewScanner(bytes.NewReader(data))
	lang := ""
	lineNo := 0
	var out []Entry
	for sc.Scan() {
		lineNo++
		line := strings.TrimRight(sc.Text(), "\r")
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if m := langHeaderRE.FindStringSubmatch(trim); m != nil {
			lang = "l_" + m[1]
			continue
		}
		if lang == "" {
			continue
		}
		m := entryRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		out = append(out, Entry{
			Key:      m[1],
			Version:  m[2],
			Value:    m[3],
			Language: lang,
			FilePath: path,
			Line:     lineNo,
		})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan loc %s: %w", path, err)
	}
	return out, nil
}

// CollectKeys walks roots for *.yml/*.yaml and returns language → key set.
func CollectKeys(roots []string) (map[string]map[string]Entry, error) {
	return CollectKeysCtx(context.Background(), roots, nil)
}

// CollectProgress reports loc walk progress.
type CollectProgress func(done, total int, path string)

// CollectKeysCtx collects loc keys with cancellation and optional progress.
func CollectKeysCtx(
	ctx context.Context, roots []string, onProgress CollectProgress,
) (map[string]map[string]Entry, error) {
	var paths []string
	for _, root := range roots {
		if root == "" {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".yml" && ext != ".yaml" {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	total := len(paths)
	if total == 0 {
		return map[string]map[string]Entry{}, nil
	}

	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	if workers > 8 {
		workers = 8
	}
	jobs := make(chan string, workers*2)
	type parsed struct {
		entries []Entry
		path    string
	}
	results := make(chan parsed, workers*2)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				if ctx.Err() != nil {
					return
				}
				entries, err := ParseFile(path)
				if err != nil {
					select {
					case <-ctx.Done():
						return
					case results <- parsed{path: path}:
					}
					continue
				}
				select {
				case <-ctx.Done():
					return
				case results <- parsed{entries: entries, path: path}:
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, p := range paths {
			select {
			case <-ctx.Done():
				return
			case jobs <- p:
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	byLang := map[string]map[string]Entry{}
	done := 0
	for r := range results {
		done++
		for _, e := range r.entries {
			if byLang[e.Language] == nil {
				byLang[e.Language] = map[string]Entry{}
			}
			byLang[e.Language][e.Key] = e
		}
		if onProgress != nil && (done == total || done%25 == 0) {
			onProgress(done, total, r.path)
		}
		if err := ctx.Err(); err != nil {
			return byLang, err
		}
	}
	return byLang, ctx.Err()
}

// FormatEntry formats a loc line for writing.
func FormatEntry(key, value string, version int) string {
	return fmt.Sprintf(" %s:%d \"%s\"", key, version, escapeLoc(value))
}

func escapeLoc(s string) string {
	return strings.ReplaceAll(s, `"`, `'`)
}

// EnsureLanguageFile ensures a loc file exists with the language header.
func EnsureLanguageFile(path, language string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body := "\uFEFF" + language + ":\n"
	return os.WriteFile(path, []byte(body), 0o644)
}

// AppendEntries appends entries under an existing language file.
func AppendEntries(path string, entries []Entry) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, e := range entries {
		ver := 0
		if e.Version != "" {
			fmt.Sscanf(e.Version, "%d", &ver)
		}
		if _, err := fmt.Fprintln(f, FormatEntry(e.Key, e.Value, ver)); err != nil {
			return err
		}
	}
	return nil
}
