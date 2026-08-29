// workspaceservice_test.go covers ListGames reading the Go registry, not SQLite.
package services

import "testing"

func TestListGamesFromRegistry(t *testing.T) {
	w := &WorkspaceService{}
	games, err := w.ListGames()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, g := range games {
		ids[g.ID] = true
	}
	for _, want := range []string{"ck3", "vic3", "eu5"} {
		if !ids[want] {
			t.Errorf("ListGames missing %s: %v", want, games)
		}
	}
}
