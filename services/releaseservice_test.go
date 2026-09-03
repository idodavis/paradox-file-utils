package services

import (
	"strings"
	"testing"
)

func TestConvertMarkdown(t *testing.T) {
	r := &ReleaseService{}
	got := r.Convert("# Hi\n", "bbcode")
	if !strings.Contains(got.Text, "[h1]Hi[/h1]") {
		t.Fatalf("%q", got.Text)
	}
}

func TestSteamPublishError(t *testing.T) {
	got := steamPublishError("SubmitItemUpdate EResult 9")
	if got == "" || !strings.Contains(got, "EResult 9") {
		t.Fatalf("eresult 9: %q", got)
	}
	msg := steamPublishError("SubmitItemUpdate EResult 25")
	if msg == "" || !strings.Contains(msg, "1 MB") {
		t.Fatalf("eresult 25: %q", msg)
	}
	if steamPublishError("SubmitItemUpdate EResult 2") != "" {
		t.Fatal("eresult 2 should not map")
	}
}

func TestSteamEResultCode(t *testing.T) {
	code, ok := steamEResultCode("CreateItem EResult 25")
	if !ok || code != 25 {
		t.Fatalf("got %d %v", code, ok)
	}
	if _, ok := steamEResultCode("no code here"); ok {
		t.Fatal("expected miss")
	}
}

func TestBumpVersion(t *testing.T) {
	r := &ReleaseService{}
	if got := r.BumpVersion("1.2.0", "patch"); got != "1.2.1" {
		t.Fatalf("%s", got)
	}
	if got := r.BumpVersion("1.2.3", "minor"); got != "1.3.0" {
		t.Fatalf("%s", got)
	}
}
