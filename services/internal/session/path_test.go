// path_test.go covers CanonPath identity on Windows vs Unix.

package session

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCanonPathWindowsFold(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "windows" {
		t.Skip("Windows path folding")
	}
	a := `C:\Steam\steamapps\common\Game\events\x.txt`
	b := `c:\steam\steamapps\common\game\events\x.txt`
	if CanonPath(a) != CanonPath(b) {
		t.Fatalf("CanonPath(%q) = %q, CanonPath(%q) = %q",
			a, CanonPath(a), b, CanonPath(b))
	}
	if !SamePath(a, b) {
		t.Fatal("SamePath should treat drive/case variants as one file")
	}
}

func TestCanonPathUnixPreservesCase(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Unix case-sensitive paths")
	}
	a := "/tmp/Mod/events/X.txt"
	if CanonPath(a) != filepath.Clean(a) {
		t.Fatalf("CanonPath = %q, want cleaned original case", CanonPath(a))
	}
	if SamePath(a, strings.ToLower(a)) {
		t.Fatal("SamePath must not fold case on Unix")
	}
}
