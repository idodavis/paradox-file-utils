package game

import "testing"

func TestAll(t *testing.T) {
	got := All()
	if len(got) != 3 || got[0].ID != "ck3" || got[1].ID != "eu5" || got[2].ID != "vic3" {
		t.Fatalf("All ids: %v %v %v", got[0], got[1], got[2])
	}
	if Get("ck3") != got[0] || OriginVanilla != "vanilla" {
		t.Fatal("Get/OriginVanilla")
	}
}
