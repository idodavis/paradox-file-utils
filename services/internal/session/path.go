// path.go canonicalizes filesystem paths so session maps, buffers, and LSP
// requests agree on Windows (slash/case) and Unix.

package session

import (
	"path/filepath"
	"runtime"
	"strings"

	"paradox-modding-tools/services/internal/catalog"
)

// CanonPath returns a cleaned path for index and buffer identity.
func CanonPath(p string) string {
	if p == "" {
		return p
	}
	return filepath.Clean(p)
}

// SamePath reports whether a and b name the same file. On Windows the
// comparison is case-insensitive after Clean.
func SamePath(a, b string) bool {
	a, b = CanonPath(a), CanonPath(b)
	if a == b {
		return true
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return false
}

// RelPath reports whether file is inside root and returns the relative path.
func RelPath(root, file string) (rel string, ok bool) {
	root, file = CanonPath(root), CanonPath(file)
	r, err := filepath.Rel(root, file)
	if err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return r, true
	}
	if runtime.GOOS != "windows" {
		return "", false
	}
	rl, fl := strings.ToLower(root), strings.ToLower(file)
	sep := string(filepath.Separator)
	if fl == rl {
		return ".", true
	}
	if !strings.HasPrefix(fl, rl+sep) {
		return "", false
	}
	return file[len(root)+1:], true
}

// KindFor returns the catalog file kind for an absolute path.
func (s *Session) KindFor(path string) string {
	_, rel, ok := s.Locate(path)
	if !ok {
		rel = path
	}
	return catalog.ClassifyRel(strings.ReplaceAll(rel, "\\", "/"), filepath.Base(path))
}
