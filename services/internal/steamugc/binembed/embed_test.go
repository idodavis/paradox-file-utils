package binembed

import "testing"

func TestHelperEmbedded(t *testing.T) {
	if len(Helper()) < 1024 {
		t.Fatal("helper not embedded; run task common:build:steamugc")
	}
	if len(APILib()) < 1024 {
		t.Fatal("steam api lib not embedded; run task common:build:steamugc")
	}
}
