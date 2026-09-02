// rgsearch.go runs the embedded ripgrep binary for IDE Find in Files.
package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"

	"paradox-modding-tools/services/internal/rgbin"
)

const (
	rgMaxResults   = 10000
	rgMaxFileBytes = "8M"
)

var (
	rgOnce sync.Once
	rgExe  string
	rgErr  error
)

// TextSearchFolder is one workspace folder plus optional glob filters.
type TextSearchFolder struct {
	Path                 string   `json:"path"`
	Includes             []string `json:"includes,omitempty"`
	Excludes             []string `json:"excludes,omitempty"`
	DisregardIgnoreFiles bool     `json:"disregardIgnoreFiles,omitempty"`
}

// TextSearchQuery is a Find-in-Files request from the workbench.
type TextSearchQuery struct {
	Pattern         string             `json:"pattern"`
	IsRegexp        bool               `json:"isRegexp"`
	IsCaseSensitive bool               `json:"isCaseSensitive"`
	IsWordMatch     bool               `json:"isWordMatch"`
	MaxResults      int                `json:"maxResults"`
	Folders         []TextSearchFolder `json:"folders"`
}

// TextSearchHit is one match line mapped for monaco (0-based columns).
type TextSearchHit struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	EndCol  int    `json:"endCol"`
	Preview string `json:"preview"`
}

// TextSearchResult is the capped hit list from TextSearch.
type TextSearchResult struct {
	Hits     []TextSearchHit `json:"hits"`
	LimitHit bool            `json:"limitHit"`
}

// FileSearchQuery is a filename search across workspace folders.
type FileSearchQuery struct {
	FilePattern string             `json:"filePattern"`
	MaxResults  int                `json:"maxResults"`
	Folders     []TextSearchFolder `json:"folders"`
}

// FileSearchHit is one path from FileSearch.
type FileSearchHit struct {
	Path string `json:"path"`
}

// FileSearchResult is the capped path list from FileSearch.
type FileSearchResult struct {
	Hits     []FileSearchHit `json:"hits"`
	LimitHit bool            `json:"limitHit"`
}

type rgJSON struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		Lines struct {
			Text string `json:"text"`
		} `json:"lines"`
		LineNumber int `json:"line_number"`
		Submatches []struct {
			Start int `json:"start"`
			End   int `json:"end"`
		} `json:"submatches"`
	} `json:"data"`
}

func rgPath() (string, error) {
	rgOnce.Do(func() {
		payload := rgbin.Binary()
		if len(payload) < 1024 {
			rgErr = fmt.Errorf("ripgrep not embedded; run task common:fetch:rg")
			return
		}
		base, err := os.UserConfigDir()
		if err != nil {
			rgErr = err
			return
		}
		dir := filepath.Join(base, appConfigDirName, "bin")
		name := "rg"
		if runtime.GOOS == "windows" {
			name = "rg.exe"
		}
		dest := filepath.Join(dir, name)
		stamp := filepath.Join(dir, "rg.version")
		if b, err := os.ReadFile(stamp); err == nil && string(b) == rgbin.Version {
			if _, err := os.Stat(dest); err == nil {
				rgExe = dest
				return
			}
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			rgErr = err
			return
		}
		if err := os.WriteFile(dest, payload, 0o755); err != nil {
			rgErr = err
			return
		}
		_ = os.WriteFile(stamp, []byte(rgbin.Version), 0o644)
		rgExe = dest
	})
	return rgExe, rgErr
}

func capResults(n int) int {
	if n <= 0 || n > rgMaxResults {
		return rgMaxResults
	}
	return n
}

func rgGlob(g string) string {
	g = strings.TrimSpace(g)
	if g == "" {
		return ""
	}
	neg := strings.HasPrefix(g, "!")
	if neg {
		g = strings.TrimSpace(g[1:])
	}
	if g != "" && !strings.Contains(g, "**") {
		g = "**/" + strings.TrimPrefix(g, "/")
	}
	if neg {
		return "!" + g
	}
	return g
}

func folderGlobs(f TextSearchFolder) []string {
	seen := map[string]bool{}
	var out []string
	add := func(g string) {
		g = rgGlob(g)
		if g == "" || seen[g] {
			return
		}
		seen[g] = true
		out = append(out, g)
	}
	for _, g := range f.Includes {
		add(g)
	}
	for _, g := range f.Excludes {
		if strings.HasPrefix(g, "!") {
			add(g)
			continue
		}
		add("!" + g)
	}
	return out
}

func appendGlobFlags(args []string, globs []string) []string {
	for _, g := range globs {
		args = append(args, "--glob", g)
	}
	return args
}

func byteToRune(s string, byteOff int) int {
	if byteOff <= 0 {
		return 0
	}
	if byteOff >= len(s) {
		return utf8.RuneCountInString(s)
	}
	return utf8.RuneCountInString(s[:byteOff])
}

