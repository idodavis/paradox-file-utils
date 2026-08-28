package services

import (
	"strings"
	"testing"
)

func TestParseWikiSectionsOR(t *testing.T) {
	wikitext := "Intro\n== OR ==\nAny trigger in this list is true.\n== limit ==\nMust be true.\n"
	sections := parseWikiSections(wikitext)
	if !strings.Contains(sections["or"], "Any trigger") {
		t.Fatalf("sections: %#v", sections)
	}
	if !strings.Contains(sections["limit"], "Must be true") {
		t.Fatalf("sections: %#v", sections)
	}
}
