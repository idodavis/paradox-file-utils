// determinism_test.go guards the scoreboard's usefulness. A count that moves
// between runs cannot be a regression threshold, and CK3 orphaned came back
// 76,041 / 76,064 / 76,071 across three sweeps of the same corpus.
//
// The two halves are separated on purpose: one session built twice from one
// cache isolates Coverage, and two caches from two scans isolates the parallel
// install walk. Whichever differs is the one to fix. Both faults it has caught
// so far were real model bugs rather than reporting artifacts — see
// GAME-SYNTAX.md §6, "Scheduling must not decide anything".
//
// By default it checks one Victoria 3 conversion, which is seconds. Widen it
// when chasing something: an empty PMT_DET_MOD runs every mod of the game,
// which is ~45s for Victoria 3 and ~11min for EU5.

package views

import (
	"context"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"paradox-modding-tools/services/internal/catalog"
	"paradox-modding-tools/services/internal/session"
)

func TestHealthIsDeterministic(t *testing.T) {
	if testing.Short() {
		t.Skip("scans a real install")
	}
	// Defaults to a Victoria 3 conversion because it is the fastest one that
	// exercises the whole path. Point it elsewhere to chase a specific drift:
	//   PMT_DET_GAME=ck3 PMT_DET_APP=1158310 PMT_DET_MOD=2962333032
	game, appID, modID := "vic3", "529340", "3199730217"
	if v := os.Getenv("PMT_DET_GAME"); v != "" {
		game, appID, modID = v, os.Getenv("PMT_DET_APP"), os.Getenv("PMT_DET_MOD")
	}
	install := os.Getenv("PMT_TEST_INSTALL_" + strings.ToUpper(game))
	if install == "" {
		t.Skipf("PMT_TEST_INSTALL_%s not set", strings.ToUpper(game))
	}
	// An empty PMT_DET_MOD checks every mod of the game. One mod passing proves
	// little: the drift chased here was visible only in a game's total, and the
	// mod carrying it was not the obvious one.
	var mods []string
	if modID != "" {
		mods = []string{modID}
	} else {
		ents, err := os.ReadDir(filepath.Join(workshopRoot, appID))
		if err != nil {
			t.Skipf("no workshop content (%v)", err)
		}
		for _, e := range ents {
			if e.IsDir() {
				mods = append(mods, e.Name())
			}
		}
	}
	scan := func(id string) (*catalog.VanillaCache, *catalog.VanillaLoc) {
		t.Helper()
		t.Cleanup(func() { _ = catalog.DropVanillaFiles(id) })
		c, vloc, err := catalog.Scan(context.Background(), catalog.ScanRequest{
			InstallID: id, GameID: game, InstallPath: install,
			Version: "det", LocLang: "english",
		})
		if err != nil {
			t.Fatalf("scan %s: %v", id, err)
		}
		return c, vloc
	}
	report := func(c *catalog.VanillaCache, vloc *catalog.VanillaLoc, mod string) HealthReport {
		s := session.NewWithLoc("det", game, "english", c, vloc,
			[]catalog.ModInput{{
				Origin: "mod", Root: filepath.Join(workshopRoot, appID, mod), Name: "m",
			}})
		defer s.Close()
		return Health(s, nil)
	}
	same := func(what, mod string, a, b HealthReport) {
		t.Helper()
		if a.Missing != b.Missing || a.Orphaned != b.Orphaned ||
			a.Untranslated != b.Untranslated || a.Dangling != b.Dangling {
			t.Errorf("%s [%s]: missing %d/%d orphaned %d/%d untranslated %d/%d dangling %d/%d",
				what, mod, a.Missing, b.Missing, a.Orphaned, b.Orphaned,
				a.Untranslated, b.Untranslated, a.Dangling, b.Dangling)
		}
	}

	c1, l1 := scan("det-1")
	c2, l2 := scan("det-2")
	for _, mod := range mods {
		same("same cache, two sessions", mod,
			report(c1, l1, mod), report(c1, l1, mod))
		same("two scans of one install", mod,
			report(c1, l1, mod), report(c2, l2, mod))
		debug.FreeOSMemory()
	}
}