func folderPath(f TextSearchFolder) string {
	p := filepath.Clean(filepath.FromSlash(f.Path))
	if p == "" || p == "." {
		return ""
	}
	return p
}

func rgBaseArgs(noIgnore bool) []string {
	args := []string{"--hidden", "--max-filesize", rgMaxFileBytes}
	if noIgnore {
		args = append(args, "--no-ignore")
	}
	return args
}

// TextSearch runs ripgrep --json, one process per folder.
func (f *FileService) TextSearch(q TextSearchQuery) (TextSearchResult, error) {
	if strings.TrimSpace(q.Pattern) == "" {
		return TextSearchResult{}, nil
	}
	max := capResults(q.MaxResults)
	out := TextSearchResult{Hits: []TextSearchHit{}}
	exe, err := rgPath()
	if err != nil {
		return TextSearchResult{}, err
	}
	for _, folder := range q.Folders {
		p := folderPath(folder)
		if p == "" {
			continue
		}
		remain := max - len(out.Hits)
		if remain <= 0 {
			out.LimitHit = true
			break
		}
		args := append(rgBaseArgs(folder.DisregardIgnoreFiles), "--json")
		if !q.IsRegexp {
			args = append(args, "-F")
		}
		if q.IsWordMatch {
			args = append(args, "-w")
		}
		if q.IsCaseSensitive {
			args = append(args, "-s")
		} else {
			args = append(args, "-i")
		}
		args = appendGlobFlags(args, folderGlobs(folder))
		args = append(args, "--", q.Pattern, p)
		hits, limit, err := collectTextHits(exe, args, remain)
		if err != nil {
			return TextSearchResult{}, err
		}
		out.Hits = append(out.Hits, hits...)
		if limit {
			out.LimitHit = true
			break
		}
	}
	return out, nil
}

func collectTextHits(exe string, args []string, max int) ([]TextSearchHit, bool, error) {
	cmd := exec.Command(exe, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false, fmt.Errorf("rg pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, false, fmt.Errorf("rg start: %w", err)
	}
	var hits []TextSearchHit
	limit := false
	sc := bufio.NewScanner(stdout)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	for sc.Scan() {
		if len(hits) >= max {
			limit = true
			_ = cmd.Process.Kill()
			break
		}
		var ev rgJSON
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Type != "match" {
			continue
		}
		line := strings.TrimRight(ev.Data.Lines.Text, "\r\n")
		start, end := 0, utf8.RuneCountInString(line)
		if len(ev.Data.Submatches) > 0 {
			start = byteToRune(line, ev.Data.Submatches[0].Start)
			end = byteToRune(line, ev.Data.Submatches[0].End)
		}
		ln := ev.Data.LineNumber
		if ln > 0 {
			ln--
		}
		hits = append(hits, TextSearchHit{
			Path: ev.Data.Path.Text, Line: ln, Col: start, EndCol: end,
			Preview: line,
		})
	}
	_ = cmd.Wait()
	return hits, limit, nil
}

// FileSearch lists files whose path contains FilePattern (ripgrep --files).
func (f *FileService) FileSearch(q FileSearchQuery) (FileSearchResult, error) {
	max := capResults(q.MaxResults)
	needle := strings.ToLower(strings.TrimSpace(q.FilePattern))
	out := FileSearchResult{Hits: []FileSearchHit{}}
	exe, err := rgPath()
	if err != nil {
		return FileSearchResult{}, err
	}
	for _, folder := range q.Folders {
		p := folderPath(folder)
		if p == "" {
			continue
		}
		remain := max - len(out.Hits)
		if remain <= 0 {
			out.LimitHit = true
			break
		}
		args := append(rgBaseArgs(folder.DisregardIgnoreFiles), "--files")
		args = appendGlobFlags(args, folderGlobs(folder))
		args = append(args, p)
		cmd := exec.Command(exe, args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return FileSearchResult{}, fmt.Errorf("rg pipe: %w", err)
		}
		if err := cmd.Start(); err != nil {
			return FileSearchResult{}, fmt.Errorf("rg start: %w", err)
		}
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			if len(out.Hits) >= max {
				out.LimitHit = true
				_ = cmd.Process.Kill()
				break
			}
			path := strings.TrimSpace(sc.Text())
			if path == "" {
				continue
			}
			if needle != "" && !strings.Contains(strings.ToLower(path), needle) &&
				!strings.Contains(strings.ToLower(filepath.Base(path)), needle) {
				continue
			}
			out.Hits = append(out.Hits, FileSearchHit{Path: path})
		}
		_ = cmd.Wait()
		if out.LimitHit {
			break
		}
	}
	return out, nil
}
