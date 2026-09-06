// scopereal_test.go checks the scope walk against real vanilla script. Vanilla
// is correct by definition, so any wrong-scope report over it is a false
// positive. Gated on PMT_TEST_INSTALL_CK3 + PMT_TEST_DOCS_CK3; slow.

package session_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/session"
)

func realInstall(t *testing.T, installEnv, docsEnv string) (install, docs string) {
	t.Helper()
	if testing.Short() {
		t.Skip("walks a full game install")
	}
	install, docs = os.Getenv(installEnv), os.Getenv(docsEnv)
	if install == "" || docs == "" {
		t.Skipf("%s / %s not set", installEnv, docsEnv)
	}
	if _, err := os.Stat(install); err != nil {
		t.Skipf("%s: %v", installEnv, err)
	}
	return install, docs
}

// The scope model must not accuse the game's own script of being wrong.
// Vanilla is the answer key: every wrong-scope report over it is a false
// positive. This is the test that keeps the diagnostic honest as the schema
// changes across patches.
func checkNoFalsePositives(t *testing.T, gameID, installEnv, docsEnv, scriptRoot string) {
	t.Helper()
	install, docs := realInstall(t, installEnv, docsEnv)
	schema := catalog.ReadSchema([]string{docs})
	if schema == nil {
		t.Fatalf("no schema under %s", docs)
	}
	root := filepath.Join(install, scriptRoot)
	cache := &catalog.VanillaCache{Schema: schema}
	catalog.PrepareCache(cache)

	s := session.NewWithLoc("ws", gameID, "english", cache, nil,
		[]catalog.ModInput{{Origin: "mod", Root: root, Order: 0}})

	// Walk the whole script root rather than assuming a fixed layout: EU5 nests
	// events/common under game/in_game/, not game/ directly.
	var files []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".txt") {
			return nil
		}
		slash := strings.ReplaceAll(p, "\\", "/")
		if strings.Contains(slash, "/events/") ||
			strings.Contains(slash, "/on_action/") || strings.Contains(slash, "/on_actions/") {
			files = append(files, p)
		}
		return nil
	})
	if len(files) == 0 {
		t.Skipf("no vanilla script under %s", root)
	}

	flagged := 0
	seen := map[string]int{}
	for _, p := range files {
		for _, m := range s.ScopeMisuses(p) {
			flagged++
			seen[m.Key+" in "+m.Scope]++
		}
	}
	t.Logf("%s: scanned %d files, %d scope reports", gameID, len(files), flagged)
	shown := 0
	for k, n := range seen {
		if shown >= 15 {
			break
		}
		t.Logf("  %s x%d", k, n)
		shown++
	}
	if flagged > 0 {
		t.Errorf("%s: %d false positives over vanilla script", gameID, flagged)
	}
}

func TestScopeNoFalsePositivesCK3(t *testing.T) {
	checkNoFalsePositives(t, "ck3", "PMT_TEST_INSTALL_CK3", "PMT_TEST_DOCS_CK3", "game")
}

func TestScopeNoFalsePositivesVic3(t *testing.T) {
	checkNoFalsePositives(t, "vic3", "PMT_TEST_INSTALL_VIC3", "PMT_TEST_DOCS_VIC3", "game")
}

func TestScopeNoFalsePositivesEU5(t *testing.T) {
	checkNoFalsePositives(t, "eu5", "PMT_TEST_INSTALL_EU5", "PMT_TEST_DOCS_EU5", "game")
}
